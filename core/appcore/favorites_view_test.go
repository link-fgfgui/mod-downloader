// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/storage"
)

func favoriteListFixture(id, name string, pinned bool, sortOrder int, createdAt int64, minecraftVersion, loader string) storage.FavoriteList {
	return storage.FavoriteList{
		ID:               id,
		Name:             name,
		Pinned:           pinned,
		SortOrder:        sortOrder,
		CreatedAt:        createdAt,
		MinecraftVersion: minecraftVersion,
		ModLoader:        loader,
	}
}

func TestBuildFavoriteOverviewFiltersScope(t *testing.T) {
	lists := []storage.FavoriteList{
		favoriteListFixture("global", "Global", false, 0, 1, "", ""),
		favoriteListFixture("match", "Match", false, 0, 2, "1.20.1", "Fabric"),
		favoriteListFixture("other-version", "Other", false, 0, 3, "1.21.1", "Fabric"),
		favoriteListFixture("other-loader", "Other", false, 0, 4, "1.20.1", "Forge"),
	}

	overview := BuildFavoriteOverview(lists, "1.20.1", "fabric")
	if len(overview.Pinned) != 0 {
		t.Fatalf("pinned = %+v, want none", overview.Pinned)
	}
	if len(overview.Ungrouped) != 2 {
		t.Fatalf("ungrouped = %+v, want global + match", overview.Ungrouped)
	}
	if overview.Ungrouped[0].ID != "global" || overview.Ungrouped[1].ID != "match" {
		t.Fatalf("ungrouped order = %q,%q", overview.Ungrouped[0].ID, overview.Ungrouped[1].ID)
	}

	// A missing scope only keeps the unscoped list.
	empty := BuildFavoriteOverview(lists, "", "")
	if len(empty.Ungrouped) != 1 || empty.Ungrouped[0].ID != "global" {
		t.Fatalf("empty scope = %+v, want only the global list", empty.Ungrouped)
	}
}

func TestBuildFavoriteOverviewOrdersByPinnedThenSortThenCreatedThenName(t *testing.T) {
	lists := []storage.FavoriteList{
		favoriteListFixture("u-sort2", "A", false, 2, 1, "", ""),
		favoriteListFixture("u-sort1", "B", false, 1, 5, "", ""),
		favoriteListFixture("u-created-old", "Same", false, 3, 10, "", ""),
		favoriteListFixture("u-created-new", "Same", false, 3, 20, "", ""),
		favoriteListFixture("u-name", "Alpha", false, 3, 20, "", ""),
		favoriteListFixture("pinned", "Z", true, 99, 99, "", ""),
	}

	overview := BuildFavoriteOverview(lists, "", "")
	if len(overview.Pinned) != 1 || overview.Pinned[0].ID != "pinned" {
		t.Fatalf("pinned = %+v", overview.Pinned)
	}
	want := []string{"u-sort1", "u-sort2", "u-created-old", "u-name", "u-created-new"}
	if len(overview.Ungrouped) != len(want) {
		t.Fatalf("ungrouped = %+v", overview.Ungrouped)
	}
	for i, id := range want {
		if overview.Ungrouped[i].ID != id {
			t.Fatalf("ungrouped[%d] = %q, want %q (full: %+v)", i, overview.Ungrouped[i].ID, id, overview.Ungrouped)
		}
	}
}

func TestFavoriteListMatchesScope(t *testing.T) {
	unscoped := favoriteListFixture("a", "A", false, 0, 0, "", "")
	if !FavoriteListMatchesScope(unscoped, "1.20.1", "fabric") || !FavoriteListMatchesScope(unscoped, "", "") {
		t.Fatal("an unscoped list must match every scope")
	}
	scoped := favoriteListFixture("b", "B", false, 0, 0, "1.20.1", "Fabric")
	if !FavoriteListMatchesScope(scoped, "1.20.1", "fabric") {
		t.Fatal("normalized tuple must match")
	}
	if FavoriteListMatchesScope(scoped, "1.20.2", "fabric") || FavoriteListMatchesScope(scoped, "1.20.1", "forge") {
		t.Fatal("mismatched tuple must not match")
	}
	if FavoriteListMatchesScope(scoped, "", "") {
		t.Fatal("a scoped list must not match an empty scope")
	}
}

func TestFilterFavoriteContents(t *testing.T) {
	contents := storage.FavoriteListContents{
		ListID: "list",
		Mods: []storage.FavoriteModEntry{
			{FavoriteMod: storage.FavoriteMod{Platform: "modrinth", ModID: "a", MinecraftVersion: "1.20.1", ModLoader: "Fabric"}},
			{FavoriteMod: storage.FavoriteMod{Platform: "modrinth", ModID: "b", MinecraftVersion: "1.21.1", ModLoader: "Fabric"}},
			{FavoriteMod: storage.FavoriteMod{Platform: "modrinth", ModID: "c", MinecraftVersion: "1.20.1", ModLoader: "Forge"}},
			{FavoriteMod: storage.FavoriteMod{Platform: "modrinth", ModID: "d", MinecraftVersion: "1.20.1", ModLoader: "fabric"}},
		},
		Refs: []storage.FavoriteListRef{{ParentListID: "list", ChildListID: "child"}},
	}

	filtered := FilterFavoriteContents(contents, "1.20.1", "fabric")
	if len(filtered.Mods) != 2 || filtered.Mods[0].ModID != "a" || filtered.Mods[1].ModID != "d" {
		t.Fatalf("filtered mods = %+v", filtered.Mods)
	}
	if len(filtered.Refs) != 1 {
		t.Fatalf("refs must be preserved: %+v", filtered.Refs)
	}

	empty := FilterFavoriteContents(contents, "", "")
	if len(empty.Mods) != 0 {
		t.Fatalf("empty scope kept %d mods", len(empty.Mods))
	}
}
