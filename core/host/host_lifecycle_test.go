// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package host

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader/core/appcore"
	"github.com/link-fgfgui/mod-downloader/core/instance"
)

// newLifecycleHost isolates every filesystem lookup the core performs: the
// config is read from the working directory, while the log and cache files use
// the user cache or temp directories. Pointing all of them at one temporary
// directory keeps the real lifecycle tests repeatable and keeps them from
// touching the developer's own state. The instance guard and HTTP server use
// ephemeral ports so a running desktop app never blocks the test suite.
func newLifecycleHost(t *testing.T) *Host {
	t.Helper()
	return newLifecycleHostWith(t, Options{})
}

// newLifecycleHostWith applies test overrides on top of the isolated defaults.
func newLifecycleHostWith(t *testing.T, opts Options) *Host {
	t.Helper()
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("XDG_CACHE_HOME", dir)
	t.Setenv("HOME", dir)
	t.Setenv("LOCALAPPDATA", dir)
	t.Setenv("TMPDIR", dir)
	opts.Version = "test"
	opts.LoadMinecraftReleases = false
	opts.LogToFileOnly = true
	opts.LogLevel = slog.LevelError
	if opts.InstanceAddr == "" {
		opts.InstanceAddr = "127.0.0.1:0"
	}
	if opts.HTTPAddr == "" {
		opts.HTTPAddr = "127.0.0.1:0"
	}
	h := New(opts)
	t.Cleanup(h.Shutdown)
	return h
}

func TestStartShutdownLifecycle(t *testing.T) {
	h := newLifecycleHost(t)

	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !h.Started() {
		t.Fatal("host does not report started after Start")
	}
	if svc := h.Service(); svc == nil {
		t.Fatal("Service() = nil after Start")
	}
	if cfg := h.Config(); cfg == nil {
		t.Fatal("Config() = nil after Start")
	}
	if got := h.Version(); got != "test" {
		t.Fatalf("Version() = %q, want %q", got, "test")
	}
	if status := h.StartupStatus(); !status.Ready || status.Error != "" {
		t.Fatalf("StartupStatus() = %+v after Start", status)
	}

	h.Shutdown()
	if h.Started() {
		t.Fatal("host still reports started after Shutdown")
	}
	if svc := h.Service(); svc != nil {
		t.Fatalf("Service() = %p after Shutdown, want nil", svc)
	}
	// A second Shutdown must be a harmless no-op.
	h.Shutdown()
}

func TestConfigIsNilBeforeStart(t *testing.T) {
	h := New(Options{Version: "test"})
	if cfg := h.Config(); cfg != nil {
		t.Fatalf("Config() = %p before Start, want nil", cfg)
	}
	if got := h.Version(); got != "test" {
		t.Fatalf("Version() = %q, want %q", got, "test")
	}
	h.Shutdown()
}

func TestHandlersRegisteredBeforeStartSurviveStart(t *testing.T) {
	h := newLifecycleHost(t)

	var events atomic.Int64
	OnEvent(h, appcore.EventDownloadCompleted, func() { events.Add(1) })

	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !h.Started() {
		t.Fatal("host does not report started after Start")
	}
	// Start must not clear handlers registered before startup.
	h.dispatch(appcore.Event{Kind: appcore.EventDownloadCompleted})
	if got := events.Load(); got != 1 {
		t.Fatalf("handler calls = %d after Start, want 1", got)
	}
	h.Shutdown()
	if h.Started() {
		t.Fatal("host still reports started after Shutdown")
	}
}

// TestSecondStartIsIdempotent verifies that calling Start on an already-started
// host returns nil and keeps the original core/service pair. Replacing the core
// would leak its background tasks and HTTP listener.
func TestSecondStartIsIdempotent(t *testing.T) {
	h := newLifecycleHost(t)

	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	first := h.Service()
	if first == nil {
		t.Fatal("Service() = nil after Start")
	}
	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	if !h.Started() {
		t.Fatal("host does not report started after a second Start")
	}
	if got := h.Service(); got != first {
		t.Fatalf("second Start replaced the core: %p != %p", got, first)
	}

	h.Shutdown()
	if h.Started() {
		t.Fatal("host still reports started after Shutdown")
	}
	if svc := h.Service(); svc != nil {
		t.Fatalf("Service() = %p after Shutdown, want nil", svc)
	}
	h.Shutdown()
}

