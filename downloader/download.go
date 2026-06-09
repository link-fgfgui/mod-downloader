package downloader

import (
	"archive/zip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mod-downloader/database"
	"mod-downloader/global"
	"mod-downloader/logging"
	"mod-downloader/minecraft"
	"mod-downloader/providers"
	appstructs "mod-downloader/structs"
	mcstructs "mod-downloader/structs/minecraft"

	"github.com/cavaliergopher/grab/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const downloadQueueUpdatedEvent = "download-queue-updated"
const downloadFailedEvent = "download-failed"

// 下载按钮的四种状态。
const (
	btnStatusNew       = "new"       // 实例内无同 modId → 标准下载
	btnStatusInstalled = "installed" // 已装且与最新版同 sha1 → 禁用
	btnStatusUpdate    = "update"    // 同 modId 同项目但非最新 → 更新
	btnStatusConflict  = "conflict"  // 同 modId 但不属于本项目任何版本 → 换源下载
)

type downloadJob struct {
	Version          appstructs.ProjectVersionResult
	Result           appstructs.SearchModResult
	TargetDir        string
	InstanceID       string
	MinecraftVersion string
	ModLoader        string
}

var downloadQueue = struct {
	sync.Mutex
	pending []downloadJob
	running bool
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

	req, instanceID, targetDir, ok := applySelectedInstance(req)
	if !ok {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "no selected version"}
	}
	if req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "invalid request"}
	}

	version, ok := downloadVersionForRequest(req)
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

