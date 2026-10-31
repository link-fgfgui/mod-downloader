package downloader

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/downloader/filetransfer"
	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func resetDownloadQueueForTest(t *testing.T) {
	t.Helper()
	ConfigureSimpleMode(false)
	downloadQueue.Lock()
	downloadQueue.nextID = 0
	downloadQueue.nextReminderID = 0
	downloadQueue.pending = nil
	downloadQueue.retryable = nil
	downloadQueue.optionalReminders = nil
	downloadQueue.cancelRequested = nil
	downloadQueue.finalizedCancel = nil
	downloadQueue.restartRequested = nil
	downloadQueue.workers = 0
	downloadQueue.maxConcurrent = 1
	downloadQueue.current = nil
	downloadQueue.progress = nil
	downloadQueue.closed = false
	downloadQueue.groups = nil
	downloadQueue.finishedGroups = nil
	downloadQueue.Unlock()
	downloadFileConcurrency.Store(filetransfer.DefaultConcurrency)
	downloadVerifySHA1.Store(false)
	resetDownloaderHooksForTest(t)
}

func TestConfigureSimpleModeClearsAndBlocksOptionalDependencyReminders(t *testing.T) {
	resetDownloadQueueForTest(t)
	t.Cleanup(func() { ConfigureSimpleMode(false) })
	downloadQueue.Lock()
	downloadQueue.optionalReminders = []optionalDependencyReminderState{{
		Snapshot: appstructs.OptionalDependencyReminder{ID: "optional-1"},
		Requests: []appstructs.ModDownloadRequest{{ProjectID: "modrinth:optional"}},
	}}
	downloadQueue.Unlock()

	emitted := make(chan appstructs.DownloadQueueState, 1)
	ConfigureSimpleMode(true, Events{OnQueueState: func(state appstructs.DownloadQueueState) {
		emitted <- state
	}})
	select {
	case state := <-emitted:
		if state.MessageCount != 0 || len(state.OptionalReminders) != 0 {
			t.Fatalf("simple mode queue state = %#v, want no optional reminders", state)
		}
	case <-time.After(time.Second):
		t.Fatal("clearing optional reminders did not emit queue state")
	}
	if results := InstallOptionalDependencies(t.Context(), "optional-1", ""); len(results) != 0 {
		t.Fatalf("stale optional install results = %#v, want none", results)
	}
}

func TestPrepareDownloadDependenciesSkipsAllDependencyWorkInSimpleMode(t *testing.T) {
	resetDownloadQueueForTest(t)
	t.Cleanup(func() { ConfigureSimpleMode(false) })
	ConfigureSimpleMode(true)

	refreshMatchingProjectVersions = func(models.ModProject, string, string) []models.ModVersion {
		t.Fatal("simple mode refreshed dependency metadata")
		return nil
	}
	installStatusPrecise = func(appstructs.ModDownloadRequest) string {
		t.Fatal("simple mode checked precise dependency status")
		return modbridge.BtnStatusNew
	}
	job := downloadJob{Version: models.ModVersion{Dependencies: []models.ModDependency{
		{DependencyProjectID: "required", DependencyType: "required"},
		{DependencyProjectID: "optional", DependencyType: "optional"},
		{DependencyProjectID: "incompatible", DependencyType: "incompatible"},
	}}}
	dependencies, prepared, err := prepareDownloadDependencies(t.Context(), job, Events{})
	if err != nil {
		t.Fatal(err)
	}
	if len(dependencies) != 0 || !prepared.dependenciesDone || len(prepared.IncompatiblePaths) != 0 {
		t.Fatalf("simple mode dependency result = %#v, %#v", dependencies, prepared)
	}
	if state := GetDownloadQueueState(); state.MessageCount != 0 || len(state.OptionalReminders) != 0 {
		t.Fatalf("simple mode created optional reminders: %#v", state)
	}
}

func resetDownloaderHooksForTest(t *testing.T) {
	t.Helper()
	refreshMatchingProjectVersions = providers.RefreshMatchingProjectVersions
	installStatusPrecise = modbridge.InstallStatusPrecise
	newFileTransferBackend = func() filetransfer.Backend { return filetransfer.NewStdlibBackend(nil) }
}

func openDownloadTestDatabase(t *testing.T) {
	t.Helper()
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatalf("storage.OpenAt() error = %v", err)
	}
	t.Cleanup(storage.Close)
	// Provider refreshes run in the background; drain them before closing the
	// DB.
	t.Cleanup(providers.DrainBackgroundTasks)
}

func seedDownloadProjectVersion(t *testing.T, projectID string, version models.ModVersion) models.ModProject {
	t.Helper()
	project := models.ModProject{
		ID:        "modrinth:" + projectID,
		Platform:  "Modrinth",
		ProjectID: projectID,
		Slug:      projectID,
		Title:     projectID,
	}
	if err := storage.UpsertModPlatform(project); err != nil {
		t.Fatalf("storage.UpsertModPlatform(%q) error = %v", projectID, err)
	}
	if err := storage.SetPlatformVersionSnapshot("Modrinth", projectID, []models.ModVersion{version}, time.Now().Unix(), []storage.ModPlatformVersionScope{{
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}}); err != nil {
		t.Fatalf("storage.SetPlatformVersionSnapshot(%q) error = %v", projectID, err)
	}
	return project
}

func waitForDownloadState(t *testing.T, match func(appstructs.DownloadQueueState) bool) appstructs.DownloadQueueState {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		state := GetDownloadQueueState()
		if match(state) {
			return state
		}
		time.Sleep(10 * time.Millisecond)
	}
	state := GetDownloadQueueState()
	t.Fatalf("download state did not match before timeout: %#v", state)
	return state
}

