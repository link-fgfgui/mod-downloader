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

// writeFabricManifest writes a minimal Fabric version manifest at
// <gameDir>/versions/<versionID>/<versionID>.json so CheckManifest recognizes it.
func writeFabricManifest(t *testing.T, gameDir, versionID, mcVersion string) {
	t.Helper()
	versionDir := filepath.Join(gameDir, "versions", versionID)
	if err := os.MkdirAll(versionDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{
		"name": "` + versionID + `",
		"id": "` + versionID + `",
		"patches": [
			{"id": "game", "version": "` + mcVersion + `"},
			{"id": "fabric", "version": "0.16.0"}
		]
	}`
	if err := os.WriteFile(filepath.Join(versionDir, versionID+".json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadVersionsFromDiskAggregatesPrismInstances(t *testing.T) {
	instancesDir := t.TempDir()

	// First Prism instance with a .minecraft subfolder.
	instance1Dir := filepath.Join(instancesDir, "FabricPack")
	gameDir1 := filepath.Join(instance1Dir, ".minecraft")
	if err := os.MkdirAll(gameDir1, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMarkerFile(t, filepath.Join(instance1Dir, "instance.cfg"))
	writeFabricManifest(t, gameDir1, "fabric-loader-1.21.1", "1.21.1")

	// Second Prism instance, no .minecraft subfolder (uses instance root as game dir).
	instance2Dir := filepath.Join(instancesDir, "BareNeoForge")
	writeMarkerFile(t, filepath.Join(instance2Dir, "mmc-pack.json"))
	writeFabricManifest(t, instance2Dir, "neoforge-1.20.1", "1.20.1")

	// A non-instance subfolder should be skipped silently.
	if err := os.MkdirAll(filepath.Join(instancesDir, "random-notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	global.SetMinecraftDir(instancesDir)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
		global.ClearLocalMods()
	})

	versions := loadVersionsFromDisk(instancesDir)
	if len(versions) != 2 {
		t.Fatalf("loadVersionsFromDisk() returned %d versions, want 2: %+v", len(versions), versions)
	}

	byID := make(map[string]structs.VersionInfo, len(versions))
	byName := make(map[string]structs.VersionInfo, len(versions))
	for _, v := range versions {
		byID[v.ID] = v
		byName[v.Name] = v
	}

	// One entry per Prism instance; Name is the instance name (sidebar display),
	// ID is the composite "<instance>/<versionFolder>" for path resolution.
	v1, ok1 := byID["FabricPack/fabric-loader-1.21.1"]
	if !ok1 {
		t.Fatalf("missing composite ID FabricPack/fabric-loader-1.21.1 in %+v", byID)
	}
	if v1.Name != "FabricPack" {
		t.Fatalf("FabricPack entry Name = %q, want instance name only", v1.Name)
	}
	if v1.MinecraftVersion != "1.21.1" || v1.ModLoader != "fabric" {
		t.Fatalf("FabricPack entry = %#v", v1)
	}
	if _, ok := byName["FabricPack"]; !ok {
		t.Fatalf("versionMap not keyed by instance name FabricPack")
	}

	v2, ok2 := byID["BareNeoForge/neoforge-1.20.1"]
	if !ok2 {
		t.Fatalf("missing composite ID BareNeoForge/neoforge-1.20.1 in %+v", byID)
	}
	if v2.Name != "BareNeoForge" {
		t.Fatalf("BareNeoForge entry Name = %q, want instance name only", v2.Name)
	}
	if v2.MinecraftVersion != "1.20.1" || v2.ModLoader != "fabric" {
		t.Fatalf("BareNeoForge entry = %#v", v2)
	}
}

func TestScanVersionModsResolvesPrismCompositeVersionDir(t *testing.T) {
	instancesDir := t.TempDir()
	instanceDir := filepath.Join(instancesDir, "MyFabric")
	gameDir := filepath.Join(instanceDir, ".minecraft")
	versionID := "fabric-loader-1.21.1"
	versionDir := filepath.Join(gameDir, "versions", versionID)
	modsDir := filepath.Join(versionDir, "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	global.SetMinecraftDir(instancesDir)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
		global.ClearLocalMods()
	})

	version := structs.VersionInfo{
		ID:               "MyFabric/" + versionID,
		Name:             "MyFabric",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}
	// No jars in the mods dir, so Mods should come back empty but the path
	// resolution must not panic or return an error.
	scanned := scanVersionMods(version, instancesDir)
	if len(scanned.Mods) != 0 {
		t.Fatalf("scanVersionMods() returned %d mods, want 0 (empty mods dir)", len(scanned.Mods))
	}
}

// writeMarkerFile is also defined in minecraft/prism_test.go but that file is
// in a different package; duplicate the helper here for the main package tests.
func writeMarkerFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
}
