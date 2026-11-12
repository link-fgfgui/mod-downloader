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

// cloneConfig returns an independent config snapshot. Pointer-valued
// preferences are cloned so callers cannot mutate the live config through a
// snapshot.
func cloneConfig(cfg *configs.Config) *configs.Config {
	if cfg == nil {
		return &configs.Config{}
	}
	clone := *cfg
	if cfg.Preferences.AutoScanUnusedDependencies != nil {
		clone.Preferences.AutoScanUnusedDependencies = configs.BoolPtr(*cfg.Preferences.AutoScanUnusedDependencies)
	}
	if cfg.Preferences.EnableJIJDetection != nil {
		clone.Preferences.EnableJIJDetection = configs.BoolPtr(*cfg.Preferences.EnableJIJDetection)
	}
	if cfg.Preferences.HardwareAcceleration != nil {
		clone.Preferences.HardwareAcceleration = configs.BoolPtr(*cfg.Preferences.HardwareAcceleration)
	}
	if cfg.Logging.Enabled != nil {
		clone.Logging.Enabled = configs.BoolPtr(*cfg.Logging.Enabled)
	}
	if cfg.Logging.Disabled != nil {
		clone.Logging.Disabled = configs.BoolPtr(*cfg.Logging.Disabled)
	}
	return &clone
}

// configSnapshot returns the current config as an immutable snapshot.
func (s *Service) configSnapshot() *configs.Config {
	return cloneConfig(s.configPtr.Load())
}

// storeConfig replaces the live config without persisting it.
func (s *Service) storeConfig(cfg *configs.Config) {
	s.configPtr.Store(cloneConfig(cfg))
}

// mutateConfig applies an in-memory-only config change. The callback must not
// call back into Service methods that touch configPtr.
func (s *Service) mutateConfig(mutate func(*configs.Config)) {
	s.configWriteMu.Lock()
	defer s.configWriteMu.Unlock()
	cfg := cloneConfig(s.configPtr.Load())
	mutate(cfg)
	s.configPtr.Store(cfg)
}

// updateConfig applies and optionally persists a config change. On a persist
// failure the live config stays unchanged and the error is returned to the
// caller, which may expose it through SettingsView.SaveError or directly. The
// callback must not call back into Service methods that touch configPtr.
func (s *Service) updateConfig(action string, save bool, mutate func(*configs.Config), attrs ...any) (*configs.Config, error) {
	s.configWriteMu.Lock()
	defer s.configWriteMu.Unlock()
	current := s.configPtr.Load()
	next := cloneConfig(current)
	mutate(next)
	if save {
		if err := configs.Save(next); err != nil {
			logging.Error(action, append(attrs, "error", err)...)
			s.saveErrMu.Lock()
			s.lastSaveError = err.Error()
			s.saveErrMu.Unlock()
			return cloneConfig(current), err
		}
		s.saveErrMu.Lock()
		s.lastSaveError = ""
		s.saveErrMu.Unlock()
	}
	s.configPtr.Store(next)
	return cloneConfig(next), nil
}

func (s *Service) recordSaveError(err error) {
	if err == nil {
		return
	}
	s.saveErrMu.Lock()
	s.lastSaveError = err.Error()
	s.saveErrMu.Unlock()
}

// GetPreferences returns the persisted theme, language and animation subset.
func (s *Service) GetPreferences() appstructs.AppPreferences {
	cfg := s.configSnapshot()
	return appstructs.AppPreferences{
		Theme:                       cfg.Preferences.Theme.Normalized().String(),
		Language:                    cfg.Preferences.Language.Normalized().String(),
		AnimationDurationMultiplier: cfg.Preferences.NormalizedAnimationDurationMultiplier(),
	}
}

