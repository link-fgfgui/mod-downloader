package appcore

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	modrinth "codeberg.org/jmansfield/go-modrinth/modrinth"
	"github.com/fsnotify/fsnotify"
	"github.com/sjet47/go-curseforge"
	curseforgeapi "github.com/sjet47/go-curseforge/api"

	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/link-fgfgui/mod-downloader-core/downloader"
	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/networking"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

const minecraftReleaseCacheTTL = 24 * time.Hour

var (
	fetchMinecraftReleaseVersions   = minecraft.FetchMinecraftReleaseVersions
	configureRemoteModIDConcurrency = modbridge.ConfigureRemoteModIDConcurrency
	configureSimpleMode             = downloader.ConfigureSimpleMode
)

// activeService enforces the documented single-Service process constraint.
// Startup registers before touching process-wide providers/storage, and
// Shutdown/Close release the slot only after all resources are drained.
var activeService struct {
	sync.Mutex
	service *Service
}

func registerActiveService(service *Service) error {
	activeService.Lock()
	defer activeService.Unlock()
	if activeService.service != nil && activeService.service != service {
		return errors.New("another appcore.Service is already running in this process")
	}
	activeService.service = service
	return nil
}

func unregisterActiveService(service *Service) {
	activeService.Lock()
	defer activeService.Unlock()
	if activeService.service == service {
		activeService.service = nil
	}
}

const APIKeyKeepSentinel = "<keep>"

var (
	favoriteCrossVersionCopyLookupProjectByID   = providers.LookupProjectByPlatformID
	favoriteCrossVersionCopyLookupProjectBySlug = providers.LookupProjectByPlatformSlug
)

const (
	LocalModBatchActionEnable  = "enable"
	LocalModBatchActionDisable = "disable"
	LocalModBatchActionInvert  = "invert"
	LocalModBatchActionDelete  = "delete"
)

func New(options Options) *Service {
	return &Service{config: options.Config, options: options}
}

func (s *Service) UserAgent() string {
	return userAgentForVersion(s.options.Version)
}

