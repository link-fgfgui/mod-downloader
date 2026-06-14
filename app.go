package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mod-downloader/configs"
	"mod-downloader/database"
	"mod-downloader/downloader"
	"mod-downloader/global"
	"mod-downloader/logging"
	"mod-downloader/minecraft"
	"mod-downloader/providers"
	appstructs "mod-downloader/structs"
	structs "mod-downloader/structs/minecraft"

	modrinth "codeberg.org/jmansfield/go-modrinth/modrinth"

	curseforge "github.com/sjet47/go-curseforge"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const minecraftDirChangedEvent = "minecraft-dir-changed"
const selectedVersionChangedEvent = "selected-version-changed"
const searchModsUpdatedEvent = "search-mods-updated"

// App struct
type App struct {
	ctx    context.Context
	config *configs.Config
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

	cfg, err := configs.Load()
	if err != nil {
		logging.Error("load config failed", "error", err)
		cfg = &configs.Config{}
	}
	a.config = cfg
	global.SetMinecraftDir(cfg.Prefers.MinecraftDir)

	if err := database.Open(); err != nil {
		logging.Error("open database failed", "error", err)
	}

	if cfg.Keys.CurseforgeApiKey != "" {
		global.SetCurseForgeClient(curseforge.NewClient(cfg.Keys.CurseforgeApiKey))
	}
	modrinthClient := modrinth.NewClient(&http.Client{Timeout: 10 * time.Second})
	modrinthClient.UserAgent = "mod-downloader"
	global.SetModrinthClient(modrinthClient)

	releaseVersions, err := minecraft.FetchMinecraftReleaseVersions()
	if err != nil {
		logging.Error("fetch minecraft release versions failed", "error", err)
		return
	}
	global.SetMinecraftReleaseVersions(releaseVersions)
}

func (a *App) SearchMods(req appstructs.SearchModsRequest) {
	providers.SearchMods(req, func(update appstructs.SearchModsUpdate) {
		runtime.EventsEmit(a.ctx, searchModsUpdatedEvent, update)
	})
}

func (a *App) ListMatchingProjectVersions(result appstructs.SearchModResult, minecraftVersion string, modLoader string) []appstructs.ProjectVersionResult {
	return providers.ListMatchingProjectVersions(result, minecraftVersion, modLoader)
}

func (a *App) GetPinnedModVersion(platform string, modID string, minecraftVersion string, modLoader string) database.PinnedMod {
	pin, _ := database.GetPinnedMod(platform, modID, minecraftVersion, modLoader)
	return pin
}

func (a *App) PinModVersion(req appstructs.ModVersionPinRequest) database.PinnedMod {
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	modID := strings.ToLower(strings.TrimSpace(req.ModID))
	versionID := strings.TrimSpace(req.VersionID)
	mcVersion := strings.TrimSpace(req.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(req.ModLoader))

	if platform == "" || modID == "" || versionID == "" || mcVersion == "" || modLoader == "" {
		return database.PinnedMod{}
	}

	existing, found := database.GetPinnedMod(platform, modID, mcVersion, modLoader)
	if found && strings.EqualFold(existing.VersionID, versionID) {
		_ = database.DeletePinnedMod(platform, modID, mcVersion, modLoader)
		return database.PinnedMod{}
	}

	pin := database.PinnedMod{
		Platform:         platform,
		ModID:            modID,
		VersionID:        versionID,
		MinecraftVersion: mcVersion,
		ModLoader:        modLoader,
	}
	if err := database.UpsertPinnedMod(pin); err != nil {
		logging.Error("upsert pinned mod failed", "platform", platform, "modID", modID, "versionID", versionID, "minecraftVersion", mcVersion, "modLoader", modLoader, "error", err)
		return database.PinnedMod{}
	}
	pin, _ = database.GetPinnedMod(platform, modID, mcVersion, modLoader)
	return pin
}

func (a *App) GetMinecraftReleaseVersions() []string {
	return minecraft.GetMinecraftReleaseVersions()
}

func (a *App) GetPreferences() AppPreferences {
	if a.config == nil {
		return AppPreferences{Theme: configs.ThemeDark.String()}
	}
	return AppPreferences{Theme: a.config.Prefers.Theme.Normalized().String()}
}

func (a *App) QueueModDownload(req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
	return downloader.QueueModDownload(a.ctx, req)
}

func (a *App) GetDownloadQueueState() appstructs.DownloadQueueState {
	return downloader.GetDownloadQueueState()
}

func (a *App) CancelDownload(id string) bool {
	return downloader.CancelDownload(a.ctx, id)
}

func (a *App) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
	return downloader.GetDownloadStates(req)
}

func (a *App) shutdown(ctx context.Context) {
	if a.config == nil {
		a.config = &configs.Config{}
	}

	a.config.Prefers.MinecraftDir = global.GetMinecraftDir()
	if err := configs.Save(a.config); err != nil {
		logging.Error("save config failed", "error", err)
	}
	database.Close()
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
	if dir == "" {
		return ""
	}

	previousDir := global.GetMinecraftDir()
	simplified := minecraft.SimplifyPathWithEnv(dir)
	global.SetMinecraftDir(dir)
	global.ClearLocalMods()
	global.InvalidateVersions()
	versions := loadVersionsFromDisk(dir)
	if len(versions) == 0 {
		logging.Warn("chosen minecraft dir has no versions", "minecraftDir", dir)
		global.SetMinecraftDir(previousDir)
		global.ClearLocalMods()
		global.InvalidateVersions()
		if strings.TrimSpace(previousDir) != "" {
			loadVersionsFromDisk(previousDir)
		}
		return ""
	}
	runtime.EventsEmit(a.ctx, minecraftDirChangedEvent, simplified)
	runtime.EventsEmit(a.ctx, selectedVersionChangedEvent, global.GetSelectedVersion())
	return simplified
}