// GetSettings returns the client-safe settings snapshot with clamped values.
func (s *Service) GetSettings() appview.SettingsView {
	cfg := s.configSnapshot()
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
	sv.Theme = cfg.Preferences.Theme.Normalized().String()
	sv.Language = cfg.Preferences.Language.Normalized().String()
	sv.SimpleMode = cfg.Preferences.SimpleMode
	sv.AnimationDurationMultiplier = cfg.Preferences.NormalizedAnimationDurationMultiplier()
	sv.AutoScanUnusedDependencies = cfg.Preferences.AutoScanUnusedDependenciesValue()
	sv.MCIMEnabled = cfg.Preferences.MCIMEnabledValue()
	sv.EnableJIJDetection = cfg.Preferences.EnableJIJDetectionValue()
	sv.HardwareAcceleration = cfg.Preferences.HardwareAccelerationValue()
	sv.RequireCtrlForMultiSelect = cfg.Preferences.RequireCtrlForMultiSelect
	sv.PreferredReleaseType = cfg.Preferences.PreferredReleaseTypeValue()
	sv.FileConcurrency = cfg.Downloads.FileConcurrencyValue()
	sv.ConcurrentDownloads = cfg.Downloads.ConcurrentDownloadsValue()
	sv.AdaptiveFileConcurrency = cfg.Downloads.AdaptiveFileConcurrency
	sv.TargetDownloadRateMiB = cfg.Downloads.TargetDownloadRateMiBValue()
	sv.VerifySHA1 = cfg.Downloads.VerifySHA1
	sv.RequestsPerSecond = cfg.API.RequestsPerSecondValue()
	sv.MinecraftDir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(cfg.Preferences.MinecraftDir))
	sv.CacheDir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(cfg.Runtime.CacheDir))
	sv.HasModrinthKey = strings.TrimSpace(cfg.Keys.ModrinthAPIKey) != ""
	sv.ModrinthKeyMask = MaskKey(cfg.Keys.ModrinthAPIKey)
	effectiveCFKey := configs.EffectiveCurseForgeAPIKey(cfg.Keys.CurseForgeAPIKey)
	sv.HasCurseForgeKey = effectiveCFKey != ""
	sv.CurseForgeKeyMask = MaskKey(effectiveCFKey)
	if sv.MinecraftDir == "" {
		sv.MinecraftDir = minecraft.SimplifyPathWithEnv(global.GetMinecraftDir())
	}
	if path, err := s.cachePathForDir(cfg.Runtime.CacheDir); err == nil {
		sv.CachePath = minecraft.SimplifyPathWithEnv(path)
	}
	sv.SaveError = s.saveErrorSnapshot()
	return configs.NormalizeSettingsView(sv)
}

// saveErrorSnapshot returns the most recent config write failure, if any.
func (s *Service) saveErrorSnapshot() string {
	s.saveErrMu.Lock()
	defer s.saveErrMu.Unlock()
	return s.lastSaveError
}

// SaveTheme validates and persists the UI theme, returning its wire value. It
// collapses a persistence failure into the returned value; callers that must
// report the failure should use SaveThemeWithError.
func (s *Service) SaveTheme(theme string) string {
	value, _ := s.SaveThemeWithError(theme)
	return value
}

// SaveThemeWithError is SaveTheme with the persistence error preserved. On a
// failed write the live config is unchanged.
func (s *Service) SaveThemeWithError(theme string) (string, error) {
	parsed := configs.ParseTheme(theme)
	if parsed == "" {
		parsed = configs.ThemeDark
	}
	value := parsed.String()
	if _, err := s.updateConfig("save theme failed", true, func(cfg *configs.Config) {
		cfg.Preferences.Theme = parsed
	}, "theme", parsed); err != nil {
		return value, err
	}
	return value, nil
}

// SaveLanguage validates and persists the UI language, returning its wire
// value. It collapses a persistence failure into the returned value; callers
// that must report the failure should use SaveLanguageWithError.
func (s *Service) SaveLanguage(language string) string {
	value, _ := s.SaveLanguageWithError(language)
	return value
}

// SaveLanguageWithError is SaveLanguage with the persistence error preserved.
// On a failed write the live config is unchanged.
func (s *Service) SaveLanguageWithError(language string) (string, error) {
	parsed := configs.ParseLanguage(language)
	if parsed == "" {
		parsed = configs.LanguageSystem
	}
	value := parsed.String()
	if _, err := s.updateConfig("save language failed", true, func(cfg *configs.Config) {
		cfg.Preferences.Language = parsed
	}, "language", parsed); err != nil {
		return value, err
	}
	return value, nil
}

// SaveAnimationSettings normalizes and persists the UI animation multiplier.
func (s *Service) SaveAnimationSettings(req appstructs.SaveAnimationSettingsRequest) appview.SettingsView {
	_, _ = s.updateConfig("save animation settings failed", true, func(cfg *configs.Config) {
		cfg.Preferences.AnimationDurationMultiplier = configs.NormalizeAnimationDurationMultiplier(req.AnimationDurationMultiplier)
	}, "animationDurationMultiplier", req.AnimationDurationMultiplier)
	return s.GetSettings()
}

