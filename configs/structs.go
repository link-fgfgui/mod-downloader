package configs

type APIKeys struct {
	CurseforgeApiKey string `toml:"curseforge_api_key" json:"curseforge_api_key" env:"CF_API_KEY"`
	ModrinthApiKey   string `toml:"modrinth_api_key" json:"modrinth_api_key" env:"MODRINTH_API_KEY"`
}

type Theme int

const (
	ThemeDark Theme = iota
	ThemeLight
	ThemeSystem
)

type Preferences struct {
	Theme        Theme  `toml:"theme" json:"theme" env:"THEME" env-default:"0"`
	MinecraftDir string `toml:"minecraft_dir" json:"minecraft_dir" env:"MINECRAFT_DIR"`
}

type Config struct {
	Keys    APIKeys     `toml:"keys" json:"keys" env-prefix:"KEYS_"`
	Prefers Preferences `toml:"preferences" json:"preferences" env-prefix:"PREFERS_"`
}
