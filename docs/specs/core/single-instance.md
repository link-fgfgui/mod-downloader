# Global Single Instance (Port Lock)

## Decision

The GUI and TUI share one process-level single-instance guard. At startup the
host binds `127.0.0.1:18801` (`httpserver.DefaultAddr`, also the browser-extension
receiver port). **The port is the lock**: startup may continue only if the bind
succeeds. A failed bind returns `instance.ErrAlreadyRunning`, and the adapter must
abort before creating a window or opening the shared SQLite/cache.

- Implementation: `core/instance` (`Acquire` / `Guard`)
- Integration point: `core/host` `Start` (both GUI and TUI go through it; no
  second implementation)
- Failure type: `instance.ErrAlreadyRunning` (detectable with `errors.Is`)
- Tests: `core/instance/instance_test.go`, `core/host/host_lifecycle_test.go`

## How it works

1. `host.Start` calls `instance.Acquire(instanceAddr)` before starting core.
2. `Acquire` runs `net.Listen` on `127.0.0.1:18801`:
   - success: returns a `Guard`; the host keeps the listener until `Shutdown`;
   - failure: returns a wrapped `ErrAlreadyRunning`; `Host.Start` fails
     immediately without initializing core.
3. The extension HTTP server no longer binds 18801 itself. When the resolved
   `HTTPAddr` equals the guard address, the host passes the guard listener to
   `httpserver.Options.Listener`, so the port is bound once.
4. `Shutdown` stops the HTTP server and core, then closes the guard; the next
   process can bind again.
5. The GUI's `App.ServiceStartup` returns the error for `ErrAlreadyRunning`, so
   Wails exits before showing a window (no second window or SQLite connection).
   `main` recognizes the error and exits quietly with status 0. The TUI's
   `host.Start` error propagates through `runWith` and exits before entering
   tview.

## Constraints

**Don't**

- Do not add a lock file, PID file, or Wails `SingleInstance` (D-Bus) as a second
  mechanism: the single-instance rule must exist once, and the port is already
  the natural lock point for the extension server.
- Do not copy the `18801` constant into `app.go`, `main.go`, TUI `internal/`, or
  the frontend. Use `instance.DefaultAddr` / `httpserver.DefaultAddr`.
- Do not bind the real 18801 in host or TUI tests: set
  `host.Options.InstanceAddr` and `HTTPAddr` to `127.0.0.1:0`, otherwise a running
  desktop instance makes the whole test suite fail.
- Do not initialize core or show the main window after `Acquire` fails: once a
  second process opens shared storage, single-instance has no meaning.

**Instead**

- Adapters only consume `ErrAlreadyRunning`: the GUI aborts before creating the
  window and exits quietly; the TUI returns the error.
- If "raise the first window / forward launch arguments" is needed later, add an
  explicit handoff endpoint on the guard port and document the protocol here.
  The current implementation only detects and rejects a second instance.
- A new frontend gets the guard automatically by reusing `host.Start`; do not
  write another startup lifecycle.

## Trade-offs

- If **another program** owns the port, startup is also rejected: this is the
  direct consequence of "the port is the lock". The error includes the underlying
  `bind: address already in use` for diagnosis. Allowing startup while an external
  program owns the port would require designing a second lock first.
- A second instance does not raise the first window. This is intentional scope
  control, avoiding a startup race by serving HTTP on the guard listener early.
