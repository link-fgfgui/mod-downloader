// Package modbridge is the convergence point between local JAR analysis (minecraft/global)
// and platform API analysis (providers/database). It provides version resolution,
// install-status determination, and display-layer platform metadata merging.
//
// Dependency direction: downloader -> modbridge -> {providers, database, global, minecraft}.
// modbridge must NOT import downloader.
package modbridge

import (
	"archive/zip"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

// Button status constants shared by downloader and modbridge.
const (
	BtnStatusNew             = "new"          // No same modId in instance -> standard download
	BtnStatusInstalled       = "installed"    // Installed and same sha1 as latest -> disabled
	BtnStatusUpdate          = "update"       // Same modId same project but not latest -> update
	BtnStatusConflict        = "conflict"     // Same modId but not from this project's versions -> switch source
	BtnStatusIncompatible    = "incompatible" // Requested version declares an incompatible installed project
	BtnStatusMetadataLoading = "metadataLoading"
	BtnStatusMetadataFailed  = "metadataFailed"
)

const (
	remoteModIDFailureTTL = 10 * time.Second
	remoteModJarTimeout   = 45 * time.Second
)

type remoteModIDCacheEntry struct {
	modIDs    []string
	expiresAt time.Time
	loading   bool
	ready     chan struct{}
}

var remoteModIDCache = struct {
	sync.Mutex
	entries map[string]*remoteModIDCacheEntry
}{entries: make(map[string]*remoteModIDCacheEntry)}

var simpleMode atomic.Bool

// preferredReleaseType selects the release channel automatic installs prefer:
// 0 = latest (no channel filter), 1 = release, 2 = beta, 3 = alpha.
var preferredReleaseType atomic.Int32

func init() {
	preferredReleaseType.Store(0)
}

func ConfigurePreferredReleaseType(value string) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "alpha":
		preferredReleaseType.Store(3)
	case "beta":
		preferredReleaseType.Store(2)
	case "release":
		preferredReleaseType.Store(1)
	default:
		preferredReleaseType.Store(0)
	}
}

func preferredReleaseTypeMatches(version models.ModVersion) bool {
	typeID := preferredReleaseType.Load()
	if typeID == 0 {
		return true
	}
	value := strings.ToLower(strings.TrimSpace(version.ReleaseType))
	if value == "" {
		return typeID == 1
	}
	switch typeID {
	case 3:
		return value == "alpha"
	case 2:
		return value == "beta"
	default:
		return value == "release"
	}
}

func preferredVersions(versions []models.ModVersion) []models.ModVersion {
	filtered := make([]models.ModVersion, 0, len(versions))
	for _, version := range versions {
		if preferredReleaseTypeMatches(version) {
			filtered = append(filtered, version)
		}
	}
	if len(filtered) == 0 {
		return versions
	}
	return filtered
}

var errSimpleMode = errors.New("remote mod ID resolution disabled by simple mode")

var parseRemoteModJarForIDs = parseRemoteModJar

type remoteModIDConcurrencyGate struct {
	cond   *sync.Cond
	limit  int
	active int
}

func newRemoteModIDConcurrencyGate(limit int) *remoteModIDConcurrencyGate {
	gate := &remoteModIDConcurrencyGate{limit: normalizedRemoteModIDConcurrency(limit)}
	gate.cond = sync.NewCond(&sync.Mutex{})
	return gate
}

func normalizedRemoteModIDConcurrency(limit int) int {
	if limit <= 0 {
		return 1
	}
	return limit
}

func (g *remoteModIDConcurrencyGate) configure(limit int) {
	limit = normalizedRemoteModIDConcurrency(limit)
	g.cond.L.Lock()
	g.limit = limit
	g.cond.Broadcast()
	g.cond.L.Unlock()
}

func (g *remoteModIDConcurrencyGate) acquire() func() {
	g.cond.L.Lock()
	for g.active >= g.limit {
		g.cond.Wait()
	}
	g.active++
	g.cond.L.Unlock()

	return func() {
		g.cond.L.Lock()
		g.active--
		g.cond.Broadcast()
		g.cond.L.Unlock()
	}
}

var remoteModIDGate = newRemoteModIDConcurrencyGate(1)

