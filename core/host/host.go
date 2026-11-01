// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package host owns the UI-independent runtime: the core service, the HTTP
// server and the typed event fan-out. GUI and TUI adapters only configure it
// and register handlers; neither reimplements lifecycle or event plumbing.
package host

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/link-fgfgui/mod-downloader/core/appcore"
	"github.com/link-fgfgui/mod-downloader/core/configs"
	"github.com/link-fgfgui/mod-downloader/core/httpserver"
	"github.com/link-fgfgui/mod-downloader/core/instance"
	"github.com/link-fgfgui/mod-downloader/core/logging"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
)

// Options configures the host lifecycle.
type Options struct {
	// Version is the normalized app version (empty falls back to "dev").
	Version string
	// LoadMinecraftReleases mirrors the GUI (true).
	LoadMinecraftReleases bool
	// Config optionally supplies a pre-loaded config.
	Config *configs.Config
	// Runtime overrides core runtime paths. The default cache dir falls back
	// to the working directory when empty.
	Runtime appcore.RuntimeOptions
	// LogLevel is the minimum level emitted by the core logger (zero = info).
	LogLevel slog.Level
	// LogToFileOnly keeps logs out of stderr; terminal UIs own the terminal.
	LogToFileOnly bool
	// DisableHTTPServer skips the extension HTTP server (one-shot CLI runs).
	DisableHTTPServer bool
	// HTTPAddr overrides httpserver.DefaultAddr when non-empty.
	HTTPAddr string
	// InstanceAddr overrides the single-instance sentinel address. It defaults
	// to instance.DefaultAddr, the extension port shared with the TUI. Tests
	// pass 127.0.0.1:0 so suites never contend for the real port.
	InstanceAddr string
}

// Host owns the core service and the HTTP server.
type Host struct {
	version               string
	loadMinecraftReleases bool
	config                *configs.Config
	runtime               appcore.RuntimeOptions
	logLevel              slog.Level
	logToFileOnly         bool
	disableHTTPServer     bool
	httpAddr              string
	instanceAddr          string

	// startMu serializes Start and Shutdown so a second Start cannot replace a
	// running core (or leak its HTTP server).
	startMu sync.Mutex
	// startCore is replaceable in tests to exercise startup failures.
	startCore func(context.Context) (*appcore.Service, error)

	mu            sync.Mutex
	core          *appcore.Service
	server        *httpserver.Server
	instanceGuard *instance.Guard
	cfg           *configs.Config
	startupErr    error
	serverErr     error
	serverNotice  string
	handlers      map[appcore.EventKind][]func(any)
}

// New builds the host without starting it.
func New(opts Options) *Host {
	h := &Host{
		version:               normalizeVersion(opts.Version),
		loadMinecraftReleases: opts.LoadMinecraftReleases,
		config:                opts.Config,
		runtime:               withDefaultRuntime(opts.Runtime),
		logLevel:              opts.LogLevel,
		logToFileOnly:         opts.LogToFileOnly,
		disableHTTPServer:     opts.DisableHTTPServer,
		httpAddr:              strings.TrimSpace(opts.HTTPAddr),
		instanceAddr:          strings.TrimSpace(opts.InstanceAddr),
		handlers:              map[appcore.EventKind][]func(any){},
	}
	if h.instanceAddr == "" {
		h.instanceAddr = instance.DefaultAddr
	}
	h.startCore = h.startCoreDefault
	return h
}

// NewWithService wraps an already constructed service without running the
// startup sequence. Adapter tests use it to exercise bindings against a
// hand-built core; production code uses New + Start.
func NewWithService(core *appcore.Service) *Host {
	h := New(Options{})
	if core != nil {
		h.core = core
		h.cfg = core.Config()
	}
	return h
}

// normalizeVersion mirrors currentAppVersion(): trim and fall back to "dev".
func normalizeVersion(version string) string {
	if v := strings.TrimSpace(version); v != "" {
		return v
	}
	return "dev"
}

// withDefaultRuntime fills the cache dir the GUI and TUI used to resolve
// separately.
func withDefaultRuntime(runtime appcore.RuntimeOptions) appcore.RuntimeOptions {
	if strings.TrimSpace(runtime.DefaultCacheDir) != "" {
		return runtime
	}
	dir, err := os.Getwd()
	if err != nil {
		logging.Error("resolve default cache dir failed", "error", err)
		return runtime
	}
	runtime.DefaultCacheDir = dir
	return runtime
}

// On registers a typed handler for one core event. The payload assertion is
// performed once here; contract drift is logged instead of panicking.
func On[T any](h *Host, kind appcore.EventKind, fn func(T)) {
	if h == nil || fn == nil {
		return
	}
	h.addHandler(kind, func(payload any) {
		typed, ok := payload.(T)
		if !ok {
			logging.Warn("invalid core event payload", "kind", kind, "payloadType", payloadType(payload))
			return
		}
		fn(typed)
	})
}

// OnEvent registers a handler for an event that carries no payload.
func OnEvent(h *Host, kind appcore.EventKind, fn func()) {
	if h == nil || fn == nil {
		return
	}
	h.addHandler(kind, func(any) { fn() })
}

func (h *Host) addHandler(kind appcore.EventKind, handler func(any)) {
	h.mu.Lock()
	h.handlers[kind] = append(h.handlers[kind], handler)
	h.mu.Unlock()
}

