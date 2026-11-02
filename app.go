// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/appcore"
	"github.com/link-fgfgui/mod-downloader/core/host"
	"github.com/link-fgfgui/mod-downloader/core/instance"
	"github.com/link-fgfgui/mod-downloader/core/logging"
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// App struct
type App struct {
	app      *application.App
	host     *host.Host
	quitFunc func()
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	a.host = host.New(host.Options{
		Version:               currentAppVersion(),
		LoadMinecraftReleases: true,
	})
	a.registerCoreEvents(a.host)
	if err := a.host.Start(ctx); err != nil {
		if errors.Is(err, instance.ErrAlreadyRunning) {
			// Wails has not shown the window yet. Abort the run instead of
			// opening a second window against the same storage.
			return err
		}
		// A partially initialized core must not be used: fail fast and leave the
		// HTTP server down so the frontend sees explicit rejections instead of
		// silent empty results. GetStartupStatus reports the typed failure.
		logging.Error("core startup failed", "error", err)
		return nil
	}
	return nil
}

func (a *App) service() *appcore.Service {
	if a.host != nil {
		if svc := a.host.Service(); svc != nil {
			return svc
		}
	}
	reason := "core not initialized"
	if a.host != nil {
		if status := a.host.StartupStatus(); status.Error != "" {
			reason = "core startup failed: " + status.Error
		}
	}
	logging.Error("app service unavailable", "reason", reason)
	panic(reason)
}

// emit forwards a core event to the Wails event bus. Events emitted before the
// application exists are dropped.
func (a *App) emit(eventName string, payload any) {
	if a.app == nil {
		return
	}
	a.app.Event.Emit(eventName, payload)
}

// GetStartupStatus reports whether the core is usable without invoking any
// business RPC. Frontend bootstrap must call this once before mounting stores.
func (a *App) GetStartupStatus() appstructs.StartupStatus {
	if a.host == nil {
		return appstructs.StartupStatus{}
	}
	return a.host.StartupStatus()
}

