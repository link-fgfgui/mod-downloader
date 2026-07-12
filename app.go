package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/appcore"
	"github.com/link-fgfgui/mod-downloader-core/configs"
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
	ctx    context.Context
	config *configs.Config
	core   *appcore.Service
	server *httpserver.Server
}

type AppPreferences struct {
	Theme                       string  `json:"theme"`
	Language                    string  `json:"language"`
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
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
		Config:                a.config,
		Runtime:               appRuntimeOptions(),
		Version:               currentAppVersion(),
		LoadMinecraftReleases: true,
		OnEvent:               a.emitCoreEvent,
	})
	if err := a.core.Startup(ctx); err != nil {
		a.config = a.core.Config()
		return
	}
	a.config = a.core.Config()

	a.server = httpserver.New(httpserver.DefaultAddr, httpserver.Options{OnEvent: a.emitHTTPServerEvent})
	if err := a.server.Start(); err != nil {
		logging.Error("start http server failed", "error", err)
	}
}

func (a *App) service() *appcore.Service {
	if a.core == nil {
		a.core = appcore.New(appcore.Options{
			Config:  a.config,
			Runtime: appRuntimeOptions(),
			Version: currentAppVersion(),
			OnEvent: a.emitCoreEvent,
		})
	}
	a.config = a.core.Config()
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

func (a *App) LookupProjectBySlug(platform, slug, mcVersion, modLoader string) models.ModProject {
	project, ok := a.service().LookupProjectBySlug(platform, slug, mcVersion, modLoader)
	if !ok {
		return models.ModProject{}
	}
	return project
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

func (a *App) PreviewFavoriteListMigration(req appcore.FavoriteMigrationRequest) appcore.FavoriteMigrationPreview {
	return a.service().PreviewFavoriteListMigration(req)
}

func (a *App) ApplyFavoriteListMigration(req appcore.FavoriteMigrationRequest) appcore.FavoriteMigrationApplyResult {
	return a.service().ApplyFavoriteListMigration(req)
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

func (a *App) GetPreferences() AppPreferences {
	prefs := a.service().GetPreferences()
	return AppPreferences{
		Theme:                       prefs.Theme,
		Language:                    prefs.Language,
		AnimationMode:               prefs.AnimationMode,
		AnimationEnabled:            prefs.AnimationEnabled,
		AnimationDurationMultiplier: prefs.AnimationDurationMultiplier,
	}
}

// SettingsView is a settings snapshot returned to the frontend. API keys use an "existence + mask" strategy,
// not sending raw keys back to the frontend; the frontend overwrites via SaveApiKeys.
type SettingsView struct {
	Theme                       string  `json:"theme"` // dark | light | system
	Language                    string  `json:"language"`
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
	AutoScanUnusedDependencies  bool    `json:"autoScanUnusedDependencies"`
	MCIMEnabled                 bool    `json:"mcimEnabled"`
	MinecraftDir                string  `json:"minecraftDir"` // simplified path (with env vars)
	CacheDir                    string  `json:"cacheDir"`
	CachePath                   string  `json:"cachePath"`
	HasCurseforgeKey            bool    `json:"hasCurseforgeKey"`
	CurseforgeKeyMask           string  `json:"curseforgeKeyMask"` // e.g. "abcd****wxyz" or ""
	HasModrinthKey              bool    `json:"hasModrinthKey"`
	ModrinthKeyMask             string  `json:"modrinthKeyMask"`
	FileConcurrency             int     `json:"fileConcurrency"`
	ConcurrentDownloads         int     `json:"concurrentDownloads"`
	AdaptiveFileConcurrency     bool    `json:"adaptiveFileConcurrency"`
	TargetDownloadRateMiB       float64 `json:"targetDownloadRateMiB"`
	RequestsPerSecond           int     `json:"requestsPerSecond"`
}

// SaveApiKeysRequest is the request structure for the frontend to save API keys.
type SaveApiKeysRequest struct {
	CurseforgeApiKey string `json:"curseforgeApiKey"`
	ModrinthApiKey   string `json:"modrinthApiKey"`
}

type SaveAnimationSettingsRequest struct {
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
}

type ExportFavoritePackwizResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

type SaveUnusedDependencyCleanupSettingsRequest struct {
	AutoScanUnusedDependencies bool `json:"autoScanUnusedDependencies"`
}

type SaveMCIMSettingsRequest struct {
	MCIMEnabled bool `json:"mcimEnabled"`
}

type SaveNetworkSettingsRequest struct {
	FileConcurrency         int     `json:"fileConcurrency"`
	ConcurrentDownloads     int     `json:"concurrentDownloads"`
	AdaptiveFileConcurrency bool    `json:"adaptiveFileConcurrency"`
	TargetDownloadRateMiB   float64 `json:"targetDownloadRateMiB"`
	RequestsPerSecond       int     `json:"requestsPerSecond"`
}

// Convention: a field value of "<keep>" means do not modify the original value (since the frontend cannot access plaintext).
// An empty string "" means clear. Any other value means overwrite.
const apiKeyKeepSentinel = appcore.APIKeyKeepSentinel

// GetSettings returns a read-only view of the current settings.
func (a *App) GetSettings() SettingsView {
	return settingsViewFromCore(a.service().GetSettings())
}

// maskKey keeps the first 4 and last 4 characters, replacing the middle with ****; fully masks if too short.
func maskKey(s string) string {
	return appcore.MaskKey(s)
}

// SaveTheme updates the theme preference and persists it immediately. Returns the updated theme string.
// Invalid values fall back to dark.
func (a *App) SaveTheme(theme string) string {
	next := a.service().SaveTheme(theme)
	a.config = a.core.Config()
	return next
}

func (a *App) SaveLanguage(language string) string {
	next := a.service().SaveLanguage(language)
	a.config = a.core.Config()
	return next
}

func (a *App) SaveAnimationSettings(req SaveAnimationSettingsRequest) SettingsView {
	next := a.service().SaveAnimationSettings(appcore.SaveAnimationSettingsRequest{
		AnimationMode:               req.AnimationMode,
		AnimationEnabled:            req.AnimationEnabled,
		AnimationDurationMultiplier: req.AnimationDurationMultiplier,
	})
	a.config = a.core.Config()
	return settingsViewFromCore(next)
}

func (a *App) SaveUnusedDependencyCleanupSettings(req SaveUnusedDependencyCleanupSettingsRequest) SettingsView {
	next := a.service().SaveUnusedDependencyCleanupSettings(appcore.SaveUnusedDependencyCleanupSettingsRequest{
		AutoScanUnusedDependencies: req.AutoScanUnusedDependencies,
	})
	a.config = a.core.Config()
	return settingsViewFromCore(next)
}

func (a *App) SaveMCIMSettings(req SaveMCIMSettingsRequest) SettingsView {
	next := a.service().SaveMCIMSettings(appcore.SaveMCIMSettingsRequest{MCIMEnabled: req.MCIMEnabled})
	a.config = a.core.Config()
	return settingsViewFromCore(next)
}

func (a *App) SaveNetworkSettings(req SaveNetworkSettingsRequest) SettingsView {
	next := a.service().SaveNetworkSettings(appcore.SaveNetworkSettingsRequest{
		FileConcurrency:         req.FileConcurrency,
		ConcurrentDownloads:     req.ConcurrentDownloads,
		AdaptiveFileConcurrency: req.AdaptiveFileConcurrency,
		TargetDownloadRateMiB:   req.TargetDownloadRateMiB,
		RequestsPerSecond:       req.RequestsPerSecond,
	})
	a.config = a.core.Config()
	return settingsViewFromCore(next)
}

func (a *App) SaveCacheDirPreference(dir string) SettingsView {
	next := a.service().SaveCacheDirPreference(dir)
	a.config = a.core.Config()
	return settingsViewFromCore(next)
}

// SaveApiKeys updates API keys and persists them immediately + reinitializes clients.
// An empty string means clear the key; the special value (see above) means keep unchanged.
func (a *App) SaveApiKeys(req SaveApiKeysRequest) SettingsView {
	next := a.service().SaveApiKeys(appcore.SaveApiKeysRequest{
		CurseforgeApiKey: req.CurseforgeApiKey,
		ModrinthApiKey:   req.ModrinthApiKey,
	})
	a.config = a.core.Config()
	return settingsViewFromCore(next)
}

func (a *App) QueueModDownload(req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
	return a.service().QueueModDownload(req)
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
	a.service().Shutdown()
	a.config = a.core.Config()
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

func (a *App) ChooseCacheDir(locale string) SettingsView {
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

func (a *App) ApplyLocalModBatchOperation(req appstructs.LocalModBatchOperationRequest) (structs.VersionInfo, error) {
	version, err := a.service().ApplyLocalModBatchOperation(req)
	if err != nil {
		return structs.VersionInfo{}, err
	}
	return version, nil
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

func settingsViewFromCore(sv appcore.SettingsView) SettingsView {
	return SettingsView{
		Theme:                       sv.Theme,
		Language:                    sv.Language,
		AnimationMode:               sv.AnimationMode,
		AnimationEnabled:            sv.AnimationEnabled,
		AnimationDurationMultiplier: sv.AnimationDurationMultiplier,
		AutoScanUnusedDependencies:  sv.AutoScanUnusedDependencies,
		MCIMEnabled:                 sv.MCIMEnabled,
		MinecraftDir:                sv.MinecraftDir,
		CacheDir:                    sv.CacheDir,
		CachePath:                   sv.CachePath,
		HasCurseforgeKey:            sv.HasCurseforgeKey,
		CurseforgeKeyMask:           sv.CurseforgeKeyMask,
		HasModrinthKey:              sv.HasModrinthKey,
		ModrinthKeyMask:             sv.ModrinthKeyMask,
		FileConcurrency:             sv.FileConcurrency,
		ConcurrentDownloads:         sv.ConcurrentDownloads,
		AdaptiveFileConcurrency:     sv.AdaptiveFileConcurrency,
		TargetDownloadRateMiB:       sv.TargetDownloadRateMiB,
		RequestsPerSecond:           sv.RequestsPerSecond,
	}
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

func loadVersionsFromDisk(mcDir string) []structs.VersionInfo {
	return appcore.New(appcore.Options{}).LoadVersionsFromDisk(mcDir)
}

func scanVersionMods(version structs.VersionInfo, mcDir string) structs.VersionInfo {
	return appcore.ScanVersionMods(version, mcDir)
}

func scanAllModDirsForHardlinkIndex(mcDir string, versions []structs.VersionInfo, generation uint64) {
	appcore.ScanAllModDirsForHardlinkIndex(mcDir, versions, generation)
}