// ConfigureRemoteModIDConcurrency updates the maximum number of uncached
// remote JAR metadata parses that may run concurrently. Existing parses are
// allowed to finish; waiting parses use the new limit.
func ConfigureRemoteModIDConcurrency(limit int) {
	remoteModIDGate.configure(limit)
}

// ConfigureSimpleMode applies the process-wide policy for remote mod ID
// resolution. Enabling it drops queued backfills; a parse already inside the
// HTTP parser is allowed to finish, while all later callers are rejected.
func ConfigureSimpleMode(enabled bool) {
	simpleMode.Store(enabled)
	if !enabled {
		return
	}
	backfillMu.Lock()
	pendingBackfills = nil
	pendingSet = make(map[string]struct{})
	backfillMu.Unlock()
}

func SimpleModeEnabled() bool {
	return simpleMode.Load()
}

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

	if pin, ok := storage.GetPinnedMod(platform, projectID, req.MinecraftVersion, req.ModLoader); ok {
		if version, found := FindVersionByID(providers.ListMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader), pin.VersionID); found {
			return []models.ModVersion{version}
		}
	}

	versions := providers.ListMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	return preferredVersions(versions)
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

// ResolveInstallTarget resolves the concrete mods directory and local-index
// instance identity for a download request. Explicit request fields are used
// first for CLI/direct-target workflows; selected GUI instance state is only a
// compatibility fallback.
func ResolveInstallTarget(req appstructs.ModDownloadRequest) (appstructs.ModDownloadRequest, string, string, bool) {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))
	req.TargetDir = cleanPath(req.TargetDir)
	req.InstanceID = strings.TrimSpace(req.InstanceID)

	if req.TargetDir != "" {
		if req.MinecraftVersion == "" || req.ModLoader == "" {
			return req, req.InstanceID, req.TargetDir, false
		}
		if req.InstanceID == "" {
			req.InstanceID = InstanceIDForTargetDir(req.TargetDir)
		}
		if req.InstanceID == "" {
			return req, "", req.TargetDir, false
		}
		return req, req.InstanceID, req.TargetDir, true
	}

	return ApplySelectedInstance(req)
}

func InstanceIDForTargetDir(targetDir string) string {
	targetDir = cleanPath(targetDir)
	if targetDir == "" {
		return ""
	}
	if !filepath.IsAbs(targetDir) {
		if abs, err := filepath.Abs(targetDir); err == nil {
			targetDir = abs
		}
	}
	return "cli:" + filepath.Clean(targetDir)
}

// InstallStatus determines the button status for a search result in the current instance
// without parsing remote JARs (used for search-list rendering).
func InstallStatus(req appstructs.ModDownloadRequest) string {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, targetDir, ok := ResolveInstallTarget(req)
	if !ok || req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return BtnStatusNew
	}
	ensureTargetModsScanned(targetDir, instanceID, req.MinecraftVersion, req.ModLoader)

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

	if len(IncompatibleConflictsCached(req)) > 0 {
		return BtnStatusIncompatible
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
		return remoteMetadataButtonStatus(version, req.ModLoader)
	}

	return BtnStatusNew
}

