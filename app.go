package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/appcore"
	"github.com/link-fgfgui/mod-downloader-core/httpserver"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const minecraftDirChangedEvent = "minecraft-dir-changed"
const selectedVersionChangedEvent = "selected-version-changed"
const searchModsUpdatedEvent = "search-mods-updated"
const downloadStatesUpdatedEvent = "download-states-updated"
const downloadQueueUpdatedEvent = "download-queue-updated"
const downloadFailedEvent = "download-failed"
const downloadCompletedEvent = "download-completed"
const usageStatsUpdatedEvent = "usage-stats-updated"

// App struct
type App struct {
	ctx        context.Context
	core       *appcore.Service
	server     *httpserver.Server
	startupErr error
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) GetAppVersion() string {
	return currentAppVersion()
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.core = appcore.New(appcore.Options{
		Runtime:               appRuntimeOptions(),
		Version:               currentAppVersion(),
		LoadMinecraftReleases: true,
		OnEvent:               a.emitCoreEvent,
	})
	if err := a.core.Startup(ctx); err != nil {
		a.startupErr = err
		// A partially initialized core must not be used: fail fast and leave the
		// HTTP server down so the frontend sees explicit rejections instead of
		// silent empty results.
		logging.Error("core startup failed", "error", err)
		return
	}

	a.server = httpserver.New(httpserver.DefaultAddr, httpserver.Options{OnEvent: a.emitHTTPServerEvent})
	if err := a.server.Start(); err != nil {
		logging.Error("start http server failed", "error", err)
	}
}

func (a *App) service() *appcore.Service {
	if a.startupErr != nil {
		reason := "core startup failed: " + a.startupErr.Error()
		logging.Error("app service unavailable", "reason", reason)
		panic(reason)
	}
	if a.core == nil {
		reason := "core not initialized"
		logging.Error("app service unavailable", "reason", reason)
		panic(reason)
	}
	return a.core
}

func appRuntimeOptions() appcore.RuntimeOptions {
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("resolve gui default cache dir failed", "error", err)
		return appcore.RuntimeOptions{}
	}
	return appcore.RuntimeOptions{DefaultCacheDir: dir}
}

func (a *App) emitCoreEvent(event appcore.Event) {
	if a.ctx == nil {
		return
	}
	eventName := ""
	switch event.Kind {
	case appcore.EventSearchModsUpdated:
		eventName = searchModsUpdatedEvent
	case appcore.EventDownloadStatesUpdated:
		eventName = downloadStatesUpdatedEvent
	case appcore.EventDownloadQueueUpdated:
		eventName = downloadQueueUpdatedEvent
	case appcore.EventDownloadFailed:
		eventName = downloadFailedEvent
	case appcore.EventDownloadCompleted:
		eventName = downloadCompletedEvent
	case appcore.EventMinecraftDirChanged:
		eventName = minecraftDirChangedEvent
	case appcore.EventSelectedVersionChanged:
		eventName = selectedVersionChangedEvent
	case appcore.EventUsageStatsUpdated:
		eventName = usageStatsUpdatedEvent
	default:
		return
	}
	if event.Payload == nil {
		runtime.EventsEmit(a.ctx, eventName)
		return
	}
	runtime.EventsEmit(a.ctx, eventName, event.Payload)
}

func (a *App) emitHTTPServerEvent(event httpserver.Event) {
	if a.ctx == nil || event.Name == "" {
		return
	}
	if event.Payload == nil {
		runtime.EventsEmit(a.ctx, event.Name)
		return
	}
	runtime.EventsEmit(a.ctx, event.Name, event.Payload)
}

func (a *App) SearchMods(req appstructs.SearchModsRequest) {
	a.service().SearchMods(req)
}

func (a *App) ListMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
	return a.service().ListMatchingProjectVersions(result, minecraftVersion, modLoader)
}

func (a *App) LookupProjectBySlug(platform, slug, mcVersion, modLoader string) (models.ModProject, error) {
	project, ok := a.service().LookupProjectBySlug(platform, slug, mcVersion, modLoader)
	if !ok {
		return models.ModProject{}, fmt.Errorf("project not found: %s/%s", platform, slug)
	}
	return project, nil
}

func (a *App) GetPinnedModVersion(platform string, modID string, minecraftVersion string, modLoader string) storage.PinnedMod {
	return a.service().GetPinnedModVersion(platform, modID, minecraftVersion, modLoader)
}

func (a *App) PinModVersion(req appstructs.ModVersionPinRequest) storage.PinnedMod {
	return a.service().PinModVersion(req)
}

