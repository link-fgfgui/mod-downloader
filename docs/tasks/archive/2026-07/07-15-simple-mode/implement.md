# Implementation Plan

1. [x] Add the `preferences.simple_mode` configuration field and configuration load/save/default tests.
2. [x] Add the core and app settings view/request contracts plus `SaveSimpleModeSettings`; apply the mode during service startup and immediately after save.
3. [x] Add atomic simple-mode policy in `modbridge`, clear pending backfills on enable, and guard cache reads, precise status, incompatible analysis, backfill scheduling, and the final pre-parser boundary.
4. [x] Make `DownloadStates` return ordinary default states in simple mode regardless of local or remote ID caches.
5. [x] Gate downloader dependency preparation and optional-dependency actions, clear existing reminders on enable, and keep single/batch file downloads on the local-parse fallback.
6. [x] Add the Wails adapter contract/method, regenerate frontend bindings, and verify every `simpleMode` field maps through the generated API.
7. [x] Add the localized Settings switch and Pinia load/save/autosave state using the existing settings patterns.
8. [x] Add focused tests for runtime toggles, queued/in-flight parser behavior, cache ignoring, no backfills, dependency/reminder gating, default button states, and basic local-parse downloads.
9. [x] Run formatting, generated-binding checks, frontend checks, core race/focused/full tests, and app build/vet/tests.

## Validation

```bash
cd core && gofmt -w configs appcore modbridge downloader
cd core && go test ./configs ./appcore ./modbridge ./downloader
cd core && go test -race ./modbridge ./downloader
cd core && go test ./...
cd core && go build ./... && go vet ./...
wails generate module
cd frontend && npm run lint && npm run build
go test ./...
go build ./... && go vet ./...
```

## Review Gates

- Search for every `VersionModIDs`, backfill, precise-status, incompatible-analysis, and optional-dependency action entry point; verify simple mode cannot reach remote parsing or disabled behavior through an alternate path.
- Prove a cached `ModIDs` field and a persisted DB cache both remain ignored in simple mode.
- Prove a parser owner rejected after waiting releases its ready channel and does not leave a stuck cache entry.
- Prove an already-entered parser may finish and normal mode still consumes its result after the mode is disabled.
- Prove enabling the mode clears reminder snapshots and stale reminder IDs cannot install optional dependencies.
- Confirm local parsing and local replacement decisions remain active after an actual file is obtained.
- Confirm generated Wails files, core contracts, app adapter contracts, and Pinia field names agree exactly.

## Risky Files And Rollback Points

- `core/modbridge/modbridge.go`: concurrency/cache ownership; keep the mode checks adjacent to cache entry creation and parser invocation.
- `core/downloader/download.go`: queue-global reminder state and dependency preflight; avoid changing normal-mode ordering.
- `core/appcore/service.go` and settings contracts: startup/save propagation must use the persisted canonical value.
- `frontend/wailsjs/`: generated output only; regenerate rather than hand-maintaining bindings.
- The core submodule and parent repository must both be reviewed and committed with the updated submodule pointer.
