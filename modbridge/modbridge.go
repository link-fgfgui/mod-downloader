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

	// Search-list rendering reads memory + DB cache only (no remote JAR parse).
	// On cache miss, marks the version for async backfill so a later refresh can
	// resolve the correct state without blocking the initial render.
	modIDs := resolveVersionModIDs(version)
	if len(modIDs) > 0 {
		for _, p := range LocalModPathsForModIDs(modIDs, instanceID) {
			if strings.ToLower(strings.TrimSpace(p.FileSHA1)) != latestSHA1 {
				return BtnStatusConflict
			}
		}
	} else {
		markBackfill(version, req.ModLoader)
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
// onBackfillComplete, when non-nil, is invoked once after any async remote
// mod-ID backfills triggered by cache misses finish; callers use it to notify
// the frontend to re-fetch states so the now-populated DB cache takes effect.
func DownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))

	states := make([]appstructs.ModDownloadButtonState, len(req.Results))
	selected := global.GetSelectedVersion()
	instanceID := versionInstanceID(selected)

	// R2: if the selected instance has never been scanned, scan it once now so
	// status decisions aren't falsely reported as "new" just because the local
	// mod index is empty. No TTL guard — already-scanned instances are skipped.
	if instanceID != "" && !global.HasLocalModPathsInInstance(instanceID) {
		ensureInstanceModsScanned(selected)
	}

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

	// R3: drain versions whose mod IDs were unavailable during this render and
	// backfill them asynchronously. The emitter fires once after all backfills
	// complete so the frontend can re-fetch with the now-warmed DB cache.
	backfill := drainPendingBackfills()
	if len(backfill) > 0 && onBackfillComplete != nil {
		go func() {
			for _, b := range backfill {
				backfillVersionModIDs(b.version, b.modLoader)
			}
			onBackfillComplete()
		}()
	}
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

	// Use only primary (strong-reference) mod IDs. JIJ / nested-jar entries must
	// not be stored as version identifiers since they would cause false conflicts
	// when the same modID appears as a JIJ in an unrelated host JAR.
	modIDs := minecraft.PrimaryModIDs(mods)

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

// resolveVersionModIDs returns version mod IDs using only synchronous sources:
// the in-memory ModIDs field first, then the persisted DB cache. It never
// triggers a remote JAR parse — use VersionModIDs for that. Intended for
// search-list InstallStatus where remote parsing must be deferred to an
// async backfill.
func resolveVersionModIDs(version models.ModVersion) []string {
	if modIDs := normalizedModIDs(version.ModIDs); len(modIDs) > 0 {
		return modIDs
	}
	if version.ID == "" {
		return nil
	}
	persisted, err := database.GetVersionModIDs(version.ID)
	if err != nil {
		logging.Warn("get version mod IDs from DB failed", "platformVersionID", version.ID, "error", err)
		return nil
	}
	return normalizedModIDs(persisted)
}

// --- async mod-ID backfill (R3) ---

// pendingBackfill records a version whose mod IDs were unavailable during a
// search-list render and should be resolved asynchronously.
type pendingBackfill struct {
	version   models.ModVersion
	modLoader string
}

var (
	backfillMu       sync.Mutex
	backfillInflight = make(map[string]struct{}) // key: version.ID — currently being parsed
	pendingBackfills []pendingBackfill
	pendingSet       = make(map[string]struct{}) // key: version.ID — already queued, not yet drained
)

// markBackfill queues a version for async mod-ID backfill. Deduplicates by
// version.ID against both the pending queue and the in-flight set so
// concurrent InstallStatus calls for the same version don't enqueue duplicate
// work within a batch or while a prior backfill is still running.
func markBackfill(version models.ModVersion, modLoader string) {
	if strings.TrimSpace(version.ID) == "" {
		return
	}
	backfillMu.Lock()
	defer backfillMu.Unlock()
	if _, queued := pendingSet[version.ID]; queued {
		return
	}
	if _, inflight := backfillInflight[version.ID]; inflight {
		return
	}
	pendingBackfills = append(pendingBackfills, pendingBackfill{version: version, modLoader: modLoader})
	pendingSet[version.ID] = struct{}{}
}

// drainPendingBackfills returns and clears the queued backfills. Called once
// per DownloadStates batch after all InstallStatus goroutines complete. The
// pending-set is cleared so a later batch can re-queue if the DB write failed.
func drainPendingBackfills() []pendingBackfill {
	backfillMu.Lock()
	defer backfillMu.Unlock()
	out := pendingBackfills
	pendingBackfills = nil
	pendingSet = make(map[string]struct{})
	return out
}

// backfillVersionModIDs resolves a version's mod IDs via VersionModIDs (which
// may parse the remote JAR and writes the result back to the DB cache) under
// an in-flight guard so the same version.ID is never parsed concurrently.
func backfillVersionModIDs(version models.ModVersion, modLoader string) {
	if strings.TrimSpace(version.ID) == "" {
		return
	}
	backfillMu.Lock()
	if _, inflight := backfillInflight[version.ID]; inflight {
		backfillMu.Unlock()
		return
	}
	backfillInflight[version.ID] = struct{}{}
	backfillMu.Unlock()
	defer func() {
		backfillMu.Lock()
		delete(backfillInflight, version.ID)
		backfillMu.Unlock()
	}()

	_ = VersionModIDs(version, modLoader)
}

// --- local mod index refresh (R2) ---

// ensureInstanceModsScanned scans the selected instance's mods directory when
// its local mod index is empty, populating global.LocalModFilePaths so
// InstallStatus has local data to compare against. Self-contained in
// modbridge (does not touch app.go's version-list cache).
func ensureInstanceModsScanned(selected mcstructs.VersionInfo) {
	instanceID := versionInstanceID(selected)
	if instanceID == "" {
		return
	}
	mcDir := global.GetMinecraftDir()
	versionDir := minecraft.VersionDirPath(mcDir, selected)
	if versionDir == "" {
		return
	}
	minecraft.ScanVersionMods(versionDir, instanceID, selected.MinecraftVersion, selected.ModLoader, mcDir)
}

// PlatformMetadataForSHA1 looks up a platform version by SHA1 and returns the associated
// project and version info for display-layer merging. This does NOT write to any cache.
func PlatformMetadataForSHA1(sha1 string) (models.ModProject, models.ModVersion, bool) {
	sha1 = strings.ToLower(strings.TrimSpace(sha1))
	if sha1 == "" {
		return models.ModProject{}, models.ModVersion{}, false
	}

	// Try to find any cached platform version with this SHA1.
	var version models.ModVersion
	var ok bool
	for _, platform := range []string{"curseforge", "modrinth"} {
		if version, ok = database.GetVersionBySHA1(platform, sha1); ok {
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
