package appcore

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/link-fgfgui/mod-downloader-core/downloader"
	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

func TestMain(m *testing.M) {
	// Direct service-method tests run without Startup, but they still rely on
	// provider refreshes being accepted. Keep one process-wide reference so a
	// per-test Service close does not disable them for the rest of the suite.
	providers.StartBackgroundTasks()
	code := m.Run()
	providers.StopBackgroundTasks()
	os.Exit(code)
}

func TestLoadMinecraftReleaseVersionsUsesFreshCache(t *testing.T) {
	t.Cleanup(func() { global.SetMinecraftReleaseVersions(nil) })
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	if err := storage.SetMinecraftReleaseVersions([]string{"1.21.1"}, time.Now().Unix()); err != nil {
		t.Fatal(err)
	}
	previousFetch := fetchMinecraftReleaseVersions
	requests := 0
	fetchMinecraftReleaseVersions = func() ([]string, error) {
		requests++
		return []string{"future"}, nil
	}
	t.Cleanup(func() { fetchMinecraftReleaseVersions = previousFetch })

	if err := loadMinecraftReleaseVersions(); err != nil {
		t.Fatal(err)
	}
	if requests != 0 {
		t.Fatalf("manifest requests = %d, want 0", requests)
	}
	if got := global.GetMinecraftReleaseVersions(); len(got) != 1 || got[0] != "1.21.1" {
		t.Fatalf("global releases = %#v", got)
	}
}

func TestDownloadCompletionEventAndUsageStat(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	events := make(chan Event, 2)
	svc := New(Options{OnEvent: func(event Event) { events <- event }})
	svc.downloadEvents().OnDownloadCompleted()
	if got := svc.GetUsageStats().DownloadsCompleted; got != 1 {
		t.Fatalf("downloads completed = %d, want 1", got)
	}
	seen := make(map[EventKind]Event)
	for range 2 {
		select {
		case event := <-events:
			seen[event.Kind] = event
		case <-time.After(time.Second):
			t.Fatal("download completion events not emitted")
		}
	}
	if event, ok := seen[EventDownloadCompleted]; !ok || event.Payload != nil {
		t.Fatalf("completion event = %#v", event)
	}
	if event, ok := seen[EventUsageStatsUpdated]; !ok {
		t.Fatal("usage stats event not emitted")
	} else if stats, ok := event.Payload.(storage.UsageStats); !ok || stats.DownloadsCompleted != 1 {
		t.Fatalf("usage stats event = %#v", event)
	}
}

func TestStartupBootstrapsForcedFileLogging(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		logging.Close()
		storage.Close()
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})
	logPath := filepath.Join(dir, "service.log")
	if err := os.WriteFile(filepath.Join(dir, "mod-downloader.toml"), []byte(`
[logging]
force_file = true
file = "`+filepath.ToSlash(logPath)+`"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := New(Options{
		Runtime:  RuntimeOptions{WorkDir: dir, CachePath: filepath.Join(dir, storage.CacheFileName)},
		LogLevel: slog.LevelDebug,
	})
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	logging.Info("bootstrap file marker")
	svc.Close()
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "load config started") || !strings.Contains(string(data), "bootstrap file marker") {
		t.Fatalf("bootstrap log = %q", data)
	}
}

func TestStartupConfiguresRemoteModIDConcurrency(t *testing.T) {
	previousConfigure := configureRemoteModIDConcurrency
	configured := make(chan int, 1)
	configureRemoteModIDConcurrency = func(limit int) {
		configured <- limit
	}
	t.Cleanup(func() {
		configureRemoteModIDConcurrency = previousConfigure
		downloader.ConfigureSimpleMode(false)
	})

	dir := t.TempDir()
	svc := New(Options{
		Config: &configs.Config{
			Prefers:   configs.Preferences{SimpleMode: true},
			Downloads: configs.DownloadConfig{ConcurrentDownloads: 3},
		},
		Runtime: RuntimeOptions{
			WorkDir:   dir,
			CachePath: filepath.Join(dir, storage.CacheFileName),
		},
	})
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(svc.Close)
	if !modbridge.SimpleModeEnabled() {
		t.Fatal("startup simple mode = false, want true")
	}

	select {
	case limit := <-configured:
		if limit != 3 {
			t.Fatalf("remote mod ID concurrency = %d, want 3", limit)
		}
	case <-time.After(time.Second):
		t.Fatal("remote mod ID concurrency was not configured during startup")
	}
}

func TestStartupBootstrapsDisabledLoggingWithoutCreatingFile(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		logging.Close()
		storage.Close()
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.WriteFile(filepath.Join(dir, "mod-downloader.toml"), []byte(`
[logging]
disabled = true
force_file = true
file = "`+filepath.ToSlash(filepath.Join(dir, "disabled.log"))+`"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := New(Options{Runtime: RuntimeOptions{WorkDir: dir, CachePath: filepath.Join(dir, storage.CacheFileName)}})
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	logging.Error("hidden bootstrap marker")
	svc.Close()
	if _, err := os.Stat(filepath.Join(dir, "disabled.log")); !os.IsNotExist(err) {
		t.Fatalf("disabled bootstrap log stat error = %v, want not exist", err)
	}
}

func TestServiceStartupShutdownIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	svc := New(Options{
		Config:  disabledLoggingConfig(),
		Runtime: RuntimeOptions{CachePath: filepath.Join(dir, storage.CacheFileName)},
	})
	ctx := context.Background()
	if err := svc.Startup(ctx); err != nil {
		t.Fatal(err)
	}
	if err := svc.Startup(ctx); err != nil {
		t.Fatalf("second Startup: %v", err)
	}
	svc.Shutdown()
	svc.Shutdown()
	svc.Close()
}

