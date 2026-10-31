package appcore

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

func (s *Service) GetPreferences() appstructs.AppPreferences {
	if s.config == nil {
		return appstructs.AppPreferences{
			Theme:                       configs.ThemeDark.String(),
			Language:                    configs.LanguageSystem.String(),
			AnimationDurationMultiplier: configs.DefaultAnimationDurationMultiplier,
		}
	}
	return appstructs.AppPreferences{
		Theme:                       s.config.Prefers.Theme.Normalized().String(),
		Language:                    s.config.Prefers.Language.Normalized().String(),
		AnimationDurationMultiplier: s.config.Prefers.NormalizedAnimationDurationMultiplier(),
	}
}

func (s *Service) GetSettings() appstructs.SettingsView {
	sv := appstructs.SettingsView{
		Theme:                       configs.ThemeDark.String(),
		Language:                    configs.LanguageSystem.String(),
		SimpleMode:                  false,
		AnimationDurationMultiplier: configs.DefaultAnimationDurationMultiplier,
		AutoScanUnusedDependencies:  configs.DefaultAutoScanUnusedDependencies,
		EnableJijDetection:          configs.DefaultEnableJijDetection,
		HardwareAcceleration:        configs.DefaultHardwareAcceleration,
		PreferredReleaseType:        configs.DefaultPreferredReleaseType,
		FileConcurrency:             configs.DefaultFileConcurrency,
		ConcurrentDownloads:         configs.DefaultConcurrentDownloads,
		TargetDownloadRateMiB:       configs.DefaultTargetDownloadRateMiB,
	}
	if s.config != nil {
		sv.Theme = s.config.Prefers.Theme.Normalized().String()
		sv.Language = s.config.Prefers.Language.Normalized().String()
		sv.SimpleMode = s.config.Prefers.SimpleMode
		sv.AnimationDurationMultiplier = s.config.Prefers.NormalizedAnimationDurationMultiplier()
		sv.AutoScanUnusedDependencies = s.config.Prefers.AutoScanUnusedDependenciesValue()
		sv.MCIMEnabled = s.config.Prefers.MCIMEnabledValue()
		sv.EnableJijDetection = s.config.Prefers.EnableJijDetectionValue()
		sv.HardwareAcceleration = s.config.Prefers.HardwareAccelerationValue()
		sv.RequireCtrlForMultiSelect = s.config.Prefers.RequireCtrlForMultiSelect
		sv.PreferredReleaseType = s.config.Prefers.PreferredReleaseTypeValue()
		sv.FileConcurrency = s.config.Downloads.FileConcurrencyValue()
		sv.ConcurrentDownloads = s.config.Downloads.ConcurrentDownloadsValue()
		sv.AdaptiveFileConcurrency = s.config.Downloads.AdaptiveFileConcurrency
		sv.TargetDownloadRateMiB = s.config.Downloads.TargetDownloadRateMiBValue()
		sv.VerifySHA1 = s.config.Downloads.VerifySHA1
		sv.RequestsPerSecond = s.config.API.RequestsPerSecondValue()
		sv.MinecraftDir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(s.config.Prefers.MinecraftDir))
		sv.CacheDir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(s.config.Runtime.CacheDir))
		sv.HasModrinthKey = strings.TrimSpace(s.config.Keys.ModrinthApiKey) != ""
		sv.ModrinthKeyMask = MaskKey(s.config.Keys.ModrinthApiKey)
	}
	effectiveCFKey := s.effectiveCurseforgeAPIKey()
	sv.HasCurseforgeKey = effectiveCFKey != ""
	sv.CurseforgeKeyMask = MaskKey(effectiveCFKey)
	if sv.MinecraftDir == "" {
		sv.MinecraftDir = minecraft.SimplifyPathWithEnv(global.GetMinecraftDir())
	}
	if path, err := s.cachePath(); err == nil {
		sv.CachePath = minecraft.SimplifyPathWithEnv(path)
	}
	sv.SaveError = s.saveErrorSnapshot()
	return sv
}

// persistConfig records config write failures so the UI can stop reporting a
// successful save when the file on disk was never updated.
func (s *Service) persistConfig(action string, attrs ...any) {
	if err := configs.Save(s.config); err != nil {
		logging.Error(action, append(attrs, "error", err)...)
		s.saveErrMu.Lock()
		s.lastSaveError = err.Error()
		s.saveErrMu.Unlock()
		return
	}
	s.saveErrMu.Lock()
	s.lastSaveError = ""
	s.saveErrMu.Unlock()
}

func (s *Service) saveErrorSnapshot() string {
	s.saveErrMu.Lock()
	defer s.saveErrMu.Unlock()
	return s.lastSaveError
}

func (s *Service) SaveTheme(theme string) string {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	parsed := configs.ParseTheme(theme)
	if parsed == "" {
		parsed = configs.ThemeDark
	}
	s.config.Prefers.Theme = parsed
	s.persistConfig("save theme failed", "theme", parsed)
	return parsed.String()
}

func (s *Service) SaveLanguage(language string) string {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	parsed := configs.ParseLanguage(language)
	if parsed == "" {
		parsed = configs.LanguageSystem
	}
	s.config.Prefers.Language = parsed
	s.persistConfig("save language failed", "language", parsed)
	return parsed.String()
}

