// Package modbridge is the convergence point between local JAR analysis (minecraft/global)
// and platform API analysis (providers/database). It provides version resolution,
// install-status determination, and display-layer platform metadata merging.
//
// Dependency direction: downloader -> modbridge -> {providers, database, global, minecraft}.
// modbridge must NOT import downloader.
package modbridge

import (
	"archive/zip"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"mod-downloader/database"
	"mod-downloader/global"
	"mod-downloader/logging"
	"mod-downloader/minecraft"
	"mod-downloader/models"
	"mod-downloader/providers"
	appstructs "mod-downloader/structs"
	mcstructs "mod-downloader/structs/minecraft"
)

// Button status constants shared by downloader and modbridge.
const (
	BtnStatusNew       = "new"       // No same modId in instance -> standard download
	BtnStatusInstalled = "installed" // Installed and same sha1 as latest -> disabled
	BtnStatusUpdate    = "update"    // Same modId same project but not latest -> update
	BtnStatusConflict  = "conflict"  // Same modId but not from this project's versions -> switch source
)

// ResolveVersion returns the best matching platform version for a download request.
func ResolveVersion(req appstructs.ModDownloadRequest) (models.ModVersion, bool) {
	versions := ResolveVersions(req)
	if len(versions) == 0 {
		return models.ModVersion{}, false
	}
	return versions[0], true
}

// ResolveVersions returns all matching platform versions for a download request,
// considering pinned versions first.
func ResolveVersions(req appstructs.ModDownloadRequest) []models.ModVersion {
	platform := strings.ToLower(strings.TrimSpace(req.Result.Platform))
	projectID := strings.TrimSpace(req.ProjectID)
	if platform == "" {
		platform, projectID = providers.SplitProjectReference(projectID)
	}
	if _, project := providers.SplitProjectReference(projectID); project != "" {
		projectID = project
	}

	if versionID := strings.TrimSpace(req.VersionID); versionID != "" {
		if version, found := FindVersionByID(providers.ListMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader), versionID); found {
			return []models.ModVersion{version}
		}
	}

	if pin, ok := database.GetPinnedMod(platform, projectID, req.MinecraftVersion, req.ModLoader); ok {
		if version, found := FindVersionByID(providers.ListMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader), pin.VersionID); found {
			return []models.ModVersion{version}
		}
	}

	versions := providers.ListMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	return versions
}

// FindVersionByID returns a version by its platform version ID from a list of versions.
func FindVersionByID(versions []models.ModVersion, versionID string) (models.ModVersion, bool) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return models.ModVersion{}, false
	}
	for _, version := range versions {
		if version.ID == versionID {
			return version, true
		}
	}
	return models.ModVersion{}, false
}

// ApplySelectedInstance overwrites the request's MinecraftVersion and ModLoader from
// the currently selected sidebar instance, and returns the resolved instanceID and targetDir.
// ok is false when no instance is selected.
func ApplySelectedInstance(req appstructs.ModDownloadRequest) (appstructs.ModDownloadRequest, string, string, bool) {
	selected := global.GetSelectedVersion()
	instanceID := versionInstanceID(selected)
	targetDir := selectedVersionModsDir(selected)
	if targetDir == "" || instanceID == "" {
		return req, instanceID, targetDir, false
	}
	if mcVersion := strings.TrimSpace(selected.MinecraftVersion); mcVersion != "" {
		req.MinecraftVersion = mcVersion
	}
	if loader := strings.ToLower(strings.TrimSpace(selected.ModLoader)); loader != "" && loader != "vanilla" {
		req.ModLoader = loader
	}
	return req, instanceID, targetDir, true
}

// InstallStatus determines the button status for a search result in the current instance
// without parsing remote JARs (used for search-list rendering).
func InstallStatus(req appstructs.ModDownloadRequest) string {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, _, ok := ApplySelectedInstance(req)
	if !ok || req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return BtnStatusNew
	}

	versions := ResolveVersions(req)
	if len(versions) == 0 {
		return BtnStatusNew
	}
	version := versions[0]

	localPaths := global.LocalModPathsInInstance(instanceID)
	if len(localPaths) == 0 {
		return BtnStatusNew
	}

	latestSHA1 := strings.ToLower(strings.TrimSpace(version.SHA1))
	if latestSHA1 != "" {
		for _, p := range localPaths {
			if strings.ToLower(strings.TrimSpace(p.FileSHA1)) == latestSHA1 {
				return BtnStatusInstalled
			}
		}
	}

	projectSHA1s := projectVersionSHA1Set(versions)
	hasProjectFile := false
	for _, p := range localPaths {
		if projectSHA1s[strings.ToLower(strings.TrimSpace(p.FileSHA1))] {
			hasProjectFile = true
			break
		}
	}
	if hasProjectFile {
		return BtnStatusUpdate
	}

	// Search-list rendering must not range-parse remote JARs; only use already persisted mod IDs.
	if modIDs := normalizedModIDs(version.ModIDs); len(modIDs) > 0 {
		for _, p := range LocalModPathsForModIDs(modIDs, instanceID) {
			if strings.ToLower(strings.TrimSpace(p.FileSHA1)) != latestSHA1 {
				return BtnStatusConflict
			}
		}
	}

	return BtnStatusNew
}