// InstallStatusPrecise may parse the remote jar to discover mod IDs.
// Use it for install-time dependency decisions, not for search-list button state.
func InstallStatusPrecise(req appstructs.ModDownloadRequest) string {
	if SimpleModeEnabled() {
		return BtnStatusNew
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, targetDir, ok := ResolveInstallTarget(req)
	if !ok || req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return BtnStatusNew
	}
	ensureTargetModsScanned(targetDir, instanceID, req.MinecraftVersion, req.ModLoader)

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

// IncompatibleConflicts resolves a request's platform-level incompatible
// dependencies against local installed mods in the target instance.
func IncompatibleConflicts(req appstructs.ModDownloadRequest) []appstructs.IncompatibleConflict {
	if SimpleModeEnabled() {
		return nil
	}
	return incompatibleConflicts(req, true)
}

// IncompatibleConflictsCached is the search-list variant. It never parses
// remote jars; missing mod IDs are queued for the existing async backfill.
func IncompatibleConflictsCached(req appstructs.ModDownloadRequest) []appstructs.IncompatibleConflict {
	if SimpleModeEnabled() {
		return nil
	}
	return incompatibleConflicts(req, false)
}

func incompatibleConflicts(req appstructs.ModDownloadRequest, precise bool) []appstructs.IncompatibleConflict {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, targetDir, ok := ResolveInstallTarget(req)
	if !ok || req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return nil
	}
	ensureTargetModsScanned(targetDir, instanceID, req.MinecraftVersion, req.ModLoader)

	version, ok := ResolveVersion(req)
	if !ok {
		return nil
	}
	if precise {
		version = hydrateDependencyMetadata(req, version, hasIncompatibleDependency)
	}
	mainTitle := projectTitle(req.Result, req.ProjectID)
	mainKey := providers.ProjectReferenceFromSearchResult(req.Result)
	if mainKey == "" {
		mainKey = req.ProjectID
	}

	conflicts := make([]appstructs.IncompatibleConflict, 0)
	seenPaths := make(map[string]struct{})
	for _, dep := range version.Dependencies {
		if !IsDependencyType(dep, "incompatible") {
			continue
		}
		depReq, ok := DependencyDownloadRequest(version.Platform, dep, req)
		if !ok {
			logging.Warn("incompatible dependency cannot be resolved", "platform", version.Platform, "projectID", dep.DependencyProjectID, "versionID", dep.DependencyVersionID, "type", dep.DependencyType)
			continue
		}
		depVersion, ok := ResolveVersion(depReq)
		if !ok {
			logging.Warn("incompatible dependency version cannot be resolved", "platform", depReq.Result.Platform, "projectID", depReq.ProjectID, "versionID", dep.DependencyVersionID)
			continue
		}
		var modIDs []string
		if precise {
			modIDs = VersionModIDs(depVersion, req.ModLoader)
		} else {
			modIDs = resolveVersionModIDs(depVersion)
			if len(modIDs) == 0 {
				markBackfill(depVersion, req.ModLoader)
			}
		}
		if len(modIDs) == 0 {
			continue
		}
		paths := LocalModPathsForModIDs(modIDs, instanceID)
		if len(paths) == 0 {
			continue
		}
		localMods := make([]appstructs.IncompatibleLocalMod, 0, len(paths))
		for _, path := range paths {
			if _, dup := seenPaths[path.Path]; dup {
				continue
			}
			seenPaths[path.Path] = struct{}{}
			localMods = append(localMods, appstructs.IncompatibleLocalMod{
				Path:     path.Path,
				FileName: filepath.Base(path.Path),
				SHA1:     path.FileSHA1,
			})
		}
		if len(localMods) == 0 {
			continue
		}
		depKey := providers.ProjectReferenceFromSearchResult(depReq.Result)
		if depKey == "" {
			depKey = depReq.ProjectID
		}
		conflicts = append(conflicts, appstructs.IncompatibleConflict{
			ProjectKey:             mainKey,
			ProjectTitle:           mainTitle,
			IncompatibleProjectKey: depKey,
			IncompatibleTitle:      projectTitle(depReq.Result, depKey),
			IncompatibleVersionID:  depReq.VersionID,
			Paths:                  localMods,
		})
	}
	return conflicts
}

func AnalyzeBatchIncompatibleConflicts(req appstructs.BatchDownloadRequest) appstructs.BatchIncompatibleAnalysis {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))
	req.TargetDir = cleanPath(req.TargetDir)
	req.InstanceID = strings.TrimSpace(req.InstanceID)

	out := appstructs.BatchIncompatibleAnalysis{}
	for _, result := range req.Results {
		key := providers.ProjectReferenceFromSearchResult(result)
		if key == "" {
			continue
		}
		conflicts := IncompatibleConflicts(appstructs.ModDownloadRequest{
			ProjectID:        key,
			Result:           result,
			MinecraftVersion: req.MinecraftVersion,
			ModLoader:        req.ModLoader,
			TargetDir:        req.TargetDir,
			InstanceID:       req.InstanceID,
		})
		if len(conflicts) == 0 {
			continue
		}
		out.Conflicts = append(out.Conflicts, appstructs.BatchIncompatibleConflict{
			Key:       key,
			Title:     projectTitle(result, key),
			Result:    result,
			Conflicts: conflicts,
		})
	}
	return out
}