func (s *Service) SaveAnimationSettings(req appstructs.SaveAnimationSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Prefers.AnimationDurationMultiplier = configs.NormalizeAnimationDurationMultiplier(req.AnimationDurationMultiplier)
	s.persistConfig("save animation settings failed", "animationDurationMultiplier", req.AnimationDurationMultiplier)
	return s.GetSettings()
}

func (s *Service) SaveSimpleModeSettings(req appstructs.SaveSimpleModeSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Prefers.SimpleMode = req.SimpleMode
	s.persistConfig("save simple mode setting failed", "simpleMode", req.SimpleMode)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

func (s *Service) SaveUnusedDependencyCleanupSettings(req appstructs.SaveUnusedDependencyCleanupSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Prefers.AutoScanUnusedDependencies = configs.BoolPtr(req.AutoScanUnusedDependencies)
	s.persistConfig("save unused dependency cleanup settings failed", "autoScanUnusedDependencies", req.AutoScanUnusedDependencies)
	return s.GetSettings()
}

func (s *Service) SaveMCIMSettings(req appstructs.SaveMCIMSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Prefers.MCIMEnabled = req.MCIMEnabled
	s.persistConfig("save MCIM settings failed", "mcimEnabled", req.MCIMEnabled)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

func (s *Service) SaveJijDetectionSettings(req appstructs.SaveJijDetectionSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Prefers.EnableJijDetection = configs.BoolPtr(req.EnableJijDetection)
	s.persistConfig("save JIJ detection setting failed", "enableJijDetection", req.EnableJijDetection)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

// SaveHardwareAccelerationSettings persists the webview GPU preference. The
// value is consumed before wails.Run on the next launch; it cannot be applied
// to the already-created webview at runtime.
func (s *Service) SaveHardwareAccelerationSettings(req appstructs.SaveHardwareAccelerationSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Prefers.HardwareAcceleration = configs.BoolPtr(req.HardwareAcceleration)
	s.persistConfig("save hardware acceleration setting failed", "hardwareAcceleration", req.HardwareAcceleration)
	return s.GetSettings()
}

func (s *Service) SavePreferredReleaseTypeSettings(req appstructs.SavePreferredReleaseTypeSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	value := configs.Preferences{PreferredReleaseType: req.PreferredReleaseType}.PreferredReleaseTypeValue()
	s.config.Prefers.PreferredReleaseType = value
	s.persistConfig("save preferred release type setting failed", "preferredReleaseType", value)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

func (s *Service) SaveNetworkSettings(req appstructs.SaveNetworkSettingsRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Downloads.FileConcurrency = configs.DownloadConfig{FileConcurrency: req.FileConcurrency}.FileConcurrencyValue()
	s.config.Downloads.ConcurrentDownloads = configs.DownloadConfig{ConcurrentDownloads: req.ConcurrentDownloads}.ConcurrentDownloadsValue()
	s.config.Downloads.AdaptiveFileConcurrency = req.AdaptiveFileConcurrency
	s.config.Downloads.TargetDownloadRateMiB = configs.DownloadConfig{TargetDownloadRateMiB: req.TargetDownloadRateMiB}.TargetDownloadRateMiBValue()
	s.config.Downloads.VerifySHA1 = req.VerifySHA1
	s.config.API.RequestsPerSecond = configs.APIConfig{RequestsPerSecond: req.RequestsPerSecond}.RequestsPerSecondValue()
	s.persistConfig("save network settings failed")
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

func (s *Service) SaveMinecraftDirPreference(dir string) string {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Prefers.MinecraftDir = strings.TrimSpace(dir)
	s.persistConfig("save minecraft dir preference failed", "minecraftDir", dir)
	global.SetMinecraftDir(minecraft.ExpandPathWithEnv(s.config.Prefers.MinecraftDir))
	return s.config.Prefers.MinecraftDir
}

func (s *Service) SaveCacheDirPreference(dir string) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	dir = strings.TrimSpace(dir)
	if dir != "" {
		dir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(dir))
	}
	s.config.Runtime.CacheDir = dir
	s.persistConfig("save cache dir preference failed", "cacheDir", dir)
	if cachePath, err := s.cachePath(); err != nil {
		logging.Error("resolve database cache path after cache dir save failed", "cacheDir", dir, "error", err)
	} else if err := storage.OpenAt(cachePath); err != nil {
		logging.Error("reopen database after cache dir save failed", "cachePath", cachePath, "error", err)
	}
	return s.GetSettings()
}

func (s *Service) SaveApiKeys(req appstructs.SaveApiKeysRequest) appstructs.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	if req.CurseforgeApiKey != APIKeyKeepSentinel {
		s.config.Keys.CurseforgeApiKey = strings.TrimSpace(req.CurseforgeApiKey)
	}
	if req.ModrinthApiKey != APIKeyKeepSentinel {
		s.config.Keys.ModrinthApiKey = strings.TrimSpace(req.ModrinthApiKey)
	}
	s.persistConfig("save api keys failed")
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}