// SaveSimpleModeSettings persists simple mode and applies its runtime effects.
func (s *Service) SaveSimpleModeSettings(req appstructs.SaveSimpleModeSettingsRequest) appview.SettingsView {
	if _, err := s.updateConfig("save simple mode setting failed", true, func(cfg *configs.Config) {
		cfg.Preferences.SimpleMode = req.SimpleMode
	}, "simpleMode", req.SimpleMode); err == nil {
		s.applyRuntimeConfig(s.downloadEvents())
	}
	return s.GetSettings()
}

// SaveUnusedDependencyCleanupSettings persists the auto-scan preference.
func (s *Service) SaveUnusedDependencyCleanupSettings(req appstructs.SaveUnusedDependencyCleanupSettingsRequest) appview.SettingsView {
	_, _ = s.updateConfig("save unused dependency cleanup settings failed", true, func(cfg *configs.Config) {
		cfg.Preferences.AutoScanUnusedDependencies = configs.BoolPtr(req.AutoScanUnusedDependencies)
	}, "autoScanUnusedDependencies", req.AutoScanUnusedDependencies)
	return s.GetSettings()
}

// SaveMCIMSettings persists the MCIM mirror toggle and reapplies clients.
func (s *Service) SaveMCIMSettings(req appstructs.SaveMCIMSettingsRequest) appview.SettingsView {
	if _, err := s.updateConfig("save MCIM settings failed", true, func(cfg *configs.Config) {
		cfg.Preferences.MCIMEnabled = req.MCIMEnabled
	}, "mcimEnabled", req.MCIMEnabled); err == nil {
		s.applyRuntimeConfig(s.downloadEvents())
	}
	return s.GetSettings()
}

// SaveJIJDetectionSettings persists the Jar-in-Jar toggle and clears cached
// parse results.
func (s *Service) SaveJIJDetectionSettings(req appstructs.SaveJIJDetectionSettingsRequest) appview.SettingsView {
	if _, err := s.updateConfig("save JIJ detection setting failed", true, func(cfg *configs.Config) {
		cfg.Preferences.EnableJIJDetection = configs.BoolPtr(req.EnableJIJDetection)
	}, "enableJijDetection", req.EnableJIJDetection); err == nil {
		s.applyRuntimeConfig(s.downloadEvents())
	}
	return s.GetSettings()
}

// SaveMultiSelectSettings persists the Ctrl-held multi-select preference. It
// is a pure UI selection behavior read from the settings snapshot; no runtime
// reconfiguration is required.
func (s *Service) SaveMultiSelectSettings(req appstructs.SaveMultiSelectSettingsRequest) appview.SettingsView {
	_, _ = s.updateConfig("save multi-select setting failed", true, func(cfg *configs.Config) {
		cfg.Preferences.RequireCtrlForMultiSelect = req.RequireCtrlForMultiSelect
	}, "requireCtrlForMultiSelect", req.RequireCtrlForMultiSelect)
	return s.GetSettings()
}

// SaveHardwareAccelerationSettings persists the webview GPU preference. The
// value is read while the app starts on the next launch; it cannot be applied
// to the already-created webview at runtime.
func (s *Service) SaveHardwareAccelerationSettings(req appstructs.SaveHardwareAccelerationSettingsRequest) appview.SettingsView {
	_, _ = s.updateConfig("save hardware acceleration setting failed", true, func(cfg *configs.Config) {
		cfg.Preferences.HardwareAcceleration = configs.BoolPtr(req.HardwareAcceleration)
	}, "hardwareAcceleration", req.HardwareAcceleration)
	return s.GetSettings()
}

// SavePreferredReleaseTypeSettings normalizes and persists the release-type
// filter applied to project versions.
func (s *Service) SavePreferredReleaseTypeSettings(req appstructs.SavePreferredReleaseTypeSettingsRequest) appview.SettingsView {
	value := configs.Preferences{PreferredReleaseType: req.PreferredReleaseType}.PreferredReleaseTypeValue()
	if _, err := s.updateConfig("save preferred release type setting failed", true, func(cfg *configs.Config) {
		cfg.Preferences.PreferredReleaseType = value
	}, "preferredReleaseType", value); err == nil {
		s.applyRuntimeConfig(s.downloadEvents())
	}
	return s.GetSettings()
}