func (a *App) GetMinecraftDir() string {
	return global.GetMinecraftDir()
}

func (a *App) ValidateMinecraftDir() bool {
	resolvedDir := global.GetMinecraftDir()
	if resolvedDir == "" {
		return false
	}
	info, err := os.Stat(resolvedDir)
	if err == nil && info.IsDir() {
		return true
	}
	return false
}

func (a *App) GetVersions() []structs.VersionInfo {
	mcDir := global.GetMinecraftDir()
	if versions, ok := global.GetVersionsForDir(mcDir); ok {
		return versions
	}
	return loadVersionsFromDisk(mcDir)
}

// GetSelectedVersion 返回当前选中的实例（含 MinecraftVersion / ModLoader），
// 供前端在挂载时主动拉取，避免错过 selected-version-changed 事件导致与实际选中实例不同步。
func (a *App) GetSelectedVersion() structs.VersionInfo {
	return global.GetSelectedVersion()
}

func (a *App) RefreshVersions() []structs.VersionInfo {
	return loadVersionsFromDisk(global.GetMinecraftDir())
}

func (a *App) RefreshSelectedVersionMods() structs.VersionInfo {
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return structs.VersionInfo{}
	}
	refreshed := refreshVersionMods(selected, global.GetMinecraftDir())
	global.SetSelectedVersion(refreshed)
	runtime.EventsEmit(a.ctx, selectedVersionChangedEvent, refreshed)
	return refreshed
}

func loadVersionsFromDisk(mcDir string) []structs.VersionInfo {
	if strings.TrimSpace(mcDir) == "" {
		global.SetVersionsForDir(mcDir, nil)
		return nil
	}

	versionsDir := filepath.Join(mcDir, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		logging.Error("read versions dir failed", "versionsDir", versionsDir, "error", err)
		global.SetVersionsForDir(mcDir, nil)
		return nil
	}

	infos := make([]structs.VersionInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		versionID := entry.Name()
		versionDir := filepath.Join(versionsDir, versionID)
		jsonPath := filepath.Join(versionDir, versionID+".json")
		info, ok := minecraft.CheckManifest(jsonPath)
		if !ok {
			continue
		}
		if !validMinecraftInstance(info) {
			logging.Warn("skip invalid minecraft version", "versionID", versionID, "minecraftVersion", info.MinecraftVersion, "modLoader", info.ModLoader)
			continue
		}
		infos = append(infos, info)
	}

	global.SetVersionsForDir(mcDir, infos)
	return infos
}

func scanAllVersionMods(versions []structs.VersionInfo, mcDir string) []structs.VersionInfo {
	global.ClearLocalMods()
	scanned := make([]structs.VersionInfo, len(versions))
	for i, version := range versions {
		scanned[i] = scanVersionMods(version, mcDir)
	}
	global.SetVersionsForDir(mcDir, scanned)
	return scanned
}

func refreshVersionMods(version structs.VersionInfo, mcDir string) structs.VersionInfo {
	global.ClearLocalModsByInstance(versionInstanceDir(version))
	refreshed := scanVersionMods(version, mcDir)
	if versions, ok := global.GetVersionsForDir(mcDir); ok {
		next := make([]structs.VersionInfo, len(versions))
		copy(next, versions)
		for i, cached := range next {
			if cached.Name == version.Name || cached.ID == version.ID {
				next[i] = refreshed
			}
		}
		global.SetVersionsForDir(mcDir, next)
	}
	return refreshed
}

func scanVersionMods(version structs.VersionInfo, mcDir string) structs.VersionInfo {
	versionDirName := versionInstanceDir(version)
	if strings.TrimSpace(mcDir) == "" || strings.TrimSpace(versionDirName) == "" {
		version.Mods = nil
		return version
	}
	versionDir := filepath.Join(mcDir, "versions", versionDirName)
	version.Mods = minecraft.ScanVersionMods(versionDir, versionDirName, version.MinecraftVersion, version.ModLoader, mcDir)
	return version
}

func (a *App) SelectVersion(versionKey string) structs.VersionInfo {
	versionKey = strings.TrimSpace(versionKey)
	if versionKey == "" {
		panic("select version failed: empty version key")
	}

	mcDir := global.GetMinecraftDir()
	if _, ok := global.GetVersionsForDir(mcDir); !ok {
		loadVersionsFromDisk(mcDir)
	}

	if version, ok := global.GetVersionByKey(versionKey); ok {
		if !validMinecraftInstance(version) {
			panic("select version failed: invalid minecraft version or mod loader")
		}
		version = refreshVersionMods(version, mcDir)
		global.SetSelectedVersion(version)
		runtime.EventsEmit(a.ctx, selectedVersionChangedEvent, version)
		return version
	}

	panic("select version failed: version not found")
}

func findVersionByKey(versions []structs.VersionInfo, key string) (structs.VersionInfo, bool) {
	for _, version := range versions {
		if version.Name == key || version.ID == key {
			return version, true
		}
	}
	return structs.VersionInfo{}, false
}

func versionInstanceDir(version structs.VersionInfo) string {
	if strings.TrimSpace(version.ID) != "" {
		return strings.TrimSpace(version.ID)
	}
	return strings.TrimSpace(version.Name)
}

func validMinecraftInstance(version structs.VersionInfo) bool {
	switch strings.ToLower(strings.TrimSpace(version.ModLoader)) {
	case "fabric", "forge", "neoforge":
		return strings.TrimSpace(version.MinecraftVersion) != ""
	default:
		return false
	}
}
