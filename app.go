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
	"mod-downloader/httpserver"
	"mod-downloader/logging"
	"mod-downloader/minecraft"
	"mod-downloader/models"
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
const downloadStatesUpdatedEvent = "download-states-updated"

// App struct
type App struct {
	ctx    context.Context
	config *configs.Config
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

	a.server = httpserver.New(ctx, httpserver.DefaultAddr)
	if err := a.server.Start(); err != nil {
		logging.Error("start http server failed", "error", err)
	}
}

func (a *App) SearchMods(req appstructs.SearchModsRequest) {
	providers.SearchMods(req, func(update appstructs.SearchModsUpdate) {
		runtime.EventsEmit(a.ctx, searchModsUpdatedEvent, update)
	})
}

func (a *App) ListMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
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

func (a *App) ListPinnedMods() []database.PinnedMod {
	return database.ListPinnedMods()
}

func (a *App) UnpinMod(platform, modID, mcVersion, modLoader string) bool {
	platform = strings.ToLower(strings.TrimSpace(platform))
	modID = strings.ToLower(strings.TrimSpace(modID))
	mcVersion = strings.TrimSpace(mcVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))
	if platform == "" || modID == "" || mcVersion == "" || modLoader == "" {
		return false
	}
	if _, found := database.GetPinnedMod(platform, modID, mcVersion, modLoader); !found {
		return false
	}
	if err := database.DeletePinnedMod(platform, modID, mcVersion, modLoader); err != nil {
		logging.Error("unpin mod failed", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader, "error", err)
		return false
	}
	return true
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

// SettingsView 是返回给前端的设置快照。API key 采用“是否存在 + 掩码”策略，
// 不把原始密钥回传给前端，前端通过 SaveApiKeys 覆盖写。
type SettingsView struct {
	Theme             string `json:"theme"`        // dark | light | system
	MinecraftDir      string `json:"minecraftDir"` // 简化路径（含环境变量）
	HasCurseforgeKey  bool   `json:"hasCurseforgeKey"`
	CurseforgeKeyMask string `json:"curseforgeKeyMask"` // 形如 "abcd****wxyz" 或 ""
	HasModrinthKey    bool   `json:"hasModrinthKey"`
	ModrinthKeyMask   string `json:"modrinthKeyMask"`
}

// SaveApiKeysRequest 是前端保存 API key 的请求结构。
type SaveApiKeysRequest struct {
	CurseforgeApiKey string `json:"curseforgeApiKey"`
	ModrinthApiKey   string `json:"modrinthApiKey"`
}

// 约定：字段值为字符串 "<keep>" 时表示不修改原值（因为前端拿不到明文）。
// 空字符串 "" 表示清除。其他值表示覆盖。
const apiKeyKeepSentinel = "<keep>"

// GetSettings 返回当前设置的只读视图。
func (a *App) GetSettings() SettingsView {
	sv := SettingsView{Theme: configs.ThemeDark.String()}
	if a.config != nil {
		sv.Theme = a.config.Prefers.Theme.Normalized().String()
		sv.MinecraftDir = minecraft.SimplifyPathWithEnv(a.config.Prefers.MinecraftDir)
		sv.HasCurseforgeKey = strings.TrimSpace(a.config.Keys.CurseforgeApiKey) != ""
		sv.CurseforgeKeyMask = maskKey(a.config.Keys.CurseforgeApiKey)
		sv.HasModrinthKey = strings.TrimSpace(a.config.Keys.ModrinthApiKey) != ""
		sv.ModrinthKeyMask = maskKey(a.config.Keys.ModrinthApiKey)
	}
	if sv.MinecraftDir == "" {
		sv.MinecraftDir = minecraft.SimplifyPathWithEnv(global.GetMinecraftDir())
	}
	return sv
}

// maskKey 保留前 4 与后 4 字符，中间以 **** 替换；过短则全掩。
func maskKey(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) <= 8 {
		return "****"
	}
	return s[:4] + "****" + s[len(s)-4:]
}

