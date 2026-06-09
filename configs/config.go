package configs

import (
	"fmt"
	"os"
	"path/filepath"

	"mod-downloader/logging"

	"github.com/BurntSushi/toml"
	"github.com/ilyakaznacheev/cleanenv"
)

const configFileName = "mod-downloader.toml"

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
			if err := cleanenv.ReadEnv(&cfg); err != nil {
				logging.Error("load config from environment failed", "error", err)
				return nil, fmt.Errorf("read env: %w", err)
			}
			logging.Info("config loaded from environment", "hasCurseforgeKey", cfg.Keys.CurseforgeApiKey != "", "hasModrinthKey", cfg.Keys.ModrinthApiKey != "", "minecraftDir", cfg.Prefers.MinecraftDir)
			return &cfg, nil
		}
		logging.Error("load config file failed", "path", path, "error", err)
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	logging.Info("config loaded from file", "path", path, "hasCurseforgeKey", cfg.Keys.CurseforgeApiKey != "", "hasModrinthKey", cfg.Keys.ModrinthApiKey != "", "minecraftDir", cfg.Prefers.MinecraftDir)
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

	f, err := os.Create(path)
	if err != nil {
		logging.Error("create config file failed", "path", path, "error", err)
		return fmt.Errorf("create config %s: %w", path, err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		logging.Error("encode config failed", "path", path, "error", err)
		return fmt.Errorf("encode config: %w", err)
	}

	logging.Info("config saved", "path", path, "hasCurseforgeKey", cfg.Keys.CurseforgeApiKey != "", "hasModrinthKey", cfg.Keys.ModrinthApiKey != "", "minecraftDir", cfg.Prefers.MinecraftDir)
	return nil
}
