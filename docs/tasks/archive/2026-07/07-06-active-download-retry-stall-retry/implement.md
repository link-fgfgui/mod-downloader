# Active Download Retry And Stall Retry Implementation Plan

## Checklist

- [x] Add queue restart-request state for active retry.
- [x] Extend `RetryDownload` to handle current running job IDs.
- [x] Mark running `DownloadQueueItem` as retryable.
- [x] Add stall detection around `grab.Response.BytesComplete()`.
- [x] Add bounded automatic stall retry in `runDownloadQueue`.
- [x] Add focused downloader tests for running retry and stall retry decision logic.
- [x] Update queue contract spec with running retry and stall retry behavior.
- [x] Run validation.

## Validation Commands

- `(cd core && go test ./...)`
- `(cd core && go build ./... && go vet ./...)`
- `go test ./...`
- `go build ./...`
- `(cd frontend && npm run build)`
- `(cd frontend && npm run lint)`

## Validation Results

- `(cd core && go test ./...)` passed.
- `(cd core && go build ./... && go vet ./...)` passed.
- `go test ./...` passed.
- `go build ./...` passed.
- `(cd frontend && npm run build)` passed.
- `(cd frontend && npm run lint)` passed.

## Risk Points

- Do not auto-retry user cancellations.
- Do not create duplicate history rows when running retry cancels an active attempt.
- Keep partial download behavior delegated to `grab`; do not delete temp files unless existing code already does so.
- Stall detection must cancel only the current transfer attempt context, not the parent queue context.
