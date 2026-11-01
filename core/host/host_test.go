// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package host

import (
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/appcore"
	"github.com/link-fgfgui/mod-downloader/core/logging"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

func TestNormalizeVersion(t *testing.T) {
	if got := normalizeVersion(" 1.2.3 "); got != "1.2.3" {
		t.Fatalf("normalizeVersion=%q", got)
	}
	if got := normalizeVersion("   "); got != "dev" {
		t.Fatalf("empty version=%q", got)
	}
}

func TestWithDefaultRuntimeUsesWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	opts := withDefaultRuntime(appcore.RuntimeOptions{})
	if opts.DefaultCacheDir != wd {
		t.Fatalf("DefaultCacheDir = %q, want %q", opts.DefaultCacheDir, wd)
	}
}

func TestWithDefaultRuntimeKeepsExplicitCacheDir(t *testing.T) {
	opts := withDefaultRuntime(appcore.RuntimeOptions{DefaultCacheDir: "/custom"})
	if opts.DefaultCacheDir != "/custom" {
		t.Fatalf("DefaultCacheDir = %q, want explicit value", opts.DefaultCacheDir)
	}
}

func TestDispatchWithoutHandlersDoesNotPanic(_ *testing.T) {
	h := New(Options{Version: "test"})
	h.dispatch(appcore.Event{Kind: appcore.EventDownloadCompleted})
}

// TestTypedHandlersReceiveEveryCoreEvent proves the whole event taxonomy is
// delivered through the typed API, including payload-less events.
func TestTypedHandlersReceiveEveryCoreEvent(t *testing.T) {
	h := New(Options{})

	var order []string
	var updates []appview.SearchModsUpdate
	var queues []appview.DownloadQueueState
	var failures []appstructs.DownloadFailedEvent
	var dirs []string
	var versions []structs.VersionInfo
	voids := 0

	On(h, appcore.EventSearchModsUpdated, func(u appview.SearchModsUpdate) {
		order = append(order, "search")
		updates = append(updates, u)
	})
	OnEvent(h, appcore.EventDownloadStatesUpdated, func() { order = append(order, "states"); voids++ })
	On(h, appcore.EventDownloadQueueUpdated, func(q appview.DownloadQueueState) {
		order = append(order, "queue")
		queues = append(queues, q)
	})
	On(h, appcore.EventDownloadFailed, func(f appstructs.DownloadFailedEvent) {
		order = append(order, "failed")
		failures = append(failures, f)
	})
	OnEvent(h, appcore.EventDownloadCompleted, func() { order = append(order, "completed"); voids++ })
	On(h, appcore.EventMinecraftDirChanged, func(dir string) {
		order = append(order, "dir")
		dirs = append(dirs, dir)
	})
	On(h, appcore.EventSelectedVersionChanged, func(v structs.VersionInfo) {
		order = append(order, "version")
		versions = append(versions, v)
	})
	OnEvent(h, appcore.EventUsageStatsUpdated, func() { order = append(order, "usage"); voids++ })
	On(h, appcore.EventExtensionModsAccepted, func(u appview.SearchModsUpdate) {
		order = append(order, "extension")
		updates = append(updates, u)
	})

	update := appview.SearchModsUpdate{RequestID: "req", Loading: true, Append: true}
	queue := appview.DownloadQueueState{Active: true, Running: 1}
	failed := appstructs.DownloadFailedEvent{FileName: "sodium.jar", Reason: "checksum"}
	version := structs.VersionInfo{ID: "inst-1", MinecraftVersion: "1.20.1", ModLoader: "Fabric"}

	h.dispatch(appcore.Event{Kind: appcore.EventSearchModsUpdated, Payload: update})
	h.dispatch(appcore.Event{Kind: appcore.EventDownloadStatesUpdated})
	h.dispatch(appcore.Event{Kind: appcore.EventDownloadQueueUpdated, Payload: queue})
	h.dispatch(appcore.Event{Kind: appcore.EventDownloadFailed, Payload: failed})
	h.dispatch(appcore.Event{Kind: appcore.EventDownloadCompleted})
	h.dispatch(appcore.Event{Kind: appcore.EventMinecraftDirChanged, Payload: "/tmp/.minecraft"})
	h.dispatch(appcore.Event{Kind: appcore.EventSelectedVersionChanged, Payload: version})
	h.dispatch(appcore.Event{Kind: appcore.EventUsageStatsUpdated})
	h.dispatch(appcore.Event{Kind: appcore.EventExtensionModsAccepted, Payload: update})

	wantOrder := []string{"search", "states", "queue", "failed", "completed", "dir", "version", "usage", "extension"}
	if strings.Join(order, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("order=%v, want %v", order, wantOrder)
	}
	if len(updates) != 2 || !reflect.DeepEqual(updates[0], update) || !reflect.DeepEqual(updates[1], update) {
		t.Fatalf("search updates=%+v", updates)
	}
	if len(queues) != 1 || !reflect.DeepEqual(queues[0], queue) {
		t.Fatalf("queues=%+v", queues)
	}
	if len(failures) != 1 || failures[0] != failed {
		t.Fatalf("failures=%+v", failures)
	}
	if len(dirs) != 1 || dirs[0] != "/tmp/.minecraft" {
		t.Fatalf("dirs=%+v", dirs)
	}
	if len(versions) != 1 || versions[0].ID != "inst-1" {
		t.Fatalf("versions=%+v", versions)
	}
	if voids != 3 {
		t.Fatalf("payload-less handlers called %d times, want 3", voids)
	}
}