func (s *Service) Startup(ctx context.Context) error {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.lifecycleState == serviceStateStarted {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := registerActiveService(s); err != nil {
		return err
	}
	s.backgroundMu.Lock()
	s.backgroundDone = false
	s.backgroundMu.Unlock()
	s.ctx = ctx
	s.lifecycleState = serviceStateStarting
	if err := s.startup(); err != nil {
		// Startup can fail after providers, storage and logging have been
		// initialized. Tear those down before returning so callers never see a
		// half-started service.
		s.stop(false)
		unregisterActiveService(s)
		s.lifecycleState = serviceStateIdle
		return err
	}
	s.lifecycleState = serviceStateStarted
	return nil
}

func (s *Service) startup() error {
	providers.StartBackgroundTasks()
	if s.config != nil {
		s.configureLoggingConfig(s.config.Logging)
	} else if !s.options.Runtime.NoConfigFile {
		if bootstrapLogging, err := configs.LoadLogging(); err == nil {
			s.configureLoggingConfig(bootstrapLogging)
		}
	}
	if s.config == nil {
		cfg, err := s.loadConfig()
		if err != nil {
			logging.Error("load config failed", "error", err)
			return &StartupError{Stage: "config", Err: err}
		}
		s.config = cfg
	}
	s.applyConfigOverrides()
	s.configureLogging()
	s.applyRuntimeConfig()
	if minecraftDir := minecraft.ExpandPathWithEnv(s.config.Prefers.MinecraftDir); strings.TrimSpace(minecraftDir) != "" {
		s.config.Prefers.MinecraftDir = minecraftDir
		global.SetMinecraftDir(minecraftDir)
	}

	cachePath, err := s.cachePath()
	if err != nil {
		logging.Error("resolve database cache path failed", "error", err)
		return &StartupError{Stage: "storage", Err: fmt.Errorf("resolve cache path: %w", err)}
	}
	if err := storage.OpenAt(cachePath); err != nil {
		logging.Error("open database failed", "path", cachePath, "error", err)
		return &StartupError{Stage: "storage", Err: fmt.Errorf("open storage %s: %w", cachePath, err)}
	}

	if s.options.LoadMinecraftReleases {
		if err := loadMinecraftReleaseVersions(); err != nil {
			return &StartupError{Stage: "releases", Err: err}
		}
	}
	// The queue may still be closed from a previous service in the same
	// process (tests, CLI embedding). Open it only after the runtime it depends
	// on is ready.
	downloader.Start()
	return nil
}

// applyRuntimeConfig pushes the current config into the process-wide runtime
// singletons. Startup and every Save* path that changes runtime behavior must
// go through this single entry point; do not call the individual setters
// elsewhere, or one save path will silently miss an action the others perform.
func (s *Service) applyRuntimeConfig(simpleModeEvents ...downloader.Events) {
	cfg := s.Config()
	downloader.Configure(downloader.Config{
		FileConcurrency:         cfg.Downloads.FileConcurrencyValue(),
		ConcurrentDownloads:     cfg.Downloads.ConcurrentDownloadsValue(),
		AdaptiveFileConcurrency: cfg.Downloads.AdaptiveFileConcurrency,
		TargetDownloadRateMiB:   cfg.Downloads.TargetDownloadRateMiBValue(),
		VerifySHA1:              cfg.Downloads.VerifySHA1,
	})
	configureSimpleMode(cfg.Prefers.SimpleMode, simpleModeEvents...)
	modbridge.ConfigurePreferredReleaseType(cfg.Prefers.PreferredReleaseTypeValue())
	minecraft.SetJijDetectionEnabled(cfg.Prefers.EnableJijDetectionValue())
	configureRemoteModIDConcurrency(cfg.Downloads.ConcurrentDownloadsValue())
	s.configureProviderClients()
}

func loadMinecraftReleaseVersions() error {
	cached, updatedAt := storage.GetMinecraftReleaseVersions()
	if len(cached) > 0 && updatedAt >= time.Now().Add(-minecraftReleaseCacheTTL).Unix() {
		global.SetMinecraftReleaseVersions(cached)
		return nil
	}

	releaseVersions, err := fetchMinecraftReleaseVersions()
	if err != nil {
		if len(cached) > 0 {
			logging.Warn("refresh minecraft release versions failed, using stale cache", "versionCount", len(cached), "updatedAt", updatedAt, "error", err)
			global.SetMinecraftReleaseVersions(cached)
			return nil
		}
		logging.Error("fetch minecraft release versions failed", "error", err)
		return err
	}
	global.SetMinecraftReleaseVersions(releaseVersions)
	if err := storage.SetMinecraftReleaseVersions(releaseVersions, time.Now().Unix()); err != nil {
		logging.Warn("cache minecraft release versions failed", "versionCount", len(releaseVersions), "error", err)
	}
	return nil
}

func (s *Service) Shutdown() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.lifecycleState != serviceStateStarted {
		if s.lifecycleState == serviceStateIdle {
			unregisterActiveService(s)
		}
		return
	}
	s.stop(true)
	unregisterActiveService(s)
	s.lifecycleState = serviceStateIdle
}

func (s *Service) Close() {
	s.lifecycleMu.Lock()
	defer s.lifecycleMu.Unlock()
	if s.lifecycleState != serviceStateStarted {
		if s.lifecycleState == serviceStateIdle {
			unregisterActiveService(s)
		}
		return
	}
	s.stop(false)
	unregisterActiveService(s)
	s.lifecycleState = serviceStateIdle
}

func (s *Service) Config() *configs.Config {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	return s.config
}

func (s *Service) GetMinecraftReleaseVersions() []string {
	return minecraft.GetMinecraftReleaseVersions()
}

func (s *Service) applyConfigOverrides() {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	if s.options.ConfigOverrides.HasMinecraftDir {
		s.config.Prefers.MinecraftDir = s.options.ConfigOverrides.MinecraftDir
	}
	if s.options.ConfigOverrides.HasCurseForgeAPIKey {
		s.config.Keys.CurseforgeApiKey = s.options.ConfigOverrides.CurseForgeAPIKey
	}
	if s.options.ConfigOverrides.HasModrinthAPIKey {
		s.config.Keys.ModrinthApiKey = s.options.ConfigOverrides.ModrinthAPIKey
	}
}

func (s *Service) loadConfig() (*configs.Config, error) {
	if s.options.Runtime.NoConfigFile {
		return configs.LoadEnv()
	}
	return configs.Load()
}

