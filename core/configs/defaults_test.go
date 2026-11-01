// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package configs

import "testing"

func TestEffectiveCurseForgeAPIKey(t *testing.T) {
	prev := DefaultCurseForgeAPIKey
	t.Cleanup(func() { DefaultCurseForgeAPIKey = prev })

	t.Run("configured wins over default", func(t *testing.T) {
		DefaultCurseForgeAPIKey = "default-key"
		if got := EffectiveCurseForgeAPIKey(" user-key "); got != "user-key" {
			t.Fatalf("got %q, want user-key", got)
		}
	})

	t.Run("falls back to default when configured empty", func(t *testing.T) {
		DefaultCurseForgeAPIKey = " default-key "
		if got := EffectiveCurseForgeAPIKey("  "); got != "default-key" {
			t.Fatalf("got %q, want default-key", got)
		}
		if got := EffectiveCurseForgeAPIKey(""); got != "default-key" {
			t.Fatalf("got %q, want default-key", got)
		}
	})

	t.Run("both empty", func(t *testing.T) {
		DefaultCurseForgeAPIKey = "  "
		if got := EffectiveCurseForgeAPIKey(""); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})
}
