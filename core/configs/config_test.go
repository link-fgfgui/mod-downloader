package configs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestPreferencesThemeReadsPrefixedEnvironment(t *testing.T) {
	t.Setenv("PREFERS_THEME", "light")
	t.Setenv("PREFERS_LANGUAGE", "en")
	t.Setenv("PREFERS_ANIMATION_DURATION_MULTIPLIER", "2.5")
	t.Setenv("THEME", "dark")

	var cfg Config
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.Theme != ThemeLight {
		t.Fatalf("theme = %q, want %q", cfg.Prefers.Theme, ThemeLight)
	}
	if cfg.Prefers.Language != LanguageEnglish {
		t.Fatalf("language = %q, want %q", cfg.Prefers.Language, LanguageEnglish)
	}
	if cfg.Prefers.NormalizedAnimationDurationMultiplier() != 2.5 {
		t.Fatalf("animation multiplier = %v, want 2.5", cfg.Prefers.NormalizedAnimationDurationMultiplier())
	}
}

func TestParseLanguageSupportsCanonicalAndCompatibleValues(t *testing.T) {
	tests := map[string]Language{
		"":        LanguageSystem,
		"system":  LanguageSystem,
		"zh":      LanguageChinese,
		"zh-CN":   LanguageChinese,
		"zh-Hans": LanguageChinese,
		"chinese": LanguageChinese,
		"en":      LanguageEnglish,
		"english": LanguageEnglish,
		" EN ":    LanguageEnglish,
	}
	for value, want := range tests {
		if got := ParseLanguage(value); got != want {
			t.Fatalf("ParseLanguage(%q) = %q, want %q", value, got, want)
		}
	}
	if got := ParseLanguage("invalid"); got != "" {
		t.Fatalf("ParseLanguage(invalid) = %q, want empty", got)
	}
	if got := Language("invalid").Normalized(); got != LanguageSystem {
		t.Fatalf("invalid language normalized to %q, want %q", got, LanguageSystem)
	}
}

func TestPreferencesLanguageReadsTOMLAndDefaultsToSystem(t *testing.T) {
	if got := (Preferences{}).Language.Normalized(); got != LanguageSystem {
		t.Fatalf("default language = %q, want %q", got, LanguageSystem)
	}

	var cfg Config
	if _, err := toml.Decode(`
[preferences]
language = "zh"
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.Language != LanguageChinese {
		t.Fatalf("language = %q, want %q", cfg.Prefers.Language, LanguageChinese)
	}
}

func TestPreferencesSimpleModeReadsTOMLAndEnvironment(t *testing.T) {
	if (Preferences{}).SimpleMode {
		t.Fatal("default simple mode = true, want false")
	}

	var cfg Config
	if _, err := toml.Decode(`
[preferences]
simple_mode = true
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Prefers.SimpleMode {
		t.Fatal("TOML simple mode = false, want true")
	}

	t.Setenv("PREFERS_SIMPLE_MODE", "true")
	cfg = Config{}
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Prefers.SimpleMode {
		t.Fatal("environment simple mode = false, want true")
	}
}

func TestPreferencesHardwareAccelerationDefaultsToEnabled(t *testing.T) {
	if !(Preferences{}).HardwareAccelerationValue() {
		t.Fatal("default hardware acceleration = false, want true")
	}

	var cfg Config
	if _, err := toml.Decode(`
[preferences]
hardware_acceleration = false
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.HardwareAccelerationValue() {
		t.Fatal("TOML hardware acceleration = true, want false")
	}

	t.Setenv("PREFERS_HARDWARE_ACCELERATION", "false")
	cfg = Config{}
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.HardwareAccelerationValue() {
		t.Fatal("environment hardware acceleration = true, want false")
	}
}

func TestLoadHardwareAccelerationReadsConfigFileBeforeCoreStartup(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	if got := LoadHardwareAcceleration(); !got {
		t.Fatal("missing config file should default hardware acceleration to true")
	}

	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(`
[preferences]
hardware_acceleration = true
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := LoadHardwareAcceleration(); !got {
		t.Fatal("config file hardware acceleration = false, want true")
	}

	t.Setenv("PREFERS_HARDWARE_ACCELERATION", "false")
	if got := LoadHardwareAcceleration(); got {
		t.Fatal("environment override should disable hardware acceleration")
	}
}

func TestSaveReplacesConfigAtomicallyAndLeavesNoTempFiles(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	if err := Save(&Config{Prefers: Preferences{MinecraftDir: "/first"}}); err != nil {
		t.Fatal(err)
	}
	if err := Save(&Config{Prefers: Preferences{MinecraftDir: "/second"}}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(contents), "/second") || strings.Contains(string(contents), "/first") {
		t.Fatalf("saved config = %q, want only the second write", contents)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("leftover temp config file: %s", entry.Name())
		}
	}
}

func TestSaveFailurePreservesOriginalConfig(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dir, 0o755)
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	original := []byte("[preferences]\nminecraft_dir = \"/keep\"\n")
	if err := os.WriteFile(filepath.Join(dir, configFileName), original, 0o644); err != nil {
		t.Fatal(err)
	}
	// A read-only directory makes CreateTemp fail, which must surface as an
	// error instead of truncating the existing file via os.Create.
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	if err := Save(&Config{Prefers: Preferences{MinecraftDir: "/replacement"}}); err == nil {
		t.Fatal("Save succeeded with an unwritable config directory")
	}
	contents, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(original) {
		t.Fatalf("config after failed save = %q, want original %q", contents, original)
	}
}

