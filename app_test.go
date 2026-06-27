package main

import (
	"os"
	"path/filepath"
	"testing"

	"mod-downloader/global"
	structs "mod-downloader/structs/minecraft"
)

func TestGetVersionsSelectsFirstVersionByDefault(t *testing.T) {
	dir := t.TempDir()
	global.SetMinecraftDir(dir)
	global.SetVersionsForDir(dir, []structs.VersionInfo{
		{
			ID:               "first",
			Name:             "First",
			MinecraftVersion: "1.21.1",
			ModLoader:        "fabric",
		},
		{
			ID:               "second",
			Name:             "Second",
			MinecraftVersion: "1.20.1",
			ModLoader:        "forge",
		},
	})
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	app := &App{}
	versions := app.GetVersions()
	if len(versions) != 2 {
		t.Fatalf("GetVersions() returned %d versions, want 2", len(versions))
	}

	selected := app.GetSelectedVersion()
	if selected.ID != "first" {
		t.Fatalf("GetSelectedVersion().ID = %q, want first", selected.ID)
	}
}

func TestScanAllModDirsForHardlinkIndexIgnoresStaleGeneration(t *testing.T) {
	dir := t.TempDir()
	modsDir := filepath.Join(dir, "versions", "first", "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		t.Fatalf("mkdir mods: %v", err)
	}
	if err := os.WriteFile(filepath.Join(modsDir, "mod.jar"), []byte("jar"), 0o644); err != nil {
		t.Fatalf("write jar: %v", err)
	}

	global.SetMinecraftDir(dir)
	generation := global.HardlinkIndexGeneration()
	global.HardlinkIndexClear()
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.HardlinkIndexClear()
	})

	scanAllModDirsForHardlinkIndex(dir, []structs.VersionInfo{{ID: "first"}}, generation)
	if _, ok := global.HardlinkIndexLookup("8a38e4e8d082c15b2104c026f910da1fe949f36d"); ok {
		t.Fatalf("stale generation added hardlink index entry")
	}
}
