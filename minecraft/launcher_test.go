package minecraft

import (
	"os"
	"path/filepath"
	"testing"

	structs "mod-downloader/structs/minecraft"
)

func TestLoadLauncherVersionsUsesStandardMinecraftFallback(t *testing.T) {
	root := t.TempDir()

	calls := 0
	got := LoadLauncherVersions(root, func(gameDir string) []structs.VersionInfo {
		calls++
		if gameDir != root {
			t.Fatalf("LoadLauncherVersions loader gameDir = %q, want %q", gameDir, root)
		}
		return []structs.VersionInfo{{
			ID:               "fabric-loader-1.21.1",
			Name:             "fabric-loader-1.21.1",
			MinecraftVersion: "1.21.1",
			ModLoader:        "fabric",
		}}
	})

	if calls != 1 {
		t.Fatalf("loader called %d times, want 1", calls)
	}
	if len(got) != 1 || got[0].ID != "fabric-loader-1.21.1" || got[0].Name != "fabric-loader-1.21.1" {
		t.Fatalf("LoadLauncherVersions() = %#v", got)
	}
}

func TestLoadLauncherVersionsAggregatesPrismInstances(t *testing.T) {
	root := t.TempDir()

	fabricDir := filepath.Join(root, "FabricPack")
	fabricGameDir := filepath.Join(fabricDir, ".minecraft")
	if err := os.MkdirAll(fabricGameDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeMarkerFile(t, filepath.Join(fabricDir, "instance.cfg"))

	bareDir := filepath.Join(root, "BareNeoForge")
	writeMarkerFile(t, filepath.Join(bareDir, "mmc-pack.json"))

	if err := os.MkdirAll(filepath.Join(root, "random-notes"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := LoadLauncherVersions(root, func(gameDir string) []structs.VersionInfo {
		switch gameDir {
		case fabricGameDir:
			return []structs.VersionInfo{{
				ID:               "fabric-loader-1.21.1",
				Name:             "fabric-loader-1.21.1",
				MinecraftVersion: "1.21.1",
				ModLoader:        "fabric",
			}}
		case bareDir:
			return []structs.VersionInfo{{
				ID:               "neoforge-1.20.1",
				Name:             "neoforge-1.20.1",
				MinecraftVersion: "1.20.1",
				ModLoader:        "neoforge",
			}}
		default:
			return nil
		}
	})

	byID := make(map[string]structs.VersionInfo, len(got))
	for _, version := range got {
		byID[version.ID] = version
	}
	if len(byID) != 2 {
		t.Fatalf("LoadLauncherVersions() returned %d versions, want 2: %#v", len(got), got)
	}
	if byID["FabricPack/fabric-loader-1.21.1"].Name != "FabricPack" {
		t.Fatalf("FabricPack entry = %#v", byID["FabricPack/fabric-loader-1.21.1"])
	}
	if byID["BareNeoForge/neoforge-1.20.1"].Name != "BareNeoForge" {
		t.Fatalf("BareNeoForge entry = %#v", byID["BareNeoForge/neoforge-1.20.1"])
	}
}
