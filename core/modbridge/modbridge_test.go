package modbridge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func openModbridgeTestDB(t *testing.T) {
	t.Helper()
	path := filepath.Join(t.TempDir(), storage.CacheFileName)
	if err := storage.OpenAt(path); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	// Provider refreshes run in the background; drain them before closing the
	// DB.
	t.Cleanup(providers.DrainBackgroundTasks)
}

func TestFindVersionByID(t *testing.T) {
	versions := []models.ModVersion{
		{ID: "v1", Name: "old"},
		{ID: "v2", Name: "new"},
	}

	version, ok := FindVersionByID(versions, "v2")
	if !ok || version.Name != "new" {
		t.Fatalf("FindVersionByID() = %#v, %v", version, ok)
	}
	if _, ok := FindVersionByID(versions, "missing"); ok {
		t.Fatal("FindVersionByID() found missing version")
	}
}

func TestApplyPlatformMetadataToModInfoPreservesJarVersion(t *testing.T) {
	info := ApplyPlatformMetadataToModInfo(mcstructs.ModInfo{
		ID:      "jar-id",
		Name:    "Jar Name",
		Version: "1.0.0-local",
	}, models.ModProject{
		Platform:  "Modrinth",
		ProjectID: "project-1",
		Title:     "Online Name",
	}, models.ModVersion{
		VersionID: "online-version-id",
		Version:   "1.0.1-online",
		FileName:  "online-name-1.0.1.jar",
	})

	if info.ID != "jar-id" || info.Name != "Jar Name" || info.Version != "1.0.0-local" {
		t.Fatalf("local metadata changed: %#v", info)
	}
	if info.OnlineVersion != "1.0.1-online" || info.OnlineVersionID != "online-version-id" || info.OnlineFileName != "online-name-1.0.1.jar" {
		t.Fatalf("online version metadata = %#v", info)
	}
}

func TestProjectVersionSHA1Set(t *testing.T) {
	set := projectVersionSHA1Set([]models.ModVersion{
		{SHA1: " ABC "},
		{SHA1: ""},
		{SHA1: "def"},
	})

	if !set["abc"] || !set["def"] || len(set) != 2 {
		t.Fatalf("sha1 set = %#v", set)
	}
}

func TestSelectedVersionModsDirUsesInstanceIDNotDisplayName(t *testing.T) {
	mcDir := t.TempDir()
	global.SetMinecraftDir(mcDir)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
	})

	got := selectedVersionModsDir(mcstructs.VersionInfo{
		ID:   "instance-folder",
		Name: "Display Name",
	})
	want := filepath.Join(mcDir, "versions", "instance-folder", "mods")
	if got != want {
		t.Fatalf("selectedVersionModsDir() = %q, want %q", got, want)
	}
}

func TestResolveInstallTargetPrefersExplicitTargetDir(t *testing.T) {
	global.SetMinecraftDir("")
	global.InvalidateVersions()
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	targetDir := filepath.Join(t.TempDir(), "mods")
	req, instanceID, resolvedDir, ok := ResolveInstallTarget(appstructs.ModDownloadRequest{
		MinecraftVersion: " 1.21.1 ",
		ModLoader:        " Fabric ",
		TargetDir:        targetDir,
	})
	if !ok {
		t.Fatal("ResolveInstallTarget() ok = false")
	}
	if resolvedDir != targetDir || req.TargetDir != targetDir {
		t.Fatalf("target dir = %q/%q, want %q", resolvedDir, req.TargetDir, targetDir)
	}
	if req.MinecraftVersion != "1.21.1" || req.ModLoader != "fabric" {
		t.Fatalf("request version/loader = %q/%q", req.MinecraftVersion, req.ModLoader)
	}
	if !strings.HasPrefix(instanceID, "cli:") || req.InstanceID != instanceID {
		t.Fatalf("instanceID = %q, req.InstanceID = %q", instanceID, req.InstanceID)
	}
}

