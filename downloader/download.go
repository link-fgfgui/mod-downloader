package downloader

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mod-downloader/global"
	"mod-downloader/logging"
	"mod-downloader/minecraft"
	"mod-downloader/modbridge"
	"mod-downloader/models"
	"mod-downloader/providers"
	appstructs "mod-downloader/structs"

	"github.com/cavaliergopher/grab/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const downloadQueueUpdatedEvent = "download-queue-updated"
const downloadFailedEvent = "download-failed"

type downloadJob struct {
	ID               string
	Version          models.ModVersion
	Result           models.ModProject
	TargetDir        string
	InstanceID       string
	MinecraftVersion string
	ModLoader        string
	cancel           context.CancelFunc
}

var downloadQueue = struct {
	sync.Mutex
	nextID  int64
	pending []downloadJob
	running bool
	current *downloadJob
}{}

func QueueModDownload(ctx context.Context, req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
	return queueModDownload(ctx, req, make(map[string]bool))
}

func queueModDownload(ctx context.Context, req appstructs.ModDownloadRequest, visited map[string]bool) appstructs.ModDownloadResult {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, targetDir, ok := modbridge.ApplySelectedInstance(req)
	if !ok {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "no selected version"}
	}
	if req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "invalid request"}
	}

	version, ok := modbridge.ResolveVersion(req)
	if !ok {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "no matching version"}
	}
	version = hydrateRequiredDependencies(req, version)
	if version.DownloadURL == "" {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "missing download url"}
	}
	if visited == nil {
		visited = make(map[string]bool)
	}
	if key := projectVersionJobKey(version); key != "" {
		if visited[key] {
			return appstructs.ModDownloadResult{Skipped: true, Reason: "already queued"}
		}
		visited[key] = true
	}

	queueMissingRequiredDependencies(ctx, req, version, instanceID, visited)
	enqueueDownload(ctx, downloadJob{
		Version:          version,
		Result:           req.Result,
		TargetDir:        targetDir,
		InstanceID:       instanceID,
		MinecraftVersion: req.MinecraftVersion,
		ModLoader:        req.ModLoader,
	})
	return appstructs.ModDownloadResult{
		Queued:    true,
		FileName:  version.FileName,
		VersionID: version.ID,
	}
}

func GetDownloadQueueState() appstructs.DownloadQueueState {
	return currentDownloadQueueState()
}

func CancelDownload(ctx context.Context, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}

	downloadQueue.Lock()
	for i, job := range downloadQueue.pending {
		if job.ID != id {
			continue
		}
		downloadQueue.pending = append(downloadQueue.pending[:i], downloadQueue.pending[i+1:]...)
		downloadQueue.Unlock()
		emitDownloadQueueState(ctx)
		return true
	}

	var cancel context.CancelFunc
	if downloadQueue.current != nil && downloadQueue.current.ID == id {
		cancel = downloadQueue.current.cancel
	}
	downloadQueue.Unlock()

	if cancel == nil {
		return false
	}
	cancel()
	emitDownloadQueueState(ctx)
	return true
}

func GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
	return modbridge.DownloadStates(req)
}