// InstallStatusPrecise may parse the remote jar to discover mod IDs.
// Use it for install-time dependency decisions, not for search-list button state.
func InstallStatusPrecise(req appstructs.ModDownloadRequest) string {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, _, ok := ApplySelectedInstance(req)
	if !ok || req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return BtnStatusNew
	}

	version, ok := ResolveVersion(req)
	if !ok {
		return BtnStatusNew
	}
	modIDs := VersionModIDs(version, req.ModLoader)
	if len(modIDs) == 0 {
		return BtnStatusNew
	}

	localPaths := LocalModPathsForModIDs(modIDs, instanceID)
	if len(localPaths) == 0 {
		return BtnStatusNew
	}

	latestSHA1 := strings.ToLower(strings.TrimSpace(version.SHA1))
	if latestSHA1 != "" {
		for _, p := range localPaths {
			if strings.ToLower(strings.TrimSpace(p.FileSHA1)) == latestSHA1 {
				return BtnStatusInstalled
			}
		}
	}

	projectSHA1s := providers.ProjectVersionSHA1Set(req.Result)
	for _, p := range localPaths {
		if projectSHA1s[strings.ToLower(strings.TrimSpace(p.FileSHA1))] {
			return BtnStatusUpdate
		}
	}
	return BtnStatusConflict
}

// DownloadStates returns button states for a list of search results.
func DownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))

	states := make([]appstructs.ModDownloadButtonState, len(req.Results))
	selected := global.GetSelectedVersion()
	instanceID := versionInstanceID(selected)
	if instanceID == "" || !global.HasLocalModPathsInInstance(instanceID) {
		for i := range req.Results {
			states[i] = defaultDownloadButtonState(req.Results[i])
		}
		return states
	}

	var wg sync.WaitGroup
	for i := range req.Results {
		result := req.Results[i]
		states[i] = defaultDownloadButtonState(result)
		if states[i].Key == "" {
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status := InstallStatus(appstructs.ModDownloadRequest{
				ProjectID:        states[idx].Key,
				Result:           result,
				MinecraftVersion: req.MinecraftVersion,
				ModLoader:        req.ModLoader,
			})
			applyButtonStatus(&states[idx], status)
		}(i)
	}
	wg.Wait()
	return states
}

// VersionModIDs returns the mod IDs for a platform version.
// It reads from the in-memory ModIDs field first, then the persisted DB cache,
// and only falls back to parsing the remote JAR (HTTP range requests) when both
// are empty. The DB read-back is what prevents duplicate range fetches when the
// same version struct is reused across calls (e.g. tryHardlinkInstall + downloadModJob).
func VersionModIDs(version models.ModVersion, modLoader string) []string {
	if modIDs := normalizedModIDs(version.ModIDs); len(modIDs) > 0 {
		return modIDs
	}
	if version.ID != "" {
		if persisted, err := database.GetVersionModIDs(version.ID); err == nil {
			if modIDs := normalizedModIDs(persisted); len(modIDs) > 0 {
				return modIDs
			}
		} else {
			logging.Warn("get version mod IDs from DB failed", "platformVersionID", version.ID, "error", err)
		}
	}

	mods, err := parseRemoteModJar(version.DownloadURL, modLoader)
	if err != nil {
		logging.Warn("remote jar parse failed for version mod IDs", "downloadURL", version.DownloadURL, "error", err)
		return nil
	}

	modIDs := make([]string, 0, len(mods))
	for _, m := range mods {
		if id := strings.TrimSpace(m.ID); id != "" {
			modIDs = append(modIDs, strings.ToLower(id))
		}
	}
	modIDs = deduplicateStrings(modIDs)

	if len(modIDs) > 0 && version.ID != "" {
		if err := database.SetVersionModIDs(version.ID, modIDs); err != nil {
			logging.Warn("persist version mod IDs failed", "platformVersionID", version.ID, "error", err)
		}
	}
	return modIDs
}

// PersistVersionModIDs persists modIDs for a platform version.
// Returns true if the IDs were persisted successfully.
func PersistVersionModIDs(platformVersionID string, modIDs []string) bool {
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" || len(modIDs) == 0 {
		return false
	}
	if err := database.SetVersionModIDs(platformVersionID, modIDs); err != nil {
		logging.Warn("persist version mod IDs failed", "platformVersionID", platformVersionID, "error", err)
		return false
	}
	return true
}