func TestResolveInstallTargetFallsBackToSelectedInstance(t *testing.T) {
	mcDir := t.TempDir()
	global.SetMinecraftDir(mcDir)
	selected := mcstructs.VersionInfo{
		ID:               "instance",
		Name:             "Display Name",
		MinecraftVersion: "1.20.1",
		ModLoader:        "neoforge",
	}
	global.SetVersions([]mcstructs.VersionInfo{selected})
	global.SetSelectedVersion(selected)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	req, instanceID, targetDir, ok := ResolveInstallTarget(appstructs.ModDownloadRequest{})
	if !ok {
		t.Fatal("ResolveInstallTarget() fallback ok = false")
	}
	if instanceID != "instance" {
		t.Fatalf("instanceID = %q, want instance", instanceID)
	}
	if targetDir != filepath.Join(mcDir, "versions", "instance", "mods") {
		t.Fatalf("targetDir = %q", targetDir)
	}
	if req.MinecraftVersion != "1.20.1" || req.ModLoader != "neoforge" {
		t.Fatalf("request version/loader = %q/%q", req.MinecraftVersion, req.ModLoader)
	}
}

func TestNormalizedModIDs(t *testing.T) {
	got := normalizedModIDs([]string{" Sodium ", "sodium", "Other", ""})
	want := []string{"sodium", "other"}
	if len(got) != len(want) {
		t.Fatalf("normalizedModIDs() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("normalizedModIDs() = %#v, want %#v", got, want)
		}
	}
}

func TestLocalModPathsForModIDsDeduplicatesByPath(t *testing.T) {
	global.ClearLocalMods()
	t.Cleanup(global.ClearLocalMods)

	base := mcstructs.ModInfo{
		FileName: "bundle",
		Path:     "versions/test/mods/bundle.jar",
		SHA1:     "same-sha1",
		Enabled:  true,
	}
	first := base
	first.ID = "firstmod"
	second := base
	second.ID = "secondmod"
	global.UpsertLocalMod(first, "test-instance", "1.20.1", "forge")
	global.UpsertLocalMod(second, "test-instance", "1.20.1", "forge")

	paths := LocalModPathsForModIDs([]string{"firstmod", "secondmod"}, "test-instance")
	if len(paths) != 1 {
		t.Fatalf("LocalModPathsForModIDs() returned %d paths, want 1", len(paths))
	}
	if paths[0].Path != base.Path {
		t.Fatalf("LocalModPathsForModIDs() path = %q, want %q", paths[0].Path, base.Path)
	}
}

func TestLocalModPathsForModIDsIncludesMultiModJarOnPartialTopLevelMatch(t *testing.T) {
	global.ClearLocalMods()
	t.Cleanup(global.ClearLocalMods)

	base := mcstructs.ModInfo{
		FileName: "bundle",
		Path:     "versions/test/mods/bundle.jar",
		SHA1:     "same-sha1",
		Enabled:  true,
	}
	first := base
	first.ID = "tmrv"
	second := base
	second.ID = "jei"
	global.UpsertLocalMod(first, "test-instance", "1.20.1", "forge")
	global.UpsertLocalMod(second, "test-instance", "1.20.1", "forge")

	paths := LocalModPathsForModIDs([]string{"jei"}, "test-instance")
	if len(paths) != 1 {
		t.Fatalf("LocalModPathsForModIDs() returned %d paths, want 1", len(paths))
	}
	if paths[0].Path != base.Path {
		t.Fatalf("LocalModPathsForModIDs() path = %q, want %q", paths[0].Path, base.Path)
	}
}

