package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/link-fgfgui/mod-downloader-core/logging"

	"github.com/BurntSushi/toml"
	"github.com/ilyakaznacheev/cleanenv"
)

const configFileName = "mod-downloader.toml"

type loggingOnlyConfig struct {
	Logging LoggingConfig `toml:"logging" env-prefix:"LOGGING_"`
}

type hardwareAccelerationConfig struct {
	Prefers Preferences `toml:"preferences"`
}

// AppConfigPath returns the path to the config file in pwd.
func AppConfigPath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads configuration from config.toml and environment variables.
// Environment variables override file values.
func Load() (*Config, error) {
	var cfg Config

	path, err := AppConfigPath()
	if err != nil {
		logging.Error("resolve config path failed", "error", err)
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	logging.Debug("load config started", "path", path)

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		if os.IsNotExist(err) {
			logging.Info("config file not found, loading from environment", "path", path)
			if err := readEnv(&cfg); err != nil {
				logging.Error("load config from environment failed", "error", err)
				return nil, fmt.Errorf("read env: %w", err)
			}
			cfg.Logging = cfg.Logging.Canonical()
			logging.Info("config loaded from environment", "hasCurseforgeKey", cfg.Keys.CurseforgeApiKey != "", "hasModrinthKey", cfg.Keys.ModrinthApiKey != "", "minecraftDir", cfg.Prefers.MinecraftDir)
			return &cfg, nil
		}
		logging.Error("load config file failed", "path", path, "error", err)
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := readHardwareAccelerationEnv(&cfg.Prefers); err != nil {
		return nil, fmt.Errorf("read hardware acceleration env: %w", err)
	}
	if err := readLoggingEnv(&cfg.Logging); err != nil {
		return nil, fmt.Errorf("read logging env: %w", err)
	}

	cfg.Logging = cfg.Logging.Canonical()
	logging.Info("config loaded from file", "path", path, "hasCurseforgeKey", cfg.Keys.CurseforgeApiKey != "", "hasModrinthKey", cfg.Keys.ModrinthApiKey != "", "minecraftDir", cfg.Prefers.MinecraftDir)
	return &cfg, nil
}

// LoadLogging reads only the logging bootstrap settings without emitting logs.
// App startup uses it before the regular config loader so disabled logging is
// honored from the first config-loader message onward.
func LoadLogging() (LoggingConfig, error) {
	var cfg loggingOnlyConfig
	path, err := AppConfigPath()
	if err != nil {
		return LoggingConfig{}, err
	}
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		if os.IsNotExist(err) {
			if err := cleanenv.ReadEnv(&cfg); err != nil {
				return LoggingConfig{}, err
			}
			if err := readLoggingEnv(&cfg.Logging); err != nil {
				return LoggingConfig{}, err
			}
			return cfg.Logging.Canonical(), nil
		}
		return LoggingConfig{}, err
	}
	if err := readLoggingEnv(&cfg.Logging); err != nil {
		return LoggingConfig{}, err
	}
	return cfg.Logging.Canonical(), nil
}

// LoadHardwareAcceleration reads only the GPU preference so the Wails window
// can be created with the right webview policy before the core starts. It
// emits no logs because logging is not configured at this point in startup.
func LoadHardwareAcceleration() bool {
	var cfg hardwareAccelerationConfig
	path, err := AppConfigPath()
	if err != nil {
		return DefaultHardwareAcceleration
	}
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		if !os.IsNotExist(err) {
			return DefaultHardwareAcceleration
		}
	}
	if err := readHardwareAccelerationEnv(&cfg.Prefers); err != nil {
		return DefaultHardwareAcceleration
	}
	if cfg.Prefers.HardwareAcceleration == nil {
		return DefaultHardwareAcceleration
	}
	return *cfg.Prefers.HardwareAcceleration
}

func readEnv(cfg *Config) error {
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return err
	}
	if err := readHardwareAccelerationEnv(&cfg.Prefers); err != nil {
		return err
	}
	return readLoggingEnv(&cfg.Logging)
}

// readHardwareAccelerationEnv applies the environment override explicitly
// because cleanenv cannot parse *bool fields from environment variables. The
// pointer keeps "unset" distinct from an explicit false in the TOML file.
func readHardwareAccelerationEnv(pref *Preferences) error {
	raw, ok := os.LookupEnv("PREFERS_HARDWARE_ACCELERATION")
	if !ok {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fmt.Errorf("parse PREFERS_HARDWARE_ACCELERATION: %w", err)
	}
	pref.HardwareAcceleration = BoolPtr(value)
	return nil
}

func readLoggingEnv(cfg *LoggingConfig) error {
	if raw, ok := os.LookupEnv("LOGGING_ENABLED"); ok {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse LOGGING_ENABLED: %w", err)
		}
		cfg.Enabled = BoolPtr(value)
	}
	if raw, ok := os.LookupEnv("LOGGING_DISABLED"); ok {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse LOGGING_DISABLED: %w", err)
		}
		cfg.Disabled = BoolPtr(value)
	}
	return nil
}

// LoadEnv reads configuration from environment variables only.
func LoadEnv() (*Config, error) {
	var cfg Config
	if err := readEnv(&cfg); err != nil {
		logging.Error("load config from environment failed", "error", err)
		return nil, fmt.Errorf("read env: %w", err)
	}
	cfg.Logging = cfg.Logging.Canonical()
	logging.Info("config loaded from environment", "hasCurseforgeKey", cfg.Keys.CurseforgeApiKey != "", "hasModrinthKey", cfg.Keys.ModrinthApiKey != "", "minecraftDir", cfg.Prefers.MinecraftDir, "cacheDir", cfg.Runtime.CacheDir)
	return &cfg, nil
}

// Save writes the configuration to mod-downloader.toml in pwd.
func Save(cfg *Config) error {
	path, err := AppConfigPath()
	if err != nil {
		logging.Error("resolve config path failed", "error", err)
		return fmt.Errorf("resolve config path: %w", err)
	}
	logging.Debug("save config started", "path", path)

	// Write to a sibling temp file first and only replace the real config after
	// the encoder has flushed and fsynced. A failed or partial write must never
	// truncate the user's existing configuration.
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		logging.Error("create temp config file failed", "path", path, "error", err)
		return fmt.Errorf("create temp config for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	canonical := *cfg
	canonical.Logging = cfg.Logging.Canonical()
	encoder := toml.NewEncoder(tmp)
	if err := encoder.Encode(&canonical); err != nil {
		_ = tmp.Close()
		logging.Error("encode config failed", "path", path, "error", err)
		return fmt.Errorf("encode config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		logging.Error("sync config file failed", "path", path, "error", err)
		return fmt.Errorf("sync config %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		logging.Error("close config file failed", "path", path, "error", err)
		return fmt.Errorf("close config %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		logging.Error("replace config file failed", "path", path, "error", err)
		return fmt.Errorf("replace config %s: %w", path, err)
	}
	cleanup = false

	logging.Debug("config saved", "path", path, "hasCurseforgeKey", cfg.Keys.CurseforgeApiKey != "", "hasModrinthKey", cfg.Keys.ModrinthApiKey != "", "minecraftDir", cfg.Prefers.MinecraftDir)
	return nil
}
