// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package global

import (
	"sync"
	"testing"

	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

func TestVersionGettersReturnSnapshots(t *testing.T) {
	dir := t.TempDir()
	version := structs.VersionInfo{
		ID:               "inst",
		Name:             "Original",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		Mods: []structs.ModInfo{{
			ID:         "sodium",
			Categories: []string{"performance"},
		}},
	}
	SetMinecraftDir(dir)
	SetVersionsForDir(dir, []structs.VersionInfo{version})
	SetSelectedVersion(version)
	t.Cleanup(func() {
		SetMinecraftDir("")
		InvalidateVersions()
	})

	versions, ok := GetVersionsForDir(dir)
	if !ok {
		t.Fatal("GetVersionsForDir missed the cache")
	}
	versions[0].Name = "Mutated"
	versions[0].Mods[0].ID = "mutated"
	versions[0].Mods[0].Categories[0] = "mutated"

	selected := GetSelectedVersion()
	if selected.Name != "Original" || selected.Mods[0].ID != "sodium" || selected.Mods[0].Categories[0] != "performance" {
		t.Fatalf("selected snapshot was mutated: %#v", selected)
	}
	byKey, ok := GetVersionByKey("inst")
	if !ok || byKey.Mods[0].ID != "sodium" {
		t.Fatalf("version by key = %#v, %v", byKey, ok)
	}
}

func TestVersionsConcurrentReadWrite(t *testing.T) {
	dir := t.TempDir()
	SetMinecraftDir(dir)
	t.Cleanup(func() {
		SetMinecraftDir("")
		InvalidateVersions()
	})

	versions := []structs.VersionInfo{{
		ID:               "inst",
		Name:             "Inst",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		Mods:             []structs.ModInfo{{ID: "sodium"}},
	}}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 200 {
			SetVersionsForDir(dir, versions)
			SetSelectedVersion(versions[0])
		}
	}()
	go func() {
		defer wg.Done()
		for range 200 {
			_ = GetVersions()
			_ = GetSelectedVersion()
			_, _ = GetVersionsForDir(dir)
			_, _ = GetVersionByKey("inst")
		}
	}()
	wg.Wait()
}

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