func TestIncompatibleConflictsCachedFindsInstalledDependency(t *testing.T) {
	openModbridgeTestDB(t)
	global.ClearLocalMods()
	t.Cleanup(global.ClearLocalMods)

	if err := storage.SetPlatformVersionSnapshot("Modrinth", "main", []models.ModVersion{{
		ID:           "main-version",
		VersionID:    "main-version",
		Platform:     "Modrinth",
		ProjectID:    "main",
		Name:         "Main",
		DownloadURL:  "https://example.invalid/main.jar",
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
		Dependencies: []models.ModDependency{{
			DependencyProjectID: "bad",
			DependencyType:      "incompatible",
		}},
	}}, time.Now().Unix(), []storage.ModPlatformVersionScope{{MinecraftVersion: "1.21.1", ModLoader: "fabric"}}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersionSnapshot("Modrinth", "bad", []models.ModVersion{{
		ID:           "bad-version",
		VersionID:    "bad-version",
		Platform:     "Modrinth",
		ProjectID:    "bad",
		Name:         "Bad",
		DownloadURL:  "https://example.invalid/bad.jar",
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
		ModIDs:       []string{"badmod"},
	}}, time.Now().Unix(), []storage.ModPlatformVersionScope{{MinecraftVersion: "1.21.1", ModLoader: "fabric"}}); err != nil {
		t.Fatal(err)
	}
	global.UpsertLocalMod(mcstructs.ModInfo{
		ID:       "badmod",
		FileName: "bad",
		Path:     "mods/bad.jar",
		SHA1:     "bad-sha1",
		Enabled:  true,
	}, "instance", "1.21.1", "fabric")

	conflicts := IncompatibleConflictsCached(appstructs.ModDownloadRequest{
		ProjectID:        "modrinth:main",
		Result:           models.ModProject{ID: "modrinth:main", Platform: "Modrinth", ProjectID: "main", Title: "Main"},
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		TargetDir:        t.TempDir(),
		InstanceID:       "instance",
	})
	if len(conflicts) != 1 {
		t.Fatalf("conflicts = %#v, want 1", conflicts)
	}
	if conflicts[0].IncompatibleProjectKey != "modrinth:bad" || len(conflicts[0].Paths) != 1 || conflicts[0].Paths[0].Path != "mods/bad.jar" {
		t.Fatalf("conflict = %#v", conflicts[0])
	}
}

func TestAnalyzeBatchIncompatibleConflictsGroupsBySelectedMod(t *testing.T) {
	openModbridgeTestDB(t)
	global.ClearLocalMods()
	t.Cleanup(global.ClearLocalMods)

	scope := []storage.ModPlatformVersionScope{{MinecraftVersion: "1.21.1", ModLoader: "fabric"}}
	now := time.Now().Unix()
	if err := storage.SetPlatformVersionSnapshot("Modrinth", "main", []models.ModVersion{{
		ID:        "main-version",
		VersionID: "main-version",
		Platform:  "Modrinth",
		ProjectID: "main",
		Dependencies: []models.ModDependency{{
			DependencyProjectID: "bad",
			DependencyType:      "incompatible",
		}},
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
	}}, now, scope); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersionSnapshot("Modrinth", "clean", []models.ModVersion{{
		ID:           "clean-version",
		VersionID:    "clean-version",
		Platform:     "Modrinth",
		ProjectID:    "clean",
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
	}}, now, scope); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersionSnapshot("Modrinth", "bad", []models.ModVersion{{
		ID:           "bad-version",
		VersionID:    "bad-version",
		Platform:     "Modrinth",
		ProjectID:    "bad",
		ModIDs:       []string{"badmod"},
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
	}}, now, scope); err != nil {
		t.Fatal(err)
	}
	global.UpsertLocalMod(mcstructs.ModInfo{ID: "badmod", FileName: "bad", Path: "mods/bad.jar", SHA1: "bad-sha1", Enabled: true}, "instance", "1.21.1", "fabric")

	analysis := AnalyzeBatchIncompatibleConflicts(appstructs.BatchDownloadRequest{
		Results: []models.ModProject{
			{ID: "modrinth:main", Platform: "Modrinth", ProjectID: "main", Title: "Main"},
			{ID: "modrinth:clean", Platform: "Modrinth", ProjectID: "clean", Title: "Clean"},
		},
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		TargetDir:        t.TempDir(),
		InstanceID:       "instance",
	})
	if len(analysis.Conflicts) != 1 {
		t.Fatalf("analysis = %#v, want one conflicted selected mod", analysis)
	}
	if analysis.Conflicts[0].Key != "modrinth:main" || len(analysis.Conflicts[0].Conflicts) != 1 {
		t.Fatalf("conflict group = %#v", analysis.Conflicts[0])
	}
}

func TestResolveVersionModIDsReadsMemoryField(t *testing.T) {
	version := models.ModVersion{ID: "v-mem", ModIDs: []string{"Sodium ", "sodium", "Lithium"}}
	got := resolveVersionModIDs(version)
	want := []string{"sodium", "lithium"}
	if len(got) != len(want) {
		t.Fatalf("resolveVersionModIDs() = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resolveVersionModIDs() = %#v, want %#v", got, want)
		}
	}
}

func TestResolveVersionModIDsReturnsNilForUnknownVersion(t *testing.T) {
	// Open an isolated DB in a temp dir so the DB-read path is exercised
	// without panicking; the version ID is intentionally absent so the read
	// returns nil (cache miss), proving resolveVersionModIDs handles DB misses.
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		storage.Close()
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})
	if err := storage.OpenAt(filepath.Join(tmp, storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}

	got := resolveVersionModIDs(models.ModVersion{ID: "v-unknown"})
	if len(got) != 0 {
		t.Fatalf("resolveVersionModIDs() = %#v, want empty for unknown version", got)
	}
}

