package downloader

import (
	"path/filepath"
	"testing"

	"mod-downloader/global"
	"mod-downloader/modbridge"
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
	if states[0].Key != "modrinth:sodium" || states[0].Status != modbridge.BtnStatusNew || states[0].Icon != "mdi-download" || states[0].Color != "primary" || states[0].Disabled {
		t.Fatalf("state = %#v", states[0])
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

func TestDependencyDownloadRequestCarriesDependencyVersionID(t *testing.T) {
	req, ok := dependencyDownloadRequest("Modrinth", models.ModDependency{
		DependencyProjectID: "fabric-api",
		DependencyVersionID: "version-123",
		DependencyType:      "required",
	}, appstructs.ModDownloadRequest{
		MinecraftVersion: "1.21.1",
		ModLoader:        "Fabric",
	})
	if !ok {
		t.Fatal("dependencyDownloadRequest() ok = false")
	}
	if req.VersionID != "version-123" {
		t.Fatalf("VersionID = %q, want version-123", req.VersionID)
	}
}