// SaveNetworkSettings clamps and persists download concurrency, rate limits and
// API request pacing, then applies them to the running clients.
func (s *Service) SaveNetworkSettings(req appstructs.SaveNetworkSettingsRequest) appview.SettingsView {
	req = configs.NormalizeNetworkSettings(req)
	if _, err := s.updateConfig("save network settings failed", true, func(cfg *configs.Config) {
		cfg.Downloads.FileConcurrency = req.FileConcurrency
		cfg.Downloads.ConcurrentDownloads = req.ConcurrentDownloads
		cfg.Downloads.AdaptiveFileConcurrency = req.AdaptiveFileConcurrency
		cfg.Downloads.TargetDownloadRateMiB = req.TargetDownloadRateMiB
		cfg.Downloads.VerifySHA1 = req.VerifySHA1
		cfg.API.RequestsPerSecond = req.RequestsPerSecond
	}); err == nil {
		s.applyRuntimeConfig(s.downloadEvents())
	}
	return s.GetSettings()
}

// SaveMinecraftDirPreference persists the Minecraft directory override and
// updates the process-wide value.
func (s *Service) SaveMinecraftDirPreference(dir string) string {
	dir = strings.TrimSpace(dir)
	if _, err := s.updateConfig("save minecraft dir preference failed", true, func(cfg *configs.Config) {
		cfg.Preferences.MinecraftDir = dir
	}, "minecraftDir", dir); err != nil {
		return s.configSnapshot().Preferences.MinecraftDir
	}
	effective := s.configSnapshot().Preferences.MinecraftDir
	global.SetMinecraftDir(minecraft.ExpandPathWithEnv(effective))
	return effective
}

// SaveCacheDirPreference opens the database at the new location before
// persisting the override. A failed open leaves the previous store and config
// untouched; the error is exposed through SettingsView.SaveError.
func (s *Service) SaveCacheDirPreference(dir string) appview.SettingsView {
	dir = strings.TrimSpace(dir)
	if dir != "" {
		dir = minecraft.SimplifyPathWithEnv(minecraft.ExpandPathWithEnv(dir))
	}
	oldDir := s.configSnapshot().Runtime.CacheDir
	cachePath, err := s.cachePathForDir(dir)
	if err != nil {
		logging.Error("resolve database cache path after cache dir save failed", "cacheDir", dir, "error", err)
		s.recordSaveError(err)
		return s.GetSettings()
	}
	if err := storage.OpenAt(cachePath); err != nil {
		logging.Error("reopen database after cache dir save failed", "cachePath", cachePath, "error", err)
		s.recordSaveError(err)
		return s.GetSettings()
	}
	if _, err := s.updateConfig("save cache dir preference failed", true, func(cfg *configs.Config) {
		cfg.Runtime.CacheDir = dir
	}, "cacheDir", dir); err != nil {
		// Config persistence failed after the store switched. Roll the store
		// back so the in-memory config and the live database stay consistent.
		if oldCachePath, pathErr := s.cachePathForDir(oldDir); pathErr == nil {
			if rollbackErr := storage.OpenAt(oldCachePath); rollbackErr != nil {
				logging.Error("rollback database after cache config save failed", "cachePath", oldCachePath, "error", rollbackErr)
			}
		}
	}
	return s.GetSettings()
}

// SaveAPIKeys persists the provider API keys unless the keep-sentinel value
// signals that the stored key must be preserved.
func (s *Service) SaveAPIKeys(req appstructs.SaveAPIKeysRequest) appview.SettingsView {
	if _, err := s.updateConfig("save api keys failed", true, func(cfg *configs.Config) {
		if req.CurseForgeAPIKey != APIKeyKeepSentinel {
			cfg.Keys.CurseForgeAPIKey = strings.TrimSpace(req.CurseForgeAPIKey)
		}
		if req.ModrinthAPIKey != APIKeyKeepSentinel {
			cfg.Keys.ModrinthAPIKey = strings.TrimSpace(req.ModrinthAPIKey)
		}
	}); err == nil {
		s.applyRuntimeConfig(s.downloadEvents())
	}
	return s.GetSettings()
}
