package appcore

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func TestLocalModsInDirScansExplicitModsDir(t *testing.T) {
	global.SetMinecraftDir("")
	global.ClearLocalMods()
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.ClearLocalMods()
	})

	modsDir := filepath.Join(t.TempDir(), "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFabricJar(t, filepath.Join(modsDir, "example.jar"))

	svc := New(Options{})
	mods, err := svc.LocalModsInDir(modsDir, "1.21.1", "fabric")
	if err != nil {
		t.Fatal(err)
	}
	if len(mods) != 1 {
		t.Fatalf("mod count = %d, want 1: %#v", len(mods), mods)
	}
	if mods[0].ID != "example" || mods[0].FileName != "example" || mods[0].Path != filepath.Join(modsDir, "example.jar") {
		t.Fatalf("mod = %#v", mods[0])
	}

	paths := global.LocalModPathsInInstance(modbridge.InstanceIDForTargetDir(modsDir))
	if len(paths) != 1 {
		t.Fatalf("indexed paths = %#v, want 1", paths)
	}
	if paths[0].Path != filepath.Join(modsDir, "example.jar") {
		t.Fatalf("indexed path = %q", paths[0].Path)
	}
}

func TestRefreshSelectedVersionModsPersistsRefreshedSnapshot(t *testing.T) {
	_, modsDir := setupSelectedLocalModInstance(t)
	svc := New(Options{})
	t.Cleanup(svc.stopModsWatcher)
	t.Cleanup(svc.stopServiceBackground)
	if got := svc.RefreshSelectedVersionMods(); len(got.Mods) != 0 {
		t.Fatalf("initial mods = %#v, want none", got.Mods)
	}

	writeFabricJar(t, filepath.Join(modsDir, "refreshed.jar"))
	refreshed := svc.RefreshSelectedVersionMods()
	if len(refreshed.Mods) != 1 || refreshed.Mods[0].ID != "example" {
		t.Fatalf("refreshed mods = %#v, want scanned mod", refreshed.Mods)
	}
	selected := svc.GetSelectedVersion()
	if len(selected.Mods) != 1 || selected.Mods[0].ID != "example" {
		t.Fatalf("selected mods after refresh = %#v, want scanned mod", selected.Mods)
	}
}

func TestSelectVersionBindsModsWatcher(t *testing.T) {
	_, modsDir := setupSelectedLocalModInstance(t)
	svc := New(Options{})
	t.Cleanup(svc.stopModsWatcher)

	if _, err := svc.SelectVersion("test"); err != nil {
		t.Fatal(err)
	}

	want, err := filepath.Abs(modsDir)
	if err != nil {
		t.Fatal(err)
	}
	svc.modsWatchMu.Lock()
	got := svc.modsWatchDir
	watching := svc.modsWatchCancel != nil
	svc.modsWatchMu.Unlock()
	if got != want || !watching {
		t.Fatalf("mods watcher = dir %q active %v, want dir %q active true", got, watching, want)
	}
}

func TestApplyLocalModBatchOperationDisableEnableInvertDelete(t *testing.T) {
	root, modsDir := setupSelectedLocalModInstance(t)
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	enabled := filepath.Join(modsDir, "enabled.jar")
	disabled := filepath.Join(modsDir, "disabled.jar.disabled")
	removed := filepath.Join(modsDir, "remove.jar")
	writeFabricJar(t, enabled)
	writeFabricJar(t, disabled)
	writeFile(t, removed)

	svc := New(Options{})
	t.Cleanup(svc.stopServiceBackground)
	version, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionDisable,
		Paths:  []string{filepath.ToSlash(filepath.Join("versions", "test", "mods", "enabled.jar"))},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertMissing(t, enabled)
	assertExists(t, enabled+".disabled")
	if len(version.Mods) != 1 || version.Mods[0].Path != filepath.Join("versions", "test", "mods", "enabled.jar.disabled") || version.Mods[0].Enabled {
		t.Fatalf("disable response mods = %#v, want disabled mod snapshot", version.Mods)
	}

	version, err = svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionEnable,
		Paths:  []string{filepath.Join(root, "versions", "test", "mods", "enabled.jar.disabled")},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertExists(t, enabled)
	assertMissing(t, enabled+".disabled")
	if len(version.Mods) != 1 || version.Mods[0].Path != filepath.Join("versions", "test", "mods", "enabled.jar") || !version.Mods[0].Enabled {
		t.Fatalf("enable response mods = %#v, want enabled mod snapshot", version.Mods)
	}

	if _, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionInvert,
		Paths:  []string{enabled, disabled},
	}); err != nil {
		t.Fatal(err)
	}
	assertExists(t, enabled+".disabled")
	assertMissing(t, enabled)
	assertExists(t, strings.TrimSuffix(disabled, ".disabled"))
	assertMissing(t, disabled)

	if _, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionDelete,
		Paths:  []string{removed},
	}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, removed)

	stats := svc.GetUsageStats()
	if stats.ModsEnabled != 2 || stats.ModsDisabled != 2 || stats.ModsDeleted != 1 {
		t.Fatalf("local mod usage stats = %#v", stats)
	}
}

