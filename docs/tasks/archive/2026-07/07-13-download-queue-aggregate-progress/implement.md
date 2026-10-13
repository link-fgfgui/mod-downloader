# Download Queue Aggregate Progress Implementation Plan

## Checklist

- [x] Add `FileSize` to `models.ModVersion`, map CurseForge and Modrinth SDK file sizes, and extend focused provider/model tests for positive and absent sizes.
- [x] Add aggregate byte/speed fields to `DownloadQueueState` and byte progress fields to `DownloadQueueItem`.
- [x] Add the mutex-owned queue progress registry and helpers for cycle start, enqueue, sample update, successful retention, attempt replacement/removal, and active-drain cleanup.
- [x] Extend the existing stall ticker with a bounded progress callback and final sample; propagate the queue item ID/event sink through the download call path without moving Wails concerns into `core`.
- [x] Project per-item progress, byte-weighted aggregate progress, and summed active speed from `currentDownloadQueueState`.
- [x] Add downloader tests for known/unknown totals, byte weighting, concurrent speed aggregation, completed contribution retention, active-drain reset, cancel/failure removal, retry replacement, short-download final sampling, and no deadlock under event callbacks.
- [x] Preserve and rerun existing dependency-analysis, running restart, stall retry, SHA1 retry, cancellation, concurrent worker, and queue-state tests.
- [x] Regenerate Wails bindings and confirm `frontend/wailsjs/go/models.ts` contains the new queue/model fields.
- [x] Extend the Pinia queue snapshot and `App.vue` leave snapshot, add aggregate and per-item progress UI, compact byte/speed formatters, responsive styling, and zh/en localization.
- [x] Verify the queue panel in known-size, unknown-size, concurrent, retryable-only, narrow viewport, and leave-transition states through the static responsive layout, leave-snapshot path, type-check, and production build; native Wails interaction was not required by project guidance.
- [x] Run the full validation matrix and record any unavailable interactive checks.

## Validation Results

- `go test -race ./downloader/...` passed.
- `(cd core && go test ./...)` passed.
- `(cd core && go build ./... && go vet ./...)` passed.
- `go test ./...`, `go build ./...`, and `go vet ./...` passed from the app repo.
- `wails generate module` passed and generated queue/model fields are present in `frontend/wailsjs/go/models.ts`.
- `(cd frontend && npm run lint)` and `(cd frontend && npm run build)` passed.
- `gofmt -l` for all changed Go files and `git diff --check` returned no output.
- Linux native `wails dev` was not started because the app backend spec reserves it for tasks requiring native-app verification; no interactive screenshot check was run.

## Validation Commands

```bash
(cd core && gofmt -w models/models.go providers/modprovider.go structs/search.go downloader/download.go '<changed test files>')
(cd core && go test -race ./downloader/...)
(cd core && go test ./...)
(cd core && go build ./... && go vet ./...)
wails generate module
(cd frontend && npm run lint)
(cd frontend && npm run build)
go test ./...
go build ./... && go vet ./...
git diff --check
git status --short
```

Use the actual changed Go test paths in the `gofmt` command. Build `frontend/dist` first if Wails generation reports the existing embed prerequisite described by the app backend spec.

## Focused Test Matrix

| Case | Expected result |
| --- | --- |
| Two known tasks of different sizes | Aggregate is `(sum complete)/(sum total)`, not average percentages |
| Pending known-size task | Item and aggregate include zero completed bytes and known total |
| Unknown-size task beside known task | Unknown item is indeterminate and excluded from aggregate bytes |
| Transfer discovers a total | Item becomes determinate and aggregate denominator expands |
| One concurrent task completes | Its full bytes remain while another active task continues |
| Final active task completes | Final sample reaches 100%, then active aggregate state resets |
| Failed/canceled attempt | Contribution is removed; retryable-only surface hides active metrics |
| Explicit/automatic retry | Old attempt is not double-counted; new attempt gets a fresh baseline |
| Multiple active transfers | Queue speed equals the latest non-negative per-job samples summed |
| No byte movement / completion | Speed returns to zero and stall behavior remains unchanged |
| Queue leave animation | Last visible metric text/progress remains stable until after leave |

## Risky Files And Rollback Points

- `core/downloader/download.go`: highest concurrency risk. Keep progress helpers small, keep callbacks outside locks, and run race tests before frontend work is considered complete.
- `core/models/models.go` and provider converters: shared cached contract. Verify old snapshots decode with `FileSize == 0` and both providers populate new data.
- `core/structs/search.go` plus `frontend/wailsjs/go/models.ts`: cross-layer payload must change together; regenerate rather than hand-edit when possible.
- `frontend/src/App.vue`: preserve `visibleQueueSnapshot`, compact action-column dimensions, and all animation modes.
- `core/` submodule pointer: do not leave the parent repository referencing an uncommitted or mismatched core state during final commit/rollback.

## Review Gate

- [ ] User has reviewed `prd.md`, `design.md`, and `implement.md` and approved implementation.
- [ ] Task is still in `planning` before review and is activated with `task.py start` only after approval.
- [ ] Phase 2 loads `trellis-before-dev` before editing production code.
- [ ] Inline Codex mode is used; JSONL context curation and implementation/check sub-agent dispatch are skipped.