func (s *Service) cachePath() (string, error) {
	if path := s.resolveRuntimePath(s.options.Runtime.CachePath); path != "" {
		return path, nil
	}
	cacheDir := firstNonEmpty(s.options.Runtime.CacheDir, s.Config().Runtime.CacheDir)
	if cacheDir != "" {
		return filepath.Join(s.resolveRuntimePath(cacheDir), storage.CacheFileName), nil
	}
	if cacheDir := s.resolveRuntimePath(s.options.Runtime.DefaultCacheDir); cacheDir != "" {
		return filepath.Join(cacheDir, storage.CacheFileName), nil
	}
	return storage.DefaultCachePath()
}

func (s *Service) applyRuntimeDownloadRequest(req appstructs.ModDownloadRequest) appstructs.ModDownloadRequest {
	if strings.TrimSpace(req.TargetDir) == "" {
		req.TargetDir = s.resolveRuntimePath(s.options.Runtime.TargetModsDir)
	} else {
		req.TargetDir = s.resolveRuntimePath(req.TargetDir)
	}
	if strings.TrimSpace(req.MinecraftVersion) == "" {
		req.MinecraftVersion = firstNonEmpty(s.options.Runtime.MinecraftVersion, s.Config().Runtime.MinecraftVersion)
	}
	if strings.TrimSpace(req.ModLoader) == "" {
		req.ModLoader = firstNonEmpty(s.options.Runtime.ModLoader, s.Config().Runtime.ModLoader)
	}
	return req
}

func (s *Service) applyRuntimeDownloadStatesRequest(req appstructs.DownloadStatesRequest) appstructs.DownloadStatesRequest {
	if strings.TrimSpace(req.TargetDir) == "" {
		req.TargetDir = s.resolveRuntimePath(s.options.Runtime.TargetModsDir)
	} else {
		req.TargetDir = s.resolveRuntimePath(req.TargetDir)
	}
	if strings.TrimSpace(req.MinecraftVersion) == "" {
		req.MinecraftVersion = firstNonEmpty(s.options.Runtime.MinecraftVersion, s.Config().Runtime.MinecraftVersion)
	}
	if strings.TrimSpace(req.ModLoader) == "" {
		req.ModLoader = firstNonEmpty(s.options.Runtime.ModLoader, s.Config().Runtime.ModLoader)
	}
	return req
}

func (s *Service) applyRuntimeBatchDownloadRequest(req appstructs.BatchDownloadRequest) appstructs.BatchDownloadRequest {
	if strings.TrimSpace(req.TargetDir) == "" {
		req.TargetDir = s.resolveRuntimePath(s.options.Runtime.TargetModsDir)
	} else {
		req.TargetDir = s.resolveRuntimePath(req.TargetDir)
	}
	if strings.TrimSpace(req.MinecraftVersion) == "" {
		req.MinecraftVersion = firstNonEmpty(s.options.Runtime.MinecraftVersion, s.Config().Runtime.MinecraftVersion)
	}
	if strings.TrimSpace(req.ModLoader) == "" {
		req.ModLoader = firstNonEmpty(s.options.Runtime.ModLoader, s.Config().Runtime.ModLoader)
	}
	return req
}

func (s *Service) resolveRuntimePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = minecraft.ExpandPathWithEnv(path)
	if filepath.IsAbs(path) || strings.TrimSpace(s.options.Runtime.WorkDir) == "" {
		return filepath.Clean(path)
	}
	return filepath.Clean(filepath.Join(s.options.Runtime.WorkDir, path))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (s *Service) effectiveCurseforgeAPIKey() string {
	if s == nil || s.config == nil {
		return configs.EffectiveCurseforgeAPIKey("")
	}
	return configs.EffectiveCurseforgeAPIKey(s.config.Keys.CurseforgeApiKey)
}

func (s *Service) configureProviderClients() {
	limiter := newRequestRateLimiter(s.Config().API.RequestsPerSecondValue())
	limitedTransport := rateLimitedTransport{limiter: limiter, base: networking.NewTransport()}
	useMCIM := s.Config().Prefers.MCIMEnabledValue()
	curseForgeAPIKey := s.effectiveCurseforgeAPIKey()
	if curseForgeAPIKey != "" || useMCIM {
		apiBaseURL := ""
		if useMCIM {
			apiBaseURL = mcimCurseForgeBaseURL
		}
		global.SetCurseForgeClient(&curseforge.Client{API: curseforgeapi.New(curseForgeTransport{
			apiKey:     curseForgeAPIKey,
			userAgent:  s.UserAgent(),
			apiBaseURL: apiBaseURL,
			base:       limitedTransport,
		})})
	} else {
		global.SetCurseForgeClient(nil)
	}
	modrinthClient := modrinth.NewClient(&http.Client{Timeout: 10 * time.Second, Transport: limitedTransport})
	modrinthClient.UserAgent = s.UserAgent()
	if useMCIM {
		modrinthClient.BaseURL, _ = url.Parse(mcimModrinthBaseURL)
	}
	global.SetModrinthClient(modrinthClient)
}