func TestStartupWritesDefaultLogOutsideWorkDir(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheDir)
	t.Setenv("HOME", cacheDir)
	t.Setenv("LOCALAPPDATA", cacheDir)
	dir := t.TempDir()
	t.Chdir(dir)
	workDir := filepath.Join(dir, "mods")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}
	enabled := true
	svc := New(Options{
		Config: &configs.Config{Logging: configs.LoggingConfig{
			Enabled:   &enabled,
			ForceFile: true,
		}},
		Runtime: RuntimeOptions{
			WorkDir:   workDir,
			CachePath: filepath.Join(dir, storage.CacheFileName),
		},
	})
	t.Cleanup(func() {
		logging.Close()
		storage.Close()
	})
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	logging.Info("default path marker")
	svc.Close()

	logPath := filepath.Join(cacheDir, "mod-downloader", "logs", logging.DefaultFileName)
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read default log: %v", err)
	}
	if !strings.Contains(string(data), "default path marker") {
		t.Fatalf("default log = %q", data)
	}
	if _, err := os.Stat(filepath.Join(workDir, logging.DefaultFileName)); !os.IsNotExist(err) {
		t.Fatalf("work dir log stat error = %v, want not exist", err)
	}
}

func TestStartupFailsOnCorruptConfigAndPreservesFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Cleanup(providers.StartBackgroundTasks)

	original := []byte("[preferences\nminecraft_dir = \"/keep\"\n")
	configPath := filepath.Join(dir, "mod-downloader.toml")
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	svc := New(Options{
		Runtime: RuntimeOptions{CachePath: filepath.Join(dir, storage.CacheFileName)},
	})
	if err := svc.Startup(context.Background()); err == nil {
		t.Fatal("Startup succeeded with a corrupt config file")
	}
	// A failed startup must never write the config back, even when the app
	// tries the normal shutdown path afterwards.
	svc.Shutdown()
	svc.Close()
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != string(original) {
		t.Fatalf("config after failed startup = %q, want original %q", contents, original)
	}
}

func TestStartupFailsWhenStorageCannotOpen(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Cleanup(providers.StartBackgroundTasks)

	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(Options{
		Config:  disabledLoggingConfig(),
		Runtime: RuntimeOptions{CachePath: filepath.Join(blocker, storage.CacheFileName)},
	})
	if err := svc.Startup(context.Background()); err == nil {
		t.Fatal("Startup succeeded with an unusable cache path")
	}
	svc.Close()
}

func TestServiceCloseAfterStartupFailure(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	previousFetch := fetchMinecraftReleaseVersions
	t.Cleanup(func() { fetchMinecraftReleaseVersions = previousFetch })
	fetchMinecraftReleaseVersions = func() ([]string, error) {
		return nil, errors.New("manifest unavailable")
	}

	svc := New(Options{
		Config:                disabledLoggingConfig(),
		Runtime:               RuntimeOptions{CachePath: filepath.Join(dir, storage.CacheFileName)},
		LoadMinecraftReleases: true,
	})
	if err := svc.Startup(context.Background()); err == nil {
		t.Fatal("Startup succeeded with a failing manifest fetch")
	}
	// Failure must leave no open files or background tasks behind; Close is
	// idempotent and must be safe after the internal cleanup.
	svc.Close()
	svc.Close()

	fetchMinecraftReleaseVersions = func() ([]string, error) { return []string{"1.21.1"}, nil }
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatalf("retry after failed startup: %v", err)
	}
	svc.Close()
}

func TestStartupRejectsSecondActiveService(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Cleanup(providers.StartBackgroundTasks)

	first := New(Options{
		Config:  disabledLoggingConfig(),
		Runtime: RuntimeOptions{CachePath: filepath.Join(dir, "first", storage.CacheFileName)},
	})
	if err := first.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}

	second := New(Options{
		Config:  disabledLoggingConfig(),
		Runtime: RuntimeOptions{CachePath: filepath.Join(dir, "second", storage.CacheFileName)},
	})
	if err := second.Startup(context.Background()); err == nil || !strings.Contains(err.Error(), "already running") {
		t.Fatalf("second Startup error = %v, want active-service rejection", err)
	}
	if second.lifecycleState != serviceStateIdle {
		t.Fatalf("second lifecycle = %v, want idle", second.lifecycleState)
	}

	first.Close()
	if err := second.Startup(context.Background()); err != nil {
		t.Fatalf("Startup after first Close: %v", err)
	}
	second.Close()
}

func TestServiceBackgroundTasksDrainBeforeStop(t *testing.T) {
	svc := New(Options{})
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	svc.startServiceBackground(func() {
		close(started)
		<-release
	})
	<-started
	go func() {
		svc.stopServiceBackground()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("stop returned before service background task finished")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stop did not return after service background task finished")
	}
	rejected := make(chan struct{}, 1)
	svc.startServiceBackground(func() { rejected <- struct{}{} })
	select {
	case <-rejected:
		t.Fatal("service background task accepted after stop")
	case <-time.After(20 * time.Millisecond):
	}
}

func disabledLoggingConfig() *configs.Config {
	enabled := false
	return &configs.Config{Logging: configs.LoggingConfig{Enabled: &enabled}}
}

// waitProviderBackgroundTasks drains provider work before a test closes
// storage, then restores the previous accepting state. Register it after the
// storage cleanup so it runs first (t.Cleanup is LIFO).
func waitProviderBackgroundTasks(t *testing.T) {
	t.Helper()
	t.Cleanup(providers.DrainBackgroundTasks)
}