func TestApplyLocalModBatchOperationDeleteClearsRelativeLocalIndex(t *testing.T) {
	_, modsDir := setupSelectedLocalModInstance(t)
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	modPath := filepath.Join(modsDir, "remove.jar")
	writeFabricJar(t, modPath)
	svc := New(Options{})
	t.Cleanup(svc.stopServiceBackground)
	if got := svc.RefreshSelectedVersionMods(); len(got.Mods) != 1 {
		t.Fatalf("initial mods = %#v, want one mod", got.Mods)
	}
	if paths := global.LocalModPathsInInstance("test"); len(paths) != 1 {
		t.Fatalf("initial local index = %#v, want one path", paths)
	}

	if _, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionDelete,
		Paths:  []string{modPath},
	}); err != nil {
		t.Fatal(err)
	}
	if paths := global.LocalModPathsInInstance("test"); len(paths) != 0 {
		t.Fatalf("local index after delete = %#v, want empty", paths)
	}
}

func TestRefreshChangedPathsClearsRelativeLocalIndex(t *testing.T) {
	_, modsDir := setupSelectedLocalModInstance(t)
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	modPath := filepath.Join(modsDir, "watched.jar")
	writeFabricJar(t, modPath)
	svc := New(Options{})
	t.Cleanup(svc.stopServiceBackground)
	if got := svc.RefreshSelectedVersionMods(); len(got.Mods) != 1 {
		t.Fatalf("initial mods = %#v, want one mod", got.Mods)
	}
	if err := os.Remove(modPath); err != nil {
		t.Fatal(err)
	}

	svc.refreshChangedPaths([]string{modPath})
	if paths := global.LocalModPathsInInstance("test"); len(paths) != 0 {
		t.Fatalf("local index after fs event = %#v, want empty", paths)
	}
}

func TestApplyLocalModBatchOperationDeduplicatesPaths(t *testing.T) {
	_, modsDir := setupSelectedLocalModInstance(t)
	source := filepath.Join(modsDir, "dup.jar")
	writeFile(t, source)

	svc := New(Options{})
	if _, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionDisable,
		Paths:  []string{source, source},
	}); err != nil {
		t.Fatal(err)
	}
	assertMissing(t, source)
	assertExists(t, source+".disabled")
}

func TestApplyLocalModBatchOperationRejectsMissingOutsideAndCollision(t *testing.T) {
	_, modsDir := setupSelectedLocalModInstance(t)
	source := filepath.Join(modsDir, "collision.jar")
	writeFile(t, source)
	writeFile(t, source+".disabled")

	svc := New(Options{})
	if _, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionDisable,
		Paths:  []string{source},
	}); err == nil {
		t.Fatal("collision disable returned nil error")
	}
	assertExists(t, source)
	assertExists(t, source+".disabled")

	if _, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionDelete,
		Paths:  []string{filepath.Join(modsDir, "missing.jar")},
	}); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("missing delete error = %v, want not found", err)
	}

	outside := filepath.Join(t.TempDir(), "outside.jar")
	writeFile(t, outside)
	if _, err := svc.ApplyLocalModBatchOperation(appstructs.LocalModBatchOperationRequest{
		Action: LocalModBatchActionDelete,
		Paths:  []string{outside},
	}); err == nil || !strings.Contains(err.Error(), "outside selected instance") {
		t.Fatalf("outside delete error = %v, want outside selected instance", err)
	}
	assertExists(t, outside)
}

func TestEnrichModMetadataPrefersOnlineDisplayFields(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	project := models.ModProject{
		Platform:   "Modrinth",
		ProjectID:  "project-1",
		Slug:       "online-project",
		Title:      "Online Project",
		IconURL:    "https://example.com/icon.png",
		Categories: []string{"library", "magic"},
	}
	if err := storage.UpsertModPlatform(project); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersions("Modrinth", "project-1", []models.ModVersion{{
		ID:        "version-1",
		VersionID: "version-1",
		ProjectID: "project-1",
		Platform:  "Modrinth",
		Version:   "1.0.1+online",
		FileName:  "online-project-1.0.1.jar",
		SHA1:      "abc123",
	}}); err != nil {
		t.Fatal(err)
	}

	enriched, misses := enrichModMetadata([]structs.ModInfo{{
		ID:      "jar-id",
		Name:    "Jar Name",
		Version: "1.0.0",
		SHA1:    "abc123",
		JijMods: []structs.JijModInfo{{ID: "nested", Name: "Nested"}},
	}})
	if len(misses) != 0 {
		t.Fatalf("misses = %#v, want none", misses)
	}
	if len(enriched) != 1 {
		t.Fatalf("enriched count = %d, want 1", len(enriched))
	}
	got := enriched[0]
	if got.ID != "jar-id" || got.Name != "Jar Name" || got.Version != "1.0.0" || len(got.JijMods) != 1 {
		t.Fatalf("jar-derived fields changed: %#v", got)
	}
	if got.OnlineName != "Online Project" || got.IconURL != "https://example.com/icon.png" || got.OnlinePlatform != "Modrinth" || got.OnlineProjectID != "project-1" || got.OnlineSlug != "online-project" {
		t.Fatalf("online fields = %#v", got)
	}
	if got.OnlineVersion != "1.0.1+online" || got.OnlineVersionID != "version-1" || got.OnlineFileName != "online-project-1.0.1.jar" {
		t.Fatalf("online version fields = %#v", got)
	}
	assertStringSlice(t, got.Categories, []string{"library", "magic"})
}

