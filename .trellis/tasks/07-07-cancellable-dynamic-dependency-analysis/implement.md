# Implementation Plan

1. Load backend guidelines with `trellis-before-dev` before editing.
2. Refactor `core/downloader.QueueModDownload` so the main install job is enqueued before dynamic dependency hydration.
3. Move dependency hydration and missing-required-dependency queueing into the active job execution path with shared context cancellation checks.
4. Preserve dependency ordering by enqueueing required dependency jobs before the main file download continues.
5. Add focused downloader tests for cancellation during dependency analysis/pre-download.
6. Run `go test ./...` in `core/`.
7. Run app-level `go test ./...` if exported signatures or app adapter behavior changes.

## Validation

- `cd core && go test ./...`
- `go test ./...`

## Rollback Points

- `core/downloader/download.go` contains the queue lifecycle changes.
- `core/downloader/download_test.go` contains the behavioral regression tests.