// TestInvalidPayloadIsDropped locks the single assertion layer: wrong payloads
// never reach a handler.
func TestInvalidPayloadIsDropped(t *testing.T) {
	h := New(Options{})
	called := 0
	On(h, appcore.EventDownloadQueueUpdated, func(appview.DownloadQueueState) { called++ })
	On(h, appcore.EventSelectedVersionChanged, func(structs.VersionInfo) { called++ })
	On(h, appcore.EventExtensionModsAccepted, func(appview.SearchModsUpdate) { called++ })

	h.dispatch(appcore.Event{Kind: appcore.EventDownloadQueueUpdated, Payload: "nope"})
	h.dispatch(appcore.Event{Kind: appcore.EventSelectedVersionChanged, Payload: appview.SearchModsUpdate{}})
	h.dispatch(appcore.Event{Kind: appcore.EventExtensionModsAccepted, Payload: 42})
	if called != 0 {
		t.Fatalf("handlers called %d times for invalid payloads", called)
	}
}

// TestUnhandledEventsAreLogged locks in the contract-drift visibility rule.
func TestUnhandledEventsAreLogged(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "host.log")
	if err := logging.Configure(logging.Options{
		NoStderr: true,
		FilePath: logPath,
		Level:    slog.LevelWarn,
	}); err != nil {
		t.Fatalf("configure logging: %v", err)
	}
	t.Cleanup(logging.Close)

	h := New(Options{})
	On(h, appcore.EventDownloadQueueUpdated, func(appview.DownloadQueueState) {})
	On(h, appcore.EventExtensionModsAccepted, func(appview.SearchModsUpdate) {})
	h.dispatch(appcore.Event{Kind: "future-core-event", Payload: 42})
	h.dispatch(appcore.Event{Kind: appcore.EventDownloadQueueUpdated, Payload: 42})
	h.dispatch(appcore.Event{Kind: appcore.EventExtensionModsAccepted, Payload: 42})
	logging.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	logText := string(data)
	for _, want := range []string{
		"unhandled core event",
		"future-core-event",
		"invalid core event payload",
		"downloadQueueUpdated",
		"extensionModsAccepted",
	} {
		if !strings.Contains(logText, want) {
			t.Fatalf("log does not contain %q:\n%s", want, logText)
		}
	}
}

func TestServerNoticeStartsEmpty(t *testing.T) {
	h := New(Options{})
	if notice := h.ServerNotice(); notice != "" {
		t.Fatalf("notice=%q", notice)
	}
	if h.Version() != "dev" {
		t.Fatalf("version=%q", h.Version())
	}
}

func TestServiceIsNilUntilStartSucceeds(t *testing.T) {
	h := New(Options{})
	if h.Started() {
		t.Fatal("host reports started before Start")
	}
	if svc := h.Service(); svc != nil {
		t.Fatalf("Service() = %p before Start, want nil", svc)
	}
	h.Shutdown() // must be a no-op before Start
}
