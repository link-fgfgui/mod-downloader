package configs

import "strings"

// DefaultCurseforgeAPIKey is empty in source and may be overwritten at link time:
//
//	go build -ldflags "-X github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey=..."
//
// Prefer EffectiveCurseforgeAPIKey for outbound requests so a non-empty user
// configuration always wins over the compiled-in default.
var DefaultCurseforgeAPIKey string

// EffectiveCurseforgeAPIKey returns the user-configured key when present,
// otherwise the compile-time default. Both sides are trimmed.
func EffectiveCurseforgeAPIKey(configured string) string {
	if key := strings.TrimSpace(configured); key != "" {
		return key
	}
	return strings.TrimSpace(DefaultCurseforgeAPIKey)
}
