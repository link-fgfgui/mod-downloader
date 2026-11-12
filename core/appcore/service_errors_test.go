// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/configs"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
)

func TestFavoriteAndPinErrorContracts(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	svc := New(Options{})

	if _, err := svc.RenameFavoriteListWithError("missing", "name"); !errors.Is(err, ErrFavoriteListNotFound) {
		t.Fatalf("rename missing list error = %v, want ErrFavoriteListNotFound", err)
	}
	if _, err := svc.UpdateFavoriteListMetadataWithError(storage.FavoriteList{ID: "missing"}); !errors.Is(err, ErrFavoriteListNotFound) {
		t.Fatalf("update missing list error = %v, want ErrFavoriteListNotFound", err)
	}
	if _, err := svc.AddFavoriteModWithError(storage.FavoriteMod{
		ListID:   "missing",
		Platform: "modrinth",
		ModID:    "sodium",
	}); !errors.Is(err, ErrFavoriteListNotFound) {
		t.Fatalf("add to missing list error = %v, want ErrFavoriteListNotFound", err)
	}
	if _, err := svc.AddFavoriteListReferenceWithError("missing", "also-missing"); !errors.Is(err, ErrFavoriteListNotFound) {
		t.Fatalf("add missing reference error = %v, want ErrFavoriteListNotFound", err)
	}

	// Deletes are idempotent: a missing target is reported as false without an
	// error so repeated UI actions do not surface a failure toast.
	if deleted, err := svc.DeleteFavoriteListWithError("missing"); err != nil || deleted {
		t.Fatalf("delete missing list = %v, %v", deleted, err)
	}
	if deleted, err := svc.RemoveFavoriteModWithError("missing", "modrinth", "sodium", "1.21.1", "fabric"); err != nil || deleted {
		t.Fatalf("remove missing mod = %v, %v", deleted, err)
	}
	if removed, err := svc.UnpinModWithError("modrinth", "sodium", "1.21.1", "fabric"); err != nil || removed {
		t.Fatalf("unpin missing pin = %v, %v", removed, err)
	}

	// Reading a missing pin is an empty result, not an error.
	if pin, err := svc.GetPinnedModVersionWithError("modrinth", "sodium", "1.21.1", "fabric"); err != nil || pin.VersionID != "" {
		t.Fatalf("missing pin = %#v, %v", pin, err)
	}
}

func TestFavoriteAndPinErrorsWhenStorageClosed(t *testing.T) {
	storage.Close()
	svc := New(Options{})

	checks := []struct {
		name string
		call func() error
	}{
		{"create list", func() error {
			_, err := svc.CreateFavoriteListForScopeWithError("list", "1.21.1", "fabric")
			return err
		}},
		{"rename list", func() error {
			_, err := svc.RenameFavoriteListWithError("id", "name")
			return err
		}},
		{"delete list", func() error {
			_, err := svc.DeleteFavoriteListWithError("id")
			return err
		}},
		{"reorder lists", func() error {
			return svc.ReorderFavoriteListsWithError([]string{"id"})
		}},
		{"add favorite mod", func() error {
			_, err := svc.AddFavoriteModWithError(storage.FavoriteMod{ListID: "id", Platform: "modrinth", ModID: "sodium"})
			return err
		}},
		{"get pin", func() error {
			_, err := svc.GetPinnedModVersionWithError("modrinth", "sodium", "1.21.1", "fabric")
			return err
		}},
		{"pin", func() error {
			_, err := svc.PinModVersionWithError(appstructs.ModVersionPinRequest{
				Platform:         "modrinth",
				ModID:            "sodium",
				VersionID:        "v1",
				MinecraftVersion: "1.21.1",
				ModLoader:        "fabric",
			})
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.call(); err == nil {
				t.Fatal("expected storage error, got nil")
			}
		})
	}
}

func TestSaveThemeAndLanguageReportPersistenceFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not enforced the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can write into read-only directories")
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(appDir, 0o755)
		if err := os.Chdir(oldwd); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	svc := New(Options{Config: &configs.Config{}})
	if err := os.Chmod(appDir, 0o555); err != nil {
		t.Fatal(err)
	}

	theme, err := svc.SaveThemeWithError("light")
	if err == nil {
		t.Fatal("SaveThemeWithError returned nil error for a read-only config directory")
	}
	if theme != configs.ThemeLight.String() {
		t.Fatalf("SaveThemeWithError value = %q, want %q", theme, configs.ThemeLight)
	}
	if got := svc.GetSettings().Theme; got != configs.ThemeDark.String() {
		t.Fatalf("live theme after failed save = %q, want %q", got, configs.ThemeDark)
	}
	if got := svc.GetSettings().SaveError; got == "" {
		t.Fatal("GetSettings().SaveError empty after failed theme save")
	}

	language, err := svc.SaveLanguageWithError("zh-CN")
	if err == nil {
		t.Fatal("SaveLanguageWithError returned nil error for a read-only config directory")
	}
	if language != configs.LanguageChinese.String() {
		t.Fatalf("SaveLanguageWithError value = %q, want %q", language, configs.LanguageChinese)
	}
	if got := svc.GetSettings().Language; got != configs.LanguageSystem.String() {
		t.Fatalf("live language after failed save = %q, want %q", got, configs.LanguageSystem)
	}
}