// TestStartFailureCanBeRetried injects a failing core startup and verifies the
// host never caches the failed service and that a later Start succeeds.
func TestStartFailureCanBeRetried(t *testing.T) {
	h := newLifecycleHost(t)
	boom := errors.New("startup failed")
	calls := 0
	h.startCore = func(context.Context) (*appcore.Service, error) {
		calls++
		return nil, boom
	}

	if err := h.Start(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("Start error = %v, want %v", err, boom)
	}
	if calls != 1 {
		t.Fatalf("startCore calls = %d, want 1", calls)
	}
	if h.Started() {
		t.Fatal("host reports started after a failed Start")
	}
	if svc := h.Service(); svc != nil {
		t.Fatalf("Service() = %p after a failed Start, want nil", svc)
	}
	if cfg := h.Config(); cfg != nil {
		t.Fatalf("Config() = %p after a failed Start, want nil", cfg)
	}
	status := h.StartupStatus()
	if status.Ready || status.Stage != "unknown" || status.Error != boom.Error() {
		t.Fatalf("StartupStatus() = %+v after a failed Start", status)
	}

	// The failure must not poison the host: retry with the real starter.
	h.startCore = h.startCoreDefault
	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("retry Start: %v", err)
	}
	if !h.Started() || h.Service() == nil {
		t.Fatal("host did not start on retry")
	}
	if status := h.StartupStatus(); !status.Ready {
		t.Fatalf("StartupStatus() = %+v after retry", status)
	}
}

// TestRestartAfterShutdown verifies the host can be started again after a
// clean shutdown instead of staying permanently closed.
func TestRestartAfterShutdown(t *testing.T) {
	h := newLifecycleHost(t)

	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("first Start: %v", err)
	}
	first := h.Service()
	h.Shutdown()
	if h.Started() {
		t.Fatal("host still started after Shutdown")
	}

	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("restart: %v", err)
	}
	second := h.Service()
	if second == nil || !h.Started() {
		t.Fatal("host did not start on restart")
	}
	if second == first {
		t.Fatal("restart reused the shut-down core")
	}
	h.Shutdown()
	if h.Started() {
		t.Fatal("host still started after the second Shutdown")
	}
}

// TestConcurrentStartKeepsOneCore serializes competing Start calls; exactly one
// core survives and every caller sees success.
func TestConcurrentStartKeepsOneCore(t *testing.T) {
	h := newLifecycleHost(t)

	const callers = 8
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- h.Start(context.Background())
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Start: %v", err)
		}
	}

	core := h.Service()
	if core == nil {
		t.Fatal("Service() = nil after concurrent Start")
	}
	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("Start after concurrent callers: %v", err)
	}
	if got := h.Service(); got != core {
		t.Fatalf("later Start replaced the core: %p != %p", got, core)
	}
}

// TestHTTPServerSharesInstanceGuard proves the extension server serves on the
// guard socket instead of binding the singleton port a second time.
func TestHTTPServerSharesInstanceGuard(t *testing.T) {
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	addr := probe.Addr().String()
	if err := probe.Close(); err != nil {
		t.Fatalf("release reserved port: %v", err)
	}

	h := newLifecycleHostWith(t, Options{InstanceAddr: addr, HTTPAddr: addr})
	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	h.mu.Lock()
	serverAddr := ""
	if h.server != nil {
		serverAddr = h.server.Addr()
	}
	h.mu.Unlock()
	if serverAddr != addr {
		t.Fatalf("HTTP server addr = %q, want guard addr %q", serverAddr, addr)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatalf("health request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

// TestStartRejectsSecondInstance proves the guard is acquired before any core
// work: a process that cannot bind the singleton port must not start.
func TestStartRejectsSecondInstance(t *testing.T) {
	occupied, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer occupied.Close()

	h := newLifecycleHostWith(t, Options{InstanceAddr: occupied.Addr().String()})
	if err := h.Start(context.Background()); !errors.Is(err, instance.ErrAlreadyRunning) {
		t.Fatalf("Start error = %v, want ErrAlreadyRunning", err)
	}
	if h.Started() {
		t.Fatal("host reports started after a guard conflict")
	}
	if svc := h.Service(); svc != nil {
		t.Fatalf("Service() = %p after a guard conflict, want nil", svc)
	}
	status := h.StartupStatus()
	if status.Ready || status.Error == "" {
		t.Fatalf("StartupStatus() = %+v after a guard conflict", status)
	}
}

// TestShutdownReleasesInstanceGuard proves a clean shutdown releases the port
// so the next launch can start.
func TestShutdownReleasesInstanceGuard(t *testing.T) {
	h := newLifecycleHost(t)
	if err := h.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	h.mu.Lock()
	addr := h.instanceGuard.Addr()
	h.mu.Unlock()
	if addr == "" {
		t.Fatal("instance guard has no address after Start")
	}

	h.Shutdown()

	guard, err := instance.Acquire(addr)
	if err != nil {
		t.Fatalf("re-acquire released guard: %v", err)
	}
	if err := guard.Close(); err != nil {
		t.Fatalf("close re-acquired guard: %v", err)
	}
}