func resetBackfillState() {
	backfillMu.Lock()
	backfillInflight = make(map[string]struct{})
	pendingBackfills = nil
	pendingSet = make(map[string]struct{})
	backfillMu.Unlock()
}

func resetRemoteModIDCache() {
	ConfigureSimpleMode(false)
	remoteModIDCache.Lock()
	remoteModIDCache.entries = make(map[string]*remoteModIDCacheEntry)
	remoteModIDCache.Unlock()
}

func TestSimpleModeIgnoresCachedModIDsAndBackfills(t *testing.T) {
	resetRemoteModIDCache()
	resetBackfillState()
	previousParse := parseRemoteModJarForIDs
	t.Cleanup(func() {
		parseRemoteModJarForIDs = previousParse
		ConfigureSimpleMode(false)
		resetRemoteModIDCache()
		resetBackfillState()
	})

	var calls atomic.Int32
	parseRemoteModJarForIDs = func(string, string) ([]mcstructs.ModInfo, error) {
		calls.Add(1)
		return []mcstructs.ModInfo{{ID: "remote"}}, nil
	}
	ConfigureSimpleMode(true)
	version := models.ModVersion{
		ID:          "simple-version",
		DownloadURL: "https://cdn.example.test/simple.jar",
		ModIDs:      []string{"cached"},
	}
	if got := VersionModIDs(version, "fabric"); len(got) != 0 {
		t.Fatalf("VersionModIDs() = %#v, want no IDs in simple mode", got)
	}
	if _, err := cachedRemoteModIDs(version.DownloadURL, "fabric"); !errors.Is(err, errSimpleMode) {
		t.Fatalf("cachedRemoteModIDs() error = %v, want %v", err, errSimpleMode)
	}
	markBackfill(version, "fabric")
	if pending := drainPendingBackfills(); len(pending) != 0 {
		t.Fatalf("simple mode queued %d backfills, want 0", len(pending))
	}
	if calls.Load() != 0 {
		t.Fatalf("remote parse calls = %d, want 0", calls.Load())
	}
}

func TestSimpleModeReturnsOrdinaryDownloadStates(t *testing.T) {
	ConfigureSimpleMode(true)
	t.Cleanup(func() { ConfigureSimpleMode(false) })

	states := DownloadStates(appstructs.DownloadStatesRequest{Results: []models.ModProject{{
		ID:        "modrinth:sodium",
		Platform:  "Modrinth",
		ProjectID: "sodium",
	}}}, func() {
		t.Fatal("simple mode must not run a backfill completion callback")
	})
	if len(states) != 1 {
		t.Fatalf("state count = %d, want 1", len(states))
	}
	state := states[0]
	if state.Key != "modrinth:sodium" || state.Status != BtnStatusNew || state.Disabled || state.Icon != "mdi-download" || state.Color != "primary" {
		t.Fatalf("simple mode state = %#v, want ordinary download state", state)
	}
}

func TestSimpleModeStopsQueuedParsesAndAllowsInflightParseToFinish(t *testing.T) {
	resetRemoteModIDCache()
	ConfigureRemoteModIDConcurrency(1)
	previousParse := parseRemoteModJarForIDs
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseAll := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(func() {
		releaseAll()
		parseRemoteModJarForIDs = previousParse
		ConfigureSimpleMode(false)
		ConfigureRemoteModIDConcurrency(1)
		resetRemoteModIDCache()
	})

	var calls atomic.Int32
	parseRemoteModJarForIDs = func(string, string) ([]mcstructs.ModInfo, error) {
		calls.Add(1)
		started <- struct{}{}
		<-release
		return []mcstructs.ModInfo{{ID: "example"}}, nil
	}
	type result struct {
		ids []string
		err error
	}
	results := make(chan result, 2)
	go func() {
		ids, err := cachedRemoteModIDs("https://cdn.example.test/inflight.jar", "fabric")
		results <- result{ids: ids, err: err}
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first remote parse did not start")
	}
	go func() {
		ids, err := cachedRemoteModIDs("https://cdn.example.test/queued.jar", "fabric")
		results <- result{ids: ids, err: err}
	}()
	select {
	case <-started:
		t.Fatal("queued parse started above the configured limit")
	case <-time.After(50 * time.Millisecond):
	}

	ConfigureSimpleMode(true)
	releaseAll()
	var succeeded, disabled int
	for range 2 {
		select {
		case got := <-results:
			switch {
			case got.err == nil && len(got.ids) == 1 && got.ids[0] == "example":
				succeeded++
			case errors.Is(got.err, errSimpleMode) && len(got.ids) == 0:
				disabled++
			default:
				t.Fatalf("unexpected parse result: %#v", got)
			}
		case <-time.After(time.Second):
			t.Fatal("simple mode left a parser caller blocked")
		}
	}
	if succeeded != 1 || disabled != 1 || calls.Load() != 1 {
		t.Fatalf("parse outcomes: succeeded=%d disabled=%d calls=%d", succeeded, disabled, calls.Load())
	}
}

