package main

import (
	"context"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/appcore"
	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/link-fgfgui/mod-downloader-core/database"
	"github.com/link-fgfgui/mod-downloader-core/httpserver"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
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

// App struct
type App struct {
	ctx    context.Context
	config *configs.Config
	core   *appcore.Service
	server *httpserver.Server
}

type AppPreferences struct {
	Theme                       string  `json:"theme"`
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.core = appcore.New(appcore.Options{
		Config:                a.config,
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
			OnEvent: a.emitCoreEvent,
		})
	}
	a.config = a.core.Config()
	return a.core
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
	case appcore.EventMinecraftDirChanged:
		eventName = minecraftDirChangedEvent
	case appcore.EventSelectedVersionChanged:
		eventName = selectedVersionChangedEvent
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

func (a *App) GetPinnedModVersion(platform string, modID string, minecraftVersion string, modLoader string) database.PinnedMod {
	return a.service().GetPinnedModVersion(platform, modID, minecraftVersion, modLoader)
}

func (a *App) PinModVersion(req appstructs.ModVersionPinRequest) database.PinnedMod {
	return a.service().PinModVersion(req)
}

func (a *App) ListPinnedMods() []database.PinnedMod {
	return a.service().ListPinnedMods()
}

func (a *App) UnpinMod(platform, modID, mcVersion, modLoader string) bool {
	return a.service().UnpinMod(platform, modID, mcVersion, modLoader)
}

func (a *App) ListFavoriteLists() []database.FavoriteList {
	return a.service().ListFavoriteLists()
}

func (a *App) CreateFavoriteList(name string) database.FavoriteList {
	return a.service().CreateFavoriteList(name)
}

func (a *App) RenameFavoriteList(id, name string) database.FavoriteList {
	return a.service().RenameFavoriteList(id, name)
}

func (a *App) DeleteFavoriteList(id string) bool {
	return a.service().DeleteFavoriteList(id)
}

func (a *App) ListFavoriteMods(listID string) []database.FavoriteMod {
	return a.service().ListFavoriteMods(listID)
}

func (a *App) ListFavoriteContents(listID string) database.FavoriteListContents {
	return a.service().ListFavoriteContents(listID)
}

func (a *App) AddFavoriteMod(mod database.FavoriteMod) database.FavoriteMod {
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

func (a *App) AddFavoriteListReference(parentListID, childListID string) database.FavoriteListRef {
	return a.service().AddFavoriteListReference(parentListID, childListID)
}

func (a *App) RemoveFavoriteListReference(parentListID, childListID string) bool {
	return a.service().RemoveFavoriteListReference(parentListID, childListID)
}

func (a *App) ListFavoriteListRefs(parentListID string) []database.FavoriteListRef {
	return a.service().ListFavoriteListRefs(parentListID)
}

func (a *App) RemoveFavoriteMod(listID, platform, modID, mcVersion, modLoader string) bool {
	return a.service().RemoveFavoriteMod(listID, platform, modID, mcVersion, modLoader)
}

func (a *App) GetMinecraftReleaseVersions() []string {
	return a.service().GetMinecraftReleaseVersions()
}

func (a *App) GetPreferences() AppPreferences {
	prefs := a.service().GetPreferences()
	return AppPreferences{
		Theme:                       prefs.Theme,
		AnimationMode:               prefs.AnimationMode,
		AnimationEnabled:            prefs.AnimationEnabled,
		AnimationDurationMultiplier: prefs.AnimationDurationMultiplier,
	}
}

// SettingsView is a settings snapshot returned to the frontend. API keys use an "existence + mask" strategy,
// not sending raw keys back to the frontend; the frontend overwrites via SaveApiKeys.
type SettingsView struct {
	Theme                       string  `json:"theme"` // dark | light | system
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
	MinecraftDir                string  `json:"minecraftDir"` // simplified path (with env vars)
	CacheDir                    string  `json:"cacheDir"`
	CachePath                   string  `json:"cachePath"`
	HasCurseforgeKey            bool    `json:"hasCurseforgeKey"`
	CurseforgeKeyMask           string  `json:"curseforgeKeyMask"` // e.g. "abcd****wxyz" or ""
	HasModrinthKey              bool    `json:"hasModrinthKey"`
	ModrinthKeyMask             string  `json:"modrinthKeyMask"`
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

func (a *App) SaveAnimationSettings(req SaveAnimationSettingsRequest) SettingsView {
	next := a.service().SaveAnimationSettings(appcore.SaveAnimationSettingsRequest{
		AnimationMode:               req.AnimationMode,
		AnimationEnabled:            req.AnimationEnabled,
		AnimationDurationMultiplier: req.AnimationDurationMultiplier,
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

func (a *App) CancelDownload(id string) bool {
	return a.service().CancelDownload(id)
}

func (a *App) RetryDownload(id string) bool {
	return a.service().RetryDownload(id)
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

func (a *App) ChooseMinecraftDir() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:           "Choose .minecraft folder",
		ShowHiddenFiles: true,
	})
	if err != nil {
		logging.Error("choose minecraft dir failed", "error", err)
		return ""
	}
	return a.service().SetMinecraftDir(dir)
}

func (a *App) ChooseCacheDir() SettingsView {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:           "Choose cache folder",
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

func (a *App) ApplyLocalModBatchOperation(req appstructs.LocalModBatchOperationRequest) structs.VersionInfo {
	version, err := a.service().ApplyLocalModBatchOperation(req)
	if err != nil {
		panic("local mod operation failed: " + err.Error())
	}
	return version
}

func (a *App) SelectVersion(versionKey string) structs.VersionInfo {
	version, err := a.service().SelectVersion(versionKey)
	if err == nil {
		return version
	}
	message := err.Error()
	if strings.HasPrefix(message, "version not found:") {
		message = "version not found"
	}
	panic("select version failed: " + message)
}

func settingsViewFromCore(sv appcore.SettingsView) SettingsView {
	return SettingsView{
		Theme:                       sv.Theme,
		AnimationMode:               sv.AnimationMode,
		AnimationEnabled:            sv.AnimationEnabled,
		AnimationDurationMultiplier: sv.AnimationDurationMultiplier,
		MinecraftDir:                sv.MinecraftDir,
		CacheDir:                    sv.CacheDir,
		CachePath:                   sv.CachePath,
		HasCurseforgeKey:            sv.HasCurseforgeKey,
		CurseforgeKeyMask:           sv.CurseforgeKeyMask,
		HasModrinthKey:              sv.HasModrinthKey,
		ModrinthKeyMask:             sv.ModrinthKeyMask,
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
