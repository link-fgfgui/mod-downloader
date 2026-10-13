# Implementation Plan: Extract core library and add CLI

## Checklist

- [x] Confirm MVP CLI command surface with the user.
- [x] Read backend specs before code changes: directory structure, quality guidelines, database guidelines, and relevant thinking guides.
- [x] Add the core service package with lifecycle options and tests for initialization paths that do not import Wails.
- [x] Move reusable helper logic from `app.go` into the service package while keeping Wails dialogs/events in the Wails adapter.
- [x] Decouple downloader event emission from Wails runtime so the service can observe queue/download events without importing Wails.
- [x] Reroute Wails `App` methods through the service and preserve frontend-facing behavior.
- [x] Add a separate CLI binary entrypoint and command package.
- [x] Implement MVP commands with script-friendly JSON output: `config`, `versions`, `search`, `install`, and `mods`.
- [x] Update README with CLI build/run examples and command overview.
- [x] Run formatting and verification.

## Validation Commands

```bash
gofmt -w <modified-go-files>
go build ./...
go vet ./...
go test ./...
```

If Wails API signatures change:

```bash
wails generate module
npm run build --prefix frontend
```

## Review Gates

- Confirm no new package in the core service or CLI path imports `github.com/wailsapp/wails/v2/pkg/runtime`.
- Confirm no type aliases, re-export files, or parallel converter functions were introduced.
- Confirm Wails UI behavior still uses the same event names where the frontend expects them.
- Confirm CLI commands return non-zero exit codes for invalid arguments, missing instances, failed provider calls, and failed downloads.

## Rollback Points

- Service package extraction should land before CLI commands are expanded.
- Downloader event decoupling should be reviewed separately because it affects UI loading/error states.
- Wails adapter reroute should preserve old method signatures where practical so frontend rollback is small.