func projectVersionJobKey(version models.ModVersion) string {
	platform := strings.ToLower(strings.TrimSpace(version.Platform))
	projectID := strings.TrimSpace(version.ProjectID)
	versionID := strings.TrimSpace(version.ID)
	if platform == "" && projectID == "" && versionID == "" {
		return ""
	}
	return platform + ":" + projectID + ":" + versionID
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func enqueueDownload(ctx context.Context, job downloadJob) {
	downloadQueue.Lock()
	downloadQueue.nextID++
	job.ID = fmt.Sprintf("download-%d", downloadQueue.nextID)
	downloadQueue.pending = append(downloadQueue.pending, job)
	shouldStart := !downloadQueue.running
	if shouldStart {
		downloadQueue.running = true
	}
	downloadQueue.Unlock()

	emitDownloadQueueState(ctx)
	if shouldStart {
		go runDownloadQueue(ctx)
	}
}

func runDownloadQueue(ctx context.Context) {
	parentCtx := ctx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	for {
		downloadQueue.Lock()
		if len(downloadQueue.pending) == 0 {
			downloadQueue.running = false
			downloadQueue.Unlock()
			emitDownloadQueueState(ctx)
			return
		}
		job := downloadQueue.pending[0]
		downloadQueue.pending = downloadQueue.pending[1:]
		jobCtx, cancel := context.WithCancel(parentCtx)
		job.cancel = cancel
		runningJob := job
		downloadQueue.current = &runningJob
		downloadQueue.Unlock()
		emitDownloadQueueState(ctx)

		if err := downloadModJob(jobCtx, job); err != nil {
			if errors.Is(err, context.Canceled) {
				logging.Info("download mod canceled", "fileName", job.Version.FileName, "versionID", job.Version.ID, "targetDir", job.TargetDir)
			} else {
				logging.Error("download mod failed", "fileName", job.Version.FileName, "versionID", job.Version.ID, "targetDir", job.TargetDir, "error", err)
				emitDownloadFailed(ctx, job, err)
			}
		}
		cancel()
		downloadQueue.Lock()
		if downloadQueue.current != nil && downloadQueue.current.ID == job.ID {
			downloadQueue.current = nil
		}
		downloadQueue.Unlock()
		emitDownloadQueueState(ctx)
	}
}

func downloadModJob(ctx context.Context, job downloadJob) error {
	if err := os.MkdirAll(job.TargetDir, 0o755); err != nil {
		return err
	}
	// Try to get mod IDs from platform version (persisted or lazy-parsed via modbridge)
	modIDs := modbridge.VersionModIDs(job.Version, job.ModLoader)
	if len(modIDs) > 0 {
		existing := modbridge.LocalModPathsForModIDs(modIDs, job.InstanceID)
		if alreadyInstalled(existing, job.Version.SHA1) {
			return nil
		}
		return downloadModToTarget(ctx, job, existing)
	}

	return downloadModWithLocalParse(ctx, job)
}

// alreadyInstalled checks whether any existing file with the same modId already has the target SHA1.
func alreadyInstalled(existing []global.LocalModFilePath, latestSHA1 string) bool {
	latestSHA1 = strings.ToLower(strings.TrimSpace(latestSHA1))
	if latestSHA1 == "" {
		return false
	}
	for _, p := range existing {
		if strings.ToLower(strings.TrimSpace(p.FileSHA1)) == latestSHA1 {
			return true
		}
	}
	return false
}

func archiveSupersededModJars(paths []global.LocalModFilePath) {
	mcDir := global.GetMinecraftDir()
	for _, p := range paths {
		abs := p.Path
		if mcDir != "" && !filepath.IsAbs(abs) {
			abs = filepath.Join(mcDir, p.Path)
		}
		archivedPath := nextOldJarPath(abs)
		if err := os.Rename(abs, archivedPath); err != nil && !os.IsNotExist(err) {
			logging.Warn("archive superseded mod jar failed", "path", abs, "archivedPath", archivedPath, "error", err)
		} else {
			logging.Info("superseded mod jar archived", "path", abs, "archivedPath", archivedPath)
		}
		global.RemoveLocalModByPath(p.Path)
	}
}

func nextOldJarPath(path string) string {
	base := path + ".old"
	if _, err := os.Stat(base); os.IsNotExist(err) {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s.old.%d", path, i)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func queueMissingRequiredDependencies(ctx context.Context, req appstructs.ModDownloadRequest, version models.ModVersion, instanceID string, visited map[string]bool) {
	for _, dep := range version.Dependencies {
		if !isRequiredDependency(dep) {
			continue
		}

		depReq, ok := dependencyDownloadRequest(version.Platform, dep, req)
		if !ok {
			logging.Warn("required dependency cannot be queued", "platform", version.Platform, "projectID", dep.DependencyProjectID, "versionID", dep.DependencyVersionID, "type", dep.DependencyType)
			continue
		}

		if modbridge.InstallStatusPrecise(depReq) != modbridge.BtnStatusNew {
			continue
		}
		result := queueModDownload(ctx, depReq, visited)
		if result.Skipped {
			logging.Warn("required dependency queue skipped", "platform", depReq.Result.Platform, "projectID", depReq.ProjectID, "versionID", dep.DependencyVersionID, "reason", result.Reason)
		}
	}
}

func hydrateRequiredDependencies(req appstructs.ModDownloadRequest, version models.ModVersion) models.ModVersion {
	if hasRequiredDependency(version.Dependencies) {
		return version
	}

	refreshed := providers.RefreshMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	if refreshedVersion, found := modbridge.FindVersionByID(refreshed, version.ID); found && hasRequiredDependency(refreshedVersion.Dependencies) {
		return refreshedVersion
	}
	return version
}

func hasRequiredDependency(deps []models.ModDependency) bool {
	for _, dep := range deps {
		if isRequiredDependency(dep) {
			return true
		}
	}
	return false
}

func isRequiredDependency(dep models.ModDependency) bool {
	return strings.EqualFold(strings.TrimSpace(dep.DependencyType), "required")
}

func dependencyDownloadRequest(platform string, dep models.ModDependency, parent appstructs.ModDownloadRequest) (appstructs.ModDownloadRequest, bool) {
	projectID := strings.TrimSpace(dep.DependencyProjectID)
	platform = strings.ToLower(strings.TrimSpace(platform))
	if projectID == "" || (platform != "curseforge" && platform != "modrinth") {
		return appstructs.ModDownloadRequest{}, false
	}

	ref := platform + ":" + projectID
	return appstructs.ModDownloadRequest{
		ProjectID: ref,
		Result: models.ModProject{
			ID:       ref,
			Platform: platform,
		},
		MinecraftVersion: parent.MinecraftVersion,
		ModLoader:        parent.ModLoader,
	}, true
}

func downloadModToTarget(ctx context.Context, job downloadJob, existing []global.LocalModFilePath) error {
	tempDir := filepath.Join(filepath.Dir(job.TargetDir), ".mod-downloader-tmp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return err
	}

	client := grab.NewClient()
	req, err := grab.NewRequest(tempDir, job.Version.DownloadURL)
	if err != nil {
		return err
	}
	if job.Version.FileName != "" {
		req.Filename = filepath.Join(tempDir, job.Version.FileName)
	}
	req = req.WithContext(ctx)

	resp := client.Do(req)
	if err := resp.Err(); err != nil {
		return err
	}
	if resp.Filename == "" {
		return nil
	}

	finalPath := filepath.Join(job.TargetDir, filepath.Base(resp.Filename))
	if downloadTargetExists(finalPath) && !pathInLocalModPaths(finalPath, existing) {
		logging.Info("download skipped because target file already exists", "path", finalPath, "versionID", job.Version.ID)
		_ = os.Remove(resp.Filename)
		return nil
	}
	archiveSupersededModJars(existing)
	if downloadTargetExists(finalPath) {
		logging.Info("download skipped because target file already exists after archiving superseded jars", "path", finalPath, "versionID", job.Version.ID)
		_ = os.Remove(resp.Filename)
		return nil
	}
	if err := os.Rename(resp.Filename, finalPath); err != nil {
		return err
	}
	upsertDownloadedMod(finalPath, job)
	return nil
}

func downloadModWithLocalParse(ctx context.Context, job downloadJob) error {
	tempDir := filepath.Join(filepath.Dir(job.TargetDir), ".mod-downloader-tmp")
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return err
	}

	client := grab.NewClient()
	req, err := grab.NewRequest(tempDir, job.Version.DownloadURL)
	if err != nil {
		return err
	}
	if job.Version.FileName != "" {
		req.Filename = filepath.Join(tempDir, job.Version.FileName)
	}
	req = req.WithContext(ctx)

	resp := client.Do(req)
	if err := resp.Err(); err != nil {
		return err
	}
	if resp.Filename == "" {
		return nil
	}

	sha1 := minecraft.FileSHA1(resp.Filename)
	mods := minecraft.ParseModJarWithSHA1(resp.Filename, sha1, job.ModLoader)
	if len(mods) == 0 {
		_ = os.Remove(resp.Filename)
		return fmt.Errorf("downloaded jar has no parseable mod metadata: %s", filepath.Base(resp.Filename))
	}

	// Extract mod IDs for future version mod ID persistence
	modIDs := make([]string, 0, len(mods))
	for _, m := range mods {
		if id := strings.TrimSpace(m.ID); id != "" {
			modIDs = append(modIDs, strings.ToLower(id))
		}
	}
	if len(modIDs) > 0 && job.Version.ID != "" {
		_ = modbridge.PersistVersionModIDs(job.Version.ID, modIDs)
	}

	existing := modbridge.LocalModPathsForModIDs(modIDs, job.InstanceID)
	if alreadyInstalled(existing, sha1) {
		_ = os.Remove(resp.Filename)
		return nil
	}
	archiveSupersededModJars(existing)

	finalPath := filepath.Join(job.TargetDir, filepath.Base(resp.Filename))
	if downloadTargetExists(finalPath) {
		logging.Info("download skipped because target file already exists", "path", finalPath, "versionID", job.Version.ID)
		_ = os.Remove(resp.Filename)
		return nil
	}
	if err := os.Rename(resp.Filename, finalPath); err != nil {
		return err
	}
	upsertDownloadedMod(finalPath, job)
	return nil
}

func downloadTargetExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	if _, err := os.Stat(path); err == nil {
		return true
	} else if err != nil && !os.IsNotExist(err) {
		logging.Warn("download target stat failed", "path", path, "error", err)
	}
	return false
}