func (a *App) ListPinnedMods() []storage.PinnedMod {
	return a.service().ListPinnedMods()
}

func (a *App) UnpinMod(platform, modID, mcVersion, modLoader string) bool {
	return a.service().UnpinMod(platform, modID, mcVersion, modLoader)
}

func (a *App) ListFavoriteLists() []storage.FavoriteList {
	return a.service().ListFavoriteLists()
}

func (a *App) CreateFavoriteList(name, minecraftVersion, modLoader string) storage.FavoriteList {
	return a.service().CreateFavoriteListForScope(name, minecraftVersion, modLoader)
}

func (a *App) RenameFavoriteList(id, name string) storage.FavoriteList {
	return a.service().RenameFavoriteList(id, name)
}

func (a *App) DeleteFavoriteList(id string) bool {
	return a.service().DeleteFavoriteList(id)
}

func (a *App) UpdateFavoriteListMetadata(list storage.FavoriteList) storage.FavoriteList {
	return a.service().UpdateFavoriteListMetadata(list)
}

func (a *App) ReorderFavoriteLists(ids []string) bool {
	return a.service().ReorderFavoriteLists(ids)
}

func (a *App) ListFavoriteMods(listID string) []storage.FavoriteMod {
	return a.service().ListFavoriteMods(listID)
}

func (a *App) ListFavoriteContents(listID string) storage.FavoriteListContents {
	return a.service().ListFavoriteContents(listID)
}

func (a *App) AddFavoriteMod(mod storage.FavoriteMod) storage.FavoriteMod {
	return a.service().AddFavoriteMod(mod)
}

func (a *App) AddFavoriteModsToLists(req appcore.FavoriteBulkAddRequest) appcore.FavoriteBulkOperationResult {
	return a.service().AddFavoriteModsToLists(req)
}

func (a *App) CopyFavoriteListToList(req appcore.FavoriteListCopyRequest) appcore.FavoriteBulkOperationResult {
	return a.service().CopyFavoriteListToList(req)
}

func (a *App) PreviewFavoriteListCrossVersionCopy(req appcore.FavoriteCrossVersionCopyRequest) appcore.FavoriteCrossVersionCopyPreview {
	return a.service().PreviewFavoriteListCrossVersionCopy(req)
}

func (a *App) ApplyFavoriteListCrossVersionCopy(req appcore.FavoriteCrossVersionCopyRequest) appcore.FavoriteCrossVersionCopyApplyResult {
	return a.service().ApplyFavoriteListCrossVersionCopy(req)
}

func (a *App) AddFavoriteListReference(parentListID, childListID string) storage.FavoriteListRef {
	return a.service().AddFavoriteListReference(parentListID, childListID)
}

func (a *App) RemoveFavoriteListReference(parentListID, childListID string) bool {
	return a.service().RemoveFavoriteListReference(parentListID, childListID)
}

func (a *App) ListFavoriteListRefs(parentListID string) []storage.FavoriteListRef {
	return a.service().ListFavoriteListRefs(parentListID)
}

func (a *App) RemoveFavoriteMod(listID, platform, modID, mcVersion, modLoader string) bool {
	return a.service().RemoveFavoriteMod(listID, platform, modID, mcVersion, modLoader)
}

func (a *App) ExportFavoriteListPackwizZip(listID, minecraftVersion, modLoader, locale string) ExportFavoritePackwizResult {
	dialogText := localizedDialogText(locale)
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:                dialogText.exportPackwizTitle,
		DefaultFilename:      a.service().FavoriteListPackwizDefaultFilename(listID),
		CanCreateDirectories: true,
		Filters: []runtime.FileFilter{{
			DisplayName: dialogText.zipArchives,
			Pattern:     "*.zip",
		}},
	})
	if err != nil {
		logging.Error("choose packwiz export path failed", "listID", listID, "error", err)
		panic("export packwiz failed: " + err.Error())
	}
	if strings.TrimSpace(path) == "" {
		return ExportFavoritePackwizResult{Canceled: true}
	}
	if filepath.Ext(path) == "" {
		path += ".zip"
	}
	result, err := a.service().ExportFavoriteListPackwizZipForScope(listID, path, minecraftVersion, modLoader)
	if err != nil {
		logging.Error("export favorite packwiz zip failed", "listID", listID, "path", path, "error", err)
		panic("export packwiz failed: " + err.Error())
	}
	return ExportFavoritePackwizResult{Path: result.Path}
}

func (a *App) GetMinecraftReleaseVersions() []string {
	return a.service().GetMinecraftReleaseVersions()
}

func (a *App) GetPreferences() appstructs.AppPreferences {
	return a.service().GetPreferences()
}