func TestLoadEnvDoesNotReadTOML(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(`
[keys]
curseforge_api_key = "from-file"
[preferences]
minecraft_dir = "/from-file"
[runtime]
minecraft_version = "1.20.1"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KEYS_CF_API_KEY", "from-env")
	t.Setenv("MINECRAFT_VERSION", "1.21.1")
	t.Setenv("MOD_LOADER", "fabric")
	t.Setenv("MOD_DOWNLOADER_CACHE_DIR", "/tmp/md-cache")

	cfg, err := LoadEnv()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Keys.CurseforgeApiKey != "from-env" {
		t.Fatalf("curseforge key = %q, want from-env", cfg.Keys.CurseforgeApiKey)
	}
	if cfg.Prefers.MinecraftDir != "" {
		t.Fatalf("minecraft dir = %q, want empty", cfg.Prefers.MinecraftDir)
	}
	if cfg.Runtime.MinecraftVersion != "1.21.1" || cfg.Runtime.ModLoader != "fabric" || cfg.Runtime.CacheDir != "/tmp/md-cache" {
		t.Fatalf("runtime = %#v", cfg.Runtime)
	}
}

func TestParseThemeSupportsLegacyNumericValues(t *testing.T) {
	tests := map[string]Theme{
		"0": ThemeDark,
		"1": ThemeLight,
		"2": ThemeSystem,
	}
	for value, want := range tests {
		if got := ParseTheme(value); got != want {
			t.Fatalf("ParseTheme(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestPreferencesThemeReadsStringTOML(t *testing.T) {
	var cfg Config
	if _, err := toml.Decode(`
[preferences]
theme = "system"
animation_duration_multiplier = 1.75
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.Theme != ThemeSystem {
		t.Fatalf("theme = %q, want %q", cfg.Prefers.Theme, ThemeSystem)
	}
	if cfg.Prefers.NormalizedAnimationDurationMultiplier() != 1.75 {
		t.Fatalf("animation multiplier = %v, want 1.75", cfg.Prefers.NormalizedAnimationDurationMultiplier())
	}
}

func TestPreferencesThemeReadsLegacyNumericTOML(t *testing.T) {
	var cfg Config
	if _, err := toml.Decode(`
[preferences]
theme = 2
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.Theme != ThemeSystem {
		t.Fatalf("theme = %q, want %q", cfg.Prefers.Theme, ThemeSystem)
	}
}

func TestPreferencesAnimationDurationDefaultsAndNormalization(t *testing.T) {
	defaults := Preferences{}
	if defaults.NormalizedAnimationDurationMultiplier() != DefaultAnimationDurationMultiplier {
		t.Fatalf("default animation multiplier = %v, want %v", defaults.NormalizedAnimationDurationMultiplier(), DefaultAnimationDurationMultiplier)
	}

	tests := map[float64]float64{
		-1:   DefaultAnimationDurationMultiplier,
		0:    DefaultAnimationDurationMultiplier,
		0.05: MinAnimationDurationMultiplier,
		0.1:  MinAnimationDurationMultiplier,
		0.25: 0.25,
		1.5:  1.5,
		5:    MaxAnimationDurationMultiplier,
	}
	for value, want := range tests {
		if got := NormalizeAnimationDurationMultiplier(value); got != want {
			t.Fatalf("NormalizeAnimationDurationMultiplier(%v) = %v, want %v", value, got, want)
		}
	}
}

func TestPreferencesAutoScanUnusedDependenciesDefaultAndTOML(t *testing.T) {
	defaults := Preferences{}
	if defaults.AutoScanUnusedDependenciesValue() {
		t.Fatal("default auto scan unused dependencies = true, want false")
	}

	var cfg Config
	if _, err := toml.Decode(`
[preferences]
auto_scan_unused_dependencies = true
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Prefers.AutoScanUnusedDependenciesValue() {
		t.Fatal("explicit auto scan unused dependencies = false, want true")
	}
}

func TestPreferencesMCIMDefaultTOMLAndEnvironment(t *testing.T) {
	if (Preferences{}).MCIMEnabledValue() {
		t.Fatal("default MCIM enabled = true, want false")
	}

	var cfg Config
	if _, err := toml.Decode(`
[preferences]
mcim_enabled = true
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Prefers.MCIMEnabledValue() {
		t.Fatal("TOML MCIM enabled = false, want true")
	}

	t.Setenv("PREFERS_MCIM_ENABLED", "true")
	cfg = Config{}
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Prefers.MCIMEnabledValue() {
		t.Fatal("environment MCIM enabled = false, want true")
	}
}

func TestNetworkConfigDefaultsAndTOML(t *testing.T) {
	defaults := Config{}
	if got := defaults.Downloads.FileConcurrencyValue(); got != DefaultFileConcurrency {
		t.Fatalf("default file concurrency = %d, want %d", got, DefaultFileConcurrency)
	}
	if got := defaults.Downloads.ConcurrentDownloadsValue(); got != DefaultConcurrentDownloads {
		t.Fatalf("default concurrent downloads = %d, want %d", got, DefaultConcurrentDownloads)
	}
	if got := defaults.Downloads.TargetDownloadRateMiBValue(); got != DefaultTargetDownloadRateMiB {
		t.Fatalf("default target download rate = %v, want %v", got, DefaultTargetDownloadRateMiB)
	}
	if got := defaults.API.RequestsPerSecondValue(); got != 0 {
		t.Fatalf("default API rate = %d, want unlimited", got)
	}

	var cfg Config
	if _, err := toml.Decode(`
[downloads]
file_concurrency = 8
concurrent_downloads = 3
adaptive_file_concurrency = true
target_download_rate_mib = 2.5
verify_sha1 = true
[api]
requests_per_second = 12
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Downloads.FileConcurrencyValue() != 8 || cfg.Downloads.ConcurrentDownloadsValue() != 3 || !cfg.Downloads.AdaptiveFileConcurrency || cfg.Downloads.TargetDownloadRateMiBValue() != 2.5 || !cfg.Downloads.VerifySHA1 {
		t.Fatalf("downloads = %#v", cfg.Downloads)
	}
	if cfg.API.RequestsPerSecondValue() != 12 {
		t.Fatalf("API = %#v", cfg.API)
	}

	t.Setenv("DOWNLOADS_VERIFY_SHA1", "true")
	cfg = Config{}
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Downloads.VerifySHA1 {
		t.Fatal("environment verify SHA1 = false, want true")
	}
}

func TestTargetDownloadRateClampsOutOfRangeValues(t *testing.T) {
	if got := (DownloadConfig{TargetDownloadRateMiB: 0.01}).TargetDownloadRateMiBValue(); got != MinTargetDownloadRateMiB {
		t.Fatalf("low target download rate = %v, want %v", got, MinTargetDownloadRateMiB)
	}
	if got := (DownloadConfig{TargetDownloadRateMiB: 10}).TargetDownloadRateMiBValue(); got != MaxTargetDownloadRateMiB {
		t.Fatalf("high target download rate = %v, want %v", got, MaxTargetDownloadRateMiB)
	}
}

func TestNetworkConfigClampsOutOfRangeValues(t *testing.T) {
	tests := []struct {
		name                string
		config              Config
		wantFileConcurrency int
		wantDownloads       int
		wantRPS             int
	}{
		{
			name:                "below minimum uses defaults",
			config:              Config{Downloads: DownloadConfig{FileConcurrency: -1, ConcurrentDownloads: -1}, API: APIConfig{RequestsPerSecond: -1}},
			wantFileConcurrency: DefaultFileConcurrency,
			wantDownloads:       DefaultConcurrentDownloads,
			wantRPS:             MinRequestsPerSecond,
		},
		{
			name:                "above maximum clamps",
			config:              Config{Downloads: DownloadConfig{FileConcurrency: 100, ConcurrentDownloads: MaxConcurrentDownloads + 1}, API: APIConfig{RequestsPerSecond: 101}},
			wantFileConcurrency: MaxFileConcurrency,
			wantDownloads:       MaxConcurrentDownloads,
			wantRPS:             MaxRequestsPerSecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.Downloads.FileConcurrencyValue(); got != tt.wantFileConcurrency {
				t.Fatalf("file concurrency = %d, want %d", got, tt.wantFileConcurrency)
			}
			if got := tt.config.Downloads.ConcurrentDownloadsValue(); got != tt.wantDownloads {
				t.Fatalf("concurrent downloads = %d, want %d", got, tt.wantDownloads)
			}
			if got := tt.config.API.RequestsPerSecondValue(); got != tt.wantRPS {
				t.Fatalf("requests per second = %d, want %d", got, tt.wantRPS)
			}
		})
	}
}

func TestLoggingConfigTOMLAndEnvironment(t *testing.T) {
	var cfg Config
	if _, err := toml.Decode(`
[logging]
enabled = false
force_file = true
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Logging.EnabledValue() || !cfg.Logging.ForceFile {
		t.Fatalf("logging TOML = %#v", cfg.Logging)
	}

	t.Setenv("LOGGING_ENABLED", "false")
	t.Setenv("LOGGING_FORCE_FILE", "true")
	cfg = Config{}
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Logging.EnabledValue() || !cfg.Logging.ForceFile {
		t.Fatalf("logging env = %#v", cfg.Logging)
	}
}

func TestLoggingConfigRotationAndFileFields(t *testing.T) {
	var cfg LoggingConfig
	if _, err := toml.Decode(`
file = "/tmp/custom.log"
max_size_mb = 12
max_backups = 4
`, &cfg); err != nil {
		t.Fatal(err)
	}
	canonical := cfg.Canonical()
	if canonical.File != "/tmp/custom.log" || canonical.MaxSizeMB != 12 || canonical.MaxBackups != 4 {
		t.Fatalf("canonical logging = %#v", canonical)
	}

	t.Setenv("LOGGING_FILE", "/tmp/from-env.log")
	t.Setenv("LOGGING_MAX_SIZE_MB", "9")
	t.Setenv("LOGGING_MAX_BACKUPS", "3")
	var env Config
	if err := readEnv(&env); err != nil {
		t.Fatal(err)
	}
	if env.Logging.File != "/tmp/from-env.log" || env.Logging.MaxSizeMB != 9 || env.Logging.MaxBackups != 3 {
		t.Fatalf("logging env = %#v", env.Logging)
	}
}

func TestLoggingEnabledTakesPrecedenceOverLegacyDisabled(t *testing.T) {
	var cfg Config
	if _, err := toml.Decode(`
[logging]
enabled = true
disabled = true
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Logging.EnabledValue() {
		t.Fatal("enabled must take precedence over legacy disabled")
	}

	t.Setenv("LOGGING_ENABLED", "true")
	t.Setenv("LOGGING_DISABLED", "true")
	cfg = Config{}
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if !cfg.Logging.EnabledValue() {
		t.Fatal("LOGGING_ENABLED must take precedence over legacy LOGGING_DISABLED")
	}
}

func TestSaveLoggingUsesCanonicalEnabledKey(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	disabled := true
	if err := Save(&Config{Logging: LoggingConfig{Disabled: &disabled}}); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(dir, configFileName))
	if err != nil {
		t.Fatal(err)
	}
	text := string(contents)
	if !strings.Contains(text, "enabled = false") {
		t.Fatalf("saved config does not contain canonical enabled key:\n%s", text)
	}
	if strings.Contains(text, "disabled =") {
		t.Fatalf("saved config contains legacy disabled key:\n%s", text)
	}
}

func TestLoadLoggingReadsFileAndEnvironmentOnly(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.WriteFile(filepath.Join(dir, configFileName), []byte(`
[logging]
disabled = true
force_file = false
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LOGGING_FORCE_FILE", "true")

	loggingConfig, err := LoadLogging()
	if err != nil {
		t.Fatal(err)
	}
	if loggingConfig.EnabledValue() || !loggingConfig.ForceFile {
		t.Fatalf("logging bootstrap = %#v", loggingConfig)
	}
}

func TestNetworkConfigReadsPrefixedEnvironment(t *testing.T) {
	t.Setenv("DOWNLOADS_FILE_CONCURRENCY", "6")
	t.Setenv("DOWNLOADS_CONCURRENT_DOWNLOADS", "2")
	t.Setenv("API_REQUESTS_PER_SECOND", "9")

	var cfg Config
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Downloads.FileConcurrency != 6 || cfg.Downloads.ConcurrentDownloads != 2 || cfg.API.RequestsPerSecond != 9 {
		t.Fatalf("config = %#v", cfg)
	}
}