func pathInLocalModPaths(path string, paths []global.LocalModFilePath) bool {
	path = filepath.Clean(path)
	mcDir := global.GetMinecraftDir()
	for _, p := range paths {
		existing := p.Path
		if mcDir != "" && !filepath.IsAbs(existing) {
			existing = filepath.Join(mcDir, existing)
		}
		if filepath.Clean(existing) == path {
			return true
		}
	}
	return false
}

func upsertDownloadedMod(path string, job downloadJob) {
	fileName := minecraft.StripJarSuffix(filepath.Base(path))
	sha1 := minecraft.FileSHA1(path)
	// Parse the downloaded JAR for mod metadata (pure local, no platform merging)
	r, err := zip.OpenReader(path)
	if err != nil {
		logging.Warn("parse downloaded jar failed", "path", path, "error", err)
		return
	}
	defer r.Close()
	mods := minecraft.ParseModZipReader(&r.Reader, filepath.Base(path), job.ModLoader)
	if len(mods) == 0 {
		logging.Warn("downloaded jar has no parseable mod metadata", "path", path)
		return
	}

	// Store JAR metadata in global memory cache (local-only, no platform enrichment)
	global.SetJarMetadata(sha1, mods)

	relPath := filepath.Base(path)
	if mcDir := global.GetMinecraftDir(); mcDir != "" {
		if rel, err := filepath.Rel(mcDir, path); err == nil {
			relPath = rel
		}
	}
	for i := range mods {
		mods[i].FileName = fileName
		mods[i].Path = relPath
		mods[i].SHA1 = sha1
		mods[i].Enabled = true
		global.UpsertLocalMod(mods[i], job.InstanceID, job.MinecraftVersion, job.ModLoader)
	}
}