func TestLoadMinecraftReleaseVersionsRefreshesExpiredCacheAndFallsBackOnError(t *testing.T) {
	t.Cleanup(func() { global.SetMinecraftReleaseVersions(nil) })
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	expiredAt := time.Now().Add(-minecraftReleaseCacheTTL - time.Minute).Unix()
	if err := storage.SetMinecraftReleaseVersions([]string{"stale"}, expiredAt); err != nil {
		t.Fatal(err)
	}
	previousFetch := fetchMinecraftReleaseVersions
	t.Cleanup(func() { fetchMinecraftReleaseVersions = previousFetch })

	fetchMinecraftReleaseVersions = func() ([]string, error) {
		return nil, errors.New("temporary manifest failure")
	}
	if err := loadMinecraftReleaseVersions(); err != nil {
		t.Fatalf("stale fallback returned error: %v", err)
	}
	if got := global.GetMinecraftReleaseVersions(); len(got) != 1 || got[0] != "stale" {
		t.Fatalf("stale global releases = %#v", got)
	}

	fetchMinecraftReleaseVersions = func() ([]string, error) {
		return []string{"1.22", "1.21.1"}, nil
	}
	if err := loadMinecraftReleaseVersions(); err != nil {
		t.Fatal(err)
	}
	got, updatedAt := storage.GetMinecraftReleaseVersions()
	if len(got) != 2 || got[0] != "1.22" || updatedAt <= expiredAt {
		t.Fatalf("refreshed release cache = %#v, %d", got, updatedAt)
	}
}

func TestCoreDoesNotDependOnWailsRuntime(t *testing.T) {
	packages := []string{
		"github.com/link-fgfgui/mod-downloader-core/appcore",
		"github.com/link-fgfgui/mod-downloader-core/httpserver",
	}
	args := append([]string{"list", "-deps"}, packages...)
	out, err := exec.Command("go", args...).Output()
	if err != nil {
		t.Fatalf("go list deps failed: %v", err)
	}
	for _, pkg := range strings.Fields(string(out)) {
		if pkg == "github.com/wailsapp/wails/v2/pkg/runtime" {
			t.Fatalf("core dependency tree includes Wails runtime")
		}
	}
}