func TestDownloadQueueStateAggregatesByteProgressAndActiveSpeed(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.current = map[string]downloadJob{
		"download-running": {ID: "download-running", Version: models.ModVersion{ID: "running"}},
	}
	downloadQueue.pending = []downloadJob{
		{ID: "download-pending", Version: models.ModVersion{ID: "pending"}},
	}
	downloadQueue.progress = map[string]downloadProgress{
		"download-complete": {BytesComplete: 100, TotalBytes: 100},
		"download-running":  {BytesComplete: 200, TotalBytes: 500, BytesPerSecond: 300},
		"download-pending":  {BytesComplete: 0, TotalBytes: 1000},
		"download-unknown":  {BytesComplete: 80, TotalBytes: 0, BytesPerSecond: 900},
	}
	downloadQueue.Unlock()

	state := GetDownloadQueueState()
	if state.BytesComplete != 300 || state.TotalBytes != 1600 || state.BytesPerSecond != 300 {
		t.Fatalf("aggregate progress = (%d/%d, %d B/s), want (300/1600, 300 B/s)", state.BytesComplete, state.TotalBytes, state.BytesPerSecond)
	}
	if len(state.Items) != 2 {
		t.Fatalf("items = %d, want active items only", len(state.Items))
	}
	for _, item := range state.Items {
		switch item.ID {
		case "download-running":
			if item.BytesComplete != 200 || item.TotalBytes != 500 {
				t.Fatalf("running item progress = %#v", item)
			}
		case "download-pending":
			if item.BytesComplete != 0 || item.TotalBytes != 1000 {
				t.Fatalf("pending item progress = %#v", item)
			}
		}
	}
}

func TestDownloadQueueProgressUpdateUsesTransferTotalAndEmitsAfterUnlock(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.current = map[string]downloadJob{
		"download-1": {ID: "download-1", Version: models.ModVersion{ID: "version-1"}},
	}
	downloadQueue.progress = map[string]downloadProgress{
		"download-1": {TotalBytes: 100},
	}
	downloadQueue.Unlock()

	emitted := make(chan appstructs.DownloadQueueState, 1)
	updateDownloadProgress("download-1", filetransfer.Progress{BytesComplete: 40, TotalBytes: 80}, 50, Events{
		OnQueueState: func(state appstructs.DownloadQueueState) {
			emitted <- state
		},
	})
	state := <-emitted
	if state.BytesComplete != 40 || state.TotalBytes != 80 || state.BytesPerSecond != 50 {
		t.Fatalf("emitted state = %#v", state)
	}
	if state.Items[0].BytesComplete != 40 || state.Items[0].TotalBytes != 80 {
		t.Fatalf("emitted item = %#v", state.Items[0])
	}
}

func TestDownloadQueueProgressClearsAfterActiveDrain(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.current = map[string]downloadJob{
		"download-1": {ID: "download-1"},
	}
	downloadQueue.progress = map[string]downloadProgress{
		"download-1": {BytesComplete: 100, TotalBytes: 100},
	}
	downloadQueue.Unlock()

	removeCurrentDownload("download-1")
	state := GetDownloadQueueState()
	if state.BytesComplete != 0 || state.TotalBytes != 0 || state.BytesPerSecond != 0 {
		t.Fatalf("drained progress = %#v, want zero aggregate", state)
	}
}

func TestGetDownloadStatesReturnsDefaultWhenSelectedInstanceHasNoLocalMods(t *testing.T) {
	resetDownloadQueueForTest(t)
	global.ClearLocalMods()
	global.SetMinecraftDir(t.TempDir())
	global.SetVersions([]mcstructs.VersionInfo{{
		ID:               "instance",
		Name:             "instance",
		MinecraftVersion: "1.21.1",
		ModLoader:        "neoforge",
	}})
	t.Cleanup(func() {
		global.ClearLocalMods()
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	states := GetDownloadStates(appstructs.DownloadStatesRequest{
		MinecraftVersion: "1.21.1",
		ModLoader:        "NeoForge",
		Results: []models.ModProject{{
			ID:       "modrinth:sodium",
			Platform: "Modrinth",
			Slug:     "sodium",
		}},
	}, nil)

	if len(states) != 1 {
		t.Fatalf("state count = %d, want 1", len(states))
	}
	if states[0].Key != "modrinth:sodium" || states[0].Status != modbridge.BtnStatusNew || states[0].Icon != "mdi-download" || states[0].Color != "primary" || states[0].Disabled {
		t.Fatalf("state = %#v", states[0])
	}
}

func TestPathInLocalModPathsMatchesRelativePaths(t *testing.T) {
	resetDownloadQueueForTest(t)
	mcDir := t.TempDir()
	global.SetMinecraftDir(mcDir)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
	})

	path := filepath.Join(mcDir, "versions", "instance", "mods", "mod.jar")
	paths := []global.LocalModFilePath{{
		Path: filepath.Join("versions", "instance", "mods", "mod.jar"),
	}}
	if !pathInLocalModPaths(path, paths) {
		t.Fatalf("pathInLocalModPaths(%q) = false, want true", path)
	}
}

func TestDependencyDownloadRequestCarriesDependencyVersionID(t *testing.T) {
	resetDownloadQueueForTest(t)
	req, ok := modbridge.DependencyDownloadRequest("Modrinth", models.ModDependency{
		DependencyProjectID: "fabric-api",
		DependencyVersionID: "version-123",
		DependencyType:      "required",
	}, appstructs.ModDownloadRequest{
		MinecraftVersion: "1.21.1",
		ModLoader:        "Fabric",
		TargetDir:        "/tmp/mods",
		InstanceID:       "cli:/tmp/mods",
	})
	if !ok {
		t.Fatal("DependencyDownloadRequest() ok = false")
	}
	if req.VersionID != "version-123" {
		t.Fatalf("VersionID = %q, want version-123", req.VersionID)
	}
	if req.TargetDir != "/tmp/mods" || req.InstanceID != "cli:/tmp/mods" {
		t.Fatalf("target = %q, instance = %q", req.TargetDir, req.InstanceID)
	}
}

func TestQueueModDownloadAcceptsExplicitTargetWithoutSelectedInstance(t *testing.T) {
	resetDownloadQueueForTest(t)
	global.SetMinecraftDir("")
	global.InvalidateVersions()
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})

	result := QueueModDownload(nil, appstructs.ModDownloadRequest{
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		TargetDir:        filepath.Join(t.TempDir(), "mods"),
	}, "")

	if !result.Skipped {
		t.Fatalf("result = %#v, want skipped invalid request", result)
	}
	if result.Reason != "invalid request" {
		t.Fatalf("reason = %q, want invalid request", result.Reason)
	}
}