func currentDownloadQueueState() appstructs.DownloadQueueState {
	downloadQueue.Lock()
	defer downloadQueue.Unlock()
	running := 0
	if downloadQueue.current != nil {
		running = 1
	}
	items := make([]appstructs.DownloadQueueItem, 0, running+len(downloadQueue.pending))
	if downloadQueue.current != nil {
		items = append(items, downloadQueueItemFromJob(*downloadQueue.current, "running", true))
	}
	for _, job := range downloadQueue.pending {
		items = append(items, downloadQueueItemFromJob(job, "pending", true))
	}
	return appstructs.DownloadQueueState{
		Active:  running > 0 || len(downloadQueue.pending) > 0,
		Pending: len(downloadQueue.pending),
		Running: running,
		Items:   items,
	}
}

func downloadQueueItemFromJob(job downloadJob, status string, cancelable bool) appstructs.DownloadQueueItem {
	title := strings.TrimSpace(job.Result.Title)
	if title == "" {
		title = firstNonEmpty(job.Version.Name, job.Version.FileName, job.Version.Version, job.Version.ID)
	}
	return appstructs.DownloadQueueItem{
		ID:               job.ID,
		Status:           status,
		Title:            title,
		FileName:         job.Version.FileName,
		VersionID:        job.Version.ID,
		Platform:         job.Version.Platform,
		MinecraftVersion: job.MinecraftVersion,
		ModLoader:        job.ModLoader,
		Cancelable:       cancelable,
	}
}

func emitDownloadQueueState(ctx context.Context) {
	if ctx == nil {
		return
	}
	runtime.EventsEmit(ctx, downloadQueueUpdatedEvent, currentDownloadQueueState())
}

func emitDownloadFailed(ctx context.Context, job downloadJob, err error) {
	if ctx == nil || err == nil {
		return
	}
	runtime.EventsEmit(ctx, downloadFailedEvent, appstructs.DownloadFailedEvent{
		FileName:  job.Version.FileName,
		VersionID: job.Version.ID,
		Reason:    err.Error(),
	})
}
