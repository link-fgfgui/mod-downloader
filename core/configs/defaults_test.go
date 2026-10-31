package configs

import "testing"

func TestEffectiveCurseforgeAPIKey(t *testing.T) {
	prev := DefaultCurseforgeAPIKey
	t.Cleanup(func() { DefaultCurseforgeAPIKey = prev })

	t.Run("configured wins over default", func(t *testing.T) {
		DefaultCurseforgeAPIKey = "default-key"
		if got := EffectiveCurseforgeAPIKey(" user-key "); got != "user-key" {
			t.Fatalf("got %q, want user-key", got)
		}
	})

	t.Run("falls back to default when configured empty", func(t *testing.T) {
		DefaultCurseforgeAPIKey = " default-key "
		if got := EffectiveCurseforgeAPIKey("  "); got != "default-key" {
			t.Fatalf("got %q, want default-key", got)
		}
		if got := EffectiveCurseforgeAPIKey(""); got != "default-key" {
			t.Fatalf("got %q, want default-key", got)
		}
	})

	t.Run("both empty", func(t *testing.T) {
		DefaultCurseforgeAPIKey = "  "
		if got := EffectiveCurseforgeAPIKey(""); got != "" {
			t.Fatalf("got %q, want empty", got)
		}
	})
}