func TestDownloadCompletedEventOnlyFiresForNewInstalledFile(t *testing.T) {
	resetDownloadQueueForTest(t)
	openDownloadTestDatabase(t)
	newFileTransferBackend = func() filetransfer.Backend {
		return successfulTransferBackend{content: []byte("downloaded jar")}
	}
	targetDir := t.TempDir()
	completed := make(chan struct{}, 2)
	events := Events{OnDownloadCompleted: func() { completed <- struct{}{} }}
	job := downloadJob{
		Version: models.ModVersion{
			ID:          "version-1",
			VersionID:   "version-1",
			Platform:    "Modrinth",
			ProjectID:   "example",
			FileName:    "example.jar",
			DownloadURL: "https://example.test/example.jar",
		},
		TargetDir:        targetDir,
		InstanceID:       "test:" + targetDir,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		dependenciesDone: true,
	}
	seedDownloadProjectVersion(t, "example", job.Version)
	if !modbridge.PersistVersionModIDs(job.Version.ID, []string{"example"}) {
		t.Fatal("PersistVersionModIDs returned false")
	}

	enqueueDownload(t.Context(), job, events)
	select {
	case <-completed:
	case <-time.After(2 * time.Second):
		t.Fatal("new install did not emit completion")
	}
	waitForDownloadState(t, func(state appstructs.DownloadQueueState) bool {
		return state.Running == 0 && state.Pending == 0
	})

	enqueueDownload(t.Context(), job, events)
	waitForDownloadState(t, func(state appstructs.DownloadQueueState) bool {
		return state.Running == 0 && state.Pending == 0
	})
	select {
	case <-completed:
		t.Fatal("existing target emitted a second completion")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestSimpleModeDownloadUsesLocalParseFallback(t *testing.T) {
	resetDownloadQueueForTest(t)
	openDownloadTestDatabase(t)
	ConfigureSimpleMode(true)
	t.Cleanup(func() { ConfigureSimpleMode(false) })
	newFileTransferBackend = func() filetransfer.Backend {
		return successfulTransferBackend{content: fabricModTestJar(t, "simple-example")}
	}

	targetDir := t.TempDir()
	version := models.ModVersion{
		ID:           "simple-version",
		VersionID:    "simple-version",
		Platform:     "Modrinth",
		ProjectID:    "simple-example",
		FileName:     "simple-example.jar",
		DownloadURL:  "https://example.test/simple-example.jar",
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
		ModIDs:       []string{"cached-id-must-be-ignored"},
	}
	seedDownloadProjectVersion(t, "simple-example", version)
	installed, err := downloadModJob(t.Context(), downloadJob{
		Version:          version,
		ProjectID:        "modrinth:simple-example",
		TargetDir:        targetDir,
		InstanceID:       "test:" + targetDir,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		dependenciesDone: true,
	}, Events{})
	if err != nil {
		t.Fatal(err)
	}
	if !installed {
		t.Fatal("simple mode local-parse download installed = false, want true")
	}
	if _, err := os.Stat(filepath.Join(targetDir, version.FileName)); err != nil {
		t.Fatalf("installed file missing: %v", err)
	}
	if got := modbridge.VersionModIDs(version, "fabric"); len(got) != 0 {
		t.Fatalf("simple mode exposed persisted local IDs: %#v", got)
	}
	ConfigureSimpleMode(false)
	if got := modbridge.VersionModIDs(models.ModVersion{ID: version.ID}, "fabric"); len(got) != 1 || got[0] != "simple-example" {
		t.Fatalf("normal mode persisted local IDs = %#v, want [simple-example]", got)
	}
}

func fabricModTestJar(t *testing.T, modID string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	entry, err := zw.Create("fabric.mod.json")
	if err != nil {
		t.Fatal(err)
	}
	metadata := fmt.Sprintf(`{"schemaVersion":1,"id":%q,"version":"1.0.0","name":"Simple Example"}`, modID)
	if _, err := entry.Write([]byte(metadata)); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestCancelDownloadDuringDependencyAnalysisKeepsRetryableQueueItem(t *testing.T) {
	resetDownloadQueueForTest(t)
	openDownloadTestDatabase(t)
	targetDir := t.TempDir()
	mainVersion := models.ModVersion{
		ID:           "main-version",
		Platform:     "Modrinth",
		ProjectID:    "main",
		FileName:     "main.jar",
		DownloadURL:  "https://example.invalid/main.jar",
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
	}
	project := seedDownloadProjectVersion(t, "main", mainVersion)

	analysisStarted := make(chan struct{})
	releaseAnalysis := make(chan struct{})
	refreshMatchingProjectVersions = func(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
		close(analysisStarted)
		<-releaseAnalysis
		mainVersion.Dependencies = []models.ModDependency{{
			DependencyProjectID: "dependency",
			DependencyType:      "required",
		}}
		return []models.ModVersion{mainVersion}
	}
	installStatusPrecise = func(req appstructs.ModDownloadRequest) string {
		return modbridge.BtnStatusNew
	}

	result := QueueModDownload(context.Background(), appstructs.ModDownloadRequest{
		ProjectID:        "modrinth:main",
		Result:           project,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		TargetDir:        targetDir,
		InstanceID:       "cli:" + targetDir,
	}, "")
	if !result.Queued {
		t.Fatalf("QueueModDownload() = %#v, want queued", result)
	}

	select {
	case <-analysisStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("dependency analysis did not start")
	}
	running := waitForDownloadState(t, func(state appstructs.DownloadQueueState) bool {
		return state.Running == 1 && len(state.Items) == 1 && state.Items[0].Cancelable
	})
	runningID := running.Items[0].ID
	if runningID == "" {
		t.Fatalf("running item has empty id: %#v", running.Items[0])
	}

	if !CancelDownload(context.Background(), runningID) {
		t.Fatal("CancelDownload() = false, want true")
	}
	close(releaseAnalysis)

	state := waitForDownloadState(t, func(state appstructs.DownloadQueueState) bool {
		return state.Running == 0 && state.Pending == 0 && len(state.Items) == 1 && state.Items[0].Status == "canceled"
	})
	item := state.Items[0]
	if item.ID != runningID || !item.Retryable || item.Cancelable {
		t.Fatalf("canceled item = %#v", item)
	}
}

func TestRetryCanceledDependencyAnalysisUsesFreshQueueID(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.nextID = 1
	downloadQueue.retryable = []retryableDownloadJob{{
		Status: "canceled",
		Reason: "canceled",
		Job: downloadJob{
			ID: "download-1",
			Version: models.ModVersion{
				ID:       "main-version",
				Platform: "modrinth",
				FileName: "main.jar",
			},
			Result: models.ModProject{
				Title: "Main",
			},
			TargetDir:        t.TempDir(),
			InstanceID:       "instance",
			MinecraftVersion: "1.21.1",
			ModLoader:        "fabric",
		},
	}}
	// Suppress the background runner; this test only verifies retry queue state.
	downloadQueue.workers = 1
	downloadQueue.Unlock()

	if !RetryDownload(nil, "download-1") {
		t.Fatal("RetryDownload() = false, want true")
	}

	state := GetDownloadQueueState()
	if state.Pending != 1 || len(state.Items) != 1 {
		t.Fatalf("state = %#v", state)
	}
	item := state.Items[0]
	if item.ID != "download-2" || item.Status != "pending" || !item.Cancelable || item.Retryable {
		t.Fatalf("item = %#v", item)
	}
}

func TestCancelPendingDownloadKeepsRetryableQueueItem(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.nextID = 1
	downloadQueue.pending = []downloadJob{{
		ID: "download-1",
		Version: models.ModVersion{
			ID:       "version-1",
			Platform: "modrinth",
			FileName: "sodium.jar",
		},
		Result: models.ModProject{
			Title: "Sodium",
		},
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}}
	downloadQueue.Unlock()

	if !CancelDownload(nil, "download-1") {
		t.Fatal("CancelDownload() = false, want true")
	}

	state := GetDownloadQueueState()
	if !state.Active || state.Pending != 0 || state.Running != 0 {
		t.Fatalf("state summary = %#v", state)
	}
	if len(state.Items) != 1 {
		t.Fatalf("item count = %d, want 1", len(state.Items))
	}
	item := state.Items[0]
	if item.ID != "download-1" || item.Status != "canceled" || !item.Retryable || item.Cancelable {
		t.Fatalf("item = %#v", item)
	}
}

func TestRetryDownloadRequeuesRetryableItemWithFreshID(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.nextID = 1
	downloadQueue.retryable = []retryableDownloadJob{{
		Status: "failed",
		Reason: "network error",
		Job: downloadJob{
			ID: "download-1",
			Version: models.ModVersion{
				ID:       "version-1",
				Platform: "modrinth",
				FileName: "sodium.jar",
			},
			Result: models.ModProject{
				Title: "Sodium",
			},
			TargetDir:        t.TempDir(),
			InstanceID:       "instance",
			MinecraftVersion: "1.21.1",
			ModLoader:        "fabric",
		},
	}}
	// Suppress the background runner; this test only needs to verify requeue state.
	downloadQueue.workers = 1
	downloadQueue.Unlock()

	if !RetryDownload(nil, "download-1") {
		t.Fatal("RetryDownload() = false, want true")
	}

	state := GetDownloadQueueState()
	if state.Pending != 1 || len(state.Items) != 1 {
		t.Fatalf("state = %#v", state)
	}
	item := state.Items[0]
	if item.ID != "download-2" || item.Status != "pending" || !item.Cancelable || item.Retryable {
		t.Fatalf("item = %#v", item)
	}
	if item.VersionID != "version-1" || item.MinecraftVersion != "1.21.1" || item.ModLoader != "fabric" {
		t.Fatalf("item lost retry context: %#v", item)
	}
}

func TestRemoveDownloadRemovesFailedAndCanceledRetryableItems(t *testing.T) {
	for _, status := range []string{"failed", "canceled"} {
		t.Run(status, func(t *testing.T) {
			resetDownloadQueueForTest(t)
			downloadQueue.Lock()
			downloadQueue.retryable = []retryableDownloadJob{
				{Status: status, Job: downloadJob{ID: "download-target"}},
				{Status: "failed", Job: downloadJob{ID: "download-other"}},
			}
			downloadQueue.progress = map[string]downloadProgress{
				"download-target": {BytesComplete: 10, TotalBytes: 20},
			}
			downloadQueue.Unlock()

			eventCount := 0
			var emitted appstructs.DownloadQueueState
			if !RemoveDownload(" download-target ", Events{OnQueueState: func(state appstructs.DownloadQueueState) {
				eventCount++
				emitted = state
			}}) {
				t.Fatal("RemoveDownload() = false, want true")
			}
			if eventCount != 1 {
				t.Fatalf("queue event count = %d, want 1", eventCount)
			}
			if len(emitted.Items) != 1 || emitted.Items[0].ID != "download-other" {
				t.Fatalf("emitted items = %#v", emitted.Items)
			}
			downloadQueue.Lock()
			_, progressExists := downloadQueue.progress["download-target"]
			downloadQueue.Unlock()
			if progressExists {
				t.Fatal("removed retryable item retained progress")
			}
		})
	}
}

func TestRemoveDownloadRemovesPendingWithoutCreatingCanceledHistory(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.pending = []downloadJob{{ID: "download-target"}, {ID: "download-other"}}
	downloadQueue.retryable = []retryableDownloadJob{{Status: "failed", Job: downloadJob{ID: "download-failed"}}}
	downloadQueue.progress = map[string]downloadProgress{"download-target": {BytesComplete: 10}}
	downloadQueue.Unlock()

	eventCount := 0
	var emitted appstructs.DownloadQueueState
	if !RemoveDownload("download-target", Events{OnQueueState: func(state appstructs.DownloadQueueState) {
		eventCount++
		emitted = state
	}}) {
		t.Fatal("RemoveDownload() = false, want true")
	}
	if eventCount != 1 {
		t.Fatalf("queue event count = %d, want 1", eventCount)
	}
	if emitted.Pending != 1 || len(emitted.Items) != 2 {
		t.Fatalf("emitted state = %#v", emitted)
	}
	if emitted.Items[0].ID != "download-other" || emitted.Items[0].Status != "pending" || emitted.Items[1].ID != "download-failed" {
		t.Fatalf("emitted items = %#v", emitted.Items)
	}
	downloadQueue.Lock()
	_, progressExists := downloadQueue.progress["download-target"]
	downloadQueue.Unlock()
	if progressExists {
		t.Fatal("removed pending item retained progress")
	}
}

func TestRemoveDownloadRejectsInvalidOrRunningItems(t *testing.T) {
	tests := []struct {
		name string
		id   string
	}{
		{name: "empty", id: "  "},
		{name: "missing", id: "download-missing"},
		{name: "unsupported retryable status", id: "download-paused"},
		{name: "running", id: "download-running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetDownloadQueueForTest(t)
			downloadQueue.Lock()
			downloadQueue.retryable = []retryableDownloadJob{{
				Status: "paused",
				Job:    downloadJob{ID: "download-paused"},
			}}
			downloadQueue.current = map[string]downloadJob{"download-running": {ID: "download-running"}}
			downloadQueue.Unlock()

			eventCount := 0
			if RemoveDownload(tt.id, Events{OnQueueState: func(appstructs.DownloadQueueState) {
				eventCount++
			}}) {
				t.Fatal("RemoveDownload() = true, want false")
			}
			if eventCount != 0 {
				t.Fatalf("queue event count = %d, want 0", eventCount)
			}
			state := GetDownloadQueueState()
			if len(state.Items) != 2 {
				t.Fatalf("item count = %d, want 2", len(state.Items))
			}
		})
	}
}

func TestRunningDownloadIsRetryableAndMarksRestartRequest(t *testing.T) {
	resetDownloadQueueForTest(t)
	canceled := false
	downloadQueue.Lock()
	downloadQueue.nextID = 1
	downloadQueue.workers = 1
	downloadQueue.current = map[string]downloadJob{"download-1": {
		ID: "download-1",
		Version: models.ModVersion{
			ID:       "version-1",
			Platform: "modrinth",
			FileName: "sodium.jar",
		},
		Result: models.ModProject{
			Title: "Sodium",
		},
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		cancel: func() {
			canceled = true
		},
	}}
	downloadQueue.Unlock()

	state := GetDownloadQueueState()
	if len(state.Items) != 1 || state.Items[0].Status != "running" || !state.Items[0].Retryable {
		t.Fatalf("running item = %#v", state.Items)
	}
	if !RetryDownload(nil, "download-1") {
		t.Fatal("RetryDownload() = false, want true")
	}
	if !canceled {
		t.Fatal("running retry did not cancel active attempt")
	}
	if !consumeDownloadRestartRequest("download-1") {
		t.Fatal("restart request was not recorded")
	}
	if consumeDownloadCancelRequest("download-1") {
		t.Fatal("running retry should not record a user-cancel request")
	}
}

func TestCancelRunningDownloadMovesItemToCanceledImmediately(t *testing.T) {
	resetDownloadQueueForTest(t)
	canceled := false
	downloadQueue.Lock()
	downloadQueue.nextID = 1
	downloadQueue.workers = 1
	downloadQueue.current = map[string]downloadJob{"download-1": {
		ID: "download-1",
		Version: models.ModVersion{
			ID:       "version-1",
			Platform: "modrinth",
			FileName: "sodium.jar",
		},
		Result: models.ModProject{
			Title: "Sodium",
		},
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		cancel: func() {
			canceled = true
		},
	}}
	downloadQueue.Unlock()

	if !CancelDownload(nil, "download-1") {
		t.Fatal("CancelDownload() = false, want true")
	}
	if !canceled {
		t.Fatal("running cancel did not cancel active attempt")
	}

	state := GetDownloadQueueState()
	if state.Running != 0 || state.Pending != 0 || len(state.Items) != 1 {
		t.Fatalf("state = %#v", state)
	}
	item := state.Items[0]
	if item.ID != "download-1" || item.Status != "canceled" || !item.Retryable || item.Cancelable {
		t.Fatalf("item = %#v", item)
	}
	if !consumeFinalizedDownloadCancel("download-1") {
		t.Fatal("finalized cancel marker was not recorded")
	}
}

func TestRetryStalledDownloadRequeuesUntilLimit(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.workers = 1
	downloadQueue.nextID = 1
	downloadQueue.Unlock()

	job := downloadJob{
		ID: "download-1",
		Version: models.ModVersion{
			ID:       "version-1",
			Platform: "modrinth",
			FileName: "sodium.jar",
		},
		Result: models.ModProject{
			Title: "Sodium",
		},
		AutoAttempt: 0,
	}
	if !retryStalledDownload(nil, job, Events{}) {
		t.Fatal("retryStalledDownload() = false, want true")
	}

	state := GetDownloadQueueState()
	if state.Pending != 1 || len(state.Items) != 1 {
		t.Fatalf("state = %#v", state)
	}
	if state.Items[0].ID != "download-2" || state.Items[0].Status != "pending" {
		t.Fatalf("item = %#v", state.Items[0])
	}

	downloadQueue.Lock()
	requeuedAttempt := downloadQueue.pending[0].AutoAttempt
	downloadQueue.Unlock()
	if requeuedAttempt != 1 {
		t.Fatalf("AutoAttempt = %d, want 1", requeuedAttempt)
	}
	if retryStalledDownload(nil, downloadJob{AutoAttempt: maxDownloadStallRetries}, Events{}) {
		t.Fatal("retryStalledDownload() = true at retry limit, want false")
	}
}

func TestDoDownloadWithStallTimeoutReturnsPromptlyWhenContextCanceled(t *testing.T) {
	resetDownloadQueueForTest(t)
	release := make(chan struct{})
	t.Cleanup(func() {
		close(release)
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-release
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := doDownloadWithStallTimeout(ctx, filetransfer.NewStdlibBackend(server.Client()), filetransfer.Request{
			URL:               server.URL,
			Destination:       filepath.Join(t.TempDir(), "blocked.jar"),
			OverwriteExisting: true,
		})
		done <- err
	}()

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v, want context.Canceled", err)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("download wait did not return promptly after context cancellation")
	}
}

type blockingTransferBackend struct{}

type successfulTransferBackend struct {
	content []byte
}

func (successfulTransferBackend) Kind() filetransfer.BackendKind {
	return filetransfer.BackendStdlib
}

func (b successfulTransferBackend) Download(_ context.Context, req filetransfer.Request, tracker *filetransfer.ProgressTracker) (filetransfer.Result, error) {
	tracker.Reset(int64(len(b.content)))
	if err := os.WriteFile(req.Destination, b.content, 0o644); err != nil {
		return filetransfer.Result{}, err
	}
	tracker.SetComplete(int64(len(b.content)))
	return filetransfer.Result{
		Path:          req.Destination,
		BytesComplete: int64(len(b.content)),
		TotalBytes:    int64(len(b.content)),
	}, nil
}

func (blockingTransferBackend) Kind() filetransfer.BackendKind {
	return filetransfer.BackendStdlib
}

type recordingBlockingTransferBackend struct {
	requests chan filetransfer.Request
}

func (*recordingBlockingTransferBackend) Kind() filetransfer.BackendKind {
	return filetransfer.BackendStdlib
}

func (b *recordingBlockingTransferBackend) Download(ctx context.Context, req filetransfer.Request, _ *filetransfer.ProgressTracker) (filetransfer.Result, error) {
	b.requests <- req
	<-ctx.Done()
	return filetransfer.Result{}, ctx.Err()
}

func TestConfigureControlsFileAndQueueConcurrency(t *testing.T) {
	resetDownloadQueueForTest(t)
	Configure(Config{FileConcurrency: 8, ConcurrentDownloads: 2})
	backend := &recordingBlockingTransferBackend{requests: make(chan filetransfer.Request, 2)}
	newFileTransferBackend = func() filetransfer.Backend { return backend }
	t.Cleanup(func() {
		newFileTransferBackend = func() filetransfer.Backend { return filetransfer.NewStdlibBackend(nil) }
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	targetDir := t.TempDir()
	for i := 1; i <= 2; i++ {
		enqueueDownload(ctx, downloadJob{
			Version: models.ModVersion{
				ID:          fmt.Sprintf("version-%d", i),
				FileName:    fmt.Sprintf("mod-%d.jar", i),
				DownloadURL: fmt.Sprintf("https://example.test/mod-%d.jar", i),
			},
			TargetDir:        targetDir,
			ModLoader:        "fabric",
			dependenciesDone: true,
		}, Events{})
	}

	for i := 0; i < 2; i++ {
		select {
		case req := <-backend.requests:
			if req.Concurrency != 8 {
				t.Fatalf("file concurrency = %d, want 8", req.Concurrency)
			}
		case <-time.After(time.Second):
			t.Fatal("two downloads did not start concurrently")
		}
	}
	state := GetDownloadQueueState()
	if state.Running != 2 || state.Pending != 0 {
		t.Fatalf("queue state = %#v, want two running downloads", state)
	}
	cancel()
	waitForDownloadState(t, func(state appstructs.DownloadQueueState) bool {
		return state.Running == 0 && state.Pending == 0
	})
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		downloadQueue.Lock()
		workers := downloadQueue.workers
		downloadQueue.Unlock()
		if workers == 0 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("download workers did not exit")
}

func (blockingTransferBackend) Download(ctx context.Context, _ filetransfer.Request, _ *filetransfer.ProgressTracker) (filetransfer.Result, error) {
	<-ctx.Done()
	return filetransfer.Result{}, ctx.Err()
}

func TestDoDownloadWithStallTimeoutCancelsBackend(t *testing.T) {
	previousTimeout := downloadStallTimeout
	previousInterval := downloadProgressPollInterval
	downloadStallTimeout = 20 * time.Millisecond
	downloadProgressPollInterval = 5 * time.Millisecond
	t.Cleanup(func() {
		downloadStallTimeout = previousTimeout
		downloadProgressPollInterval = previousInterval
	})

	_, err := doDownloadWithStallTimeout(context.Background(), blockingTransferBackend{}, filetransfer.Request{
		URL:         "https://example.invalid/file.jar",
		Destination: filepath.Join(t.TempDir(), "file.jar"),
	})
	if !errors.Is(err, errDownloadStalled) {
		t.Fatalf("err = %v, want errDownloadStalled", err)
	}
}

func TestDoDownloadWithStallTimeoutEmitsFinalProgressForShortTransfer(t *testing.T) {
	content := []byte("short transfer")
	var progress filetransfer.Progress
	_, err := doDownloadWithStallTimeout(t.Context(), successfulTransferBackend{content: content}, filetransfer.Request{
		URL:         "https://example.invalid/file.jar",
		Destination: filepath.Join(t.TempDir(), "file.jar"),
	}, func(value filetransfer.Progress, _ int64) {
		progress = value
	})
	if err != nil {
		t.Fatal(err)
	}
	if progress.BytesComplete != int64(len(content)) || progress.TotalBytes != int64(len(content)) {
		t.Fatalf("final progress = %#v, want %d/%d", progress, len(content), len(content))
	}
}

func TestDownloadFileToTempUsesJobFilenameHeadersAndContent(t *testing.T) {
	content := []byte("downloaded-content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if got := req.Header.Get("User-Agent"); got != "mod-downloader/v1.2.3" {
			t.Errorf("User-Agent = %q", got)
		}
		w.Header().Set("Content-Length", fmt.Sprint(len(content)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	root := t.TempDir()
	path, err := downloadFileToTemp(t.Context(), downloadJob{
		Version: models.ModVersion{
			DownloadURL: server.URL + "/fallback-name.jar",
			FileName:    "provided-name.jar",
		},
		TargetDir: filepath.Join(root, "versions", "test", "mods"),
		UserAgent: "mod-downloader/v1.2.3",
	})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "provided-name.jar" {
		t.Fatalf("path = %q", path)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content = %q", got)
	}
}

func TestDownloadFileToTempPreservesProviderURL(t *testing.T) {
	resetDownloadQueueForTest(t)
	requests := make(chan filetransfer.Request, 1)
	newFileTransferBackend = func() filetransfer.Backend {
		return requestCaptureTransferBackend{requests: requests}
	}
	t.Cleanup(func() {
		newFileTransferBackend = func() filetransfer.Backend { return filetransfer.NewStdlibBackend(nil) }
	})
	const providerURL = "https://cdn.modrinth.com/data/project/version/mod.jar?download=1"
	if _, err := downloadFileToTemp(t.Context(), downloadJob{
		Version:   models.ModVersion{DownloadURL: providerURL, FileName: "mod.jar"},
		TargetDir: filepath.Join(t.TempDir(), "mods"),
	}); err != nil {
		t.Fatal(err)
	}
	if req := <-requests; req.URL != providerURL {
		t.Fatalf("transfer URL = %q, want provider URL %q", req.URL, providerURL)
	}
}

func TestDownloadFileToTempRetriesSHA1Mismatch(t *testing.T) {
	resetDownloadQueueForTest(t)
	Configure(Config{VerifySHA1: true})
	t.Cleanup(func() { Configure(Config{}) })

	bad := []byte("corrupt download")
	good := []byte("verified download")
	digest := sha1.Sum(good)
	backend := &sequencedTransferBackend{contents: [][]byte{bad, good}}
	newFileTransferBackend = func() filetransfer.Backend { return backend }
	t.Cleanup(func() {
		newFileTransferBackend = func() filetransfer.Backend { return filetransfer.NewStdlibBackend(nil) }
	})

	path, err := downloadFileToTemp(t.Context(), downloadJob{
		Version: models.ModVersion{
			DownloadURL: "https://example.test/mod.jar",
			FileName:    "mod.jar",
			SHA1:        hex.EncodeToString(digest[:]),
		},
		TargetDir: filepath.Join(t.TempDir(), "mods"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if backend.calls != 2 {
		t.Fatalf("download attempts = %d, want 2", backend.calls)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, good) {
		t.Fatalf("downloaded content = %q, want %q", content, good)
	}
}

func TestDownloadFileToTempRemovesFileAfterSHA1RetriesExhausted(t *testing.T) {
	resetDownloadQueueForTest(t)
	Configure(Config{VerifySHA1: true})
	t.Cleanup(func() { Configure(Config{}) })

	digest := sha1.Sum([]byte("expected"))
	backend := &sequencedTransferBackend{contents: [][]byte{
		[]byte("bad 1"), []byte("bad 2"), []byte("bad 3"),
	}}
	newFileTransferBackend = func() filetransfer.Backend { return backend }
	t.Cleanup(func() {
		newFileTransferBackend = func() filetransfer.Backend { return filetransfer.NewStdlibBackend(nil) }
	})

	root := t.TempDir()
	path, err := downloadFileToTemp(t.Context(), downloadJob{
		Version: models.ModVersion{
			DownloadURL: "https://example.test/mod.jar",
			FileName:    "mod.jar",
			SHA1:        hex.EncodeToString(digest[:]),
		},
		TargetDir: filepath.Join(root, "mods"),
	})
	if err == nil {
		t.Fatal("downloadFileToTemp() error = nil, want SHA1 mismatch")
	}
	if backend.calls != 3 {
		t.Fatalf("download attempts = %d, want 3", backend.calls)
	}
	if path != "" {
		t.Fatalf("path = %q, want empty", path)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".mod-downloader-tmp", "mod.jar")); !os.IsNotExist(statErr) {
		t.Fatalf("temporary download file still exists, stat error = %v", statErr)
	}
}

type sequencedTransferBackend struct {
	contents [][]byte
	calls    int
}

func (*sequencedTransferBackend) Kind() filetransfer.BackendKind {
	return filetransfer.BackendStdlib
}

func (b *sequencedTransferBackend) Download(_ context.Context, req filetransfer.Request, _ *filetransfer.ProgressTracker) (filetransfer.Result, error) {
	content := b.contents[b.calls]
	b.calls++
	if err := os.WriteFile(req.Destination, content, 0o644); err != nil {
		return filetransfer.Result{}, err
	}
	return filetransfer.Result{Path: req.Destination}, nil
}

type requestCaptureTransferBackend struct {
	requests chan<- filetransfer.Request
}

func (requestCaptureTransferBackend) Kind() filetransfer.BackendKind {
	return filetransfer.BackendStdlib
}

func (b requestCaptureTransferBackend) Download(_ context.Context, req filetransfer.Request, _ *filetransfer.ProgressTracker) (filetransfer.Result, error) {
	b.requests <- req
	return filetransfer.Result{Path: req.Destination}, nil
}

func TestSetDownloadRequestHeaders(t *testing.T) {
	headers := downloadHeaders(downloadJob{
		Version:          models.ModVersion{DownloadURL: "https://edge.forgecdn.net/files/example.jar"},
		CurseForgeAPIKey: "secret",
		UserAgent:        "mod-downloader/v1.2.3",
	})
	if got := headers["User-Agent"]; got != "mod-downloader/v1.2.3" {
		t.Fatalf("User-Agent = %q", got)
	}
	if got := headers["x-api-key"]; got != "secret" {
		t.Fatalf("x-api-key = %q", got)
	}
}

func TestDiscardFetchFromNetworkSetsUserAgent(t *testing.T) {
	got := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		got <- req.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	DiscardFetchFromNetwork(server.URL, "", "mod-downloader/v1.2.3")
	if userAgent := <-got; userAgent != "mod-downloader/v1.2.3" {
		t.Fatalf("User-Agent = %q", userAgent)
	}
}

func TestOptionalDependencyReminderStateDismissAndClear(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.nextReminderID = 1
	downloadQueue.optionalReminders = []optionalDependencyReminderState{{
		Snapshot: appstructs.OptionalDependencyReminder{
			ID:             "optional-1",
			MainProjectKey: "modrinth:main",
			MainTitle:      "Main",
			Dependencies: []appstructs.OptionalDependencyCandidate{{
				ProjectKey: "modrinth:optional",
				Title:      "Optional",
			}},
		},
	}}
	downloadQueue.Unlock()

	state := GetDownloadQueueState()
	if state.Active || state.MessageCount != 1 || len(state.OptionalReminders) != 1 {
		t.Fatalf("state = %#v", state)
	}
	if !DismissOptionalDependencyReminder("optional-1") {
		t.Fatal("DismissOptionalDependencyReminder() = false, want true")
	}
	state = GetDownloadQueueState()
	if state.MessageCount != 0 || len(state.OptionalReminders) != 0 {
		t.Fatalf("state after dismiss = %#v", state)
	}

	downloadQueue.Lock()
	downloadQueue.optionalReminders = []optionalDependencyReminderState{{Snapshot: appstructs.OptionalDependencyReminder{ID: "optional-2"}}}
	downloadQueue.Unlock()
	if !ClearOptionalDependencyReminders() {
		t.Fatal("ClearOptionalDependencyReminders() = false, want true")
	}
	if state = GetDownloadQueueState(); state.MessageCount != 0 {
		t.Fatalf("state after clear = %#v", state)
	}
}

func TestInstallOptionalDependenciesConsumesReminder(t *testing.T) {
	resetDownloadQueueForTest(t)
	downloadQueue.Lock()
	downloadQueue.optionalReminders = []optionalDependencyReminderState{{
		Snapshot: appstructs.OptionalDependencyReminder{ID: "optional-1"},
		Requests: []appstructs.ModDownloadRequest{{
			ProjectID:        "",
			MinecraftVersion: "1.21.1",
			ModLoader:        "fabric",
			TargetDir:        t.TempDir(),
			InstanceID:       "cli:test",
		}},
	}}
	downloadQueue.Unlock()

	results := InstallOptionalDependencies(nil, "optional-1", "")
	if len(results) != 1 || !results[0].Skipped {
		t.Fatalf("results = %#v, want one skipped result", results)
	}
	state := GetDownloadQueueState()
	if state.MessageCount != 0 || len(state.OptionalReminders) != 0 {
		t.Fatalf("state after install = %#v", state)
	}
}

// shutdownBlockingBackend blocks until its context is canceled so shutdown
// tests can observe running jobs being drained.
type shutdownBlockingBackend struct {
	started  chan struct{}
	canceled chan struct{}
	once     sync.Once
}

func (*shutdownBlockingBackend) Kind() filetransfer.BackendKind {
	return filetransfer.BackendStdlib
}

func (b *shutdownBlockingBackend) Download(ctx context.Context, _ filetransfer.Request, _ *filetransfer.ProgressTracker) (filetransfer.Result, error) {
	b.once.Do(func() { close(b.started) })
	<-ctx.Done()
	select {
	case <-b.canceled:
	default:
		close(b.canceled)
	}
	return filetransfer.Result{}, ctx.Err()
}

func blockingDownloadJob(t *testing.T) downloadJob {
	t.Helper()
	return downloadJob{
		Version: models.ModVersion{
			ID:           "blocking-version",
			VersionID:    "blocking-version",
			Platform:     "Modrinth",
			ProjectID:    "blocking",
			FileName:     "blocking.jar",
			DownloadURL:  "https://example.test/blocking.jar",
			GameVersions: []string{"1.21.1"},
			Loaders:      []string{"fabric"},
			ModIDs:       []string{"blocking-mod"},
		},
		ProjectID:        "modrinth:blocking",
		TargetDir:        t.TempDir(),
		InstanceID:       "test:blocking",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		dependenciesDone: true,
	}
}

func TestShutdownCancelsRunningWorkersAndRejectsNewJobs(t *testing.T) {
	resetDownloadQueueForTest(t)
	t.Cleanup(Start)
	backend := &shutdownBlockingBackend{started: make(chan struct{}), canceled: make(chan struct{})}
	newFileTransferBackend = func() filetransfer.Backend { return backend }
	downloadQueue.Lock()
	downloadQueue.maxConcurrent = 1
	downloadQueue.Unlock()

	queueID, queued := enqueueDownload(t.Context(), blockingDownloadJob(t), Events{})
	if !queued {
		t.Fatal("enqueueDownload() = false, want true")
	}
	select {
	case <-backend.started:
	case <-time.After(2 * time.Second):
		t.Fatal("download backend never started")
	}

	waitDone := make(chan DownloadWaitResult, 1)
	waitErr := make(chan error, 1)
	go func() {
		result, err := WaitForDownload(context.Background(), queueID)
		if err != nil {
			waitErr <- err
			return
		}
		waitDone <- result
	}()

	shutdownDone := make(chan struct{})
	go func() {
		Shutdown()
		close(shutdownDone)
	}()
	select {
	case <-shutdownDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not return with a running worker")
	}
	select {
	case <-backend.canceled:
	case <-time.After(time.Second):
		t.Fatal("running transfer was not canceled by Shutdown")
	}
	select {
	case result := <-waitDone:
		if result.Status != "canceled" || result.Reason != "shutdown" {
			t.Fatalf("waiter result = %#v, want canceled/shutdown", result)
		}
	case err := <-waitErr:
		t.Fatalf("WaitForDownload() error = %v, want terminal result", err)
	case <-time.After(time.Second):
		t.Fatal("WaitForDownload did not unblock on Shutdown")
	}

	if !IsClosed() {
		t.Fatal("IsClosed() = false after Shutdown")
	}
	if _, ok := enqueueDownload(t.Context(), blockingDownloadJob(t), Events{}); ok {
		t.Fatal("enqueueDownload() accepted a job after Shutdown")
	}
	if RetryDownload(nil, "download-1") {
		t.Fatal("RetryDownload() succeeded after Shutdown")
	}
	state := GetDownloadQueueState()
	if state.Active || state.Pending != 0 || state.Running != 0 {
		t.Fatalf("state after Shutdown = %#v", state)
	}
	if _, err := WaitForDownload(context.Background(), "download-unknown"); !errors.Is(err, errDownloadQueueUnavailable) {
		t.Fatalf("WaitForDownload() after Shutdown error = %v, want unavailable", err)
	}
}

func TestWaitForDownloadTracksItsOwnQueueEntry(t *testing.T) {
	resetDownloadQueueForTest(t)
	t.Cleanup(Start)
	backend := &shutdownBlockingBackend{started: make(chan struct{}), canceled: make(chan struct{})}
	newFileTransferBackend = func() filetransfer.Backend { return backend }
	downloadQueue.Lock()
	downloadQueue.maxConcurrent = 1
	downloadQueue.Unlock()

	firstID, queued := enqueueDownload(t.Context(), blockingDownloadJob(t), Events{})
	if !queued {
		t.Fatal("first enqueueDownload() = false")
	}
	secondID, queued := enqueueDownload(t.Context(), blockingDownloadJob(t), Events{})
	if !queued {
		t.Fatal("second enqueueDownload() = false")
	}
	select {
	case <-backend.started:
	case <-time.After(2 * time.Second):
		t.Fatal("download backend never started")
	}

	waiting := make(chan DownloadWaitResult, 1)
	go func() {
		result, err := WaitForDownload(context.Background(), secondID)
		if err != nil {
			waiting <- DownloadWaitResult{Status: "error: " + err.Error()}
			return
		}
		waiting <- result
	}()

	if !CancelDownload(nil, secondID) {
		t.Fatal("CancelDownload(second) = false")
	}
	select {
	case result := <-waiting:
		if result.Status != "canceled" {
			t.Fatalf("second install result = %#v, want canceled", result)
		}
	case <-time.After(time.Second):
		t.Fatal("WaitForDownload(second) did not observe cancellation")
	}
	// The first, unrelated job must still be running.
	state := GetDownloadQueueState()
	if state.Running != 1 {
		t.Fatalf("state after canceling the second job = %#v, want first job still running", state)
	}
	if firstID == "" || firstID == secondID {
		t.Fatalf("queue ids = %q / %q, want distinct ids", firstID, secondID)
	}
	Shutdown()
}