// GetSettings returns a read-only view of the current settings.
func (a *App) GetSettings() appstructs.SettingsView {
	return a.service().GetSettings()
}

// SaveTheme updates the theme preference and persists it immediately. Returns the updated theme string.
// Invalid values fall back to dark.
func (a *App) SaveTheme(theme string) string {
	return a.service().SaveTheme(theme)
}

func (a *App) SaveLanguage(language string) string {
	return a.service().SaveLanguage(language)
}

func (a *App) SaveAnimationSettings(req appstructs.SaveAnimationSettingsRequest) appstructs.SettingsView {
	return a.service().SaveAnimationSettings(req)
}

func (a *App) SaveSimpleModeSettings(req appstructs.SaveSimpleModeSettingsRequest) appstructs.SettingsView {
	return a.service().SaveSimpleModeSettings(req)
}

func (a *App) SaveUnusedDependencyCleanupSettings(req appstructs.SaveUnusedDependencyCleanupSettingsRequest) appstructs.SettingsView {
	return a.service().SaveUnusedDependencyCleanupSettings(req)
}

func (a *App) SaveMCIMSettings(req appstructs.SaveMCIMSettingsRequest) appstructs.SettingsView {
	return a.service().SaveMCIMSettings(req)
}

func (a *App) SaveJijDetectionSettings(req appstructs.SaveJijDetectionSettingsRequest) appstructs.SettingsView {
	return a.service().SaveJijDetectionSettings(req)
}

func (a *App) SavePreferredReleaseTypeSettings(req appstructs.SavePreferredReleaseTypeSettingsRequest) appstructs.SettingsView {
	return a.service().SavePreferredReleaseTypeSettings(req)
}

func (a *App) SaveHardwareAccelerationSettings(req appstructs.SaveHardwareAccelerationSettingsRequest) appstructs.SettingsView {
	return a.service().SaveHardwareAccelerationSettings(req)
}

func (a *App) SaveNetworkSettings(req appstructs.SaveNetworkSettingsRequest) appstructs.SettingsView {
	return a.service().SaveNetworkSettings(req)
}

func (a *App) SaveCacheDirPreference(dir string) appstructs.SettingsView {
	return a.service().SaveCacheDirPreference(dir)
}

// SaveApiKeys updates API keys and persists them immediately + reinitializes clients.
// An empty string means clear the key; the special value (see above) means keep unchanged.
func (a *App) SaveApiKeys(req appstructs.SaveApiKeysRequest) appstructs.SettingsView {
	return a.service().SaveApiKeys(req)
}

func (a *App) QueueModDownload(req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
	return a.service().QueueModDownload(req)
}

func (a *App) CheckDependencyFulfillment(req appstructs.DependencyPreflightRequest) appstructs.DependencyPreflightResult {
	return a.service().CheckDependencyFulfillment(req)
}

func (a *App) AnalyzeLocalModDisableImpact(req appstructs.LocalModDisableImpactRequest) (appstructs.LocalModDisableImpactResult, error) {
	return a.service().AnalyzeLocalModDisableImpact(req)
}

func (a *App) RestoreCachedDependency(req appstructs.RestoreCachedDependencyRequest) appstructs.ModDownloadResult {
	return a.service().RestoreCachedDependency(req)
}

func (a *App) GetDownloadQueueState() appstructs.DownloadQueueState {
	return a.service().GetDownloadQueueState()
}

func (a *App) GetUsageStats() storage.UsageStats {
	return a.service().GetUsageStats()
}

func (a *App) CancelDownload(id string) bool {
	return a.service().CancelDownload(id)
}

func (a *App) RetryDownload(id string) bool {
	return a.service().RetryDownload(id)
}

func (a *App) RemoveDownload(id string) bool {
	return a.service().RemoveDownload(id)
}

func (a *App) DismissOptionalDependencyReminder(id string) bool {
	return a.service().DismissOptionalDependencyReminder(id)
}

func (a *App) ClearOptionalDependencyReminders() bool {
	return a.service().ClearOptionalDependencyReminders()
}

func (a *App) InstallOptionalDependencies(id string) []appstructs.ModDownloadResult {
	return a.service().InstallOptionalDependencies(id)
}

func (a *App) AnalyzeBatchIncompatibleConflicts(req appstructs.BatchDownloadRequest) appstructs.BatchIncompatibleAnalysis {
	return a.service().AnalyzeBatchIncompatibleConflicts(req)
}

func (a *App) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
	return a.service().GetDownloadStates(req)
}

