// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"reflect"
	"strings"
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/global"
	mcstructs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

func TestBuildLocalModGroupsAggregatesByFileName(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{ID: "sodium", FileName: "sodium.jar", Path: "a/sodium.jar", Enabled: true},
		{ID: "sodium-extra", FileName: "sodium.jar", Path: "b/sodium.jar", Enabled: false},
		{ID: "lithium", FileName: "lithium.jar", Path: "lithium.jar", Enabled: true},
	}

	groups := BuildLocalModGroups(mods, "")
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	if groups[0].Primary.ID != "sodium" {
		t.Fatalf("primary = %q, want sodium (first seen)", groups[0].Primary.ID)
	}
	if len(groups[0].Strong) != 2 {
		t.Fatalf("strong count = %d, want 2", len(groups[0].Strong))
	}
	if groups[1].Primary.ID != "lithium" {
		t.Fatalf("second group primary = %q, want lithium", groups[1].Primary.ID)
	}
}

func TestBuildLocalModGroupsFallsBackToPathThenID(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{ID: "a", Path: "a.jar"},
		{ID: "b", Path: "a.jar"},
		{ID: "c"},
		{ID: "c"},
	}

	groups := BuildLocalModGroups(mods, "")
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	if len(groups[0].Strong) != 2 || groups[0].Primary.ID != "a" {
		t.Fatalf("path group = %+v, want a+b", groups[0].Strong)
	}
	if len(groups[1].Strong) != 2 || groups[1].Primary.ID != "c" {
		t.Fatalf("id group = %+v, want c+c", groups[1].Strong)
	}
}

func TestBuildLocalModGroupsDeduplicatesJIJCaseInsensitively(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{
			ID:       "sodium",
			FileName: "sodium.jar",
			JIJMods: []mcstructs.JIJModInfo{
				{ID: "embeddium", Name: "Embeddium"},
				{ID: " Embeddium ", Name: "Embeddium dup"},
				{ID: "   ", Name: "blank"},
			},
		},
		{
			ID:       "sodium-extra",
			FileName: "sodium.jar",
			JIJMods:  []mcstructs.JIJModInfo{{ID: "EMBEDDIUM", Name: "dup again"}},
		},
	}

	groups := BuildLocalModGroups(mods, "")
	if len(groups) != 1 {
		t.Fatalf("group count = %d, want 1", len(groups))
	}
	if len(groups[0].JIJ) != 1 {
		t.Fatalf("jij = %+v, want one de-duplicated entry", groups[0].JIJ)
	}
	if groups[0].JIJ[0].ID != "embeddium" {
		t.Fatalf("jij id = %q, want first spelling", groups[0].JIJ[0].ID)
	}
}

func TestBuildLocalModGroupsCompatibility(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{ID: "fabric-mod", FileName: "a.jar", Loaders: []string{"Fabric", " quilt "}},
		{ID: "forge-mod", FileName: "b.jar", Loaders: []string{"Forge"}},
		{ID: "unknown", FileName: "c.jar"},
	}

	groups := BuildLocalModGroups(mods, "Fabric")
	if groups[0].Incompatible {
		t.Fatalf("fabric mod flagged incompatible: %+v", groups[0])
	}
	if !groups[1].Incompatible {
		t.Fatalf("forge mod not flagged incompatible: %+v", groups[1])
	}
	if groups[2].Incompatible {
		t.Fatalf("mod without loader declarations flagged incompatible: %+v", groups[2])
	}
	if want := []string{"fabric", "quilt"}; !reflect.DeepEqual(groups[0].Loaders, want) {
		t.Fatalf("loaders = %v, want %v", groups[0].Loaders, want)
	}

	unselected := BuildLocalModGroups(mods, "")
	for _, group := range unselected {
		if group.Incompatible {
			t.Fatalf("empty selected loader flagged %q incompatible", group.Primary.ID)
		}
	}
}