func TestCachedRemoteModIDsDeduplicatesConcurrentParses(t *testing.T) {
	resetRemoteModIDCache()
	ConfigureRemoteModIDConcurrency(1)
	previousParse := parseRemoteModJarForIDs
	t.Cleanup(func() {
		parseRemoteModJarForIDs = previousParse
		ConfigureRemoteModIDConcurrency(1)
		resetRemoteModIDCache()
	})

	var calls atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	parseRemoteModJarForIDs = func(string, string) ([]mcstructs.ModInfo, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return []mcstructs.ModInfo{{ID: "ExampleMod"}}, nil
	}

	const workers = 8
	results := make(chan []string, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids, err := cachedRemoteModIDs("https://cdn.example.test/mod.jar", "fabric")
			if err != nil {
				t.Errorf("cachedRemoteModIDs() error = %v", err)
			}
			results <- ids
		}()
	}
	<-started
	close(release)
	wg.Wait()
	close(results)

	if calls.Load() != 1 {
		t.Fatalf("remote parse calls = %d, want 1", calls.Load())
	}
	for ids := range results {
		if len(ids) != 1 || ids[0] != "examplemod" {
			t.Fatalf("cached mod IDs = %#v, want [examplemod]", ids)
		}
	}
}

func TestCachedRemoteModIDsHonorsUpdatedConcurrencyLimit(t *testing.T) {
	resetRemoteModIDCache()
	ConfigureRemoteModIDConcurrency(2)
	previousParse := parseRemoteModJarForIDs
	t.Cleanup(func() {
		parseRemoteModJarForIDs = previousParse
		ConfigureRemoteModIDConcurrency(1)
		resetRemoteModIDCache()
	})

	var active atomic.Int32
	var maxActive atomic.Int32
	started := make(chan struct{}, 3)
	release := make(chan struct{}, 3)
	parseRemoteModJarForIDs = func(string, string) ([]mcstructs.ModInfo, error) {
		current := active.Add(1)
		for {
			observed := maxActive.Load()
			if current <= observed || maxActive.CompareAndSwap(observed, current) {
				break
			}
		}
		started <- struct{}{}
		<-release
		active.Add(-1)
		return []mcstructs.ModInfo{{ID: "ExampleMod"}}, nil
	}

	urls := []string{
		"https://cdn.example.test/one.jar",
		"https://cdn.example.test/two.jar",
		"https://cdn.example.test/three.jar",
	}
	errs := make(chan error, len(urls))
	for _, downloadURL := range urls {
		go func() {
			_, err := cachedRemoteModIDs(downloadURL, "fabric")
			errs <- err
		}()
	}

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for initial remote JAR parses")
		}
	}
	select {
	case <-started:
		t.Fatal("third remote JAR parse started above configured limit")
	case <-time.After(50 * time.Millisecond):
	}

	ConfigureRemoteModIDConcurrency(1)
	release <- struct{}{}
	select {
	case <-started:
		t.Fatal("waiting parse started before active work fell below lowered limit")
	case <-time.After(50 * time.Millisecond):
	}

	release <- struct{}{}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("waiting parse did not start after capacity became available")
	}
	release <- struct{}{}

	for range urls {
		select {
		case err := <-errs:
			if err != nil {
				t.Fatalf("cachedRemoteModIDs() error = %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for remote JAR parse result")
		}
	}
	if got := maxActive.Load(); got != 2 {
		t.Fatalf("maximum concurrent remote JAR parses = %d, want 2", got)
	}
}