// Quit asks the native shell to exit; the startup failure screen uses it.
func (a *App) Quit() {
	if a.quitFunc != nil {
		a.quitFunc()
		return
	}
	if a.app != nil {
		a.app.Quit()
	}
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

// ListFavoriteOverview returns the scope-filtered, core-ordered favorites
// projection. The frontend renders pinned/ungrouped as-is.
func (a *App) ListFavoriteOverview(minecraftVersion string, modLoader string) appview.FavoriteOverview {
	return a.service().ListFavoriteOverview(minecraftVersion, modLoader)
}

// ListFavoriteContentsForScope returns a list's contents filtered to one scope.
func (a *App) ListFavoriteContentsForScope(listID string, minecraftVersion string, modLoader string) storage.FavoriteListContents {
	return a.service().ListFavoriteContentsForScope(listID, minecraftVersion, modLoader)
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

func (a *App) RemoveFavoriteMod(listID, platform, modID, mcVersion, modLoader string) bool {
	return a.service().RemoveFavoriteMod(listID, platform, modID, mcVersion, modLoader)
}

func (a *App) ExportFavoriteListPackwizZip(listID, minecraftVersion, modLoader, locale string) ExportFavoritePackwizResult {
	dialogText := localizedDialogText(locale)
	path, err := a.app.Dialog.SaveFileWithOptions(&application.SaveFileDialogOptions{
		Title:                dialogText.exportPackwizTitle,
		Filename:             a.service().FavoriteListPackwizDefaultFilename(listID),
		CanCreateDirectories: true,
		Window:               a.dialogWindow(),
		Filters: []application.FileFilter{{
			DisplayName: dialogText.zipArchives,
			Pattern:     "*.zip",
		}},
	}).PromptForSingleSelection()
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

// GetSettings returns a read-only view of the current settings.
func (a *App) GetSettings() appview.SettingsView {
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

func (a *App) SaveAnimationSettings(req appstructs.SaveAnimationSettingsRequest) appview.SettingsView {
	return a.service().SaveAnimationSettings(req)
}

func (a *App) SaveSimpleModeSettings(req appstructs.SaveSimpleModeSettingsRequest) appview.SettingsView {
	return a.service().SaveSimpleModeSettings(req)
}

func (a *App) SaveUnusedDependencyCleanupSettings(req appstructs.SaveUnusedDependencyCleanupSettingsRequest) appview.SettingsView {
	return a.service().SaveUnusedDependencyCleanupSettings(req)
}

func (a *App) SaveMCIMSettings(req appstructs.SaveMCIMSettingsRequest) appview.SettingsView {
	return a.service().SaveMCIMSettings(req)
}

func (a *App) SaveJIJDetectionSettings(req appstructs.SaveJIJDetectionSettingsRequest) appview.SettingsView {
	return a.service().SaveJIJDetectionSettings(req)
}

func (a *App) SaveMultiSelectSettings(req appstructs.SaveMultiSelectSettingsRequest) appview.SettingsView {
	return a.service().SaveMultiSelectSettings(req)
}

func (a *App) SavePreferredReleaseTypeSettings(req appstructs.SavePreferredReleaseTypeSettingsRequest) appview.SettingsView {
	return a.service().SavePreferredReleaseTypeSettings(req)
}

func (a *App) SaveHardwareAccelerationSettings(req appstructs.SaveHardwareAccelerationSettingsRequest) appview.SettingsView {
	return a.service().SaveHardwareAccelerationSettings(req)
}

func (a *App) SaveNetworkSettings(req appstructs.SaveNetworkSettingsRequest) appview.SettingsView {
	return a.service().SaveNetworkSettings(req)
}

func (a *App) SaveCacheDirPreference(dir string) appview.SettingsView {
	return a.service().SaveCacheDirPreference(dir)
}

// SaveAPIKeys updates API keys and persists them immediately + reinitializes clients.
// An empty string means clear the key; the special value (see above) means keep unchanged.
func (a *App) SaveAPIKeys(req appstructs.SaveAPIKeysRequest) appview.SettingsView {
	return a.service().SaveAPIKeys(req)
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

func (a *App) GetDownloadQueueState() appview.DownloadQueueState {
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

func (a *App) GetDownloadStates(req appstructs.DownloadStatesRequest) []appview.ModDownloadButtonState {
	return a.service().GetDownloadStates(req)
}

func (a *App) ServiceShutdown() error {
	if a.host != nil {
		a.host.Shutdown()
	}
	return nil
}

func (a *App) ChooseMinecraftDir(locale string) (DirectorySelectionResult, error) {
	dir, err := a.app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                localizedDialogText(locale).chooseMinecraftDir,
		Directory:            a.service().GetMinecraftDir(),
		ShowHiddenFiles:      true,
		CanChooseDirectories: true,
		CanChooseFiles:       false,
		Window:               a.dialogWindow(),
	}).PromptForSingleSelection()
	if err != nil {
		logging.Error("choose minecraft dir failed", "error", err)
		return DirectorySelectionResult{}, fmt.Errorf("open folder picker: %w", err)
	}
	if strings.TrimSpace(dir) == "" {
		return DirectorySelectionResult{Canceled: true}, nil
	}
	chosen := a.service().SetMinecraftDir(dir)
	if strings.TrimSpace(chosen) == "" {
		return DirectorySelectionResult{Invalid: true}, nil
	}
	// SetMinecraftDir only validates and switches the in-memory directory.
	// Persist the successful choice so it survives the next launch.
	a.service().SaveMinecraftDirPreference(chosen)
	return DirectorySelectionResult{Path: chosen}, nil
}

// dialogWindow returns the active window for native dialogs. Server mode has
// no window and the dialog implementation accepts nil there.
func (a *App) dialogWindow() application.Window {
	if a == nil || a.app == nil || a.app.Window == nil {
		return nil
	}
	return a.app.Window.Current()
}

func (a *App) ChooseCacheDir(locale string) (appview.SettingsView, error) {
	dir, err := a.app.Dialog.OpenFileWithOptions(&application.OpenFileDialogOptions{
		Title:                localizedDialogText(locale).chooseCacheDir,
		Directory:            a.GetSettings().CacheDir,
		ShowHiddenFiles:      true,
		CanChooseDirectories: true,
		CanChooseFiles:       false,
		Window:               a.dialogWindow(),
	}).PromptForSingleSelection()
	if err != nil {
		logging.Error("choose cache dir failed", "error", err)
		return a.GetSettings(), fmt.Errorf("open folder picker: %w", err)
	}
	if strings.TrimSpace(dir) == "" {
		return a.GetSettings(), nil
	}
	return a.SaveCacheDirPreference(dir), nil
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

// GetLocalModView returns the core-built Manage projection for one instance:
// grouped rows, the dependency forest and reverse dependents. Grouping, JIJ
// de-duplication, search text and loader compatibility all live in core so the
// GUI and TUI render the same rows.
func (a *App) GetLocalModView(versionKey string, modLoader string) appview.LocalModView {
	return a.service().GetLocalModView(versionKey, modLoader)
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