// DownloadStates returns button states for a list of search results.
// onBackfillComplete, when non-nil, is invoked once after any async remote
// mod-ID backfills triggered by cache misses finish; callers use it to notify
// the frontend to re-fetch states so the now-populated DB cache takes effect.
func DownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))
	req.TargetDir = cleanPath(req.TargetDir)
	req.InstanceID = strings.TrimSpace(req.InstanceID)
	if SimpleModeEnabled() {
		states := make([]appstructs.ModDownloadButtonState, len(req.Results))
		for i := range req.Results {
			states[i] = defaultDownloadButtonState(req.Results[i])
		}
		return states
	}

	states := make([]appstructs.ModDownloadButtonState, len(req.Results))
	targetReq, instanceID, targetDir, ok := ResolveInstallTarget(appstructs.ModDownloadRequest{
		MinecraftVersion: req.MinecraftVersion,
		ModLoader:        req.ModLoader,
		TargetDir:        req.TargetDir,
		InstanceID:       req.InstanceID,
	})
	req.MinecraftVersion = targetReq.MinecraftVersion
	req.ModLoader = targetReq.ModLoader

	if ok {
		ensureTargetModsScanned(targetDir, instanceID, req.MinecraftVersion, req.ModLoader)
	}

	if !ok || instanceID == "" || !global.HasLocalModPathsInInstance(instanceID) {
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
			downloadReq := appstructs.ModDownloadRequest{
				ProjectID:        states[idx].Key,
				Result:           result,
				MinecraftVersion: req.MinecraftVersion,
				ModLoader:        req.ModLoader,
				TargetDir:        targetDir,
				InstanceID:       instanceID,
			}
			status := InstallStatus(downloadReq)
			applyButtonStatus(&states[idx], status)
			if status == BtnStatusConflict {
				states[idx].ConflictFileName = conflictingLocalFileName(downloadReq, instanceID)
			}
		}(i)
	}
	wg.Wait()

	// R3: drain versions whose mod IDs were unavailable during this render and
	// backfill them asynchronously. The emitter fires once after all backfills
	// complete so the frontend can re-fetch with the now-warmed DB cache.
	backfill := drainPendingBackfills()
	if len(backfill) > 0 && onBackfillComplete != nil {
		go func() {
			runRemoteModIDBackfills(backfill)
			onBackfillComplete()
		}()
	}
	return states
}

func conflictingLocalFileName(req appstructs.ModDownloadRequest, instanceID string) string {
	versions := ResolveVersions(req)
	if len(versions) == 0 {
		return ""
	}
	latestSHA1 := strings.ToLower(strings.TrimSpace(versions[0].SHA1))
	for _, path := range LocalModPathsForModIDs(resolveVersionModIDs(versions[0]), instanceID) {
		if strings.ToLower(strings.TrimSpace(path.FileSHA1)) != latestSHA1 {
			return filepath.Base(path.Path)
		}
	}
	return ""
}

// VersionModIDs returns the mod IDs for a platform version.
// It reads from the in-memory ModIDs field first, then the persisted DB cache,
// and only falls back to parsing the remote JAR (HTTP range requests) when both
// are empty. The DB read-back is what prevents duplicate range fetches when the
// same version struct is reused across calls (e.g. tryHardlinkInstall + downloadModJob).
func VersionModIDs(version models.ModVersion, modLoader string) []string {
	if SimpleModeEnabled() {
		return nil
	}
	if modIDs := normalizedModIDs(version.ModIDs); len(modIDs) > 0 {
		return modIDs
	}
	if version.ID != "" {
		if persisted, err := storage.GetVersionModIDs(version.ID); err == nil {
			if modIDs := normalizedModIDs(persisted); len(modIDs) > 0 {
				return modIDs
			}
		} else {
			logging.Warn("get version mod IDs from DB failed", "platformVersionID", version.ID, "error", err)
		}
	}

	downloadURL := providers.EffectiveDownloadURL(version)
	modIDs, err := cachedRemoteModIDs(downloadURL, modLoader)
	if err != nil {
		logging.Warn("remote jar parse failed for version mod IDs", "downloadURL", downloadURL, "error", err)
		return nil
	}

	if len(modIDs) > 0 && version.ID != "" {
		if err := storage.SetVersionModIDs(version.ID, modIDs); err != nil {
			logging.Warn("persist version mod IDs failed", "platformVersionID", version.ID, "error", err)
		}
	}
	return modIDs
}

