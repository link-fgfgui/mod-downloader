// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/configs"
	"github.com/link-fgfgui/mod-downloader/core/global"
	"github.com/link-fgfgui/mod-downloader/core/logging"
	"github.com/link-fgfgui/mod-downloader/core/minecraft"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// GetPreferences returns the persisted theme, language and animation subset.
func (s *Service) GetPreferences() appstructs.AppPreferences {
	if s.config == nil {
		return appstructs.AppPreferences{
			Theme:                       configs.ThemeDark.String(),
			Language:                    configs.LanguageSystem.String(),
			AnimationDurationMultiplier: configs.DefaultAnimationDurationMultiplier,
		}
	}
	return appstructs.AppPreferences{
		Theme:                       s.config.Preferences.Theme.Normalized().String(),
		Language:                    s.config.Preferences.Language.Normalized().String(),
		AnimationDurationMultiplier: s.config.Preferences.NormalizedAnimationDurationMultiplier(),
	}
}

// GetSettings returns the client-safe settings snapshot with clamped values.
func (s *Service) GetSettings() appview.SettingsView {
	sv := appview.SettingsView{
		Theme:                       configs.ThemeDark.String(),
		Language:                    configs.LanguageSystem.String(),
		SimpleMode:                  false,
		AnimationDurationMultiplier: configs.DefaultAnimationDurationMultiplier,
		AutoScanUnusedDependencies:  configs.DefaultAutoScanUnusedDependencies,
		EnableJIJDetection:          configs.DefaultEnableJIJDetection,
		HardwareAcceleration:        configs.DefaultHardwareAcceleration,
		PreferredReleaseType:        configs.DefaultPreferredReleaseType,
		FileConcurrency:             configs.DefaultFileConcurrency,
		ConcurrentDownloads:         configs.DefaultConcurrentDownloads,
		TargetDownloadRateMiB:       configs.DefaultTargetDownloadRateMiB,
	}
	if s.config != nil {
		sv.Theme = s.config.Preferences.Theme.Normalized().String()
		sv.Language = s.config.Preferences.Language.Normalized().String()
		sv.SimpleMode = s.config.Preferences.SimpleMode
		sv.AnimationDurationMultiplier = s.config.Preferences.NormalizedAnimationDurationMultiplier()
		sv.AutoScanUnusedDependencies = s.config.Preferences.AutoScanUnusedDependenciesValue()
		sv.MCIMEnabled = s.config.Preferences.MCIMEnabledValue()
		sv.EnableJIJDetection = s.config.Preferences.EnableJIJDetectionValue()
		sv.HardwareAcceleration = s.config.Preferences.HardwareAccelerationValue()
		sv.RequireCtrlForMultiSelect = s.config.Preferences.RequireCtrlForMultiSelect
		sv.PreferredReleaseType = s.config.Preferences.PreferredReleaseTypeValue()
		sv.FileConcurrency = s.config.Downloads.FileConcurrencyValue()
		sv.ConcurrentDownloads = s.config.Downloads.ConcurrentDownloadsValue()
		sv.AdaptiveFileConcurrency = s.config.Downloads.AdaptiveFileConcurrency
		sv.TargetDownloadRateMiB = s.config.Downloads.TargetDownloadRateMiBValue()
		sv.VerifySHA1 = s.config.Downloads.VerifySHA1
		sv.RequestsPerSecond = s.config.API.RequestsPerSecondValue()
		sv.MinecraftDir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(s.config.Preferences.MinecraftDir))
		sv.CacheDir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(s.config.Runtime.CacheDir))
		sv.HasModrinthKey = strings.TrimSpace(s.config.Keys.ModrinthAPIKey) != ""
		sv.ModrinthKeyMask = MaskKey(s.config.Keys.ModrinthAPIKey)
	}
	effectiveCFKey := s.effectiveCurseForgeAPIKey()
	sv.HasCurseForgeKey = effectiveCFKey != ""
	sv.CurseForgeKeyMask = MaskKey(effectiveCFKey)
	if sv.MinecraftDir == "" {
		sv.MinecraftDir = minecraft.SimplifyPathWithEnv(global.GetMinecraftDir())
	}
	if path, err := s.cachePath(); err == nil {
		sv.CachePath = minecraft.SimplifyPathWithEnv(path)
	}
	sv.SaveError = s.saveErrorSnapshot()
	return configs.NormalizeSettingsView(sv)
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

// saveErrorSnapshot returns the most recent config write failure, if any.
func (s *Service) saveErrorSnapshot() string {
	s.saveErrMu.Lock()
	defer s.saveErrMu.Unlock()
	return s.lastSaveError
}

// SaveTheme validates and persists the UI theme, returning its wire value.
func (s *Service) SaveTheme(theme string) string {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	parsed := configs.ParseTheme(theme)
	if parsed == "" {
		parsed = configs.ThemeDark
	}
	s.config.Preferences.Theme = parsed
	s.persistConfig("save theme failed", "theme", parsed)
	return parsed.String()
}

// SaveLanguage validates and persists the UI language, returning its wire
// value.
func (s *Service) SaveLanguage(language string) string {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	parsed := configs.ParseLanguage(language)
	if parsed == "" {
		parsed = configs.LanguageSystem
	}
	s.config.Preferences.Language = parsed
	s.persistConfig("save language failed", "language", parsed)
	return parsed.String()
}

