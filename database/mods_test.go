package database

import (
	"os"
	"path/filepath"
	"testing"

	"mod-downloader/models"
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
	if err := Open(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(tmp, databaseFileName)
	return path
}

func reopenTestDB(t *testing.T, path string) {
	t.Helper()
	Close()
	if err := Open(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
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
