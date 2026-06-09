package global

import (
	"testing"

	structs "mod-downloader/structs/minecraft"
)

func TestLocalModPathsInInstanceByModIDSupportsMultipleIDsPerSHA1(t *testing.T) {
	ClearLocalMods()
	defer ClearLocalMods()

	base := structs.ModInfo{
		FileName: "bundle",
		Path:     "versions/test/mods/bundle.jar",
		SHA1:     "same-sha1",
		Enabled:  true,
	}
	first := base
	first.ID = "firstmod"
	second := base
	second.ID = "secondmod"

	UpsertLocalMod(first, "test-instance", "1.20.1", "forge")
	UpsertLocalMod(second, "test-instance", "1.20.1", "forge")

	for _, modID := range []string{"firstmod", "secondmod"} {
		paths := LocalModPathsInInstanceByModID("test-instance", modID)
		if len(paths) != 1 {
			t.Fatalf("LocalModPathsInInstanceByModID(%q) returned %d paths, want 1", modID, len(paths))
		}
		if paths[0].Path != base.Path {
			t.Fatalf("LocalModPathsInInstanceByModID(%q) path = %q, want %q", modID, paths[0].Path, base.Path)
		}
	}
}