func TestRunRemoteModIDBackfillsUsesConfiguredConcurrency(t *testing.T) {
	openModbridgeTestDB(t)
	resetBackfillState()
	resetRemoteModIDCache()
	ConfigureRemoteModIDConcurrency(2)
	previousParse := parseRemoteModJarForIDs
	t.Cleanup(func() {
		parseRemoteModJarForIDs = previousParse
		ConfigureRemoteModIDConcurrency(1)
		resetRemoteModIDCache()
		resetBackfillState()
	})

	started := make(chan struct{}, 3)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseAll := func() {
		releaseOnce.Do(func() { close(release) })
	}
	t.Cleanup(releaseAll)
	parseRemoteModJarForIDs = func(string, string) ([]mcstructs.ModInfo, error) {
		started <- struct{}{}
		<-release
		return []mcstructs.ModInfo{{ID: "ExampleMod"}}, nil
	}

	done := make(chan struct{})
	go func() {
		runRemoteModIDBackfills([]pendingBackfill{
			{version: models.ModVersion{ID: "v-one", DownloadURL: "https://cdn.example.test/one.jar"}, modLoader: "fabric"},
			{version: models.ModVersion{ID: "v-two", DownloadURL: "https://cdn.example.test/two.jar"}, modLoader: "fabric"},
			{version: models.ModVersion{ID: "v-three", DownloadURL: "https://cdn.example.test/three.jar"}, modLoader: "fabric"},
		})
		close(done)
	}()

	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("batch did not start remote JAR backfills concurrently")
		}
	}
	select {
	case <-started:
		t.Fatal("batch exceeded configured remote JAR concurrency")
	case <-time.After(50 * time.Millisecond):
	}

	releaseAll()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("remote JAR backfill batch did not complete")
	}
}

func TestRemoteModJarTimeout(t *testing.T) {
	if remoteModJarTimeout != 45*time.Second {
		t.Fatalf("remoteModJarTimeout = %s, want 45s", remoteModJarTimeout)
	}
}