func (s *Service) configureLogging() {
	s.configureLoggingConfig(s.Config().Logging)
}

func (s *Service) configureLoggingConfig(config configs.LoggingConfig) {
	path := strings.TrimSpace(config.File)
	if path == "" {
		path = logging.DefaultPath()
	} else {
		path = minecraft.ExpandPathWithEnv(path)
	}
	maxSizeBytes := int64(config.MaxSizeMB) * 1024 * 1024
	maxBackups := config.MaxBackups
	if maxBackups == 0 {
		maxBackups = logging.DefaultMaxBackups
	}
	if err := logging.Configure(logging.Options{
		Disabled:     !config.EnabledValue(),
		ForceFile:    config.ForceFile,
		FilePath:     path,
		Level:        s.options.LogLevel,
		NoStderr:     s.options.LogToFileOnly,
		MaxSizeBytes: maxSizeBytes,
		MaxBackups:   maxBackups,
	}); err != nil {
		logging.Error("configure logging failed", "path", path, "error", err)
	}
}

func (s *Service) downloadEvents() downloader.Events {
	return downloader.Events{
		OnQueueState: func(state appstructs.DownloadQueueState) {
			s.emit(EventDownloadQueueUpdated, state)
		},
		OnDownloadFailed: func(event appstructs.DownloadFailedEvent) {
			s.emit(EventDownloadFailed, event)
		},
		OnDownloadCompleted: func() {
			s.incrementUsageStat(storage.UsageStatDownloadsCompleted, 1)
			s.emit(EventDownloadCompleted, nil)
		},
	}
}

func (s *Service) incrementUsageStat(key storage.UsageStatKey, delta int64) {
	if err := storage.IncrementUsageStat(key, delta); err != nil {
		logging.Warn("increment usage stat failed", "key", key, "delta", delta, "error", err)
		return
	}
	s.emit(EventUsageStatsUpdated, storage.GetUsageStats())
}

func (s *Service) emit(kind EventKind, payload any) {
	if s.options.OnEvent != nil {
		s.options.OnEvent(Event{Kind: kind, Payload: payload})
	}
}

func MaskKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

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
		if !ValidMinecraftInstance(info) {
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

func FindVersionByKey(versions []structs.VersionInfo, key string) (structs.VersionInfo, bool) {
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

func ValidMinecraftInstance(version structs.VersionInfo) bool {
	switch strings.ToLower(strings.TrimSpace(version.ModLoader)) {
	case "fabric", "forge", "neoforge":
		return strings.TrimSpace(version.MinecraftVersion) != ""
	default:
		return false
	}
}

func ScanVersionMods(version structs.VersionInfo, mcDir string) structs.VersionInfo {
	version = initializeActualModLoader(version)
	versionDirName := versionInstanceDir(version)
	if strings.TrimSpace(mcDir) == "" || strings.TrimSpace(versionDirName) == "" {
		version.Mods = nil
		return reconcileConnectorVersion(version)
	}
	versionDir := minecraft.VersionDirPath(mcDir, version)
	version.Mods = minecraft.ScanVersionMods(versionDir, versionDirName, version.MinecraftVersion, version.ActualModLoader, mcDir)
	return reconcileConnectorVersion(version)
}

func (s *Service) refreshAndSelectVersionMods(version structs.VersionInfo, mcDir string) structs.VersionInfo {
	refreshed := s.refreshVersionMods(version, mcDir)
	enriched, missingFiles := enrichModMetadata(refreshed.Mods)
	setOnlineMetadataLoading(enriched, missingFiles, true)
	refreshed.Mods = enriched
	global.SetSelectedVersion(refreshed)
	s.emit(EventSelectedVersionChanged, refreshed)
	s.resolveMissingModMetadata(refreshed, missingFiles)
	return refreshed
}

func (s *Service) stopModsWatcher() {
	s.modsWatchRestartMu.Lock()
	defer s.modsWatchRestartMu.Unlock()
	s.modsWatchMu.Lock()
	cancel := s.modsWatchCancel
	done := s.modsWatchDone
	s.modsWatchCancel = nil
	s.modsWatchDone = nil
	s.modsWatchDir = ""
	s.modsWatchMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		<-done
	}
}

func (s *Service) ensureModsWatcher(version structs.VersionInfo) {
	dir := filepath.Join(minecraft.VersionDirPath(global.GetMinecraftDir(), version), "mods")
	dir, err := filepath.Abs(dir)
	if err != nil || strings.TrimSpace(dir) == "" {
		return
	}
	// Serialize replacement so concurrent refreshes cannot leave two watchers
	// running, and refuse to start one after shutdown has begun.
	s.modsWatchRestartMu.Lock()
	defer s.modsWatchRestartMu.Unlock()
	s.backgroundMu.Lock()
	ending := s.backgroundDone
	s.backgroundMu.Unlock()
	if ending {
		return
	}
	s.modsWatchMu.Lock()
	if s.modsWatchDir == dir && s.modsWatchCancel != nil {
		s.modsWatchMu.Unlock()
		return
	}
	previousCancel := s.modsWatchCancel
	previousDone := s.modsWatchDone
	s.modsWatchCancel = nil
	s.modsWatchDone = nil
	s.modsWatchDir = ""
	s.modsWatchMu.Unlock()
	// Stop the previous watcher before replacing it, otherwise two watchers can
	// briefly refresh the same directory concurrently.
	if previousCancel != nil {
		previousCancel()
	}
	if previousDone != nil {
		<-previousDone
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	s.modsWatchMu.Lock()
	s.modsWatchCancel = cancel
	s.modsWatchDir = dir
	s.modsWatchDone = done
	s.modsWatchMu.Unlock()
	go func() {
		defer close(done)
		s.watchModsDir(ctx, dir, version.ID, version.Name)
	}()
}

func (s *Service) watchModsDir(ctx context.Context, dir, versionID, versionName string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logging.Warn("create mods watcher failed", "dir", dir, "error", err)
		return
	}
	defer watcher.Close()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		logging.Warn("create mods directory for watcher failed", "dir", dir, "error", err)
		return
	}
	if err := watcher.Add(dir); err != nil {
		logging.Warn("watch mods directory failed", "dir", dir, "error", err)
		return
	}
	var timer *time.Timer
	pending := make(map[string]bool)
	var pendingMu sync.Mutex
	trigger := func(path string) {
		pendingMu.Lock()
		pending[path] = true
		pendingMu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(250*time.Millisecond, func() {
			selected := global.GetSelectedVersion()
			if selected.ID != versionID && selected.Name != versionName {
				return
			}
			s.modsWatchMu.Lock()
			activeDir := s.modsWatchDir
			s.modsWatchMu.Unlock()
			if ctx.Err() != nil || activeDir != dir {
				return
			}
			pendingMu.Lock()
			paths := make([]string, 0, len(pending))
			for path := range pending {
				paths = append(paths, path)
			}
			pending = make(map[string]bool)
			pendingMu.Unlock()
			s.startServiceBackground(func() {
				s.refreshChangedPaths(paths)
			})
		})
	}
	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logging.Warn("mods watcher error", "dir", dir, "error", err)
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 {
				trigger(event.Name)
			}
		}
	}
}

func ScanAllModDirsForHardlinkIndex(mcDir string, versions []structs.VersionInfo, generation uint64) {
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

func (s *Service) stop(saveConfig bool) {
	s.stopServiceBackground()
	s.stopModsWatcher()
	// Drain download workers before providers, storage and logging go away:
	// running jobs still touch all three while finishing.
	downloader.Shutdown()
	if saveConfig {
		if s.config == nil {
			s.config = &configs.Config{}
		}
		s.config.Prefers.MinecraftDir = global.GetMinecraftDir()
		if err := configs.Save(s.config); err != nil {
			logging.Error("save config failed", "error", err)
		}
	}
	providers.StopBackgroundTasks()
	storage.Close()
	logging.Close()
}

func (s *Service) startServiceBackground(task func()) {
	s.backgroundMu.Lock()
	if s.backgroundDone {
		s.backgroundMu.Unlock()
		return
	}
	s.backgroundWG.Add(1)
	s.backgroundMu.Unlock()
	go func() {
		defer s.backgroundWG.Done()
		task()
	}()
}

func (s *Service) stopServiceBackground() {
	s.backgroundMu.Lock()
	s.backgroundDone = true
	s.backgroundMu.Unlock()
	s.backgroundWG.Wait()
}
