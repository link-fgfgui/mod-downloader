package configs

import (
	"fmt"
	"strings"
)

type APIKeys struct {
	CurseforgeApiKey string `toml:"curseforge_api_key" json:"curseforge_api_key" env:"CF_API_KEY"`
	ModrinthApiKey   string `toml:"modrinth_api_key" json:"modrinth_api_key" env:"MODRINTH_API_KEY"`
}

type Theme string

const (
	ThemeDark   Theme = "dark"
	ThemeLight  Theme = "light"
	ThemeSystem Theme = "system"
)

func (t *Theme) UnmarshalText(text []byte) error {
	theme := ParseTheme(string(text))
	if theme == "" {
		return fmt.Errorf("invalid theme %q: expected dark, light, or system", strings.TrimSpace(string(text)))
	}
	*t = theme
	return nil
}

func (t Theme) MarshalText() ([]byte, error) {
	return []byte(t.Normalized().String()), nil
}

func (t Theme) String() string {
	return string(t)
}

func (t Theme) Normalized() Theme {
	if normalized := ParseTheme(string(t)); normalized != "" {
		return normalized
	}
	return ThemeDark
}

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

type Preferences struct {
	Theme        Theme  `toml:"theme" json:"theme" env:"THEME" env-default:"dark"`
	MinecraftDir string `toml:"minecraft_dir" json:"minecraft_dir" env:"MINECRAFT_DIR"`
}

type Config struct {
	Keys    APIKeys     `toml:"keys" json:"keys" env-prefix:"KEYS_"`
	Prefers Preferences `toml:"preferences" json:"preferences" env-prefix:"PREFERS_"`
}
