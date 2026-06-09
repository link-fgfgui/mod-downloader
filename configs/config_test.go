package configs

import (
	"testing"

	"github.com/BurntSushi/toml"
)

func TestPreferencesThemeReadsPrefixedEnvironment(t *testing.T) {
	t.Setenv("PREFERS_THEME", "light")
	t.Setenv("THEME", "dark")

	var cfg Config
	if err := readEnv(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.Theme != ThemeLight {
		t.Fatalf("theme = %q, want %q", cfg.Prefers.Theme, ThemeLight)
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
`, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Prefers.Theme != ThemeSystem {
		t.Fatalf("theme = %q, want %q", cfg.Prefers.Theme, ThemeSystem)
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