func TestConfigSnapshotDoesNotSharePointerFields(t *testing.T) {
	svc := New(Options{Config: &configs.Config{
		Preferences: configs.Preferences{
			AutoScanUnusedDependencies: configs.BoolPtr(true),
			EnableJIJDetection:         configs.BoolPtr(true),
			HardwareAcceleration:       configs.BoolPtr(true),
		},
		Logging: configs.LoggingConfig{Enabled: configs.BoolPtr(true)},
	}})

	snapshot := svc.Config()
	*snapshot.Preferences.AutoScanUnusedDependencies = false
	*snapshot.Preferences.EnableJIJDetection = false
	*snapshot.Preferences.HardwareAcceleration = false
	*snapshot.Logging.Enabled = false

	fresh := svc.Config()
	if !*fresh.Preferences.AutoScanUnusedDependencies ||
		!*fresh.Preferences.EnableJIJDetection ||
		!*fresh.Preferences.HardwareAcceleration ||
		!*fresh.Logging.Enabled {
		t.Fatalf("snapshot mutation leaked into live config: %#v", fresh)
	}
}

func TestConfigSnapshotsConcurrentReadWrite(t *testing.T) {
	svc := New(Options{Config: &configs.Config{}})
	var wg sync.WaitGroup
	for worker := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 100 {
				svc.mutateConfig(func(cfg *configs.Config) {
					cfg.Preferences.RequireCtrlForMultiSelect = (worker+i)%2 == 0
					cfg.Preferences.Theme = configs.ThemeDark
				})
				cfg := svc.Config()
				_ = cfg.Preferences.RequireCtrlForMultiSelect
				_ = svc.GetSettings()
			}
		}()
	}
	wg.Wait()
}

func TestSaveCacheDirPreferenceKeepsConfigWhenStoreOpenFails(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		storage.Close()
		if err := os.Chdir(oldwd); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	initialDir := filepath.Join(t.TempDir(), "initial-cache")
	svc := New(Options{Config: &configs.Config{
		Runtime: configs.RuntimeConfig{CacheDir: initialDir},
	}})
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)
	sentinel := svc.CreateFavoriteListForScope("Sentinel", "1.21.1", "fabric")
	if sentinel.ID == "" {
		t.Fatal("failed to seed initial store")
	}

	// A regular file where the cache directory should be makes OpenAt fail
	// before the config is persisted.
	badDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(badDir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	view := svc.SaveCacheDirPreference(badDir)
	if view.SaveError == "" {
		t.Fatal("SaveError empty after store open failure")
	}
	if got := svc.configSnapshot().Runtime.CacheDir; got != initialDir {
		t.Fatalf("config cache dir = %q, want %q", got, initialDir)
	}
	if lists := svc.ListFavoriteLists(); len(lists) != 1 || lists[0].ID != sentinel.ID {
		t.Fatalf("live store after failed switch = %#v", lists)
	}
}

func TestSaveCacheDirPreferenceRollsBackOnConfigSaveFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not enforced the same way on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can write into read-only directories")
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(appDir, 0o755)
		storage.Close()
		if err := os.Chdir(oldwd); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	initialDir := filepath.Join(t.TempDir(), "initial-cache")
	svc := New(Options{Config: &configs.Config{
		Runtime: configs.RuntimeConfig{CacheDir: initialDir},
	}})
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)

	initialCachePath, err := svc.cachePath()
	if err != nil {
		t.Fatal(err)
	}
	sentinel := svc.CreateFavoriteListForScope("Sentinel", "1.21.1", "fabric")
	if sentinel.ID == "" {
		t.Fatal("failed to seed initial store")
	}

	if err := os.Chmod(appDir, 0o555); err != nil {
		t.Fatal(err)
	}
	nextDir := filepath.Join(t.TempDir(), "next-cache")
	view := svc.SaveCacheDirPreference(nextDir)
	if view.SaveError == "" {
		t.Fatal("SaveError empty after config save failure")
	}
	if got := svc.configSnapshot().Runtime.CacheDir; got != initialDir {
		t.Fatalf("config cache dir = %q, want rollback to %q", got, initialDir)
	}
	if got, err := svc.cachePath(); err != nil || got != initialCachePath {
		t.Fatalf("cache path after rollback = %q, %v; want %q", got, err, initialCachePath)
	}
	if lists := svc.ListFavoriteLists(); len(lists) != 1 || lists[0].ID != sentinel.ID {
		t.Fatalf("live store after rollback = %#v, want seeded list", lists)
	}

	// The failed save must be visible to the next settings read as well.
	if got := svc.GetSettings().SaveError; got == "" {
		t.Fatal("GetSettings().SaveError empty after config save failure")
	}
}