// PlatformMetadataForSHA1 looks up a platform version by SHA1 and returns the associated
// project and version info for display-layer merging. This does NOT write to any cache.
func PlatformMetadataForSHA1(sha1 string) (models.ModProject, models.ModVersion, bool) {
	sha1 = strings.ToLower(strings.TrimSpace(sha1))
	if sha1 == "" {
		return models.ModProject{}, models.ModVersion{}, false
	}

	// Try to find a platform version with this SHA1 from the database
	var version models.ModVersion
	var ok bool
	for _, platform := range []string{"curseforge", "modrinth"} {
		if version, ok = database.GetLatestProjectBySHA1(platform, sha1); ok {
			break
		}
	}
	if !ok {
		return models.ModProject{}, models.ModVersion{}, false
	}

	project, found := database.GetModPlatform(version.Platform, version.ProjectID)
	if !found {
		return models.ModProject{}, version, false
	}
	return project, version, true
}

// parseRemoteModJar downloads a JAR via HTTP range requests and parses its mod metadata.
func parseRemoteModJar(downloadURL string, modLoader string) ([]mcstructs.ModInfo, error) {
	reader, err := minecraft.NewHTTPRangeReaderAt(downloadURL)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return nil, err
	}
	mods := minecraft.ParseModZipReader(zr, downloadURL, modLoader)
	if len(mods) == 0 {
		return nil, fmt.Errorf("no parseable mod metadata")
	}
	return mods, nil
}

// --- private helpers ---

func defaultDownloadButtonState(result models.ModProject) appstructs.ModDownloadButtonState {
	return appstructs.ModDownloadButtonState{Key: providers.ProjectReferenceFromSearchResult(result), Status: BtnStatusNew, Icon: "mdi-download", Color: "primary"}
}

func applyButtonStatus(state *appstructs.ModDownloadButtonState, status string) {
	state.Status = status
	switch status {
	case BtnStatusInstalled:
		state.Disabled = true
		state.Icon = "mdi-check"
		state.Color = ""
	case BtnStatusUpdate:
		state.Icon = "mdi-arrow-up-bold-circle"
		state.Color = "warning"
	case BtnStatusConflict:
		state.Icon = "mdi-download"
		state.Color = "warning"
	default:
		state.Icon = "mdi-download"
		state.Color = "primary"
	}
}

func projectVersionSHA1Set(versions []models.ModVersion) map[string]bool {
	out := make(map[string]bool, len(versions))
	for _, version := range versions {
		if sha1 := strings.ToLower(strings.TrimSpace(version.SHA1)); sha1 != "" {
			out[sha1] = true
		}
	}
	return out
}

// LocalModPathsForModIDs returns local file paths in the instance that match any of the given mod IDs.
func LocalModPathsForModIDs(modIDs []string, instanceID string) []global.LocalModFilePath {
	out := make([]global.LocalModFilePath, 0)
	seen := make(map[string]struct{})
	for _, id := range modIDs {
		for _, p := range global.LocalModPathsInInstanceByModID(instanceID, id) {
			if _, dup := seen[p.Path]; dup {
				continue
			}
			seen[p.Path] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

// FilterFullyCoveredPaths 实现 jij 弱引用规则：仅保留那些 modID 集合被 newModIDs 完全覆盖的
// 已有本地 mod 路径。如果已安装 mod 提供了 newModIDs 中不存在的 modID，则跳过替换。
func FilterFullyCoveredPaths(newModIDs []string, existing []global.LocalModFilePath) []global.LocalModFilePath {
	if len(existing) == 0 {
		return existing
	}
	newSet := make(map[string]struct{}, len(newModIDs))
	for _, id := range newModIDs {
		if s := strings.ToLower(strings.TrimSpace(id)); s != "" {
			newSet[s] = struct{}{}
		}
	}

	out := make([]global.LocalModFilePath, 0, len(existing))
	for _, p := range existing {
		existingIDs := global.LocalModIDsBySHA1(p.FileSHA1)
		if modIDsCoveredBy(existingIDs, newSet) {
			out = append(out, p)
		} else {
			logging.Info("jij weak-ref: skip archive, existing mod has uncovered mod IDs",
				"path", p.Path, "existingModIDs", existingIDs, "newModIDs", newModIDs)
		}
	}
	return out
}

func modIDsCoveredBy(ids []string, superSet map[string]struct{}) bool {
	for _, id := range ids {
		if _, ok := superSet[id]; !ok {
			return false
		}
	}
	return true
}

func selectedVersionModsDir(selected mcstructs.VersionInfo) string {
	mcDir := global.GetMinecraftDir()
	versionDir := minecraft.VersionDirPath(mcDir, selected)
	if versionDir == "" {
		return ""
	}
	return filepath.Join(versionDir, "mods")
}

func versionInstanceID(version mcstructs.VersionInfo) string {
	if strings.TrimSpace(version.ID) != "" {
		return strings.TrimSpace(version.ID)
	}
	return strings.TrimSpace(version.Name)
}

func normalizedModIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if s := strings.ToLower(strings.TrimSpace(id)); s != "" {
			out = append(out, s)
		}
	}
	return deduplicateStrings(out)
}

func deduplicateStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