// cachedRemoteModIDs deduplicates remote JAR parsing by URL and loader. This
// covers both concurrent status checks and different platform versions that
// point at the same physical file. Failed parses are cached briefly so a
// repeated UI refresh cannot immediately replay the same timed-out requests.
func cachedRemoteModIDs(downloadURL, modLoader string) ([]string, error) {
	key := remoteModIDCacheKey(downloadURL, modLoader)
	if strings.TrimSpace(downloadURL) == "" {
		return nil, fmt.Errorf("empty download url")
	}
	for {
		if SimpleModeEnabled() {
			return nil, errSimpleMode
		}
		remoteModIDCache.Lock()
		entry := remoteModIDCache.entries[key]
		if entry != nil {
			if entry.loading {
				ready := entry.ready
				remoteModIDCache.Unlock()
				<-ready
				continue
			}
			if entry.expiresAt.IsZero() || time.Now().Before(entry.expiresAt) {
				modIDs := append([]string(nil), entry.modIDs...)
				remoteModIDCache.Unlock()
				if len(modIDs) == 0 {
					return nil, fmt.Errorf("remote jar metadata unavailable (cached)")
				}
				return modIDs, nil
			}
		}
		entry = &remoteModIDCacheEntry{loading: true, ready: make(chan struct{})}
		remoteModIDCache.entries[key] = entry
		remoteModIDCache.Unlock()

		release := remoteModIDGate.acquire()
		if SimpleModeEnabled() {
			release()
			remoteModIDCache.Lock()
			if current := remoteModIDCache.entries[key]; current == entry {
				delete(remoteModIDCache.entries, key)
				entry.loading = false
				close(entry.ready)
			}
			remoteModIDCache.Unlock()
			return nil, errSimpleMode
		}
		mods, err := func() ([]mcstructs.ModInfo, error) {
			defer release()
			return parseRemoteModJarForIDs(downloadURL, modLoader)
		}()
		modIDs := minecraft.PrimaryModIDs(mods)
		remoteModIDCache.Lock()
		entry.modIDs = append([]string(nil), modIDs...)
		if err != nil || len(modIDs) == 0 {
			entry.expiresAt = time.Now().Add(remoteModIDFailureTTL)
		}
		entry.loading = false
		close(entry.ready)
		remoteModIDCache.Unlock()
		if err != nil {
			return nil, err
		}
		if len(modIDs) == 0 {
			return nil, fmt.Errorf("no parseable mod metadata")
		}
		return modIDs, nil
	}
}

func remoteModIDCacheKey(downloadURL, modLoader string) string {
	return strings.TrimSpace(downloadURL) + "\x00" + strings.ToLower(strings.TrimSpace(modLoader))
}

func remoteModIDBackfillSuppressed(downloadURL, modLoader string) bool {
	key := remoteModIDCacheKey(downloadURL, modLoader)
	remoteModIDCache.Lock()
	defer remoteModIDCache.Unlock()
	entry := remoteModIDCache.entries[key]
	return entry != nil && !entry.loading && len(entry.modIDs) == 0 && time.Now().Before(entry.expiresAt)
}

func remoteMetadataButtonStatus(version models.ModVersion, modLoader string) string {
	downloadURL := providers.EffectiveDownloadURL(version)
	if strings.TrimSpace(downloadURL) == "" {
		return BtnStatusMetadataFailed
	}
	key := remoteModIDCacheKey(downloadURL, modLoader)
	remoteModIDCache.Lock()
	entry := remoteModIDCache.entries[key]
	if entry != nil && !entry.loading && len(entry.modIDs) == 0 && time.Now().Before(entry.expiresAt) {
		remoteModIDCache.Unlock()
		return BtnStatusMetadataFailed
	}
	remoteModIDCache.Unlock()
	return BtnStatusMetadataLoading
}

