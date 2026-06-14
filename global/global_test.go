package global

import (
	"testing"

	structs "mod-downloader/structs/minecraft"
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