func (a *App) shutdown(ctx context.Context) {
	if a.server != nil {
		a.server.Stop()
	}
	if a.core != nil {
		a.core.Shutdown()
	}
}

func (a *App) ChooseMinecraftDir(locale string) string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:           localizedDialogText(locale).chooseMinecraftDir,
		ShowHiddenFiles: true,
	})
	if err != nil {
		logging.Error("choose minecraft dir failed", "error", err)
		return ""
	}
	return a.service().SetMinecraftDir(dir)
}

func (a *App) ChooseCacheDir(locale string) appstructs.SettingsView {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:           localizedDialogText(locale).chooseCacheDir,
		ShowHiddenFiles: true,
	})
	if err != nil {
		logging.Error("choose cache dir failed", "error", err)
		return a.GetSettings()
	}
	if strings.TrimSpace(dir) == "" {
		return a.GetSettings()
	}
	return a.SaveCacheDirPreference(dir)
}

func (a *App) GetMinecraftDir() string {
	return a.service().GetMinecraftDir()
}

func (a *App) ValidateMinecraftDir() bool {
	return a.service().ValidateMinecraftDir()
}

func (a *App) GetVersions() []structs.VersionInfo {
	return a.service().GetVersions()
}

// GetSelectedVersion returns the currently selected instance (including MinecraftVersion / ModLoader),
// for the frontend to fetch on mount, avoiding missed selected-version-changed events that could cause desync.
func (a *App) GetSelectedVersion() structs.VersionInfo {
	return a.service().GetSelectedVersion()
}

func (a *App) RefreshVersions() []structs.VersionInfo {
	return a.service().RefreshVersions()
}

func (a *App) RefreshSelectedVersionMods() structs.VersionInfo {
	return a.service().RefreshSelectedVersionMods()
}

func (a *App) ToggleConnectorLoader() (structs.VersionInfo, error) {
	return a.service().ToggleConnectorLoader()
}

func (a *App) ApplyLocalModBatchOperation(req appstructs.LocalModBatchOperationRequest) (structs.VersionInfo, error) {
	version, err := a.service().ApplyLocalModBatchOperation(req)
	if err != nil {
		return structs.VersionInfo{}, err
	}
	return version, nil
}

// OpenLocalModLocation reveals a scanned local mod in the OS file manager.
// Only paths that belong to the currently selected instance are accepted, so
// the renderer cannot turn this into an arbitrary-file-open primitive.
func (a *App) OpenLocalModLocation(path string) error {
	target := filepath.Clean(strings.TrimSpace(path))
	if target == "" || target == "." {
		return fmt.Errorf("missing mod path")
	}
	selected := a.service().GetSelectedVersion()
	for _, mod := range selected.Mods {
		if mod.Path == "" {
			continue
		}
		if filepath.Clean(mod.Path) == target {
			return openInFileManager(filepath.Dir(target))
		}
	}
	return fmt.Errorf("path is not a mod in the selected instance")
}

func openInFileManager(dir string) error {
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dir)
	case "darwin":
		cmd = exec.Command("open", dir)
	default:
		cmd = exec.Command("xdg-open", dir)
	}
	return cmd.Start()
}

func (a *App) ScanUnusedDependencies(req appstructs.UnusedDependencyScanRequest) (appstructs.UnusedDependencyScanResult, error) {
	result, err := a.service().ScanUnusedDependencies(req)
	if err != nil {
		return appstructs.UnusedDependencyScanResult{}, err
	}
	return result, nil
}

func (a *App) SelectVersion(versionKey string) (structs.VersionInfo, error) {
	version, err := a.service().SelectVersion(versionKey)
	if err == nil {
		return version, nil
	}
	message := err.Error()
	if strings.HasPrefix(message, "version not found:") {
		message = "version not found"
	}
	return structs.VersionInfo{}, fmt.Errorf("select version failed: %s", message)
}

type dialogText struct {
	chooseMinecraftDir string
	chooseCacheDir     string
	exportPackwizTitle string
	zipArchives        string
}

func localizedDialogText(locale string) dialogText {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "zh") {
		return dialogText{
			chooseMinecraftDir: "选择 .minecraft 文件夹",
			chooseCacheDir:     "选择缓存文件夹",
			exportPackwizTitle: "导出 packwiz 整合包",
			zipArchives:        "ZIP 压缩包 (*.zip)",
		}
	}
	return dialogText{
		chooseMinecraftDir: "Choose .minecraft folder",
		chooseCacheDir:     "Choose cache folder",
		exportPackwizTitle: "Export packwiz modpack",
		zipArchives:        "ZIP archives (*.zip)",
	}
}
