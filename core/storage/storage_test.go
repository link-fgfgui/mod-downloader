package storage

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/link-fgfgui/mod-downloader-core/models"
)

func openTestDB(t *testing.T) string {
	t.Helper()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		Close()
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})
	path := filepath.Join(tmp, metadataCacheFileName)
	if err := OpenAt(path); err != nil {
		t.Fatal(err)
	}
	return path
}

func reopenTestDB(t *testing.T, path string) {
	t.Helper()
	Close()
	if err := OpenAt(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultCachePathUsesTempDir(t *testing.T) {
	path, err := DefaultCachePath()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := filepath.Clean(os.TempDir())
	if rel, err := filepath.Rel(tempDir, path); err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		t.Fatalf("DefaultCachePath() = %q, want under %q", path, tempDir)
	}
	if filepath.Base(path) != metadataCacheFileName {
		t.Fatalf("DefaultCachePath() base = %q, want %q", filepath.Base(path), metadataCacheFileName)
	}
}

func TestOpenAtCreatesParentDirectoryOnClose(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "cache", metadataCacheFileName)
	if err := OpenAt(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(path)); err != nil {
		t.Fatalf("parent after open: %v", err)
	}
	if _, err := os.Stat(UserDataPathForCachePath(path)); err != nil {
		t.Fatalf("user database after open: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cache file before close error = %v, want not exist", err)
	}
	Close()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("cache file after close: %v", err)
	}
}

func TestUsageStatsAccumulateAndPersist(t *testing.T) {
	path := openTestDB(t)

	if got := GetUsageStats(); got != (UsageStats{}) {
		t.Fatalf("initial stats = %#v, want zero", got)
	}
	for key, delta := range map[UsageStatKey]int64{
		UsageStatDownloadsCompleted: 3,
		UsageStatModsEnabled:        2,
		UsageStatModsDisabled:       4,
		UsageStatModsDeleted:        1,
		UsageStatFavoritesAdded:     5,
		UsageStatPackwizExports:     2,
	} {
		if err := IncrementUsageStat(key, delta); err != nil {
			t.Fatalf("IncrementUsageStat(%q, %d): %v", key, delta, err)
		}
	}
	if err := IncrementUsageStat(UsageStatDownloadsCompleted, 2); err != nil {
		t.Fatal(err)
	}
	if err := IncrementUsageStat(UsageStatKey("unknown"), 99); err != nil {
		t.Fatal(err)
	}
	if err := IncrementUsageStat(UsageStatModsDeleted, -1); err != nil {
		t.Fatal(err)
	}

	want := UsageStats{
		DownloadsCompleted: 5,
		ModsEnabled:        2,
		ModsDisabled:       4,
		ModsDeleted:        1,
		FavoritesAdded:     5,
		PackwizExports:     2,
	}
	if got := GetUsageStats(); got != want {
		t.Fatalf("stats = %#v, want %#v", got, want)
	}

	reopenTestDB(t, path)
	if got := GetUsageStats(); got != want {
		t.Fatalf("stats after reopen = %#v, want %#v", got, want)
	}
	var unknown int
	if err := userDB.db.QueryRow(`SELECT COUNT(*) FROM usage_stats WHERE key = 'unknown'`).Scan(&unknown); err != nil || unknown != 0 {
		t.Fatalf("unknown usage stat rows = %d, %v", unknown, err)
	}
}

func TestLocalMetadataMissesNormalizeAndPersist(t *testing.T) {
	path := openTestDB(t)
	if err := SetLocalMetadataMisses(" Modrinth ", []string{" ABC123 ", "abc123", ""}, 1234); err != nil {
		t.Fatal(err)
	}
	if got, ok := GetLocalMetadataMissCheckedAt("modrinth", "abc123"); !ok || got != 1234 {
		t.Fatalf("metadata miss before reopen = %d, %v", got, ok)
	}

	reopenTestDB(t, path)
	if got, ok := GetLocalMetadataMissCheckedAt("MODRINTH", " ABC123 "); !ok || got != 1234 {
		t.Fatalf("metadata miss after reopen = %d, %v", got, ok)
	}
	if _, ok := GetLocalMetadataMissCheckedAt("modrinth", "missing"); ok {
		t.Fatal("unexpected metadata miss for unknown identity")
	}
}

