package modbridge

import (
	"os"
	"path/filepath"
	"testing"

	"mod-downloader/database"
	"mod-downloader/global"
	"mod-downloader/models"
	mcstructs "mod-downloader/structs/minecraft"
)

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
		database.Close()
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	})
	if err := database.Open(); err != nil {
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