// SaveAnimationSettings normalizes and persists the UI animation multiplier.
func (s *Service) SaveAnimationSettings(req appstructs.SaveAnimationSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Preferences.AnimationDurationMultiplier = configs.NormalizeAnimationDurationMultiplier(req.AnimationDurationMultiplier)
	s.persistConfig("save animation settings failed", "animationDurationMultiplier", req.AnimationDurationMultiplier)
	return s.GetSettings()
}

// SaveSimpleModeSettings persists simple mode and applies its runtime effects.
func (s *Service) SaveSimpleModeSettings(req appstructs.SaveSimpleModeSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Preferences.SimpleMode = req.SimpleMode
	s.persistConfig("save simple mode setting failed", "simpleMode", req.SimpleMode)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

// SaveUnusedDependencyCleanupSettings persists the auto-scan preference.
func (s *Service) SaveUnusedDependencyCleanupSettings(req appstructs.SaveUnusedDependencyCleanupSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Preferences.AutoScanUnusedDependencies = configs.BoolPtr(req.AutoScanUnusedDependencies)
	s.persistConfig("save unused dependency cleanup settings failed", "autoScanUnusedDependencies", req.AutoScanUnusedDependencies)
	return s.GetSettings()
}

// SaveMCIMSettings persists the MCIM mirror toggle and reapplies clients.
func (s *Service) SaveMCIMSettings(req appstructs.SaveMCIMSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Preferences.MCIMEnabled = req.MCIMEnabled
	s.persistConfig("save MCIM settings failed", "mcimEnabled", req.MCIMEnabled)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

// SaveJIJDetectionSettings persists the Jar-in-Jar toggle and clears cached
// parse results.
func (s *Service) SaveJIJDetectionSettings(req appstructs.SaveJIJDetectionSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Preferences.EnableJIJDetection = configs.BoolPtr(req.EnableJIJDetection)
	s.persistConfig("save JIJ detection setting failed", "enableJijDetection", req.EnableJIJDetection)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

// SaveHardwareAccelerationSettings persists the webview GPU preference. The
// value is read while the app starts on the next launch; it cannot be applied
// to the already-created webview at runtime.
func (s *Service) SaveHardwareAccelerationSettings(req appstructs.SaveHardwareAccelerationSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Preferences.HardwareAcceleration = configs.BoolPtr(req.HardwareAcceleration)
	s.persistConfig("save hardware acceleration setting failed", "hardwareAcceleration", req.HardwareAcceleration)
	return s.GetSettings()
}

// SavePreferredReleaseTypeSettings normalizes and persists the release-type
// filter applied to project versions.
func (s *Service) SavePreferredReleaseTypeSettings(req appstructs.SavePreferredReleaseTypeSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	value := configs.Preferences{PreferredReleaseType: req.PreferredReleaseType}.PreferredReleaseTypeValue()
	s.config.Preferences.PreferredReleaseType = value
	s.persistConfig("save preferred release type setting failed", "preferredReleaseType", value)
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

// SaveNetworkSettings clamps and persists download concurrency, rate limits and
// API request pacing, then applies them to the running clients.
func (s *Service) SaveNetworkSettings(req appstructs.SaveNetworkSettingsRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	req = configs.NormalizeNetworkSettings(req)
	s.config.Downloads.FileConcurrency = req.FileConcurrency
	s.config.Downloads.ConcurrentDownloads = req.ConcurrentDownloads
	s.config.Downloads.AdaptiveFileConcurrency = req.AdaptiveFileConcurrency
	s.config.Downloads.TargetDownloadRateMiB = req.TargetDownloadRateMiB
	s.config.Downloads.VerifySHA1 = req.VerifySHA1
	s.config.API.RequestsPerSecond = req.RequestsPerSecond
	s.persistConfig("save network settings failed")
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}

// SaveMinecraftDirPreference persists the Minecraft directory override and
// updates the process-wide value.
func (s *Service) SaveMinecraftDirPreference(dir string) string {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	s.config.Preferences.MinecraftDir = strings.TrimSpace(dir)
	s.persistConfig("save minecraft dir preference failed", "minecraftDir", dir)
	global.SetMinecraftDir(minecraft.ExpandPathWithEnv(s.config.Preferences.MinecraftDir))
	return s.config.Preferences.MinecraftDir
}

// SaveCacheDirPreference persists the cache directory override and reopens the
// database at the new location.
func (s *Service) SaveCacheDirPreference(dir string) appview.SettingsView {
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

// SaveAPIKeys persists the provider API keys unless the keep-sentinel value
// signals that the stored key must be preserved.
func (s *Service) SaveAPIKeys(req appstructs.SaveAPIKeysRequest) appview.SettingsView {
	if s.config == nil {
		s.config = &configs.Config{}
	}
	if req.CurseForgeAPIKey != APIKeyKeepSentinel {
		s.config.Keys.CurseForgeAPIKey = strings.TrimSpace(req.CurseForgeAPIKey)
	}
	if req.ModrinthAPIKey != APIKeyKeepSentinel {
		s.config.Keys.ModrinthAPIKey = strings.TrimSpace(req.ModrinthAPIKey)
	}
	s.persistConfig("save api keys failed")
	s.applyRuntimeConfig(s.downloadEvents())
	return s.GetSettings()
}
