package main

import (
	"context"
	"strings"

	"mod-downloader/appcore"
	"mod-downloader/configs"
	"mod-downloader/database"
	"mod-downloader/httpserver"
	"mod-downloader/logging"
	"mod-downloader/models"
	appstructs "mod-downloader/structs"
	structs "mod-downloader/structs/minecraft"

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
	Theme string `json:"theme"`
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

	a.server = httpserver.New(ctx, httpserver.DefaultAddr)
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

func (a *App) GetMinecraftReleaseVersions() []string {
	return a.service().GetMinecraftReleaseVersions()
}

func (a *App) GetPreferences() AppPreferences {
	prefs := a.service().GetPreferences()
	return AppPreferences{Theme: prefs.Theme}
}

// SettingsView is a settings snapshot returned to the frontend. API keys use an "existence + mask" strategy,
// not sending raw keys back to the frontend; the frontend overwrites via SaveApiKeys.
type SettingsView struct {
	Theme             string `json:"theme"`        // dark | light | system
	MinecraftDir      string `json:"minecraftDir"` // simplified path (with env vars)
	HasCurseforgeKey  bool   `json:"hasCurseforgeKey"`
	CurseforgeKeyMask string `json:"curseforgeKeyMask"` // e.g. "abcd****wxyz" or ""
	HasModrinthKey    bool   `json:"hasModrinthKey"`
	ModrinthKeyMask   string `json:"modrinthKeyMask"`
}

// SaveApiKeysRequest is the request structure for the frontend to save API keys.
type SaveApiKeysRequest struct {
	CurseforgeApiKey string `json:"curseforgeApiKey"`
	ModrinthApiKey   string `json:"modrinthApiKey"`
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
		Theme:             sv.Theme,
		MinecraftDir:      sv.MinecraftDir,
		HasCurseforgeKey:  sv.HasCurseforgeKey,
		CurseforgeKeyMask: sv.CurseforgeKeyMask,
		HasModrinthKey:    sv.HasModrinthKey,
		ModrinthKeyMask:   sv.ModrinthKeyMask,
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
