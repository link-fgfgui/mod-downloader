// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package configs

import (
	"fmt"
	"math"
	"strings"
)

// APIKeys holds the platform credentials read from config or environment.
// Values are never exposed verbatim through adapter RPCs.
type APIKeys struct {
	CurseForgeAPIKey string `toml:"curseforge_api_key" json:"curseforge_api_key" env:"CF_API_KEY"`
	ModrinthAPIKey   string `toml:"modrinth_api_key" json:"modrinth_api_key" env:"MODRINTH_API_KEY"`
}

// Theme is a validated UI color theme.
type Theme string

// Supported theme values. ThemeSystem follows the operating system.
const (
	ThemeDark   Theme = "dark"
	ThemeLight  Theme = "light"
	ThemeSystem Theme = "system"
)

// UnmarshalText decodes a theme name, rejecting unknown values.
func (t *Theme) UnmarshalText(text []byte) error {
	theme := ParseTheme(string(text))
	if theme == "" {
		return fmt.Errorf("invalid theme %q: expected dark, light, or system", strings.TrimSpace(string(text)))
	}
	*t = theme
	return nil
}

// MarshalText encodes the normalized theme name.
func (t Theme) MarshalText() ([]byte, error) {
	return []byte(t.Normalized().String()), nil
}

// String returns the theme's wire value.
func (t Theme) String() string {
	return string(t)
}

// Normalized returns the parsed theme, falling back to ThemeDark for an
// empty or unknown value.
func (t Theme) Normalized() Theme {
	if normalized := ParseTheme(string(t)); normalized != "" {
		return normalized
	}
	return ThemeDark
}

// ParseTheme converts a stored value or numeric legacy index into a Theme.
// It returns "" when the value is not a recognized theme.
func ParseTheme(value string) Theme {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "0", string(ThemeDark):
		return ThemeDark
	case "1", string(ThemeLight):
		return ThemeLight
	case "2", string(ThemeSystem):
		return ThemeSystem
	default:
		return ""
	}
}

// Language is a validated UI language.
type Language string

// Supported language values. LanguageSystem follows the operating system.
const (
	LanguageSystem  Language = "system"
	LanguageChinese Language = "zh"
	LanguageEnglish Language = "en"
)

// UnmarshalText decodes a language name, rejecting unknown values.
func (l *Language) UnmarshalText(text []byte) error {
	language := ParseLanguage(string(text))
	if language == "" {
		return fmt.Errorf("invalid language %q: expected system, zh, or en", strings.TrimSpace(string(text)))
	}
	*l = language
	return nil
}

// MarshalText encodes the normalized language name.
func (l Language) MarshalText() ([]byte, error) {
	return []byte(l.Normalized().String()), nil
}

// String returns the language's wire value.
func (l Language) String() string {
	return string(l)
}

// Normalized returns the parsed language, falling back to LanguageSystem for
// an empty or unknown value.
func (l Language) Normalized() Language {
	if normalized := ParseLanguage(string(l)); normalized != "" {
		return normalized
	}
	return LanguageSystem
}

// ParseLanguage converts a stored value or English/Chinese alias into a
// Language. It returns "" when the value is not recognized.
func ParseLanguage(value string) Language {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(LanguageSystem):
		return LanguageSystem
	case "zh-cn", "zh-hans", "chinese", string(LanguageChinese):
		return LanguageChinese
	case "english", string(LanguageEnglish):
		return LanguageEnglish
	default:
		return ""
	}
}

// Preferences is the persisted user preference set. Pointer bool fields keep
// "unset" distinct from an explicit false.
type Preferences struct {
	Theme                       Theme    `toml:"theme" json:"theme" env:"THEME" env-default:"dark"`
	Language                    Language `toml:"language" json:"language" env:"LANGUAGE" env-default:"system"`
	MinecraftDir                string   `toml:"minecraft_dir" json:"minecraft_dir" env:"MINECRAFT_DIR"`
	SimpleMode                  bool     `toml:"simple_mode" json:"simple_mode" env:"SIMPLE_MODE"`
	AnimationDurationMultiplier float64  `toml:"animation_duration_multiplier" json:"animation_duration_multiplier" env:"ANIMATION_DURATION_MULTIPLIER"`
	AutoScanUnusedDependencies  *bool    `toml:"auto_scan_unused_dependencies" json:"auto_scan_unused_dependencies"`
	MCIMEnabled                 bool     `toml:"mcim_enabled" json:"mcim_enabled" env:"MCIM_ENABLED"`
	PreferredReleaseType        string   `toml:"preferred_release_type" json:"preferred_release_type" env:"PREFERRED_RELEASE_TYPE"`
	EnableJIJDetection          *bool    `toml:"enable_jij_detection" json:"enable_jij_detection"`
	HardwareAcceleration        *bool    `toml:"hardware_acceleration" json:"hardware_acceleration"`
	// RequireCtrlForMultiSelect makes a plain row click select a single item;
	// adding to the selection then requires Ctrl (or Cmd). Defaults to false.
	RequireCtrlForMultiSelect bool `toml:"require_ctrl_for_multi_select" json:"require_ctrl_for_multi_select" env:"REQUIRE_CTRL_FOR_MULTI_SELECT"`
}

