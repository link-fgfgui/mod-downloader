# Make dynamic dependency analysis cancellable

## Goal

Users can cancel an install while the app is still analyzing or refreshing required dependencies, before the actual JAR download has started.

## Background

- Download cancellation already exists for pending and running queue jobs through `CancelDownload`; running jobs cancel their `context.CancelFunc` and are kept as retryable canceled queue entries (`core/downloader/download.go:130`, `core/downloader/download.go:297`).
- Required dependency discovery is performed before the main job is enqueued: `queueModDownload` resolves the version, calls `hydrateRequiredDependencies`, queues missing required dependencies, then enqueues the requested mod (`core/downloader/download.go:75`, `core/downloader/download.go:490`).
- `hydrateRequiredDependencies` can call provider APIs through `providers.RefreshMatchingProjectVersions` when the selected version does not already include required dependency metadata (`core/downloader/download.go:517`, `core/providers/service.go:215`).
- The frontend install button stays in a local loading state until `QueueModDownload` returns (`frontend/src/stores/downloadSearch.ts:201`), but no download queue item exists during pre-enqueue dependency analysis, so the existing queue cancel action cannot affect that phase.

## Requirements

- Add a user-visible, cancelable queue state for pre-download dependency analysis so the user can cancel after starting an install even if the job has not reached file download yet.
- Canceling during dependency analysis must stop the install from enqueueing dependencies or the main download after cancellation is observed.
- Canceled analysis attempts must appear as retryable canceled items, consistent with current canceled pending/running downloads.
- Existing pending/running download cancellation and retry behavior must keep working.
- Keep dependency analysis optional to cancel, not mandatory to skip: if the user does not cancel, required dependency discovery and queuing should behave as it does today.

## Acceptance Criteria

- [x] Starting an install creates a queue item promptly, with status indicating analysis/running and `cancelable: true`, before provider dependency refresh work can block the flow.
- [x] Calling `CancelDownload` for that queue item during dependency analysis returns `true`, cancels the active attempt, and the canceled item is retryable.
- [x] After cancellation, no dependency jobs and no main mod job are enqueued by that canceled attempt.
- [x] Retrying the canceled analysis item starts a fresh install attempt with a fresh queue ID and normal dependency analysis/download behavior.
- [x] Existing tests for canceling pending downloads, canceling running downloads, retrying, and stalled-download retry continue to pass.
- [x] New backend tests cover cancellation during the analysis/pre-download phase.

## Verification

- `cd core && go test ./...`
- `cd core && go build ./... && go vet ./...`
- `go test ./...`

## Out Of Scope

- Adding a global setting to disable dependency discovery.
- Reworking provider APIs to make every network request internally context-aware; this task only requires the install attempt to stop once cancellation is observed at downloader boundaries.
- Changing how required dependencies are selected or filtered.
