package downloader

import (
	"path/filepath"
	"testing"

	"mod-downloader/global"
	"mod-downloader/models"
	appstructs "mod-downloader/structs"
	mcstructs "mod-downloader/structs/minecraft"
)

func TestGetDownloadStatesReturnsDefaultWhenSelectedInstanceHasNoLocalMods(t *testing.T) {
	global.ClearLocalMods()
	global.SetMinecraftDir(t.TempDir())
	global.SetVersions([]mcstructs.VersionInfo{{
		ID:               "instance",
		Name:             "instance",
		MinecraftVersion: "1.21.1",
		ModLoader:        "neoforge",
	}})
	t.Cleanup(func() {
		global.ClearLocalMods()
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	states := GetDownloadStates(appstructs.DownloadStatesRequest{
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
		Results: []models.ModProject{{
			ID:       "modrinth:sodium",
			Platform: "Modrinth",
			Slug:     "sodium",
		}},
	})

	if len(states) != 1 {
		t.Fatalf("state count = %d, want 1", len(states))
	}
	if states[0].Key != "modrinth:sodium" || states[0].Status != btnStatusNew || states[0].Icon != "mdi-download" || states[0].Color != "primary" || states[0].Disabled {
		t.Fatalf("state = %#v", states[0])
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

func TestPathInLocalModPathsMatchesRelativePaths(t *testing.T) {
	mcDir := t.TempDir()
	global.SetMinecraftDir(mcDir)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
	})

	path := filepath.Join(mcDir, "versions", "instance", "mods", "mod.jar")
	paths := []global.LocalModFilePath{{
		Path: filepath.Join("versions", "instance", "mods", "mod.jar"),
	}}
	if !pathInLocalModPaths(path, paths) {
		t.Fatalf("pathInLocalModPaths(%q) = false, want true", path)
	}
}
