# App Backend Guidelines

> Package-scoped entry point for the Wails app shell in `mod-downloader`.

The app package owns Wails runtime integration, frontend bindings, and adapter
code in `app.go`. Reusable domain logic belongs in the `core/` submodule.

## Required References

- [Shared Backend Guidelines](../../backend/index.md)
- [Directory Structure](../../backend/directory-structure.md)
- [Quality Guidelines](../../backend/quality-guidelines.md)
- [Build Version](./build-version.md)

## Pre-Development Checklist

- Check whether the change belongs in `app.go` / frontend bindings or in the
  `core/` submodule.
- Keep Wails runtime imports in the app adapter layer only.
- Preserve `replace github.com/link-fgfgui/mod-downloader-core => ./core`.
- Inject release identity through `APP_VERSION` and Go `-X`; see
  [Build Version](./build-version.md).

## Quality Check

- Run `go test ./...` from the app repo after app adapter changes.
- Run `go build ./...` when Go signatures or imports change.
- If public Wails API signatures change, regenerate frontend bindings.

## Convention: Linux Wails Dev And Browser Automation

- Unless a task explicitly requires native-app verification, do not start
  `wails dev`; use the normal non-native checks instead.
- When native-app verification is explicitly required on Linux, start it with
  both WebKit build-tag arguments:

  ```bash
  wails dev -tags webkit2_41
  ```

- For browser automation against a running Wails development instance, parse
  the Wails dev server URL/port from the `wails dev` process output. Do not use
  the Vite port: it is an internal frontend development endpoint and is not the
  browser-facing Wails runtime under test.

Wrong:

```bash
wails dev
# Automation opens the Vite URL/port.
```

Correct:

```bash
wails dev -tags webkit2_41
# Automation waits for output, extracts the Wails dev URL/port, and opens it.
```

## Scenario: Wails API And Frontend Binding Changes

### 1. Scope / Trigger

Use this when adding, removing, or changing any public method on `App` in
`app.go`, or changing a Go request/response struct consumed by Wails-generated
frontend code.

### 2. Signatures

Wails adapter methods live on `*App`:

```go
func (a *App) QueueModDownload(req structs.ModDownloadRequest) structs.ModDownloadResult
func (a *App) GetDownloadQueueState() structs.DownloadQueueState
```

Core logic stays behind `appcore.Service`; `app.go` forwards only.

### 3. Contracts

- Public Wails method names are frontend API names. Keep them stable unless the
  task owns a frontend migration.
- Shared payload structs live in `core/structs` or `core/models`, not in
  `app.go`.
- `app.go` may import Wails runtime; `core/appcore` and lower layers must not.
- `main.go` embeds `all:frontend/dist`, so Wails generation and Go build need
  `frontend/dist` to exist with at least one file.

### 4. Validation & Error Matrix

- Missing `frontend/dist` -> `wails generate module` / `go build` fails with a
  `pattern all:frontend/dist: no matching files found` error.
- Changed `App` method or shared payload but stale `frontend/wailsjs` -> frontend
  type-check may compile against the wrong API.
- Frontend dependencies absent -> `cd frontend && npm run build` fails before
  type-check; install dependencies or report that the frontend build could not
  be run.

### 5. Good/Base/Bad Cases

- Good: add `func (a *App) NewAction(req structs.NewRequest)` in `app.go`,
  forward to `a.service().NewAction(req)`, run `wails generate module`, then
  run frontend type-check/build.
- Base: when no production build exists yet, create or build `frontend/dist`
  before running Wails generation because the embed pattern is evaluated while
  loading the Go package.
- Bad: hand-editing `frontend/wailsjs` without attempting Wails generation when
  the generator is available.
- Bad: importing `github.com/wailsapp/wails/v2/pkg/runtime` into `core/appcore`
  to emit UI events directly.

### 6. Tests Required

- `wails generate module` after public Wails API or payload changes.
- `cd frontend && npm run build` to verify generated TypeScript consumers.
- `go build ./... && go vet ./... && go test ./...` from the app repo after
  adapter changes.
- `cd core && go test ./...` when the adapter exposes changed core behavior.

### 7. Wrong vs Correct

Wrong:

```go
// core/appcore/service.go
import "github.com/wailsapp/wails/v2/pkg/runtime"
```

Correct:

```go
// app.go
func (a *App) NewAction(req structs.NewRequest) structs.NewResult {
    return a.service().NewAction(req)
}
```
