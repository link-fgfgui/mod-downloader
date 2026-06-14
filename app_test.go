package main

import (
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