// PersistVersionModIDs persists modIDs for a platform version.
// Returns true if the IDs were persisted successfully.
func PersistVersionModIDs(platformVersionID string, modIDs []string) bool {
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" || len(modIDs) == 0 {
		return false
	}
	if err := storage.SetVersionModIDs(platformVersionID, modIDs); err != nil {
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
	persisted, err := storage.GetVersionModIDs(version.ID)
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
	if SimpleModeEnabled() {
		return
	}
	if strings.TrimSpace(version.ID) == "" {
		return
	}
	if remoteModIDBackfillSuppressed(version.DownloadURL, modLoader) {
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
	if SimpleModeEnabled() {
		return
	}
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

func runRemoteModIDBackfills(backfills []pendingBackfill) {
	if SimpleModeEnabled() {
		return
	}
	var wg sync.WaitGroup
	for _, backfill := range backfills {
		wg.Add(1)
		go func(item pendingBackfill) {
			defer wg.Done()
			backfillVersionModIDs(item.version, item.modLoader)
		}(backfill)
	}
	wg.Wait()
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
	modLoader := selected.ActualModLoader
	if strings.TrimSpace(modLoader) == "" {
		modLoader = selected.ModLoader
	}
	minecraft.ScanVersionMods(versionDir, instanceID, selected.MinecraftVersion, modLoader, mcDir)
}

func ensureTargetModsScanned(targetDir, instanceID, minecraftVersion, modLoader string) {
	instanceID = strings.TrimSpace(instanceID)
	targetDir = cleanPath(targetDir)
	if instanceID == "" || targetDir == "" || global.HasLocalModPathsInInstance(instanceID) {
		return
	}
	minecraft.ScanModsDir(targetDir, instanceID, minecraftVersion, modLoader, "")
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
		if version, ok = storage.GetVersionBySHA1(platform, sha1); ok {
			break
		}
	}
	if !ok {
		return models.ModProject{}, models.ModVersion{}, false
	}

	project, found := storage.GetModPlatform(version.Platform, version.ProjectID)
	if !found {
		return models.ModProject{}, version, false
	}
	return project, version, true
}

// ApplyPlatformMetadataToModInfo adds online display metadata to local JAR metadata
// without replacing local identity fields used by install/status logic.
func ApplyPlatformMetadataToModInfo(info mcstructs.ModInfo, project models.ModProject, version models.ModVersion) mcstructs.ModInfo {
	if title := strings.TrimSpace(project.Title); title != "" {
		info.OnlineName = title
	}
	if iconURL := strings.TrimSpace(project.IconURL); iconURL != "" {
		info.IconURL = iconURL
	}
	if platform := strings.TrimSpace(project.Platform); platform != "" {
		info.OnlinePlatform = platform
	}
	if projectID := strings.TrimSpace(project.ProjectID); projectID != "" {
		info.OnlineProjectID = projectID
	}
	if slug := strings.TrimSpace(project.Slug); slug != "" {
		info.OnlineSlug = slug
	}
	if onlineVersion := strings.TrimSpace(version.Version); onlineVersion != "" {
		info.OnlineVersion = onlineVersion
	}
	if versionID := strings.TrimSpace(version.ID); versionID != "" {
		info.OnlineVersionID = versionID
	} else if versionID := strings.TrimSpace(version.VersionID); versionID != "" {
		info.OnlineVersionID = versionID
	}
	if fileName := strings.TrimSpace(version.FileName); fileName != "" {
		info.OnlineFileName = fileName
	}
	if categories := models.NormalizeCategories(project.Categories); len(categories) > 0 {
		info.Categories = categories
	}
	return info
}

func ApplyPlatformMetadataToModInfos(mods []mcstructs.ModInfo, project models.ModProject, version models.ModVersion) []mcstructs.ModInfo {
	if len(mods) == 0 {
		return nil
	}
	out := make([]mcstructs.ModInfo, len(mods))
	for i, mod := range mods {
		out[i] = ApplyPlatformMetadataToModInfo(mod, project, version)
	}
	return out
}

// ApplyProjectMetadataToModInfo preserves the project-only API for existing core consumers.
func ApplyProjectMetadataToModInfo(info mcstructs.ModInfo, project models.ModProject) mcstructs.ModInfo {
	return ApplyPlatformMetadataToModInfo(info, project, models.ModVersion{})
}

// ApplyProjectMetadataToModInfos preserves the project-only API for existing core consumers.
func ApplyProjectMetadataToModInfos(mods []mcstructs.ModInfo, project models.ModProject) []mcstructs.ModInfo {
	return ApplyPlatformMetadataToModInfos(mods, project, models.ModVersion{})
}

// parseRemoteModJar downloads a JAR via HTTP range requests and parses its mod metadata.
func parseRemoteModJar(downloadURL string, modLoader string) ([]mcstructs.ModInfo, error) {
	started := time.Now()
	reader, err := minecraft.NewHTTPRangeReaderAtWithTimeout(downloadURL, remoteModJarTimeout)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return nil, err
	}
	mods := minecraft.ParseModZipReader(zr, downloadURL, modLoader)
	requests, bytesFetched := reader.RangeStats()
	logging.Debug("remote jar metadata parsed", "url", downloadURL, "host", remoteHost(downloadURL), "size", reader.Size(), "rangeRequests", requests, "rangeBytes", bytesFetched, "modCount", len(mods), "elapsedMs", time.Since(started).Milliseconds())
	if len(mods) == 0 {
		return nil, fmt.Errorf("no parseable mod metadata")
	}
	return mods, nil
}

func remoteHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
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
	case BtnStatusIncompatible:
		state.Icon = "mdi-alert-octagon-outline"
		state.Color = "error"
	case BtnStatusMetadataLoading:
		state.Icon = "mdi-progress-clock"
		state.Color = "surface-variant"
	case BtnStatusMetadataFailed:
		state.Icon = "mdi-refresh"
		state.Color = "surface-variant"
	default:
		state.Icon = "mdi-download"
		state.Color = "primary"
	}
}

func DependencyDownloadRequest(platform string, dep models.ModDependency, parent appstructs.ModDownloadRequest) (appstructs.ModDownloadRequest, bool) {
	projectID := strings.TrimSpace(dep.DependencyProjectID)
	platform = strings.ToLower(strings.TrimSpace(platform))
	if projectID == "" || (platform != "curseforge" && platform != "modrinth") {
		return appstructs.ModDownloadRequest{}, false
	}

	ref := platform + ":" + projectID
	project := models.ModProject{
		ID:        ref,
		Platform:  platform,
		ProjectID: projectID,
	}
	if cached, ok := providers.GetProjectByID(ref); ok {
		project = cached
		if strings.TrimSpace(project.ID) == "" {
			project.ID = ref
		}
		if strings.TrimSpace(project.Platform) == "" {
			project.Platform = platform
		}
		if strings.TrimSpace(project.ProjectID) == "" {
			project.ProjectID = projectID
		}
	}
	return appstructs.ModDownloadRequest{
		ProjectID:        ref,
		Result:           project,
		MinecraftVersion: parent.MinecraftVersion,
		ModLoader:        parent.ModLoader,
		VersionID:        strings.TrimSpace(dep.DependencyVersionID),
		TargetDir:        parent.TargetDir,
		InstanceID:       parent.InstanceID,
	}, true
}

func IsDependencyType(dep models.ModDependency, dependencyType string) bool {
	return strings.EqualFold(strings.TrimSpace(dep.DependencyType), dependencyType)
}

func hydrateDependencyMetadata(req appstructs.ModDownloadRequest, version models.ModVersion, match func([]models.ModDependency) bool) models.ModVersion {
	if match(version.Dependencies) {
		return version
	}
	refreshed := providers.RefreshMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	if refreshedVersion, found := FindVersionByID(refreshed, version.ID); found && match(refreshedVersion.Dependencies) {
		return refreshedVersion
	}
	return version
}

func hasIncompatibleDependency(deps []models.ModDependency) bool {
	for _, dep := range deps {
		if IsDependencyType(dep, "incompatible") {
			return true
		}
	}
	return false
}

func projectTitle(project models.ModProject, fallback string) string {
	if title := strings.TrimSpace(project.Title); title != "" {
		return title
	}
	if slug := strings.TrimSpace(project.Slug); slug != "" {
		return slug
	}
	if projectID := strings.TrimSpace(project.ProjectID); projectID != "" {
		return projectID
	}
	if fallback = strings.TrimSpace(fallback); fallback != "" {
		return fallback
	}
	return "Mod"
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

func cleanPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
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