// applySelectedInstance 用当前选中的实例覆盖请求的 MC 版本与 mod loader，
// 使"版本匹配 / 下载目标目录 / 安装标记"三者始终以侧边栏选中的实例为准。
// instanceID(实例文件夹名)与 targetDir 来自选中实例；未选实例时 ok 为 false。
// 注意：MC 版本必须取 selected.MinecraftVersion，绝不能用 instanceID(实例名)——
// 实例名不是合法 MC 版本，会导致版本匹配恒为空。
func applySelectedInstance(req appstructs.ModDownloadRequest) (appstructs.ModDownloadRequest, string, string, bool) {
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

func GetDownloadQueueState() appstructs.DownloadQueueState {
	return currentDownloadQueueState()
}

func GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
	req.MinecraftVersion = strings.TrimSpace(req.MinecraftVersion)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))

	states := make([]appstructs.ModDownloadButtonState, len(req.Results))
	var wg sync.WaitGroup
	for i := range req.Results {
		result := req.Results[i]
		key := providers.ProjectReferenceFromSearchResult(result)
		states[i] = appstructs.ModDownloadButtonState{Key: key, Status: btnStatusNew, Icon: "mdi-download", Color: "primary"}
		if key == "" {
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			status := localModButtonStatus(appstructs.ModDownloadRequest{
				ProjectID:        key,
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

// applyButtonStatus 把状态映射成按钮的禁用/图标/颜色。
func applyButtonStatus(state *appstructs.ModDownloadButtonState, status string) {
	state.Status = status
	switch status {
	case btnStatusInstalled:
		state.Disabled = true
		state.Icon = "mdi-check"
		state.Color = ""
	case btnStatusUpdate:
		state.Icon = "mdi-arrow-up-bold-circle"
		state.Color = "warning"
	case btnStatusConflict:
		state.Icon = "mdi-download"
		state.Color = "warning"
	default:
		state.Icon = "mdi-download"
		state.Color = "primary"
	}
}

// localModButtonStatus 判定某搜索结果在当前选中实例下的按钮状态。
func localModButtonStatus(req appstructs.ModDownloadRequest) string {
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if req.ProjectID == "" {
		req.ProjectID = providers.ProjectReferenceFromSearchResult(req.Result)
	}

	req, instanceID, _, ok := applySelectedInstance(req)
	if !ok || req.ProjectID == "" || req.MinecraftVersion == "" || req.ModLoader == "" {
		return btnStatusNew
	}

	version, ok := downloadVersionForRequest(req)
	if !ok {
		return btnStatusNew
	}
	mods, ok := metadataForProjectVersion(version, req.ModLoader)
	if !ok {
		return btnStatusNew
	}

	localPaths := localModPathsForMods(mods, instanceID)
	if len(localPaths) == 0 {
		return btnStatusNew
	}

	latestSHA1 := strings.ToLower(strings.TrimSpace(version.SHA1))
	if latestSHA1 != "" {
		for _, p := range localPaths {
			if strings.ToLower(strings.TrimSpace(p.FileSHA1)) == latestSHA1 {
				return btnStatusInstalled
			}
		}
	}

	projectSHA1s := providers.ProjectVersionSHA1Set(req.Result)
	for _, p := range localPaths {
		if projectSHA1s[strings.ToLower(strings.TrimSpace(p.FileSHA1))] {
			return btnStatusUpdate
		}
	}
	return btnStatusConflict
}

// localModPathsForMods 汇总实例内与给定 mod 集合任一 modId 匹配的本地文件记录（按路径去重）。
func localModPathsForMods(mods []mcstructs.ModInfo, instanceID string) []global.LocalModFilePath {
	out := make([]global.LocalModFilePath, 0)
	seen := make(map[string]struct{})
	for _, m := range mods {
		for _, p := range global.LocalModPathsInInstanceByModID(instanceID, m.ID) {
			if _, dup := seen[p.Path]; dup {
				continue
			}
			seen[p.Path] = struct{}{}
			out = append(out, p)
		}
	}
	return out
}

func downloadVersionForRequest(req appstructs.ModDownloadRequest) (appstructs.ProjectVersionResult, bool) {
	platform := strings.ToLower(strings.TrimSpace(req.Result.Platform))
	projectID := strings.TrimSpace(req.ProjectID)
	if platform == "" {
		platform, projectID = providers.SplitProjectReference(projectID)
	}
	if _, project := providers.SplitProjectReference(projectID); project != "" {
		projectID = project
	}

	if pin, ok := database.GetPinnedMod(platform, projectID, req.MinecraftVersion, req.ModLoader); ok {
		if version, found := findProjectVersionByID(providers.ListMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader), pin.VersionID); found {
			return version, true
		}
	}

	versions := providers.ListMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	if len(versions) == 0 {
		return appstructs.ProjectVersionResult{}, false
	}
	return versions[0], true
}

func findProjectVersionByID(versions []appstructs.ProjectVersionResult, versionID string) (appstructs.ProjectVersionResult, bool) {
	versionID = strings.TrimSpace(versionID)
	if versionID == "" {
		return appstructs.ProjectVersionResult{}, false
	}
	for _, version := range versions {
		if version.ID == versionID {
			return version, true
		}
	}
	return appstructs.ProjectVersionResult{}, false
}

func selectedVersionModsDir(selected mcstructs.VersionInfo) string {
	mcDir := global.GetMinecraftDir()
	versionDirName := versionInstanceID(selected)
	if strings.TrimSpace(mcDir) == "" || strings.TrimSpace(versionDirName) == "" {
		return ""
	}
	return filepath.Join(mcDir, "versions", versionDirName, "mods")
}

func versionInstanceID(version mcstructs.VersionInfo) string {
	if strings.TrimSpace(version.Name) != "" {
		return strings.TrimSpace(version.Name)
	}
	return strings.TrimSpace(version.ID)
}

func queueMissingRequiredDependencies(ctx context.Context, req appstructs.ModDownloadRequest, version appstructs.ProjectVersionResult, instanceID string, visited map[string]bool) {
	for _, dep := range version.Dependencies {
		if !isRequiredDependency(dep) {
			continue
		}

		depReq, ok := dependencyDownloadRequest(version.Platform, dep, req)
		if !ok {
			logging.Warn("required dependency cannot be queued", "platform", version.Platform, "projectID", dep.ProjectID, "versionID", dep.VersionID, "type", dep.Type)
			continue
		}

		if localModButtonStatus(depReq) != btnStatusNew {
			continue
		}
		result := queueModDownload(ctx, depReq, visited)
		if result.Skipped {
			logging.Warn("required dependency queue skipped", "platform", depReq.Result.Platform, "projectID", depReq.ProjectID, "versionID", dep.VersionID, "reason", result.Reason)
		}
	}
}

func hydrateRequiredDependencies(req appstructs.ModDownloadRequest, version appstructs.ProjectVersionResult) appstructs.ProjectVersionResult {
	if hasRequiredDependency(version.Dependencies) {
		return version
	}

	refreshed := providers.RefreshMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	if refreshedVersion, found := findProjectVersionByID(refreshed, version.ID); found && hasRequiredDependency(refreshedVersion.Dependencies) {
		return refreshedVersion
	}
	return version
}

func hasRequiredDependency(deps []appstructs.ProjectDependency) bool {
	for _, dep := range deps {
		if isRequiredDependency(dep) {
			return true
		}
	}
	return false
}

func isRequiredDependency(dep appstructs.ProjectDependency) bool {
	return strings.EqualFold(strings.TrimSpace(dep.Type), "required")
}

func dependencyDownloadRequest(platform string, dep appstructs.ProjectDependency, parent appstructs.ModDownloadRequest) (appstructs.ModDownloadRequest, bool) {
	projectID := strings.TrimSpace(dep.ProjectID)
	platform = strings.ToLower(strings.TrimSpace(platform))
	if projectID == "" || (platform != "curseforge" && platform != "modrinth") {
		return appstructs.ModDownloadRequest{}, false
	}

	ref := platform + ":" + projectID
	return appstructs.ModDownloadRequest{
		ProjectID: ref,
		Result: appstructs.SearchModResult{
			ID:       ref,
			Platform: platform,
		},
		MinecraftVersion: parent.MinecraftVersion,
		ModLoader:        parent.ModLoader,
	}, true
}

func projectVersionJobKey(version appstructs.ProjectVersionResult) string {
	platform := strings.ToLower(strings.TrimSpace(version.Platform))
	projectID := strings.TrimSpace(version.ProjectID)
	versionID := strings.TrimSpace(version.ID)
	if platform == "" && projectID == "" && versionID == "" {
		return ""
	}
	return platform + ":" + projectID + ":" + versionID
}

func enqueueDownload(ctx context.Context, job downloadJob) {
	downloadQueue.Lock()
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
		downloadQueue.Unlock()
		emitDownloadQueueState(ctx)

		if err := downloadModJob(job); err != nil {
			logging.Error("download mod failed", "fileName", job.Version.FileName, "versionID", job.Version.ID, "targetDir", job.TargetDir, "error", err)
			emitDownloadFailed(ctx, job, err)
		}
	}
}

func downloadModJob(job downloadJob) error {
	if err := os.MkdirAll(job.TargetDir, 0o755); err != nil {
		return err
	}
	if mods, ok := metadataForProjectVersionWithResult(job.Version, job.Result, job.ModLoader); ok {
		existing := localModPathsForMods(mods, job.InstanceID)
		if alreadyInstalled(existing, job.Version.SHA1) {
			return nil
		}
		archiveSupersededModJars(existing)
		return downloadModToTarget(job, mods)
	}

	return downloadModWithLocalParse(job)
}

// alreadyInstalled 判断现有同 modId 文件里是否已有与目标版本完全相同的 jar（同 sha1）。
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

func metadataForProjectVersion(version appstructs.ProjectVersionResult, modLoader string) ([]mcstructs.ModInfo, bool) {
	return metadataForProjectVersionWithResult(version, appstructs.SearchModResult{}, modLoader)
}

func metadataForProjectVersionWithResult(version appstructs.ProjectVersionResult, result appstructs.SearchModResult, modLoader string) ([]mcstructs.ModInfo, bool) {
	if mods, ok := database.GetJarMetadata(version.SHA1); ok {
		return applyPlatformMetadata(mods, version, result), true
	}
	mods, err := parseRemoteModJar(version.DownloadURL, modLoader)
	if err != nil {
		logging.Warn("range parse mod jar failed, fallback to full download", "downloadURL", version.DownloadURL, "modLoader", modLoader, "error", err)
		return nil, false
	}
	mods = applyPlatformMetadata(mods, version, result)
	_ = database.SetJarMetadata(version.SHA1, mods)
	return mods, true
}

func applyPlatformMetadata(mods []mcstructs.ModInfo, version appstructs.ProjectVersionResult, result appstructs.SearchModResult) []mcstructs.ModInfo {
	if len(mods) == 0 {
		return mods
	}

	name := strings.TrimSpace(version.Name)
	if strings.TrimSpace(result.Title) != "" {
		name = strings.TrimSpace(result.Title)
	}
	if name == "" {
		name = strings.TrimSpace(version.FileName)
	}
	versionText := strings.TrimSpace(version.Version)
	description := strings.TrimSpace(result.Description)

	out := make([]mcstructs.ModInfo, len(mods))
	copy(out, mods)
	for i := range out {
		out[i].Name = name
		out[i].Version = versionText
		out[i].Description = description
	}
	return out
}

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

func downloadModToTarget(job downloadJob, mods []mcstructs.ModInfo) error {
	client := grab.NewClient()
	req, err := grab.NewRequest(job.TargetDir, job.Version.DownloadURL)
	if err != nil {
		return err
	}
	if job.Version.FileName != "" {
		req.Filename = filepath.Join(job.TargetDir, job.Version.FileName)
	}

	resp := client.Do(req)
	if err := resp.Err(); err != nil {
		return err
	}
	if resp.Filename != "" {
		upsertDownloadedMod(resp.Filename, mods, job)
	}
	return nil
}

func downloadModWithLocalParse(job downloadJob) error {
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
	existing := localModPathsForMods(mods, job.InstanceID)
	if alreadyInstalled(existing, sha1) {
		_ = os.Remove(resp.Filename)
		return nil
	}
	archiveSupersededModJars(existing)

	finalPath := filepath.Join(job.TargetDir, filepath.Base(resp.Filename))
	if err := os.Rename(resp.Filename, finalPath); err != nil {
		return err
	}
	upsertDownloadedMod(finalPath, mods, job)
	return nil
}

func upsertDownloadedMod(path string, mods []mcstructs.ModInfo, job downloadJob) {
	fileName := minecraft.StripJarSuffix(filepath.Base(path))
	sha1 := minecraft.FileSHA1(path)
	mods = applyPlatformMetadata(mods, job.Version, job.Result)
	_ = database.SetJarMetadata(sha1, mods)
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
	if downloadQueue.running {
		running = 1
	}
	return appstructs.DownloadQueueState{
		Active:  downloadQueue.running || len(downloadQueue.pending) > 0,
		Pending: len(downloadQueue.pending),
		Running: running,
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
