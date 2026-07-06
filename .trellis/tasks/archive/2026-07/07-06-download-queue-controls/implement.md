# Download Queue Controls Implementation Plan

## Checklist

- [x] Update core structs with retry/error fields on `DownloadQueueItem`.
- [x] Extend `core/downloader` with bounded retryable history for failed/canceled jobs.
- [x] Add `RetryDownload` in downloader, appcore service, and Wails app adapter.
- [x] Add focused downloader tests for cancel history and retry behavior.
- [x] Regenerate or synchronize Wails frontend bindings for new fields/API.
- [x] Update `downloadQueue` Pinia store with cancel/retry actions and derived counts.
- [x] Replace the floating button in `App.vue` with a toggleable queue panel.
- [x] Add zh/en i18n keys for queue status and actions.
- [x] Run backend and frontend validation.

## Validation Commands

- `go test ./...`
- `(cd core && go test ./...)`
- `(cd frontend && npm run build)`

If Go API signatures are changed, run `wails generate module` when available. If Wails generation is unavailable in this environment, manually keep `frontend/wailsjs/go/main/App.{js,d.ts}` and `frontend/wailsjs/go/models.ts` synchronized and report that generation could not be run.

## Validation Results

- `wails generate module` passed after `frontend/dist` existed.
- `go test ./...` passed from repo root.
- `(cd core && go test ./...)` passed.
- `(cd core && go build ./... && go vet ./...)` passed.
- `go build ./...` passed from repo root.
- `(cd frontend && npm run build)` passed.
- `(cd frontend && npm run lint)` passed.

## Risk Points

- Running cancellation is asynchronous: the queue may emit once when cancellation is requested and again when the job exits.
- Retry must copy backend job data before clearing history; otherwise target/version context can be lost.
- Queue state must not keep completed successful jobs, or the floating button becomes noisy.
- Frontend panel should remain compact and must not obscure main content more than necessary.

## Review Gate

Before implementation, confirm:

- `prd.md`, `design.md`, and `implement.md` exist.
- The active task is still `planning`.
- The user already requested implementation in the original task.