// Preference defaults and bounds. Boundary tests derive from these values.
const (
	DefaultAnimationDurationMultiplier = 1.0
	MinAnimationDurationMultiplier     = 0.1
	MaxAnimationDurationMultiplier     = 3.0
	DefaultAutoScanUnusedDependencies  = false
	DefaultPreferredReleaseType        = "latest"
	DefaultEnableJIJDetection          = true
	DefaultHardwareAcceleration        = true
)

// BoolPtr returns a pointer to value, used for tri-state config fields.
func BoolPtr(value bool) *bool {
	return &value
}

// NormalizedAnimationDurationMultiplier returns the clamped animation speed.
func (p Preferences) NormalizedAnimationDurationMultiplier() float64 {
	return NormalizeAnimationDurationMultiplier(p.AnimationDurationMultiplier)
}

// AutoScanUnusedDependenciesValue resolves the tri-state auto-scan preference.
func (p Preferences) AutoScanUnusedDependenciesValue() bool {
	if p.AutoScanUnusedDependencies == nil {
		return DefaultAutoScanUnusedDependencies
	}
	return *p.AutoScanUnusedDependencies
}

// MCIMEnabledValue reports whether the MCIM CurseForge mirror is enabled.
func (p Preferences) MCIMEnabledValue() bool {
	return p.MCIMEnabled
}

// PreferredReleaseTypeValue returns the validated release channel, falling
// back to the default for unknown values.
func (p Preferences) PreferredReleaseTypeValue() string {
	switch strings.ToLower(strings.TrimSpace(p.PreferredReleaseType)) {
	case "latest", "alpha", "beta", "release":
		return strings.ToLower(strings.TrimSpace(p.PreferredReleaseType))
	default:
		return DefaultPreferredReleaseType
	}
}

// EnableJIJDetectionValue resolves the tri-state Jar-in-Jar preference.
func (p Preferences) EnableJIJDetectionValue() bool {
	if p.EnableJIJDetection == nil {
		return DefaultEnableJIJDetection
	}
	return *p.EnableJIJDetection
}

// HardwareAccelerationValue reports whether the webview should force GPU
// rendering. It defaults to true and is read before the Wails window exists,
// so changing it only takes effect on the next launch.
func (p Preferences) HardwareAccelerationValue() bool {
	if p.HardwareAcceleration == nil {
		return DefaultHardwareAcceleration
	}
	return *p.HardwareAcceleration
}

// NormalizeAnimationDurationMultiplier clamps a multiplier to the supported
// range and falls back to the default for non-finite or non-positive values.
func NormalizeAnimationDurationMultiplier(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return DefaultAnimationDurationMultiplier
	}
	if value < MinAnimationDurationMultiplier {
		return MinAnimationDurationMultiplier
	}
	if value > MaxAnimationDurationMultiplier {
		return MaxAnimationDurationMultiplier
	}
	return value
}

// RuntimeConfig holds the runtime paths and selected instance scope.
type RuntimeConfig struct {
	MinecraftVersion string `toml:"minecraft_version" json:"minecraftVersion" env:"MINECRAFT_VERSION"`
	ModLoader        string `toml:"mod_loader" json:"modLoader" env:"MOD_LOADER"`
	CacheDir         string `toml:"cache_dir" json:"cacheDir" env:"MOD_DOWNLOADER_CACHE_DIR"`
}

// Download and API defaults plus the inclusive bounds accepted by the settings
// UI. The core clamps every persisted value to these limits.
const (
	DefaultFileConcurrency       = 4
	DefaultConcurrentDownloads   = 16
	MinFileConcurrency           = 1
	MaxFileConcurrency           = 32
	MinConcurrentDownloads       = 1
	MaxConcurrentDownloads       = 256
	MinRequestsPerSecond         = 0
	MaxRequestsPerSecond         = 100
	DefaultTargetDownloadRateMiB = 1.0
	MinTargetDownloadRateMiB     = 0.1
	MaxTargetDownloadRateMiB     = 5.0
)

// DownloadConfig holds the persisted download queue and file-transfer limits.
type DownloadConfig struct {
	FileConcurrency         int     `toml:"file_concurrency" json:"fileConcurrency" env:"FILE_CONCURRENCY"`
	ConcurrentDownloads     int     `toml:"concurrent_downloads" json:"concurrentDownloads" env:"CONCURRENT_DOWNLOADS"`
	AdaptiveFileConcurrency bool    `toml:"adaptive_file_concurrency" json:"adaptiveFileConcurrency" env:"ADAPTIVE_FILE_CONCURRENCY"`
	TargetDownloadRateMiB   float64 `toml:"target_download_rate_mib" json:"targetDownloadRateMiB" env:"TARGET_DOWNLOAD_RATE_MIB"`
	VerifySHA1              bool    `toml:"verify_sha1" json:"verifySha1" env:"VERIFY_SHA1"`
}