func TestMinecraftReleaseVersionsNormalizeAndPersist(t *testing.T) {
	path := openTestDB(t)
	if err := SetMinecraftReleaseVersions([]string{" 1.21.1 ", "1.20.1", "1.21.1", ""}, 5678); err != nil {
		t.Fatal(err)
	}
	versions, updatedAt := GetMinecraftReleaseVersions()
	if updatedAt != 5678 || len(versions) != 2 || versions[0] != "1.21.1" || versions[1] != "1.20.1" {
		t.Fatalf("release cache before reopen = %#v, %d", versions, updatedAt)
	}

	reopenTestDB(t, path)
	versions, updatedAt = GetMinecraftReleaseVersions()
	if updatedAt != 5678 || len(versions) != 2 || versions[0] != "1.21.1" || versions[1] != "1.20.1" {
		t.Fatalf("release cache after reopen = %#v, %d", versions, updatedAt)
	}
	versions[0] = "mutated"
	again, _ := GetMinecraftReleaseVersions()
	if again[0] != "1.21.1" {
		t.Fatalf("release cache returned shared slice: %#v", again)
	}
}

func TestCachePlatformVersionsAndDependencies(t *testing.T) {
	path := openTestDB(t)

	if err := UpsertModPlatform(models.ModProject{
		Platform:  "Modrinth",
		ProjectID: "sodium",
		Slug:      "sodium-slug",
		Title:     "Sodium",
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetPlatformVersionSnapshot("Modrinth", "sodium", []models.ModVersion{
		{
			VersionID:    "v1",
			Name:         "1.0",
			SHA1:         "abc",
			PublishedAt:  10,
			GameVersions: []string{"1.21.1"},
			Loaders:      []string{"neoforge"},
			Dependencies: []models.ModDependency{{
				DependencyProjectID: "fabric-api",
				DependencyType:      "required",
			}},
		},
	}, 100, []ModPlatformVersionScope{{
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
	}}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cache file before close error = %v, want not exist", err)
	}

	reopenTestDB(t, path)

	if p, ok := GetModPlatformBySlug("Modrinth", "sodium-slug"); !ok || p.ProjectID != "sodium" {
		t.Fatalf("slug lookup = %#v, %v", p, ok)
	}
	if ts, ok := GetPlatformVersionScopeUpdatedAt("Modrinth", "sodium", ModPlatformVersionScope{MinecraftVersion: "1.21.1", ModLoader: "neoforge"}); !ok || ts != 100 {
		t.Fatalf("scope timestamp = %d, %v", ts, ok)
	}

	versions, err := GetPlatformVersions("Modrinth", "sodium")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].VersionID != "v1" || len(versions[0].Dependencies) != 1 {
		t.Fatalf("versions = %#v", versions)
	}
	if versions[0].Dependencies[0].PlatformVersionID != versions[0].ID {
		t.Fatalf("dependency platform version id = %q, want %q", versions[0].Dependencies[0].PlatformVersionID, versions[0].ID)
	}

	deps, err := GetVersionDependencies(versions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(deps) != 1 || deps[0].DependencyProjectID != "fabric-api" {
		t.Fatalf("dependencies = %#v", deps)
	}
}

func TestPlatformCacheKeysAreCaseInsensitive(t *testing.T) {
	openTestDB(t)

	if err := UpsertModPlatform(models.ModProject{
		Platform:  "Modrinth",
		ProjectID: "sodium",
		Slug:      "sodium-slug",
		Title:     "Sodium",
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetPlatformVersionSnapshot("Modrinth", "sodium", []models.ModVersion{
		{
			VersionID:   "v1",
			SHA1:        "abc",
			PublishedAt: 10,
		},
	}, 100, []ModPlatformVersionScope{{
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
	}}); err != nil {
		t.Fatal(err)
	}

	if p, ok := GetModPlatform("modrinth", "sodium"); !ok || p.Title != "Sodium" || p.Platform != "Modrinth" {
		t.Fatalf("GetModPlatform lower-case = %#v, %v", p, ok)
	}
	if p, ok := GetModPlatformBySlug("modrinth", "sodium-slug"); !ok || p.ProjectID != "sodium" {
		t.Fatalf("GetModPlatformBySlug lower-case = %#v, %v", p, ok)
	}
	if versions, err := GetPlatformVersions("modrinth", "sodium"); err != nil || len(versions) != 1 || versions[0].VersionID != "v1" {
		t.Fatalf("GetPlatformVersions lower-case = %#v, %v", versions, err)
	}
	if ts, ok := GetPlatformVersionScopeUpdatedAt("modrinth", "sodium", ModPlatformVersionScope{MinecraftVersion: "1.21.1", ModLoader: "neoforge"}); !ok || ts != 100 {
		t.Fatalf("GetPlatformVersionScopeUpdatedAt lower-case = %d, %v", ts, ok)
	}
	if version, ok := GetLatestProjectBySHA1("modrinth", "abc"); !ok || version.VersionID != "v1" || version.Platform != "Modrinth" {
		t.Fatalf("GetLatestProjectBySHA1 lower-case = %#v, %v", version, ok)
	}
}

func TestCachePinnedMods(t *testing.T) {
	path := openTestDB(t)

	if err := UpsertPinnedMod(PinnedMod{
		Platform:         "Modrinth",
		ModID:            "Sodium",
		VersionID:        "v1",
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
	}); err != nil {
		t.Fatal(err)
	}

	reopenTestDB(t, path)

	if pin, ok := GetPinnedMod("modrinth", "sodium", "1.21.1", "neoforge"); !ok || pin.VersionID != "v1" {
		t.Fatalf("pin = %#v, %v", pin, ok)
	}
}

func TestListPinnedMods(t *testing.T) {
	path := openTestDB(t)

	if pins := ListPinnedMods(); len(pins) != 0 {
		t.Fatalf("empty list = %#v, want empty", pins)
	}

	inserts := []PinnedMod{
		{Platform: "CurseForge", ModID: "jei", VersionID: "v3", MinecraftVersion: "1.20.1", ModLoader: "forge"},
		{Platform: "Modrinth", ModID: "sodium", VersionID: "v1", MinecraftVersion: "1.21.1", ModLoader: "fabric"},
		{Platform: "Modrinth", ModID: "sodium", VersionID: "v2", MinecraftVersion: "1.21.1", ModLoader: "neoforge"},
	}
	for _, p := range inserts {
		if err := UpsertPinnedMod(p); err != nil {
			t.Fatal(err)
		}
	}

	pins := ListPinnedMods()
	if len(pins) != 3 {
		t.Fatalf("len = %d, want 3", len(pins))
	}

	want := []PinnedMod{
		{Platform: "curseforge", ModID: "jei", VersionID: "v3", MinecraftVersion: "1.20.1", ModLoader: "forge"},
		{Platform: "modrinth", ModID: "sodium", VersionID: "v1", MinecraftVersion: "1.21.1", ModLoader: "fabric"},
		{Platform: "modrinth", ModID: "sodium", VersionID: "v2", MinecraftVersion: "1.21.1", ModLoader: "neoforge"},
	}
	for i, p := range pins {
		if p.Platform != want[i].Platform || p.ModID != want[i].ModID || p.VersionID != want[i].VersionID || p.MinecraftVersion != want[i].MinecraftVersion || p.ModLoader != want[i].ModLoader {
			t.Fatalf("pin[%d] = %#v, want %#v", i, p, want[i])
		}
	}

	pins[0].VersionID = "mutated"
	if got, ok := GetPinnedMod("curseforge", "jei", "1.20.1", "forge"); !ok || got.VersionID != "v3" {
		t.Fatalf("returned value was not a copy: %#v", got)
	}

	reopenTestDB(t, path)

	if len(ListPinnedMods()) != 3 {
		t.Fatalf("after reopen len = %d, want 3", len(ListPinnedMods()))
	}
}

func TestFavoriteListsAndModsPersist(t *testing.T) {
	path := openTestDB(t)

	first, err := CreateFavoriteListForScope(" Utilities ", "1.21.1", "Fabric")
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateFavoriteList("Performance")
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == "" || second.ID == "" || first.ID == second.ID {
		t.Fatalf("list IDs = %q, %q", first.ID, second.ID)
	}
	if first.Name != "Utilities" || second.SortOrder <= first.SortOrder {
		t.Fatalf("lists = %#v, %#v", first, second)
	}

	renamed, ok, err := UpdateFavoriteListName(first.ID, "Utility Mods")
	if err != nil || !ok || renamed.Name != "Utility Mods" {
		t.Fatalf("rename = %#v, %v, %v", renamed, ok, err)
	}

	item, ok, err := UpsertFavoriteMod(FavoriteMod{
		ListID:           first.ID,
		Platform:         "Modrinth",
		ModID:            "Sodium",
		VersionID:        "v1",
		MinecraftVersion: "1.21.1",
		ModLoader:        "Fabric",
		Title:            "Sodium",
		Slug:             "sodium",
		IconURL:          "https://example.com/icon.png",
		Categories:       []string{"Performance", "performance", ""},
	})
	if err != nil || !ok {
		t.Fatalf("upsert favorite mod = %#v, %v, %v", item, ok, err)
	}
	if item.Platform != "modrinth" || item.ModID != "sodium" || item.ModLoader != "fabric" {
		t.Fatalf("normalized item = %#v", item)
	}
	if len(item.Categories) != 1 || item.Categories[0] != "performance" {
		t.Fatalf("categories = %#v", item.Categories)
	}

	updated, ok, err := UpsertFavoriteMod(FavoriteMod{
		ListID:           first.ID,
		Platform:         "modrinth",
		ModID:            "sodium",
		VersionID:        "v2",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		Title:            "Sodium Updated",
	})
	if err != nil || !ok {
		t.Fatalf("update favorite mod = %#v, %v, %v", updated, ok, err)
	}
	if updated.ID != item.ID || updated.VersionID != "v2" || updated.Title != "Sodium Updated" {
		t.Fatalf("updated item = %#v, original = %#v", updated, item)
	}

	reopenTestDB(t, path)

	lists := ListFavoriteLists()
	if len(lists) != 2 {
		t.Fatalf("lists after reopen = %#v", lists)
	}
	if lists[0].ID != first.ID || lists[0].Name != "Utility Mods" || lists[1].ID != second.ID {
		t.Fatalf("list order after reopen = %#v", lists)
	}

	mods := ListFavoriteMods(first.ID)
	if len(mods) != 1 || mods[0].ID != item.ID || mods[0].VersionID != "v2" || mods[0].Title != "Sodium Updated" {
		t.Fatalf("mods after reopen = %#v", mods)
	}

	mods[0].Categories = []string{"mutated"}
	if again := ListFavoriteMods(first.ID); len(again) != 1 || len(again[0].Categories) != 0 {
		t.Fatalf("returned favorite mods are not copies: %#v", again)
	}
}

func TestFavoriteListDeleteCascadesItems(t *testing.T) {
	openTestDB(t)

	list, err := CreateFavoriteList("To Delete")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := UpsertFavoriteMod(FavoriteMod{
		ListID:   list.ID,
		Platform: "modrinth",
		ModID:    "sodium",
	}); err != nil || !ok {
		t.Fatalf("upsert favorite mod: %v, %v", ok, err)
	}
	if deleted, err := DeleteFavoriteList(list.ID); err != nil || !deleted {
		t.Fatalf("delete favorite list = %v, %v", deleted, err)
	}
	if lists := ListFavoriteLists(); len(lists) != 0 {
		t.Fatalf("lists after delete = %#v", lists)
	}
	if mods := ListFavoriteMods(list.ID); len(mods) != 0 {
		t.Fatalf("mods after delete = %#v", mods)
	}
}

func TestFavoriteModRequiresExistingList(t *testing.T) {
	openTestDB(t)

	mod, ok, err := UpsertFavoriteMod(FavoriteMod{
		ListID:   "missing",
		Platform: "modrinth",
		ModID:    "sodium",
	})
	if err != nil || ok || mod.ID != "" {
		t.Fatalf("upsert without list = %#v, %v, %v", mod, ok, err)
	}
}

func TestSQLiteSchemaV3DiscardsExistingFavorites(t *testing.T) {
	tmp := t.TempDir()
	cachePath := filepath.Join(tmp, metadataCacheFileName)
	userPath := UserDataPathForCachePath(cachePath)
	db, err := sql.Open(sqliteDriverName, userPath)
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY, applied_at INTEGER NOT NULL)`,
		`CREATE TABLE pinned_mods (
			id TEXT PRIMARY KEY,
			platform TEXT NOT NULL,
			mod_id TEXT NOT NULL,
			version_id TEXT NOT NULL,
			minecraft_version TEXT NOT NULL,
			mod_loader TEXT NOT NULL,
			UNIQUE(platform, mod_id, minecraft_version, mod_loader)
		)`,
		`CREATE TABLE usage_stats (
			key TEXT PRIMARY KEY,
			value INTEGER NOT NULL DEFAULT 0 CHECK(value >= 0),
			updated_at INTEGER NOT NULL
		)`,
		`CREATE TABLE favorite_lists (
			id TEXT PRIMARY KEY,
			group_id TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL,
			minecraft_version TEXT NOT NULL DEFAULT '',
			mod_loader TEXT NOT NULL DEFAULT '',
			icon_kind TEXT NOT NULL DEFAULT '',
			icon_value TEXT NOT NULL DEFAULT '',
			icon_url TEXT NOT NULL DEFAULT '',
			pinned INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			system INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE favorite_mods (
			id TEXT PRIMARY KEY,
			list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			platform TEXT NOT NULL,
			mod_id TEXT NOT NULL,
			version_id TEXT NOT NULL DEFAULT '',
			minecraft_version TEXT NOT NULL DEFAULT '',
			mod_loader TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			slug TEXT NOT NULL DEFAULT '',
			icon_url TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			categories_json TEXT NOT NULL DEFAULT '[]',
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			UNIQUE(list_id, platform, mod_id, minecraft_version, mod_loader)
		)`,
		`CREATE TABLE favorite_groups (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL
		)`,
		`CREATE TABLE favorite_list_refs (
			id TEXT PRIMARY KEY,
			parent_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			child_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sort_order INTEGER NOT NULL,
			UNIQUE(parent_list_id, child_list_id),
			CHECK(parent_list_id <> child_list_id)
		)`,
		`INSERT INTO schema_migrations(version, applied_at) VALUES (1, 100), (2, 101)`,
		`INSERT INTO pinned_mods(id, platform, mod_id, version_id, minecraft_version, mod_loader)
			 VALUES ('pin', 'modrinth', 'sodium', 'v2', '1.21.1', 'fabric')`,
		`INSERT INTO usage_stats(key, value, updated_at) VALUES ('downloads_completed', 7, 100)`,
		`INSERT INTO favorite_lists(id, name, minecraft_version, mod_loader, created_at, updated_at, sort_order, system)
			 VALUES ('legacy-list', 'Legacy', '1.20.1', 'fabric', 10, 11, 1, 0)`,
		`INSERT INTO favorite_mods(id, list_id, platform, mod_id, version_id, minecraft_version, mod_loader, title, categories_json, created_at, updated_at)
			 VALUES ('legacy-mod', 'legacy-list', 'modrinth', 'sodium', 'v1', '1.20.1', 'fabric', 'Sodium', '["performance"]', 12, 13)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	if err := OpenAt(cachePath); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(Close)

	if lists := ListFavoriteLists(); len(lists) != 0 {
		t.Fatalf("legacy lists were not discarded = %#v", lists)
	}
	if groups := ListFavoriteGroups(); len(groups) != 0 {
		t.Fatalf("legacy groups were not discarded = %#v", groups)
	}
	if pin, ok := GetPinnedMod("modrinth", "sodium", "1.21.1", "fabric"); !ok || pin.VersionID != "v2" {
		t.Fatalf("pin was not preserved = %#v, %v", pin, ok)
	}
	if stats := GetUsageStats(); stats.DownloadsCompleted != 7 {
		t.Fatalf("usage stats were not preserved = %#v", stats)
	}
	var version int
	if err := userDB.db.QueryRow(`SELECT version FROM schema_migrations WHERE version = 3`).Scan(&version); err != nil || version != 3 {
		t.Fatalf("schema version 3 = %d, %v", version, err)
	}
}

func TestFavoriteGroupsAndListMetadataPersist(t *testing.T) {
	path := openTestDB(t)

	first, err := CreateFavoriteGroup(" Utilities ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreateFavoriteGroup("Performance")
	if err != nil {
		t.Fatal(err)
	}
	if first.Name != "Utilities" || second.SortOrder <= first.SortOrder {
		t.Fatalf("groups = %#v, %#v", first, second)
	}
	if err := ReorderFavoriteGroups([]string{second.ID, first.ID}); err != nil {
		t.Fatal(err)
	}
	renamed, ok, err := UpdateFavoriteGroupName(first.ID, "Utility Mods")
	if err != nil || !ok || renamed.Name != "Utility Mods" {
		t.Fatalf("rename group = %#v, %v, %v", renamed, ok, err)
	}

	listA, err := CreateFavoriteList("A")
	if err != nil {
		t.Fatal(err)
	}
	listB, err := CreateFavoriteList("B")
	if err != nil {
		t.Fatal(err)
	}
	updated, ok, err := UpdateFavoriteListMetadata(FavoriteList{
		ID:        listB.ID,
		GroupID:   first.ID,
		IconKind:  "MDI",
		IconValue: " mdi-star ",
		IconURL:   " https://example.com/icon.png ",
		Pinned:    true,
	})
	if err != nil || !ok {
		t.Fatalf("update list metadata = %#v, %v, %v", updated, ok, err)
	}
	if updated.GroupID != first.ID || updated.IconKind != "mdi" || updated.IconValue != "mdi-star" || updated.IconURL != "https://example.com/icon.png" || !updated.Pinned {
		t.Fatalf("normalized list metadata = %#v", updated)
	}
	if err := ReorderFavoriteLists([]string{listB.ID, listA.ID}); err != nil {
		t.Fatal(err)
	}

	reopenTestDB(t, path)

	groups := ListFavoriteGroups()
	if len(groups) != 2 || groups[0].ID != second.ID || groups[1].ID != first.ID || groups[1].Name != "Utility Mods" {
		t.Fatalf("groups after reopen = %#v", groups)
	}
	lists := ListFavoriteLists()
	if len(lists) != 2 || lists[0].ID != listB.ID || !lists[0].Pinned || lists[0].GroupID != first.ID {
		t.Fatalf("lists after reopen = %#v", lists)
	}
	if deleted, err := DeleteFavoriteGroup(first.ID); err != nil || !deleted {
		t.Fatalf("delete group = %v, %v", deleted, err)
	}
	lists = ListFavoriteLists()
	for _, list := range lists {
		if list.ID == listB.ID && list.GroupID != "" {
			t.Fatalf("group id not cleared after delete: %#v", list)
		}
	}
}

func TestFavoriteListReferencesAndResolvedContents(t *testing.T) {
	openTestDB(t)

	root, err := CreateFavoriteList("Root")
	if err != nil {
		t.Fatal(err)
	}
	child, err := CreateFavoriteList("Child")
	if err != nil {
		t.Fatal(err)
	}
	grandchild, err := CreateFavoriteList("Grandchild")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := UpsertFavoriteMod(FavoriteMod{ListID: root.ID, Platform: "modrinth", ModID: "sodium", Title: "Direct Sodium"}); err != nil || !ok {
		t.Fatalf("upsert root mod: %v, %v", ok, err)
	}
	if _, ok, err := UpsertFavoriteMod(FavoriteMod{ListID: child.ID, Platform: "modrinth", ModID: "sodium", Title: "Referenced Sodium"}); err != nil || !ok {
		t.Fatalf("upsert duplicate child mod: %v, %v", ok, err)
	}
	if _, ok, err := UpsertFavoriteMod(FavoriteMod{ListID: child.ID, Platform: "modrinth", ModID: "lithium", Title: "Lithium"}); err != nil || !ok {
		t.Fatalf("upsert child mod: %v, %v", ok, err)
	}
	if _, ok, err := UpsertFavoriteMod(FavoriteMod{ListID: grandchild.ID, Platform: "curseforge", ModID: "jei", Title: "JEI"}); err != nil || !ok {
		t.Fatalf("upsert grandchild mod: %v, %v", ok, err)
	}

	if _, ok, err := CreateFavoriteListRef(root.ID, child.ID); err != nil || !ok {
		t.Fatalf("create root child ref: %v, %v", ok, err)
	}
	if _, ok, err := CreateFavoriteListRef(child.ID, grandchild.ID); err != nil || !ok {
		t.Fatalf("create child grandchild ref: %v, %v", ok, err)
	}
	if _, ok, err := CreateFavoriteListRef(root.ID, root.ID); !errors.Is(err, errFavoriteListRefCycle) || ok {
		t.Fatalf("self ref = %v, %v", ok, err)
	}
	if _, ok, err := CreateFavoriteListRef(grandchild.ID, root.ID); !errors.Is(err, errFavoriteListRefCycle) || ok {
		t.Fatalf("cycle ref = %v, %v", ok, err)
	}

	refs := ListFavoriteListRefs(root.ID)
	if len(refs) != 1 || refs[0].ChildListID != child.ID {
		t.Fatalf("root refs = %#v", refs)
	}
	contents := ListFavoriteContents(root.ID)
	if contents.ListID != root.ID || len(contents.Mods) != 3 {
		t.Fatalf("contents = %#v", contents)
	}
	if contents.Mods[0].ModID != "sodium" || contents.Mods[0].Title != "Direct Sodium" || contents.Mods[0].Referenced {
		t.Fatalf("direct duplicate was not preferred: %#v", contents.Mods)
	}
	foundLithium := false
	foundJEI := false
	for _, mod := range contents.Mods {
		if mod.ModID == "lithium" {
			foundLithium = mod.Referenced && mod.SourceListID == child.ID && mod.SourceListName == "Child"
		}
		if mod.ModID == "jei" {
			foundJEI = mod.Referenced && mod.SourceListID == grandchild.ID
		}
	}
	if !foundLithium || !foundJEI {
		t.Fatalf("referenced mods missing metadata: %#v", contents.Mods)
	}
	if deleted, err := DeleteFavoriteListRef(root.ID, child.ID); err != nil || !deleted {
		t.Fatalf("delete ref = %v, %v", deleted, err)
	}
	contents = ListFavoriteContents(root.ID)
	if len(contents.Mods) != 1 || contents.Mods[0].ModID != "sodium" {
		t.Fatalf("contents after ref delete = %#v", contents)
	}
}

func TestFavoriteListScopeRoundTrip(t *testing.T) {
	openTestDB(t)

	created, err := CreateFavoriteListForScope("Scoped", "1.21.1", "Fabric")
	if err != nil {
		t.Fatal(err)
	}
	if created.MinecraftVersion != "1.21.1" || created.ModLoader != "fabric" {
		t.Fatalf("created scope = %#v", created)
	}

	lists := ListFavoriteLists()
	if len(lists) != 1 || lists[0].ID != created.ID || lists[0].MinecraftVersion != "1.21.1" || lists[0].ModLoader != "fabric" {
		t.Fatalf("listed scope = %#v", lists)
	}
}

func TestFavoriteListNameIsUniqueWithinScope(t *testing.T) {
	openTestDB(t)

	created, err := CreateFavoriteListForScope("Performance", "1.21.1", "Fabric")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateFavoriteListForScope(" performance ", "1.21.1", "fabric"); !errors.Is(err, ErrFavoriteListNameConflict) {
		t.Fatalf("duplicate create error = %v", err)
	}
	otherScope, err := CreateFavoriteListForScope("Performance", "1.20.1", "fabric")
	if err != nil || otherScope.ID == "" {
		t.Fatalf("other scope create = %#v, %v", otherScope, err)
	}
	found, ok := GetFavoriteListByNameAndScope("PERFORMANCE", "1.21.1", "FABRIC")
	if !ok || found.ID != created.ID {
		t.Fatalf("scope lookup = %#v, %v", found, ok)
	}
}

func TestFavoriteModUsesParentListScope(t *testing.T) {
	openTestDB(t)

	list, err := CreateFavoriteListForScope("Scoped", "1.21.1", "Fabric")
	if err != nil {
		t.Fatal(err)
	}
	saved, ok, err := UpsertFavoriteMod(FavoriteMod{
		ListID:           list.ID,
		Platform:         "modrinth",
		ModID:            "sodium",
		MinecraftVersion: "1.20.1",
		ModLoader:        "forge",
	})
	if err != nil || !ok || saved.MinecraftVersion != "1.21.1" || saved.ModLoader != "fabric" {
		t.Fatalf("canonicalized favorite = %#v, %v, %v", saved, ok, err)
	}
	_, err = userDB.db.Exec(
		`INSERT INTO favorite_mods(id, list_id, platform, mod_id, minecraft_version, mod_loader, created_at, updated_at)
		 VALUES ('invalid-scope', ?, 'modrinth', 'lithium', '1.20.1', 'forge', 1, 1)`,
		list.ID,
	)
	if err == nil || !strings.Contains(err.Error(), "scope must match") {
		t.Fatalf("raw mismatched scope error = %v", err)
	}
}

func TestCreateFavoriteListWithModsForScopeIsAtomic(t *testing.T) {
	openTestDB(t)

	list, saved, err := CreateFavoriteListWithModsForScope("Performance", "1.21.1", "Fabric", []FavoriteMod{
		{Platform: "modrinth", ModID: "sodium", MinecraftVersion: "wrong", ModLoader: "wrong", Title: "Sodium"},
		{Platform: "modrinth", ModID: "sodium", Title: "Duplicate Sodium"},
		{Platform: "modrinth", ModID: "lithium", Title: "Lithium"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if list.MinecraftVersion != "1.21.1" || list.ModLoader != "fabric" || len(saved) != 2 {
		t.Fatalf("atomic create = %#v, %#v", list, saved)
	}
	for _, mod := range saved {
		if mod.ListID != list.ID || mod.MinecraftVersion != list.MinecraftVersion || mod.ModLoader != list.ModLoader {
			t.Fatalf("saved mod scope = %#v", mod)
		}
	}
	if _, _, err := CreateFavoriteListWithModsForScope("performance", "1.21.1", "fabric", []FavoriteMod{{Platform: "modrinth", ModID: "ferrite-core"}}); !errors.Is(err, ErrFavoriteListNameConflict) {
		t.Fatalf("duplicate atomic create error = %v", err)
	}
	if lists := ListFavoriteLists(); len(lists) != 1 {
		t.Fatalf("duplicate atomic create left a list = %#v", lists)
	}
	if _, _, err := CreateFavoriteListWithModsForScope("Invalid", "1.21.1", "fabric", []FavoriteMod{{ModID: "missing-platform"}}); err == nil {
		t.Fatal("invalid atomic create succeeded")
	}
	if lists := ListFavoriteLists(); len(lists) != 1 {
		t.Fatalf("invalid atomic create left a list = %#v", lists)
	}
}

func TestCacheVersionModIDs(t *testing.T) {
	path := openTestDB(t)

	if err := UpsertModPlatform(models.ModProject{
		Platform:  "Modrinth",
		ProjectID: "sodium",
		Slug:      "sodium-slug",
		Title:     "Sodium",
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetPlatformVersionSnapshot("Modrinth", "sodium", []models.ModVersion{
		{
			VersionID:    "v1",
			Name:         "1.0",
			SHA1:         "abc",
			PublishedAt:  10,
			GameVersions: []string{"1.21.1"},
			Loaders:      []string{"neoforge"},
		},
	}, 100, []ModPlatformVersionScope{{
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
	}}); err != nil {
		t.Fatal(err)
	}

	versions, err := GetPlatformVersions("Modrinth", "sodium")
	if err != nil || len(versions) != 1 {
		t.Fatalf("get versions: %v, count=%d", err, len(versions))
	}

	if err := SetVersionModIDs(versions[0].ID, []string{"sodium", "Sodium", "other", ""}); err != nil {
		t.Fatal(err)
	}

	// Direct read-back by version ID must reflect the persisted, deduplicated IDs.
	direct, err := GetVersionModIDs(versions[0].ID)
	if err != nil {
		t.Fatalf("GetVersionModIDs error: %v", err)
	}
	if len(direct) != 2 || direct[0] != "sodium" || direct[1] != "other" {
		t.Fatalf("GetVersionModIDs = %#v", direct)
	}

	// Unknown version ID must return (nil, nil) — callers treat this as a cache miss.
	miss, err := GetVersionModIDs("does-not-exist")
	if err != nil || miss != nil {
		t.Fatalf("GetVersionModIDs unknown = %#v, %v", miss, err)
	}

	reopenTestDB(t, path)

	versions, err = GetPlatformVersions("Modrinth", "sodium")
	if err != nil || len(versions) != 1 {
		t.Fatalf("get versions after reopen: %v, count=%d", err, len(versions))
	}
	modIDs := versions[0].ModIDs
	if len(modIDs) != 2 || modIDs[0] != "sodium" || modIDs[1] != "other" {
		t.Fatalf("modIDs = %#v", modIDs)
	}

	// Direct read-back must survive reopen.
	direct, err = GetVersionModIDs(versions[0].ID)
	if err != nil {
		t.Fatalf("GetVersionModIDs after reopen error: %v", err)
	}
	if len(direct) != 2 || direct[0] != "sodium" || direct[1] != "other" {
		t.Fatalf("GetVersionModIDs after reopen = %#v", direct)
	}
}

func TestFindCachedModVersionsByModIDFiltersAndSurvivesReopen(t *testing.T) {
	path := openTestDB(t)
	for _, project := range []models.ModProject{
		{Platform: "Modrinth", ProjectID: "architectury", Title: "Architectury API"},
		{Platform: "CurseForge", ProjectID: "419699", Title: "Architectury API CF"},
	} {
		if err := UpsertModPlatform(project); err != nil {
			t.Fatal(err)
		}
	}
	if err := SetPlatformVersions("Modrinth", "architectury", []models.ModVersion{
		{ID: "mr-old", VersionID: "mr-old", ProjectID: "architectury", DownloadURL: "https://example.com/old.jar", PublishedAt: 1, GameVersions: []string{"1.21.1"}, Loaders: []string{"fabric"}, ModIDs: []string{"architectury"}},
		{ID: "mr-new", VersionID: "mr-new", ProjectID: "architectury", DownloadURL: "https://example.com/new.jar", PublishedAt: 2, GameVersions: []string{"1.21.1"}, Loaders: []string{"fabric"}, ModIDs: []string{"Architectury"}},
		{ID: "mr-forge", VersionID: "mr-forge", ProjectID: "architectury", DownloadURL: "https://example.com/forge.jar", PublishedAt: 3, GameVersions: []string{"1.21.1"}, Loaders: []string{"forge"}, ModIDs: []string{"architectury"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetPlatformVersions("CurseForge", "419699", []models.ModVersion{{
		ID: "cf-new", VersionID: "cf-new", ProjectID: "419699", DownloadURL: "https://example.com/cf.jar", PublishedAt: 4, GameVersions: []string{"1.21.1"}, Loaders: []string{"fabric"}, ModIDs: []string{"architectury"},
	}}); err != nil {
		t.Fatal(err)
	}

	assertCandidates := func() {
		candidates := FindCachedModVersionsByModID(" ARCHITECTURY ", "1.21.1", "Fabric")
		if len(candidates) != 2 {
			t.Fatalf("candidate count = %d, want 2: %#v", len(candidates), candidates)
		}
		if candidates[0].Project.Platform != "CurseForge" || candidates[0].Version.ID != "cf-new" {
			t.Fatalf("first candidate = %#v", candidates[0])
		}
		if candidates[1].Project.Platform != "Modrinth" || candidates[1].Version.ID != "mr-new" {
			t.Fatalf("second candidate = %#v", candidates[1])
		}
		if miss := FindCachedModVersionsByModID("missing", "1.21.1", "fabric"); len(miss) != 0 {
			t.Fatalf("cache miss = %#v", miss)
		}
	}

	assertCandidates()
	reopenTestDB(t, path)
	assertCandidates()
}
