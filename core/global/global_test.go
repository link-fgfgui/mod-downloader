package global

import (
	"testing"

	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func TestGetSelectedVersionReturnsEmptyWhenSelectedKeyMissing(t *testing.T) {
	dir := t.TempDir()
	SetMinecraftDir(dir)
	SetVersionsForDir(dir, []structs.VersionInfo{{
		ID:               "selected",
		Name:             "selected",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}})
	SetSelectedVersion(structs.VersionInfo{
		ID:               "selected",
		Name:             "selected",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	})
	SetVersionsForDir(dir, []structs.VersionInfo{{
		ID:               "other",
		Name:             "other",
		MinecraftVersion: "1.20.1",
		ModLoader:        "forge",
	}})
	t.Cleanup(func() {
		SetMinecraftDir("")
		InvalidateVersions()
	})

	got := GetSelectedVersion()
	if got.ID != "" || got.Name != "" || got.MinecraftVersion != "" || got.ModLoader != "" {
		t.Fatalf("GetSelectedVersion() = %#v, want empty", got)
	}
}

func TestSelectedVersionKeyPrefersIDOverDisplayName(t *testing.T) {
	dir := t.TempDir()
	SetMinecraftDir(dir)
	SetVersionsForDir(dir, []structs.VersionInfo{{
		ID:               "instance-folder",
		Name:             "Display Name",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}})
	SetSelectedVersion(structs.VersionInfo{
		ID:               "instance-folder",
		Name:             "Display Name",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	})
	t.Cleanup(func() {
		SetMinecraftDir("")
		InvalidateVersions()
	})

	got := GetSelectedVersion()
	if got.ID != "instance-folder" || got.Name != "Display Name" {
		t.Fatalf("GetSelectedVersion() = %#v, want selected by ID", got)
	}
}

func TestSetSelectedVersionUpdatesCachedSnapshot(t *testing.T) {
	dir := t.TempDir()
	version := structs.VersionInfo{
		ID:               "instance-folder",
		Name:             "Display Name",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}
	SetMinecraftDir(dir)
	SetVersionsForDir(dir, []structs.VersionInfo{version})
	version.Mods = []structs.ModInfo{{ID: "new-mod"}}
	SetSelectedVersion(version)
	t.Cleanup(func() {
		SetMinecraftDir("")
		InvalidateVersions()
	})

	selected := GetSelectedVersion()
	if len(selected.Mods) != 1 || selected.Mods[0].ID != "new-mod" {
		t.Fatalf("GetSelectedVersion().Mods = %#v, want refreshed mod", selected.Mods)
	}
	versions, ok := GetVersionsForDir(dir)
	if !ok || len(versions) != 1 || len(versions[0].Mods) != 1 || versions[0].Mods[0].ID != "new-mod" {
		t.Fatalf("GetVersionsForDir() = %#v, %v, want refreshed mod", versions, ok)
	}
}