func TestApplyResolvedModMetadataClearsLoadingWithoutMatch(t *testing.T) {
	mods := []structs.ModInfo{{
		ID: "local", SHA1: "ABC123", OnlineMetadataLoading: true,
	}, {
		ID: "unrelated", SHA1: "other", OnlineMetadataLoading: false,
	}}
	updated, changed := applyResolvedModMetadata(mods, map[string]uint32{"abc123": 42}, nil, nil)
	if !changed || updated[0].OnlineMetadataLoading {
		t.Fatalf("pending metadata was not cleared: changed=%v mods=%#v", changed, updated)
	}
	if mods[0].OnlineMetadataLoading != true {
		t.Fatalf("input slice was mutated: %#v", mods)
	}
	if updated[1].ID != mods[1].ID || updated[1].SHA1 != mods[1].SHA1 || updated[1].OnlineMetadataLoading != mods[1].OnlineMetadataLoading {
		t.Fatalf("unrelated mod changed: %#v", updated[1])
	}
}

func TestApplyResolvedModMetadataAppliesMatchAndClearsLoading(t *testing.T) {
	mods := []structs.ModInfo{{ID: "local", Version: "jar-version", SHA1: "abc123", OnlineMetadataLoading: true}}
	updated, changed := applyResolvedModMetadata(
		mods,
		map[string]uint32{"abc123": 42},
		map[string]models.ModProject{"abc123": {Platform: "CurseForge", ProjectID: "123", Title: "Online"}},
		map[string]models.ModVersion{"abc123": {ID: "456", Version: "online-version"}},
	)
	if !changed || updated[0].OnlineMetadataLoading || updated[0].OnlineName != "Online" || updated[0].OnlineVersionID != "456" {
		t.Fatalf("resolved metadata = changed=%v mods=%#v", changed, updated)
	}
	if updated[0].Version != "jar-version" {
		t.Fatalf("local version changed: %#v", updated[0])
	}
}

func assertStringSlice(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("strings = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("strings[%d] = %q, want %q (all: %#v)", i, got[i], want[i], got)
		}
	}
}

func setupSelectedLocalModInstance(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	version := structs.VersionInfo{
		ID:               "test",
		Name:             "test",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}
	global.SetMinecraftDir(root)
	global.SetVersionsForDir(root, []structs.VersionInfo{version})
	global.SetSelectedVersion(version)
	global.ClearLocalMods()
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
		global.ClearLocalMods()
	})

	modsDir := filepath.Join(root, "versions", "test", "mods")
	if err := os.MkdirAll(modsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, modsDir
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("jar"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected path to exist %q: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected path to be missing %q: %v", path, err)
	}
}

func writeFabricJar(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("fabric.mod.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(`{"id":"example","name":"Example","version":"1.0.0"}`)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestLocalModsInDirRejectsMissingDir(t *testing.T) {
	svc := New(Options{})
	_, err := svc.LocalModsInDir(filepath.Join(t.TempDir(), "missing"), "", "")
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("LocalModsInDir error = %v, want explicit missing-directory error", err)
	}
}

func TestEnsureSelectedVersionSelectsFirstOnlyWhenUnset(t *testing.T) {
	previous := global.GetSelectedVersion()
	previousDir := global.GetMinecraftDir()
	t.Cleanup(func() {
		global.SetSelectedVersion(previous)
		global.SetMinecraftDir(previousDir)
		global.InvalidateVersions()
	})
	dir := t.TempDir()
	versions := []structs.VersionInfo{
		{ID: "first", Name: "First", MinecraftVersion: "1.21.1", ModLoader: "fabric"},
		{ID: "second", Name: "Second", MinecraftVersion: "1.20.1", ModLoader: "forge"},
	}
	global.SetMinecraftDir(dir)
	global.SetVersionsForDir(dir, versions)

	global.SetSelectedVersion(structs.VersionInfo{})
	ensureSelectedVersion(versions)
	if selected := global.GetSelectedVersion(); selected.ID != "first" {
		t.Fatalf("selected = %q, want first", selected.ID)
	}

	global.SetSelectedVersion(versions[1])
	ensureSelectedVersion(versions)
	if selected := global.GetSelectedVersion(); selected.ID != "second" {
		t.Fatalf("existing selection overwritten: %q", selected.ID)
	}
}