func TestBuildLocalModGroupsDisplayCategoriesAndSearchText(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{
			ID:          "sodium",
			Name:        "Sodium",
			FileName:    "sodium.jar",
			Path:        "/instances/1.20.1-fabric/mods/sodium.jar",
			OnlineName:  "Sodium Renderer",
			Description: "Fast renderer",
			Categories:  []string{" Optimization ", "optimization", ""},
			JIJMods:     []mcstructs.JIJModInfo{{ID: "embeddium", Name: "Embeddium"}},
		},
	}

	groups := BuildLocalModGroups(mods, "Fabric")
	group := groups[0]
	if group.DisplayName != "Sodium Renderer" {
		t.Fatalf("display name = %q, want online name", group.DisplayName)
	}
	if want := []string{"optimization"}; !reflect.DeepEqual(group.Categories, want) {
		t.Fatalf("categories = %v, want %v", group.Categories, want)
	}
	for _, needle := range []string{"sodium renderer", "fast renderer", "optimization", "embeddium"} {
		if !strings.Contains(group.SearchText, needle) {
			t.Fatalf("search text missing %q: %q", needle, group.SearchText)
		}
	}
	if strings.Contains(group.SearchText, "1.20.1-fabric") {
		t.Fatalf("search text must not index instance paths: %q", group.SearchText)
	}
}

func TestBuildLocalModGroupsDisplayNameFallbacks(t *testing.T) {
	groups := BuildLocalModGroups([]mcstructs.ModInfo{
		{ID: "a", Name: "Name A", FileName: "a.jar"},
		{ID: "b", FileName: "b.jar"},
	}, "")
	if groups[0].DisplayName != "Name A" {
		t.Fatalf("display name = %q, want name fallback", groups[0].DisplayName)
	}
	if groups[1].DisplayName != "b" {
		t.Fatalf("display name = %q, want id fallback", groups[1].DisplayName)
	}
}

func TestGetLocalModGroupsUsesSelectedVersion(t *testing.T) {
	global.SetMinecraftDir("")
	global.SetVersions(nil)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.SetVersions(nil)
	})

	version := mcstructs.VersionInfo{
		ID:               "inst",
		Name:             "inst",
		MinecraftVersion: "1.20.1",
		ModLoader:        "Fabric",
		Mods: []mcstructs.ModInfo{
			{ID: "sodium", FileName: "sodium.jar", Loaders: []string{"Fabric"}},
			{ID: "forge-mod", FileName: "forge.jar", Loaders: []string{"Forge"}},
		},
	}
	global.SetVersions([]mcstructs.VersionInfo{version})
	global.SetSelectedVersion(version)

	service := New(Options{})
	groups := service.GetLocalModGroups("inst", "")
	if len(groups) != 2 {
		t.Fatalf("group count = %d, want 2", len(groups))
	}
	if groups[0].Incompatible {
		t.Fatalf("fabric mod flagged incompatible: %+v", groups[0])
	}
	if !groups[1].Incompatible {
		t.Fatalf("selected loader fallback did not mark the forge mod incompatible: %+v", groups[1])
	}
}

func TestGetLocalModGroupsFallsBackForStaleVersionKey(t *testing.T) {
	global.SetMinecraftDir("")
	global.SetVersions(nil)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.SetVersions(nil)
	})

	version := mcstructs.VersionInfo{
		ID:        "inst",
		Name:      "inst",
		ModLoader: "Fabric",
		Mods:      []mcstructs.ModInfo{{ID: "sodium", FileName: "sodium.jar"}},
	}
	global.SetVersions([]mcstructs.VersionInfo{version})
	global.SetSelectedVersion(version)

	groups := New(Options{}).GetLocalModGroups("removed-instance", "")
	if len(groups) != 1 || groups[0].Primary.ID != "sodium" {
		t.Fatalf("stale key should fall back to the selected instance, got %+v", groups)
	}
}
