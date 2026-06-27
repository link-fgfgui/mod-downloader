package modbridge

import (
	"path/filepath"
	"testing"

	"mod-downloader/global"
	"mod-downloader/models"
	mcstructs "mod-downloader/structs/minecraft"
)

func TestFindVersionByID(t *testing.T) {
	versions := []models.ModVersion{
		{ID: "v1", Name: "old"},
		{ID: "v2", Name: "new"},
	}

	version, ok := FindVersionByID(versions, "v2")
	if !ok || version.Name != "new" {
		t.Fatalf("FindVersionByID() = %#v, %v", version, ok)
	}
	if _, ok := FindVersionByID(versions, "missing"); ok {
		t.Fatal("FindVersionByID() found missing version")
	}
}

func TestProjectVersionSHA1Set(t *testing.T) {
	set := projectVersionSHA1Set([]models.ModVersion{
		{SHA1: " ABC "},
		{SHA1: ""},
		{SHA1: "def"},
	})

	if !set["abc"] || !set["def"] || len(set) != 2 {
		t.Fatalf("sha1 set = %#v", set)
	}
}

func TestSelectedVersionModsDirUsesInstanceIDNotDisplayName(t *testing.T) {
	mcDir := t.TempDir()
	global.SetMinecraftDir(mcDir)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
	})

	got := selectedVersionModsDir(mcstructs.VersionInfo{
		ID:   "instance-folder",
		Name: "Display Name",
	})
	want := filepath.Join(mcDir, "versions", "instance-folder", "mods")
	if got != want {
		t.Fatalf("selectedVersionModsDir() = %q, want %q", got, want)
	}
}

func TestNormalizedModIDs(t *testing.T) {
	got := normalizedModIDs([]string{" Sodium ", "sodium", "Other", ""})
	want := []string{"sodium", "other"}
	if len(got) != len(want) {
		t.Fatalf("normalizedModIDs() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizedModIDs() = %#v, want %#v", got, want)
		}
	}
}

func TestLocalModPathsForModIDsDeduplicatesByPath(t *testing.T) {
	global.ClearLocalMods()
	t.Cleanup(global.ClearLocalMods)

	base := mcstructs.ModInfo{
		FileName: "bundle",
		Path:     "versions/test/mods/bundle.jar",
		SHA1:     "same-sha1",
		Enabled:  true,
	}
	first := base
	first.ID = "firstmod"
	second := base
	second.ID = "secondmod"
	global.UpsertLocalMod(first, "test-instance", "1.20.1", "forge")
	global.UpsertLocalMod(second, "test-instance", "1.20.1", "forge")

	paths := LocalModPathsForModIDs([]string{"firstmod", "secondmod"}, "test-instance")
	if len(paths) != 1 {
		t.Fatalf("LocalModPathsForModIDs() returned %d paths, want 1", len(paths))
	}
	if paths[0].Path != base.Path {
		t.Fatalf("LocalModPathsForModIDs() path = %q, want %q", paths[0].Path, base.Path)
	}
}