// SaveTheme 更新主题偏好并立即持久化。返回更新后的主题字符串。
// 非法值回退为 dark。
func (a *App) SaveTheme(theme string) string {
	if a.config == nil {
		a.config = &configs.Config{}
	}
	parsed := configs.ParseTheme(theme)
	if parsed == "" {
		parsed = configs.ThemeDark
	}
	a.config.Prefers.Theme = parsed
	if err := configs.Save(a.config); err != nil {
		logging.Error("save theme failed", "theme", parsed, "error", err)
	}
	return parsed.String()
}

// SaveApiKeys 更新 API key 并立即持久化 + 重初始化客户端。
// 传空字符串表示清除该 key；传特殊值（见上）表示保持不变。
func (a *App) SaveApiKeys(req SaveApiKeysRequest) SettingsView {
	if a.config == nil {
		a.config = &configs.Config{}
	}
	if req.CurseforgeApiKey != apiKeyKeepSentinel {
		a.config.Keys.CurseforgeApiKey = strings.TrimSpace(req.CurseforgeApiKey)
	}
	if req.ModrinthApiKey != apiKeyKeepSentinel {
		a.config.Keys.ModrinthApiKey = strings.TrimSpace(req.ModrinthApiKey)
	}
	if err := configs.Save(a.config); err != nil {
		logging.Error("save api keys failed", "error", err)
	}
	// 重初始化 CurseForge 客户端
	if strings.TrimSpace(a.config.Keys.CurseforgeApiKey) != "" {
		global.SetCurseForgeClient(curseforge.NewClient(a.config.Keys.CurseforgeApiKey))
	} else {
		global.SetCurseForgeClient(nil)
	}
	// Modrinth 当前不使用 key，保留字段供未来使用
	return a.GetSettings()
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
	return downloader.GetDownloadStates(req, func() {
		runtime.EventsEmit(a.ctx, downloadStatesUpdatedEvent)
	})
}

func (a *App) shutdown(ctx context.Context) {
	if a.server != nil {
		a.server.Stop()
	}
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
	global.HardlinkIndexClear()
	global.InvalidateVersions()
	versions := loadVersionsFromDisk(dir)
	if len(versions) == 0 {
		logging.Warn("chosen minecraft dir has no versions", "minecraftDir", dir)
		global.SetMinecraftDir(previousDir)
		global.ClearLocalMods()
		global.HardlinkIndexClear()
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
		ensureSelectedVersion(versions)
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

	infos := minecraft.LoadLauncherVersions(mcDir, loadMinecraftDirVersions)
	global.SetVersionsForDir(mcDir, infos)
	ensureSelectedVersion(infos)
	generation := global.HardlinkIndexGeneration()
	go scanAllModDirsForHardlinkIndex(mcDir, infos, generation)
	return infos
}

// loadMinecraftDirVersions scans <gameDir>/versions/*/ and returns the
// recognized Minecraft instances. gameDir is the directory that directly
// contains the versions/ subfolder (i.e. a .minecraft folder, or a Prism
// instance's .minecraft subfolder).
func loadMinecraftDirVersions(gameDir string) []structs.VersionInfo {
	versionsDir := filepath.Join(gameDir, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		logging.Error("read versions dir failed", "versionsDir", versionsDir, "error", err)
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
	return infos
}

func ensureSelectedVersion(versions []structs.VersionInfo) {
	if len(versions) == 0 {
		return
	}
	selected := global.GetSelectedVersion()
	if selected.ID != "" || selected.Name != "" {
		return
	}
	global.SetSelectedVersion(versions[0])
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
	// versionDirName is the composite ID (e.g. "MyInstance/fabric-loader-...")
	// for Prism versions; it serves as the unique instanceID for local mod
	// storage. versionDir is the absolute path to the actual on-disk version
	// folder, which for Prism lives under <instances>/<instance>/.minecraft/.
	versionDir := minecraft.VersionDirPath(mcDir, version)
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

func scanAllModDirsForHardlinkIndex(mcDir string, versions []structs.VersionInfo, generation uint64) {
	if global.GetMinecraftDir() != mcDir || global.HardlinkIndexGeneration() != generation {
		return
	}
	minecraft.ScanAllModDirsForHardlink(mcDir, versions, func(sha1, path string) bool {
		if global.GetMinecraftDir() != mcDir {
			return false
		}
		return global.HardlinkIndexAddForGeneration(sha1, path, generation)
	})
}
