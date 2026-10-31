package appcore

import (
	"archive/zip"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/storage"
)

func TestFavoritePackwizScopeRejectsMixedOrMissingScope(t *testing.T) {
	_, err := favoritePackwizScope([]storage.FavoriteMod{{
		Title:            "Sodium",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}, {
		Title:            "JEI",
		MinecraftVersion: "1.20.1",
		ModLoader:        "fabric",
	}})
	if err == nil || !strings.Contains(err.Error(), "multiple minecraft versions") {
		t.Fatalf("mixed minecraft error = %v", err)
	}

	_, err = favoritePackwizScope([]storage.FavoriteMod{{
		Title:            "Sodium",
		MinecraftVersion: "1.21.1",
	}})
	if err == nil || !strings.Contains(err.Error(), "missing minecraft version or mod loader") {
		t.Fatalf("missing scope error = %v", err)
	}
}

func TestExportFavoriteListPackwizZipRejectsEmptyList(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	waitProviderBackgroundTasks(t)

	svc := New(Options{})
	list := svc.CreateFavoriteList("Empty")
	_, err := svc.ExportFavoriteListPackwizZip(list.ID, filepath.Join(t.TempDir(), "empty.zip"))
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("empty list error = %v", err)
	}
}

func TestExportFavoriteListPackwizZipWritesPack(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	waitProviderBackgroundTasks(t)

	storeFavoritePackwizVersion(t, "modrinth", "sodium", "version-1", "1.21.1", "fabric", "sodium.jar")

	svc := New(Options{})
	list := svc.CreateFavoriteListForScope("Performance", "1.21.1", "fabric")
	saved := svc.AddFavoriteMod(storage.FavoriteMod{
		ListID:           list.ID,
		Platform:         "modrinth",
		ModID:            "sodium",
		VersionID:        "version-1",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		Title:            "Sodium",
	})
	if saved.ID == "" {
		t.Fatalf("favorite mod not saved: %#v", saved)
	}

	out := filepath.Join(t.TempDir(), "performance")
	result, err := svc.ExportFavoriteListPackwizZip(list.ID, out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(result.Path, ".zip") {
		t.Fatalf("result path = %q", result.Path)
	}
	paths := zipPaths(t, result.Path)
	for _, path := range []string{"pack.toml", "index.toml", "mods/sodium.pw.toml"} {
		if !paths[path] {
			t.Fatalf("missing %s from exported zip paths %#v", path, paths)
		}
	}
	stats := svc.GetUsageStats()
	if stats.FavoritesAdded != 1 || stats.PackwizExports != 1 {
		t.Fatalf("usage stats after favorite export = %#v", stats)
	}
}

func TestExportFavoriteListPackwizZipForScopeFiltersItems(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	waitProviderBackgroundTasks(t)

	storeFavoritePackwizVersion(t, "modrinth", "sodium", "version-1", "1.21.1", "fabric", "sodium.jar")

	svc := New(Options{})
	list := svc.CreateFavoriteListForScope("Scoped", "1.21.1", "fabric")
	svc.AddFavoriteMod(storage.FavoriteMod{
		ListID:           list.ID,
		Platform:         "modrinth",
		ModID:            "sodium",
		VersionID:        "version-1",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		Title:            "Sodium",
	})
	result, err := svc.ExportFavoriteListPackwizZipForScope(list.ID, filepath.Join(t.TempDir(), "scoped"), "1.21.1", "Fabric")
	if err != nil {
		t.Fatal(err)
	}
	paths := zipPaths(t, result.Path)
	if !paths["mods/sodium.pw.toml"] {
		t.Fatalf("missing scoped mod from exported zip paths %#v", paths)
	}
}

func TestExportFavoriteListPackwizZipIncludesRecursiveReferences(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	waitProviderBackgroundTasks(t)

	for _, modID := range []string{"sodium", "lithium", "ferrite-core"} {
		storeFavoritePackwizVersion(t, "modrinth", modID, modID+"-version", "1.21.1", "fabric", modID+".jar")
	}

	svc := New(Options{})
	root := svc.CreateFavoriteListForScope("Root", "1.21.1", "fabric")
	child := svc.CreateFavoriteListForScope("Child", "1.21.1", "fabric")
	grandchild := svc.CreateFavoriteListForScope("Grandchild", "1.21.1", "fabric")
	for _, entry := range []struct {
		listID string
		modID  string
	}{
		{listID: root.ID, modID: "sodium"},
		{listID: child.ID, modID: "lithium"},
		{listID: grandchild.ID, modID: "ferrite-core"},
	} {
		svc.AddFavoriteMod(storage.FavoriteMod{
			ListID: entry.listID, Platform: "modrinth", ModID: entry.modID,
			VersionID: entry.modID + "-version", MinecraftVersion: "1.21.1", ModLoader: "fabric", Title: entry.modID,
		})
	}
	if ref := svc.AddFavoriteListReference(root.ID, child.ID); ref.ID == "" {
		t.Fatal("root to child reference was not created")
	}
	if ref := svc.AddFavoriteListReference(child.ID, grandchild.ID); ref.ID == "" {
		t.Fatal("child to grandchild reference was not created")
	}

	result, err := svc.ExportFavoriteListPackwizZip(root.ID, filepath.Join(t.TempDir(), "recursive.zip"))
	if err != nil {
		t.Fatal(err)
	}
	paths := zipPaths(t, result.Path)
	for _, path := range []string{"mods/sodium.pw.toml", "mods/lithium.pw.toml", "mods/ferrite-core.pw.toml"} {
		if !paths[path] {
			t.Fatalf("missing recursively referenced mod %s from %#v", path, paths)
		}
	}
}

func TestEnsureZipPathAppendsExtension(t *testing.T) {
	if got := ensureZipPath("/tmp/pack"); got != "/tmp/pack.zip" {
		t.Fatalf("ensureZipPath without extension = %q", got)
	}
	if got := ensureZipPath("/tmp/pack.ZIP"); got != "/tmp/pack.ZIP" {
		t.Fatalf("ensureZipPath with extension = %q", got)
	}
}

func storeFavoritePackwizVersion(t *testing.T, platform, projectID, versionID, minecraftVersion, modLoader, fileName string) {
	t.Helper()
	now := time.Now().Unix()
	if err := storage.UpsertModPlatform(models.ModProject{
		ID:        models.ProjectKey(platform, projectID),
		Platform:  platform,
		ProjectID: projectID,
		Title:     projectID,
		CachedAt:  now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersionSnapshot(platform, projectID, []models.ModVersion{{
		ID:           versionID,
		Platform:     platform,
		ProjectID:    projectID,
		VersionID:    versionID,
		Name:         versionID,
		FileName:     fileName,
		DownloadURL:  "https://cdn.modrinth.com/data/" + fileName,
		SHA1:         "abc123",
		GameVersions: []string{minecraftVersion},
		Loaders:      []string{modLoader},
	}}, now, []storage.ModPlatformVersionScope{{
		MinecraftVersion: minecraftVersion,
		ModLoader:        modLoader,
	}}); err != nil {
		t.Fatal(err)
	}
}

func zipPaths(t *testing.T, path string) map[string]bool {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	paths := make(map[string]bool)
	for _, file := range zr.File {
		paths[file.Name] = true
	}
	return paths
}