// Start acquires the process-wide instance guard, creates and starts the core
// service, then starts the HTTP server. A guard conflict is fatal (another
// process owns the extension port); an HTTP listen failure on an explicitly
// overridden address is logged and surfaced as a non-fatal notice. Calling
// Start on an already-started host is a no-op so callers cannot replace a
// running core.
func (h *Host) Start(ctx context.Context) error {
	h.startMu.Lock()
	defer h.startMu.Unlock()
	if h.Started() {
		return nil
	}

	guard, err := instance.Acquire(h.instanceAddr)
	if err != nil {
		h.mu.Lock()
		h.startupErr = err
		h.serverErr = nil
		h.serverNotice = ""
		h.mu.Unlock()
		return err
	}

	core, err := h.startCore(ctx)
	if err != nil {
		if core != nil {
			// Startup already tears down partially initialized resources; Close
			// is called again defensively and is a no-op when there is nothing
			// to do.
			core.Close()
		}
		// Release the guard so a retry (after fixing the config) can start.
		_ = guard.Close()
		h.mu.Lock()
		h.startupErr = err
		h.serverErr = nil
		h.serverNotice = ""
		h.mu.Unlock()
		return err
	}

	h.mu.Lock()
	h.core = core
	h.cfg = core.Config()
	h.instanceGuard = guard
	h.startupErr = nil
	h.serverErr = nil
	h.mu.Unlock()

	if h.disableHTTPServer {
		return nil
	}
	addr := h.httpAddr
	if addr == "" {
		addr = httpserver.DefaultAddr
	}
	options := httpserver.Options{Accept: core.AcceptExtensionProjects}
	if guard.Addr() == addr {
		// The guard already owns the extension port; serve on its socket
		// instead of binding the same address twice.
		options.Listener = guard.Listener()
	}
	server := httpserver.New(addr, options)
	h.mu.Lock()
	h.server = server
	h.mu.Unlock()
	if err := server.Start(); err != nil {
		logging.Error("start http server failed", "error", err)
		h.mu.Lock()
		h.serverErr = err
		h.serverNotice = fmt.Sprintf("http server unavailable on %s: %v", addr, err)
		h.mu.Unlock()
	}
	return nil
}

// startCoreDefault creates and starts a core service. It returns the service
// even when Startup fails so Start can close partially initialized resources.
func (h *Host) startCoreDefault(ctx context.Context) (*appcore.Service, error) {
	core := appcore.New(appcore.Options{
		Config:                h.config,
		Runtime:               h.runtime,
		Version:               h.version,
		LoadMinecraftReleases: h.loadMinecraftReleases,
		OnEvent:               h.dispatch,
		LogLevel:              h.logLevel,
		LogToFileOnly:         h.logToFileOnly,
	})
	if err := core.Startup(ctx); err != nil {
		return core, err
	}
	return core, nil
}

// dispatch forwards a core event to every registered handler. Events with no
// handler are logged so contract drift stays visible.
func (h *Host) dispatch(event appcore.Event) {
	h.mu.Lock()
	handlers := append([]func(any){}, h.handlers[event.Kind]...)
	h.mu.Unlock()
	if len(handlers) == 0 {
		logging.Warn("unhandled core event", "kind", event.Kind, "payloadType", payloadType(event.Payload))
		return
	}
	for _, handler := range handlers {
		handler(event.Payload)
	}
}

// Service returns the started core service, or nil before Start succeeds.
// Callers must not assume a usable service before Started reports true.
func (h *Host) Service() *appcore.Service {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.core != nil {
		h.cfg = h.core.Config()
	}
	return h.core
}

// Config returns the latest loaded config.
func (h *Host) Config() *configs.Config {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.core != nil {
		h.cfg = h.core.Config()
	}
	return h.cfg
}

// Version returns the normalized version string.
func (h *Host) Version() string { return h.version }

// Started reports whether the core service finished Startup.
func (h *Host) Started() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.core != nil
}

// StartupStatus is the typed readiness snapshot exposed to adapters.
func (h *Host) StartupStatus() appstructs.StartupStatus {
	h.mu.Lock()
	core := h.core
	startupErr := h.startupErr
	serverErr := h.serverErr
	h.mu.Unlock()

	status := appstructs.StartupStatus{Ready: core != nil && startupErr == nil}
	if startupErr != nil {
		status.Stage = "unknown"
		status.Error = startupErr.Error()
		var typed *appcore.StartupError
		if errors.As(startupErr, &typed) {
			status.Stage = typed.Stage
			status.Error = typed.Err.Error()
		}
	}
	if serverErr != nil {
		status.ExtensionServerError = serverErr.Error()
	}
	return status
}

// ServerNotice returns a non-blocking notice about HTTP server availability.
func (h *Host) ServerNotice() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.serverNotice
}

// Shutdown stops the server then the core, releases the instance guard and
// clears all handles.
func (h *Host) Shutdown() {
	h.startMu.Lock()
	defer h.startMu.Unlock()
	h.mu.Lock()
	server := h.server
	core := h.core
	guard := h.instanceGuard
	h.server = nil
	h.core = nil
	h.instanceGuard = nil
	h.serverErr = nil
	h.serverNotice = ""
	h.mu.Unlock()
	if server != nil {
		server.Stop()
	}
	if core != nil {
		core.Shutdown()
	}
	if guard != nil {
		// The HTTP server shares this listener; closing it again is harmless.
		_ = guard.Close()
	}
}

func payloadType(payload any) string {
	return fmt.Sprintf("%T", payload)
}
