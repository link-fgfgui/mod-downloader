// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package configs

import "strings"

// DefaultCurseForgeAPIKey is empty in source and may be overwritten at link time:
//
//	go build -ldflags "-X github.com/link-fgfgui/mod-downloader/core/configs.DefaultCurseForgeAPIKey=..."
//
// Prefer EffectiveCurseForgeAPIKey for outbound requests so a non-empty user
// configuration always wins over the compiled-in default.
var DefaultCurseForgeAPIKey string

// EffectiveCurseForgeAPIKey returns the user-configured key when present,
// otherwise the compile-time default. Both sides are trimmed.
func EffectiveCurseForgeAPIKey(configured string) string {
	if key := strings.TrimSpace(configured); key != "" {
		return key
	}
	return strings.TrimSpace(DefaultCurseForgeAPIKey)
}