// FileConcurrencyValue returns the clamped per-file chunk concurrency.
func (c DownloadConfig) FileConcurrencyValue() int {
	if c.FileConcurrency < MinFileConcurrency {
		return DefaultFileConcurrency
	}
	if c.FileConcurrency > MaxFileConcurrency {
		return MaxFileConcurrency
	}
	return c.FileConcurrency
}

// ConcurrentDownloadsValue returns the clamped number of simultaneous mod
// downloads.
func (c DownloadConfig) ConcurrentDownloadsValue() int {
	if c.ConcurrentDownloads < MinConcurrentDownloads {
		return DefaultConcurrentDownloads
	}
	if c.ConcurrentDownloads > MaxConcurrentDownloads {
		return MaxConcurrentDownloads
	}
	return c.ConcurrentDownloads
}

// TargetDownloadRateMiBValue returns the clamped per-download bandwidth target.
func (c DownloadConfig) TargetDownloadRateMiBValue() float64 {
	if math.IsNaN(c.TargetDownloadRateMiB) || math.IsInf(c.TargetDownloadRateMiB, 0) || c.TargetDownloadRateMiB <= 0 {
		return DefaultTargetDownloadRateMiB
	}
	if c.TargetDownloadRateMiB < MinTargetDownloadRateMiB {
		return MinTargetDownloadRateMiB
	}
	if c.TargetDownloadRateMiB > MaxTargetDownloadRateMiB {
		return MaxTargetDownloadRateMiB
	}
	return c.TargetDownloadRateMiB
}

// APIConfig holds the outbound provider request rate limit.
type APIConfig struct {
	RequestsPerSecond int `toml:"requests_per_second" json:"requestsPerSecond" env:"REQUESTS_PER_SECOND"`
}

// LoggingConfig controls file logging and rotation.
type LoggingConfig struct {
	Enabled   *bool `toml:"enabled,omitempty" json:"enabled,omitempty" env:"-"`
	Disabled  *bool `toml:"disabled,omitempty" json:"-" env:"-"`
	ForceFile bool  `toml:"force_file" json:"forceFile" env:"FORCE_FILE"`
	// File overrides the log file path. An empty value uses the per-user cache
	// directory (see logging.DefaultPath).
	File string `toml:"file,omitempty" json:"file,omitempty" env:"FILE"`
	// MaxSizeMB is the rotation threshold in MiB. Zero uses the default (6 MiB).
	MaxSizeMB int `toml:"max_size_mb,omitempty" json:"maxSizeMb,omitempty" env:"MAX_SIZE_MB"`
	// MaxBackups is the number of historical files to keep. Zero uses the
	// default (2); a negative value disables backups.
	MaxBackups int `toml:"max_backups,omitempty" json:"maxBackups,omitempty" env:"MAX_BACKUPS"`
}

// EnabledValue resolves the enabled/disabled aliases, defaulting to enabled.
func (c LoggingConfig) EnabledValue() bool {
	if c.Enabled != nil {
		return *c.Enabled
	}
	if c.Disabled != nil {
		return !*c.Disabled
	}
	return true
}

// Canonical collapses the enabled/disabled aliases into a single Enabled field
// and trims the file path before the config is written back.
func (c LoggingConfig) Canonical() LoggingConfig {
	return LoggingConfig{
		Enabled:    BoolPtr(c.EnabledValue()),
		ForceFile:  c.ForceFile,
		File:       strings.TrimSpace(c.File),
		MaxSizeMB:  c.MaxSizeMB,
		MaxBackups: c.MaxBackups,
	}
}

// RequestsPerSecondValue returns the clamped provider request rate.
func (c APIConfig) RequestsPerSecondValue() int {
	if c.RequestsPerSecond < MinRequestsPerSecond {
		return MinRequestsPerSecond
	}
	if c.RequestsPerSecond > MaxRequestsPerSecond {
		return MaxRequestsPerSecond
	}
	return c.RequestsPerSecond
}

// Config is the complete application configuration persisted to TOML or
// assembled from environment variables.
type Config struct {
	Keys        APIKeys        `toml:"keys" json:"keys" env-prefix:"KEYS_"`
	Preferences Preferences    `toml:"preferences" json:"preferences" env-prefix:"PREFERS_"`
	Runtime     RuntimeConfig  `toml:"runtime" json:"runtime"`
	Downloads   DownloadConfig `toml:"downloads" json:"downloads" env-prefix:"DOWNLOADS_"`
	API         APIConfig      `toml:"api" json:"api" env-prefix:"API_"`
	Logging     LoggingConfig  `toml:"logging" json:"logging" env-prefix:"LOGGING_"`
}