func TestStartupNoConfigFileUsesEnvironmentOnly(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		storage.Close()
		global.SetMinecraftDir("")
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	if err := os.WriteFile(filepath.Join(dir, "mod-downloader.toml"), []byte(`
[keys]
curseforge_api_key = "from-file"
[preferences]
minecraft_dir = "/from-file"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KEYS_CF_API_KEY", "from-env")

	cachePath := filepath.Join(t.TempDir(), "cache", storage.CacheFileName)
	svc := New(Options{Runtime: RuntimeOptions{
		NoConfigFile: true,
		CachePath:    cachePath,
	}})
	if err := svc.Startup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if svc.Config().Keys.CurseforgeApiKey != "from-env" {
		t.Fatalf("curseforge key = %q, want from-env", svc.Config().Keys.CurseforgeApiKey)
	}
	if got := global.GetMinecraftDir(); got != "" {
		t.Fatalf("global minecraft dir = %q, want empty", got)
	}
	svc.Close()
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("cache path after close: %v", err)
	}
}

func TestFavoriteServiceWorkflow(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	svc := New(Options{})
	list := svc.CreateFavoriteList("Performance")
	if list.ID == "" || list.Name != "Performance" {
		t.Fatalf("created list = %#v", list)
	}

	renamed := svc.RenameFavoriteList(list.ID, "Performance Mods")
	if renamed.ID != list.ID || renamed.Name != "Performance Mods" {
		t.Fatalf("renamed list = %#v", renamed)
	}

	mod := svc.AddFavoriteMod(storage.FavoriteMod{
		ListID:           list.ID,
		Platform:         "Modrinth",
		ModID:            "Sodium",
		MinecraftVersion: "1.21.1",
		ModLoader:        "Fabric",
		Title:            "Sodium",
	})
	if mod.ID == "" || mod.Platform != "modrinth" || mod.ModID != "sodium" {
		t.Fatalf("added mod = %#v", mod)
	}
	if mods := svc.ListFavoriteMods(list.ID); len(mods) != 1 || mods[0].ID != mod.ID {
		t.Fatalf("listed mods = %#v", mods)
	}

	if !svc.RemoveFavoriteMod(list.ID, "Modrinth", "Sodium", "1.21.1", "Fabric") {
		t.Fatal("RemoveFavoriteMod returned false")
	}
	if mods := svc.ListFavoriteMods(list.ID); len(mods) != 0 {
		t.Fatalf("mods after remove = %#v", mods)
	}
	if !svc.DeleteFavoriteList(list.ID) {
		t.Fatal("DeleteFavoriteList returned false")
	}
	if lists := svc.ListFavoriteLists(); len(lists) != 0 {
		t.Fatalf("lists after delete = %#v", lists)
	}
}

func TestFavoriteBulkAddCopiesSelectedMods(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	svc := New(Options{})
	source := svc.CreateFavoriteList("Source")
	target := svc.CreateFavoriteList("Target")
	mod := svc.AddFavoriteMod(storage.FavoriteMod{
		ListID:           source.ID,
		Platform:         "Modrinth",
		ModID:            "Sodium",
		MinecraftVersion: "1.21.1",
		ModLoader:        "Fabric",
		Title:            "Sodium",
	})
	if mod.ID == "" {
		t.Fatalf("added source mod = %#v", mod)
	}

	result := svc.AddFavoriteModsToLists(FavoriteBulkAddRequest{
		TargetListIDs: []string{target.ID, "missing", target.ID},
		Mods:          []storage.FavoriteMod{mod},
	})
	if result.Added != 1 || result.Updated != 0 || result.Skipped != 1 || len(result.Errors) != 1 {
		t.Fatalf("bulk add result = %#v", result)
	}
	targetMods := svc.ListFavoriteMods(target.ID)
	if len(targetMods) != 1 || targetMods[0].ID == mod.ID || targetMods[0].Title != "Sodium" {
		t.Fatalf("target mods = %#v, source = %#v", targetMods, mod)
	}

	mod.Title = "Sodium Updated"
	result = svc.AddFavoriteModsToLists(FavoriteBulkAddRequest{
		TargetListIDs: []string{target.ID},
		Mods:          []storage.FavoriteMod{mod},
	})
	if result.Added != 0 || result.Updated != 1 || result.Skipped != 0 {
		t.Fatalf("bulk update result = %#v", result)
	}
	targetMods = svc.ListFavoriteMods(target.ID)
	if len(targetMods) != 1 || targetMods[0].Title != "Sodium Updated" {
		t.Fatalf("target mods after update = %#v", targetMods)
	}
	if stats := svc.GetUsageStats(); stats.FavoritesAdded != 2 {
		t.Fatalf("favorite usage stats = %#v, want 2 additions", stats)
	}
}

func TestFavoriteListCopyUsesResolvedReferences(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	svc := New(Options{})
	source := svc.CreateFavoriteList("Source")
	child := svc.CreateFavoriteList("Child")
	target := svc.CreateFavoriteList("Target")
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: source.ID, Platform: "modrinth", ModID: "sodium", Title: "Sodium"})
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: child.ID, Platform: "curseforge", ModID: "jei", Title: "JEI"})
	ref := svc.AddFavoriteListReference(source.ID, child.ID)
	if ref.ID == "" {
		t.Fatal("AddFavoriteListReference returned empty ref")
	}

	result := svc.CopyFavoriteListToList(FavoriteListCopyRequest{SourceListID: source.ID, TargetListID: target.ID})
	if result.Added != 2 || result.Updated != 0 || result.Skipped != 0 {
		t.Fatalf("copy result = %#v", result)
	}
	targetMods := svc.ListFavoriteMods(target.ID)
	if len(targetMods) != 2 {
		t.Fatalf("target mods = %#v", targetMods)
	}

	self := svc.CopyFavoriteListToList(FavoriteListCopyRequest{SourceListID: source.ID, TargetListID: source.ID})
	if self.Skipped != 2 || len(self.Errors) != 1 {
		t.Fatalf("self copy result = %#v", self)
	}
}

func TestFavoriteListReferenceServiceMethods(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	svc := New(Options{})
	parent := svc.CreateFavoriteList("Parent")
	child := svc.CreateFavoriteList("Child")

	ref := svc.AddFavoriteListReference(parent.ID, child.ID)
	if ref.ID == "" || ref.ParentListID != parent.ID || ref.ChildListID != child.ID {
		t.Fatalf("ref = %#v", ref)
	}
	refs := svc.ListFavoriteListRefs(parent.ID)
	if len(refs) != 1 || refs[0].ID != ref.ID {
		t.Fatalf("refs = %#v", refs)
	}
	if cycle := svc.AddFavoriteListReference(child.ID, parent.ID); cycle.ID != "" {
		t.Fatalf("cycle ref = %#v", cycle)
	}
	if !svc.RemoveFavoriteListReference(parent.ID, child.ID) {
		t.Fatal("RemoveFavoriteListReference returned false")
	}
	if refs := svc.ListFavoriteListRefs(parent.ID); len(refs) != 0 {
		t.Fatalf("refs after remove = %#v", refs)
	}
}

func TestFavoriteListMetadataAndReorderServiceMethods(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	svc := New(Options{})
	listA := svc.CreateFavoriteList("A")
	listB := svc.CreateFavoriteList("B")
	updated := svc.UpdateFavoriteListMetadata(storage.FavoriteList{
		ID:        listA.ID,
		IconKind:  "mdi",
		IconValue: "mdi-star",
		Pinned:    true,
	})
	if updated.ID != listA.ID || updated.IconKind != "mdi" || !updated.Pinned {
		t.Fatalf("updated list metadata = %#v", updated)
	}
	if !svc.ReorderFavoriteLists([]string{listB.ID, listA.ID}) {
		t.Fatal("ReorderFavoriteLists returned false")
	}
	lists := svc.ListFavoriteLists()
	if len(lists) != 2 || lists[0].ID != listA.ID || lists[0].SortOrder <= lists[1].SortOrder {
		t.Fatalf("lists after pinned metadata and reorder = %#v", lists)
	}
}

func TestFavoriteCrossVersionCopyPreviewAndApplyAllMatch(t *testing.T) {
	svc := openFavoriteCrossVersionCopyService(t)
	storeFavoriteCrossVersionCopyProject(t, "Modrinth", "sodium", "sodium", []models.ModVersion{
		favoriteCrossVersionCopyVersion("Modrinth", "sodium", "sodium-1.21.1", "1.21.1", "fabric", 20),
	})

	source := svc.CreateFavoriteListForScope("Source", "1.20.1", "fabric")
	sourceMod := svc.AddFavoriteMod(storage.FavoriteMod{
		ListID:           source.ID,
		Platform:         "modrinth",
		ModID:            "sodium",
		VersionID:        "sodium-1.20.1",
		MinecraftVersion: "1.20.1",
		ModLoader:        "fabric",
		Title:            "Sodium",
		Slug:             "sodium",
	})

	req := FavoriteCrossVersionCopyRequest{
		SourceListID:     source.ID,
		MinecraftVersion: "1.21.1",
		ModLoader:        "Fabric",
	}
	preview := svc.PreviewFavoriteListCrossVersionCopy(req)
	if len(preview.Matched) != 1 || len(preview.Conflicts) != 0 || len(preview.Errors) != 0 {
		t.Fatalf("preview = %#v", preview)
	}
	if preview.TargetListName != source.Name || preview.MinecraftVersion != "1.21.1" || preview.ModLoader != "fabric" || preview.NameConflict {
		t.Fatalf("preview target = %#v", preview)
	}
	if lists := svc.ListFavoriteLists(); len(lists) != 1 {
		t.Fatalf("preview wrote a target list: %#v", lists)
	}
	if got := preview.Matched[0].Target; got.ListID != "" || got.VersionID != "sodium-1.21.1" || got.MinecraftVersion != "1.21.1" || got.ModLoader != "fabric" {
		t.Fatalf("matched target = %#v", got)
	}

	applied := svc.ApplyFavoriteListCrossVersionCopy(req)
	if !applied.Applied || applied.Result.Added != 1 || applied.Result.Updated != 0 || applied.Result.Skipped != 0 {
		t.Fatalf("apply result = %#v", applied)
	}
	if applied.TargetList.ID == "" || applied.TargetList.Name != source.Name || applied.TargetList.MinecraftVersion != "1.21.1" || applied.TargetList.ModLoader != "fabric" {
		t.Fatalf("target list = %#v", applied.TargetList)
	}
	targetMods := svc.ListFavoriteMods(applied.TargetList.ID)
	if len(targetMods) != 1 || targetMods[0].ID == sourceMod.ID || targetMods[0].VersionID != "sodium-1.21.1" {
		t.Fatalf("target mods = %#v, source = %#v", targetMods, sourceMod)
	}
	sourceMods := svc.ListFavoriteMods(source.ID)
	if len(sourceMods) != 1 || sourceMods[0].VersionID != "sodium-1.20.1" || sourceMods[0].MinecraftVersion != "1.20.1" {
		t.Fatalf("source mods changed = %#v", sourceMods)
	}
}

func TestFavoriteCrossVersionCopyPartialConflictRequiresIgnore(t *testing.T) {
	svc := openFavoriteCrossVersionCopyService(t)
	storeFavoriteCrossVersionCopyProject(t, "Modrinth", "sodium", "sodium", []models.ModVersion{
		favoriteCrossVersionCopyVersion("Modrinth", "sodium", "sodium-1.21.1", "1.21.1", "fabric", 20),
	})
	storeFavoriteCrossVersionCopyProject(t, "Modrinth", "lithium", "lithium", []models.ModVersion{
		favoriteCrossVersionCopyVersion("Modrinth", "lithium", "lithium-1.20.1", "1.20.1", "fabric", 10),
	})

	source := svc.CreateFavoriteListForScope("Source", "1.20.1", "fabric")
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: source.ID, Platform: "modrinth", ModID: "sodium", Slug: "sodium", Title: "Sodium"})
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: source.ID, Platform: "modrinth", ModID: "lithium", Slug: "lithium", Title: "Lithium"})

	req := FavoriteCrossVersionCopyRequest{
		SourceListID:     source.ID,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}
	preview := svc.PreviewFavoriteListCrossVersionCopy(req)
	if len(preview.Matched) != 1 || len(preview.Conflicts) != 1 {
		t.Fatalf("preview = %#v", preview)
	}

	blocked := svc.ApplyFavoriteListCrossVersionCopy(req)
	if blocked.Applied || blocked.Result.Added != 0 || blocked.Result.Updated != 0 {
		t.Fatalf("blocked apply result = %#v", blocked)
	}
	if lists := svc.ListFavoriteLists(); len(lists) != 1 {
		t.Fatalf("non-ignore apply wrote target list: %#v", lists)
	}

	req.IgnoreConflicts = true
	applied := svc.ApplyFavoriteListCrossVersionCopy(req)
	if !applied.Applied || applied.Result.Added != 1 || applied.Result.Skipped != 1 {
		t.Fatalf("ignore-conflict apply result = %#v", applied)
	}
	targetMods := svc.ListFavoriteMods(applied.TargetList.ID)
	if len(targetMods) != 1 || targetMods[0].ModID != "sodium" {
		t.Fatalf("target mods = %#v", targetMods)
	}
}

func TestFavoriteCrossVersionCopyAllConflictWritesNothing(t *testing.T) {
	svc := openFavoriteCrossVersionCopyService(t)
	storeFavoriteCrossVersionCopyProject(t, "Modrinth", "lithium", "lithium", []models.ModVersion{
		favoriteCrossVersionCopyVersion("Modrinth", "lithium", "lithium-1.20.1", "1.20.1", "fabric", 10),
	})

	source := svc.CreateFavoriteListForScope("Source", "1.20.1", "fabric")
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: source.ID, Platform: "modrinth", ModID: "lithium", Slug: "lithium", Title: "Lithium"})

	applied := svc.ApplyFavoriteListCrossVersionCopy(FavoriteCrossVersionCopyRequest{
		SourceListID:     source.ID,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		IgnoreConflicts:  true,
	})
	if applied.Result.Added != 0 || applied.Result.Updated != 0 || applied.Result.Skipped != 1 {
		t.Fatalf("all-conflict apply result = %#v", applied)
	}
	if lists := svc.ListFavoriteLists(); len(lists) != 1 {
		t.Fatalf("all-conflict apply wrote target list: %#v", lists)
	}
}

func TestFavoriteCrossVersionCopyMissingProjectLookupConflict(t *testing.T) {
	svc := openFavoriteCrossVersionCopyService(t)
	stubFavoriteCrossVersionCopyProjectLookup(t)
	source := svc.CreateFavoriteListForScope("Source", "1.20.1", "fabric")
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: source.ID, Platform: "modrinth", ModID: "missing-project", Title: "Missing"})

	preview := svc.PreviewFavoriteListCrossVersionCopy(FavoriteCrossVersionCopyRequest{
		SourceListID:     source.ID,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	})
	if len(preview.Matched) != 0 || len(preview.Conflicts) != 1 || preview.Conflicts[0].Reason != "project not found" {
		t.Fatalf("preview = %#v", preview)
	}
}

func TestFavoriteCrossVersionCopyUsesReferencedListContents(t *testing.T) {
	svc := openFavoriteCrossVersionCopyService(t)
	storeFavoriteCrossVersionCopyProject(t, "CurseForge", "238222", "jei", []models.ModVersion{
		favoriteCrossVersionCopyVersion("CurseForge", "238222", "jei-1.21.1", "1.21.1", "neoforge", 20),
	})

	source := svc.CreateFavoriteListForScope("Source", "1.20.1", "neoforge")
	child := svc.CreateFavoriteListForScope("Child", "1.20.1", "neoforge")
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: child.ID, Platform: "curseforge", ModID: "238222", Slug: "jei", Title: "JEI"})
	if ref := svc.AddFavoriteListReference(source.ID, child.ID); ref.ID == "" {
		t.Fatal("AddFavoriteListReference returned empty ref")
	}

	preview := svc.PreviewFavoriteListCrossVersionCopy(FavoriteCrossVersionCopyRequest{
		SourceListID:     source.ID,
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
	})
	if len(preview.Matched) != 1 || len(preview.Conflicts) != 0 {
		t.Fatalf("preview = %#v", preview)
	}
	if !preview.Matched[0].Source.Referenced || preview.Matched[0].Source.SourceListID != child.ID {
		t.Fatalf("matched source did not come from referenced list: %#v", preview.Matched[0].Source)
	}
}

func TestFavoriteCrossVersionCopyRejectsTargetNameConflict(t *testing.T) {
	svc := openFavoriteCrossVersionCopyService(t)
	storeFavoriteCrossVersionCopyProject(t, "Modrinth", "sodium", "sodium", []models.ModVersion{
		favoriteCrossVersionCopyVersion("Modrinth", "sodium", "sodium-1.21.1", "1.21.1", "fabric", 20),
	})
	source := svc.CreateFavoriteListForScope("Performance", "1.20.1", "fabric")
	svc.AddFavoriteMod(storage.FavoriteMod{ListID: source.ID, Platform: "modrinth", ModID: "sodium", Slug: "sodium"})
	req := FavoriteCrossVersionCopyRequest{
		SourceListID:     source.ID,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}
	preview := svc.PreviewFavoriteListCrossVersionCopy(req)
	if preview.NameConflict || len(preview.Errors) != 0 {
		t.Fatalf("initial preview = %#v", preview)
	}
	conflicting := svc.CreateFavoriteListForScope("performance", "1.21.1", "fabric")
	if conflicting.ID == "" {
		t.Fatal("failed to create concurrent target")
	}
	applied := svc.ApplyFavoriteListCrossVersionCopy(req)
	if applied.Applied || !applied.Preview.NameConflict || len(applied.Result.Errors) == 0 {
		t.Fatalf("conflicting apply = %#v", applied)
	}
	if lists := svc.ListFavoriteLists(); len(lists) != 2 {
		t.Fatalf("conflicting apply created another list = %#v", lists)
	}
}

func TestFavoriteCrossVersionCopyRejectsSourceScope(t *testing.T) {
	svc := openFavoriteCrossVersionCopyService(t)
	source := svc.CreateFavoriteListForScope("Performance", "1.21.1", "Fabric")
	preview := svc.PreviewFavoriteListCrossVersionCopy(FavoriteCrossVersionCopyRequest{
		SourceListID:     source.ID,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	})
	if len(preview.Errors) != 1 || preview.Errors[0] != "target scope must differ from source scope" {
		t.Fatalf("same-scope preview = %#v", preview)
	}
}

func openFavoriteCrossVersionCopyService(t *testing.T) *Service {
	t.Helper()
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		providers.DrainBackgroundTasks()
		storage.Close()
	})
	return New(Options{})
}

func storeFavoriteCrossVersionCopyProject(t *testing.T, platform, projectID, slug string, versions []models.ModVersion) {
	t.Helper()
	if err := storage.UpsertModPlatform(models.ModProject{
		ID:        models.ProjectKey(platform, projectID),
		Platform:  platform,
		ProjectID: projectID,
		Slug:      slug,
		Title:     projectID,
	}); err != nil {
		t.Fatal(err)
	}
	scopes := []storage.ModPlatformVersionScope{
		{MinecraftVersion: "1.21.1", ModLoader: "fabric"},
		{MinecraftVersion: "1.21.1", ModLoader: "neoforge"},
	}
	if err := storage.SetPlatformVersionSnapshot(platform, projectID, versions, time.Now().Unix(), scopes); err != nil {
		t.Fatal(err)
	}
}

func favoriteCrossVersionCopyVersion(platform, projectID, versionID, minecraftVersion, modLoader string, publishedAt int64) models.ModVersion {
	return models.ModVersion{
		ID:           versionID,
		Platform:     platform,
		ProjectID:    projectID,
		VersionID:    versionID,
		Name:         versionID,
		PublishedAt:  publishedAt,
		GameVersions: []string{minecraftVersion},
		Loaders:      []string{modLoader},
	}
}

func stubFavoriteCrossVersionCopyProjectLookup(t *testing.T) {
	t.Helper()
	lookupByID := favoriteCrossVersionCopyLookupProjectByID
	lookupBySlug := favoriteCrossVersionCopyLookupProjectBySlug
	favoriteCrossVersionCopyLookupProjectByID = func(platform, projectID, mcVersion, modLoader string) (models.ModProject, bool) {
		return models.ModProject{}, false
	}
	favoriteCrossVersionCopyLookupProjectBySlug = func(platform, slug, mcVersion, modLoader string) (models.ModProject, bool) {
		return models.ModProject{}, false
	}
	t.Cleanup(func() {
		favoriteCrossVersionCopyLookupProjectByID = lookupByID
		favoriteCrossVersionCopyLookupProjectBySlug = lookupBySlug
	})
}

func TestSaveCacheDirPreferencePersistsAndReopensDatabase(t *testing.T) {
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
			t.Fatal(err)
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

	nextDir := filepath.Join(t.TempDir(), "custom-cache")
	view := svc.SaveCacheDirPreference(nextDir)
	if got := minecraft.ExpandPathWithEnv(svc.Config().Runtime.CacheDir); got != nextDir {
		t.Fatalf("config cache dir = %q, want %q", got, nextDir)
	}
	wantCachePath := filepath.Join(nextDir, storage.CacheFileName)
	if got := minecraft.ExpandPathWithEnv(view.CachePath); got != wantCachePath {
		t.Fatalf("settings cache path = %q, want %q", got, wantCachePath)
	}
	if _, err := os.Stat(storage.UserDataPathForCachePath(wantCachePath)); err != nil {
		t.Fatalf("user database at custom cache dir: %v", err)
	}

	loaded, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := minecraft.ExpandPathWithEnv(loaded.Runtime.CacheDir); got != nextDir {
		t.Fatalf("persisted cache dir = %q, want %q", got, nextDir)
	}
}

func TestSaveUnusedDependencyCleanupSettingsPersists(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	svc := New(Options{Config: &configs.Config{}})
	if svc.GetSettings().AutoScanUnusedDependencies {
		t.Fatal("default auto scan unused dependencies = true, want false")
	}
	view := svc.SaveUnusedDependencyCleanupSettings(appstructs.SaveUnusedDependencyCleanupSettingsRequest{
		AutoScanUnusedDependencies: true,
	})
	if !view.AutoScanUnusedDependencies {
		t.Fatal("view auto scan unused dependencies = false, want true")
	}
	if svc.Config().Prefers.AutoScanUnusedDependencies == nil || !*svc.Config().Prefers.AutoScanUnusedDependencies {
		t.Fatalf("config auto scan unused dependencies = %#v, want true", svc.Config().Prefers.AutoScanUnusedDependencies)
	}

	loaded, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Prefers.AutoScanUnusedDependenciesValue() {
		t.Fatal("persisted auto scan unused dependencies = false, want true")
	}
}

func TestSaveSimpleModeSettingsPersistsAndAppliesImmediately(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		downloader.ConfigureSimpleMode(false)
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	svc := New(Options{Config: &configs.Config{}})
	if svc.GetSettings().SimpleMode {
		t.Fatal("default simple mode = true, want false")
	}
	view := svc.SaveSimpleModeSettings(appstructs.SaveSimpleModeSettingsRequest{SimpleMode: true})
	if !view.SimpleMode || !svc.Config().Prefers.SimpleMode || !modbridge.SimpleModeEnabled() {
		t.Fatalf("enabled simple mode was not applied: view=%#v config=%#v runtime=%v", view, svc.Config().Prefers, modbridge.SimpleModeEnabled())
	}

	loaded, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Prefers.SimpleMode {
		t.Fatal("persisted simple mode = false, want true")
	}

	view = svc.SaveSimpleModeSettings(appstructs.SaveSimpleModeSettingsRequest{SimpleMode: false})
	if view.SimpleMode || modbridge.SimpleModeEnabled() {
		t.Fatalf("disabled simple mode remained active: view=%#v runtime=%v", view, modbridge.SimpleModeEnabled())
	}
}

func TestSaveLanguagePersistsAndNormalizes(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	svc := New(Options{Config: &configs.Config{}})
	if got := svc.GetPreferences().Language; got != configs.LanguageSystem.String() {
		t.Fatalf("default preference language = %q, want system", got)
	}
	if got := svc.SaveLanguage("zh-CN"); got != configs.LanguageChinese.String() {
		t.Fatalf("SaveLanguage(zh-CN) = %q, want zh", got)
	}
	if got := svc.GetSettings().Language; got != configs.LanguageChinese.String() {
		t.Fatalf("settings language = %q, want zh", got)
	}

	loaded, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := loaded.Prefers.Language.Normalized(); got != configs.LanguageChinese {
		t.Fatalf("persisted language = %q, want zh", got)
	}
	if got := svc.SaveLanguage("invalid"); got != configs.LanguageSystem.String() {
		t.Fatalf("SaveLanguage(invalid) = %q, want system", got)
	}
}

func TestSaveMCIMSettingsPersistsAndReconfiguresProviders(t *testing.T) {
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		global.SetModrinthClient(nil)
		global.SetCurseForgeClient(nil)
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	svc := New(Options{Config: &configs.Config{}})
	if svc.GetSettings().MCIMEnabled {
		t.Fatal("default MCIM enabled = true, want false")
	}
	view := svc.SaveMCIMSettings(appstructs.SaveMCIMSettingsRequest{MCIMEnabled: true})
	if !view.MCIMEnabled || global.GetCurseForgeClient() == nil {
		t.Fatalf("MCIM settings view = %#v", view)
	}
	client := global.GetModrinthClient()
	if client == nil || client.BaseURL == nil || client.BaseURL.String() != mcimModrinthBaseURL {
		t.Fatalf("Modrinth MCIM client = %#v", client)
	}

	loaded, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Prefers.MCIMEnabledValue() {
		t.Fatal("persisted MCIM enabled = false, want true")
	}

	view = svc.SaveMCIMSettings(appstructs.SaveMCIMSettingsRequest{MCIMEnabled: false})
	if view.MCIMEnabled || global.GetCurseForgeClient() != nil {
		t.Fatalf("disabled MCIM settings view = %#v", view)
	}
	client = global.GetModrinthClient()
	if client == nil || client.BaseURL == nil || client.BaseURL.String() == mcimModrinthBaseURL {
		t.Fatalf("Modrinth official client = %#v", client)
	}
}

func TestDefaultCurseforgeAPIKeyFallback(t *testing.T) {
	prevDefault := configs.DefaultCurseforgeAPIKey
	t.Cleanup(func() {
		configs.DefaultCurseforgeAPIKey = prevDefault
		global.SetCurseForgeClient(nil)
		global.SetModrinthClient(nil)
	})
	configs.DefaultCurseforgeAPIKey = "compiled-default-key"

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	// Empty user config falls back to compiled default for settings and providers.
	svc := New(Options{Config: &configs.Config{}})
	svc.configureProviderClients()
	sv := svc.GetSettings()
	if !sv.HasCurseforgeKey {
		t.Fatal("HasCurseforgeKey = false with compiled default, want true")
	}
	if sv.CurseforgeKeyMask != MaskKey("compiled-default-key") {
		t.Fatalf("CurseforgeKeyMask = %q, want masked default", sv.CurseforgeKeyMask)
	}
	if global.GetCurseForgeClient() == nil {
		t.Fatal("CurseForge client is nil with compiled default")
	}
	if got := svc.Config().Keys.CurseforgeApiKey; got != "" {
		t.Fatalf("configured key = %q, want empty (default must not be written into config)", got)
	}

	// Explicit user key wins over compiled default.
	svc.config.Keys.CurseforgeApiKey = "user-override-key"
	svc.configureProviderClients()
	sv = svc.GetSettings()
	if sv.CurseforgeKeyMask != MaskKey("user-override-key") {
		t.Fatalf("CurseforgeKeyMask = %q, want user mask", sv.CurseforgeKeyMask)
	}

	// Clear stores empty config and falls back to default again; default is not persisted.
	sv = svc.SaveApiKeys(appstructs.SaveApiKeysRequest{CurseforgeApiKey: ""})
	if svc.Config().Keys.CurseforgeApiKey != "" {
		t.Fatalf("configured key after clear = %q, want empty", svc.Config().Keys.CurseforgeApiKey)
	}
	if !sv.HasCurseforgeKey {
		t.Fatal("HasCurseforgeKey = false after clear with default, want true")
	}
	if sv.CurseforgeKeyMask != MaskKey("compiled-default-key") {
		t.Fatalf("CurseforgeKeyMask after clear = %q, want default mask", sv.CurseforgeKeyMask)
	}
	loaded, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Keys.CurseforgeApiKey != "" {
		t.Fatalf("persisted key = %q, want empty (must not write default into toml)", loaded.Keys.CurseforgeApiKey)
	}
}

func TestSaveNetworkSettingsClampsAndPersists(t *testing.T) {
	previousConfigure := configureRemoteModIDConcurrency
	configured := make(chan int, 1)
	configureRemoteModIDConcurrency = func(limit int) {
		configured <- limit
	}
	t.Cleanup(func() {
		configureRemoteModIDConcurrency = previousConfigure
	})

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	appDir := t.TempDir()
	if err := os.Chdir(appDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})

	svc := New(Options{Config: &configs.Config{}})
	view := svc.SaveNetworkSettings(appstructs.SaveNetworkSettingsRequest{
		FileConcurrency:         100,
		ConcurrentDownloads:     -1,
		AdaptiveFileConcurrency: true,
		TargetDownloadRateMiB:   10,
		VerifySHA1:              true,
		RequestsPerSecond:       101,
	})
	if view.FileConcurrency != configs.MaxFileConcurrency ||
		view.ConcurrentDownloads != configs.DefaultConcurrentDownloads ||
		!view.AdaptiveFileConcurrency ||
		view.TargetDownloadRateMiB != configs.MaxTargetDownloadRateMiB ||
		!view.VerifySHA1 ||
		view.RequestsPerSecond != configs.MaxRequestsPerSecond {
		t.Fatalf("network settings view = %#v", view)
	}

	loaded, err := configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Downloads.FileConcurrency != configs.MaxFileConcurrency ||
		loaded.Downloads.ConcurrentDownloads != configs.DefaultConcurrentDownloads ||
		!loaded.Downloads.AdaptiveFileConcurrency ||
		loaded.Downloads.TargetDownloadRateMiB != configs.MaxTargetDownloadRateMiB ||
		!loaded.Downloads.VerifySHA1 ||
		loaded.API.RequestsPerSecond != configs.MaxRequestsPerSecond {
		t.Fatalf("persisted network settings = %#v", loaded)
	}
	select {
	case limit := <-configured:
		if limit != configs.DefaultConcurrentDownloads {
			t.Fatalf("saved remote mod ID concurrency = %d, want %d", limit, configs.DefaultConcurrentDownloads)
		}
	case <-time.After(time.Second):
		t.Fatal("remote mod ID concurrency was not configured after saving network settings")
	}

	view = svc.SaveNetworkSettings(appstructs.SaveNetworkSettingsRequest{
		FileConcurrency:         configs.DefaultFileConcurrency,
		ConcurrentDownloads:     configs.DefaultConcurrentDownloads,
		AdaptiveFileConcurrency: true,
		TargetDownloadRateMiB:   1.1,
		VerifySHA1:              true,
	})
	if view.TargetDownloadRateMiB != 1.1 {
		t.Fatalf("decimal target download rate in settings view = %v, want 1.1", view.TargetDownloadRateMiB)
	}
	loaded, err = configs.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Downloads.TargetDownloadRateMiB != 1.1 {
		t.Fatalf("persisted decimal target download rate = %v, want 1.1", loaded.Downloads.TargetDownloadRateMiB)
	}
}

func TestSettingsViewExposesRequireCtrlForMultiSelect(t *testing.T) {
	svc := New(Options{Config: &configs.Config{}})
	if svc.GetSettings().RequireCtrlForMultiSelect {
		t.Fatal("RequireCtrlForMultiSelect = true by default, want false")
	}
	svc.config.Prefers.RequireCtrlForMultiSelect = true
	if !svc.GetSettings().RequireCtrlForMultiSelect {
		t.Fatal("RequireCtrlForMultiSelect = false after enabling in config")
	}
}