func TestCachedRemoteModIDsCachesFailures(t *testing.T) {
	resetRemoteModIDCache()
	previousParse := parseRemoteModJarForIDs
	t.Cleanup(func() {
		parseRemoteModJarForIDs = previousParse
		resetRemoteModIDCache()
	})

	var calls atomic.Int32
	parseRemoteModJarForIDs = func(string, string) ([]mcstructs.ModInfo, error) {
		calls.Add(1)
		return nil, errors.New("timeout")
	}

	for range 2 {
		if _, err := cachedRemoteModIDs("https://cdn.example.test/bad.jar", "neoforge"); err == nil {
			t.Fatal("cachedRemoteModIDs() error = nil, want failure")
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("remote parse calls = %d, want 1 during failure TTL", calls.Load())
	}
}

func TestRemoteMetadataButtonStatusReflectsFailureTTL(t *testing.T) {
	resetRemoteModIDCache()
	t.Cleanup(resetRemoteModIDCache)

	version := models.ModVersion{DownloadURL: "https://cdn.example.test/status.jar"}
	if got := remoteMetadataButtonStatus(version, "fabric"); got != BtnStatusMetadataLoading {
		t.Fatalf("initial metadata status = %q, want %q", got, BtnStatusMetadataLoading)
	}

	key := remoteModIDCacheKey(version.DownloadURL, "fabric")
	remoteModIDCache.Lock()
	remoteModIDCache.entries[key] = &remoteModIDCacheEntry{expiresAt: time.Now().Add(time.Second)}
	remoteModIDCache.Unlock()
	if got := remoteMetadataButtonStatus(version, "fabric"); got != BtnStatusMetadataFailed {
		t.Fatalf("cached failure metadata status = %q, want %q", got, BtnStatusMetadataFailed)
	}

	remoteModIDCache.Lock()
	remoteModIDCache.entries[key].expiresAt = time.Now().Add(-time.Second)
	remoteModIDCache.Unlock()
	if got := remoteMetadataButtonStatus(version, "fabric"); got != BtnStatusMetadataLoading {
		t.Fatalf("expired failure metadata status = %q, want %q", got, BtnStatusMetadataLoading)
	}
}

func TestRemoteMetadataButtonStatusUsesCurseForgeCDNFallback(t *testing.T) {
	resetRemoteModIDCache()
	t.Cleanup(resetRemoteModIDCache)

	version := models.ModVersion{
		Platform: "CurseForge",
		ID:       "12345678",
		FileName: "example.jar",
	}
	if got := remoteMetadataButtonStatus(version, "forge"); got != BtnStatusMetadataLoading {
		t.Fatalf("fallback metadata status = %q, want %q", got, BtnStatusMetadataLoading)
	}
}

func TestApplyButtonStatusStylesMetadataStatesAsRetryable(t *testing.T) {
	for _, status := range []string{BtnStatusMetadataLoading, BtnStatusMetadataFailed} {
		state := defaultDownloadButtonState(models.ModProject{})
		applyButtonStatus(&state, status)
		if state.Status != status || state.Disabled || state.Loading || state.Color != "surface-variant" {
			t.Fatalf("metadata state for %q = %#v, want enabled gray non-spinner button", status, state)
		}
	}
}

func TestMarkBackfillDeduplicatesByVersionID(t *testing.T) {
	resetBackfillState()
	t.Cleanup(resetBackfillState)

	version := models.ModVersion{ID: "v-dedup"}
	markBackfill(version, "fabric")
	markBackfill(version, "fabric")
	markBackfill(version, "forge")

	pending := drainPendingBackfills()
	if len(pending) != 1 {
		t.Fatalf("drainPendingBackfills() returned %d entries, want 1", len(pending))
	}
	if pending[0].version.ID != "v-dedup" {
		t.Fatalf("pending entry version ID = %q, want %q", pending[0].version.ID, "v-dedup")
	}

	// After draining, the queue is empty; a fresh mark re-enqueues.
	markBackfill(version, "fabric")
	pending = drainPendingBackfills()
	if len(pending) != 1 {
		t.Fatalf("drainPendingBackfills() after re-mark returned %d entries, want 1", len(pending))
	}
}

func TestBackfillVersionModIDsGuardsInflight(t *testing.T) {
	resetBackfillState()
	t.Cleanup(resetBackfillState)

	version := models.ModVersion{ID: "v-inflight-guard"}
	// Pre-mark as in-flight to simulate a concurrent backfill already running.
	backfillMu.Lock()
	backfillInflight[version.ID] = struct{}{}
	backfillMu.Unlock()

	// This call must early-return WITHOUT registering the deferred marker
	// cleanup, so the pre-set marker remains. A non-guarded path would proceed
	// to VersionModIDs (which with an empty DownloadURL fails fast) and then
	// delete the marker via defer — observable as a missing entry here.
	backfillVersionModIDs(version, "fabric")

	backfillMu.Lock()
	_, stillInflight := backfillInflight[version.ID]
	backfillMu.Unlock()
	if !stillInflight {
		t.Fatal("backfillVersionModIDs removed in-flight marker — guard did not early-return")
	}
}

func TestMarkBackfillSkipsEmptyVersionID(t *testing.T) {
	resetBackfillState()
	t.Cleanup(resetBackfillState)

	markBackfill(models.ModVersion{ID: ""}, "fabric")
	markBackfill(models.ModVersion{ID: "  "}, "fabric")

	pending := drainPendingBackfills()
	if len(pending) != 0 {
		t.Fatalf("drainPendingBackfills() returned %d entries, want 0 for empty version ID", len(pending))
	}
}

func TestPreferredVersionsUsesFallbackWhenPreferredTypeUnavailable(t *testing.T) {
	versions := []models.ModVersion{
		{ID: "stable", ReleaseType: "release"},
		{ID: "beta", ReleaseType: "beta"},
	}

	ConfigurePreferredReleaseType("latest")
	got := preferredVersions(versions)
	if len(got) != len(versions) {
		t.Fatalf("preferred latest versions = %#v, want all versions", got)
	}

	ConfigurePreferredReleaseType("beta")
	got = preferredVersions(versions)
	if len(got) != 1 || got[0].ID != "beta" {
		t.Fatalf("preferred beta versions = %#v, want beta only", got)
	}

	ConfigurePreferredReleaseType("alpha")
	got = preferredVersions(versions)
	if len(got) != len(versions) {
		t.Fatalf("missing preferred alpha fallback = %#v, want all versions", got)
	}

	ConfigurePreferredReleaseType("release")
	got = preferredVersions(versions)
	if len(got) != 1 || got[0].ID != "stable" {
		t.Fatalf("preferred release versions = %#v, want stable only", got)
	}
	ConfigurePreferredReleaseType("latest")
}
