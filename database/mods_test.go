package database

import (
	"os"
	"path/filepath"
	"testing"

	structs "mod-downloader/structs/minecraft"

	"github.com/tidwall/buntdb"
)

func openTestDB(t *testing.T) {
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
	if _, err := os.Stat(filepath.Join(tmp, databaseFileName)); err != nil {
		t.Fatal(err)
	}
	var config buntdb.Config
	if err := db.ReadConfig(&config); err != nil {
		t.Fatal(err)
	}
	if config.SyncPolicy != buntdb.Never {
		t.Fatalf("sync policy = %v, want %v", config.SyncPolicy, buntdb.Never)
	}
}

func TestBuntDBPlatformVersionsAndDependencies(t *testing.T) {
	openTestDB(t)

	if err := UpsertModPlatform(ModPlatform{
		Platform:  "Modrinth",
		ProjectID: "sodium",
		Slug:      "sodium-slug",
		Name:      "Sodium",
	}); err != nil {
		t.Fatal(err)
	}
	if err := SetPlatformVersionSnapshot("Modrinth", "sodium", []ModPlatformVersion{
		{
			VersionID:    "v1",
			Name:         "1.0",
			SHA1:         "abc",
			PublishedAt:  10,
			GameVersions: []string{"1.21.1"},
			Loaders:      []string{"neoforge"},
			Dependencies: []ModDependency{{
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

func TestBuntDBPinnedModsAndJarMetadata(t *testing.T) {
	openTestDB(t)

	if err := UpsertPinnedMod(PinnedMod{
		Platform:         "Modrinth",
		ModID:            "Sodium",
		VersionID:        "v1",
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
	}); err != nil {
		t.Fatal(err)
	}
	if pin, ok := GetPinnedMod("modrinth", "sodium", "1.21.1", "neoforge"); !ok || pin.VersionID != "v1" {
		t.Fatalf("pin = %#v, %v", pin, ok)
	}

	if err := SetJarMetadata("sha1", []structs.ModInfo{{ID: "jei"}, {ID: "jei"}, {ID: "tmrv"}}); err != nil {
		t.Fatal(err)
	}
	mods, ok := GetJarMetadata("sha1")
	if !ok || len(mods) != 2 || mods[0].ID != "jei" || mods[1].ID != "tmrv" {
		t.Fatalf("jar metadata = %#v, %v", mods, ok)
	}
}
