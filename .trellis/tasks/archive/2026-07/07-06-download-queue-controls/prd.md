# Download queue controls

## Goal

Clicking the floating download button opens a live download queue panel. Users can inspect individual downloads and act on them without leaving the current page: cancel running or pending downloads, and retry downloads that failed or were canceled.

This request was made as a planning-and-implementation task: "点击下载按钮展开队列,可以取消或重试某个下载".

## Confirmed Facts

- `app.go` already exposes `GetDownloadQueueState()` and `CancelDownload(id string)` through Wails.
- `core/appcore.Service` forwards queue state and cancellation to `core/downloader`.
- `core/downloader` already tracks one running job plus pending jobs, emits `download-queue-updated`, and emits `download-failed` for failed running jobs.
- `DownloadQueueState.Items` currently contains active queue items only. `DownloadQueueItem` includes `id`, `status`, `title`, `fileName`, `versionId`, `platform`, `minecraftVersion`, `modLoader`, and `cancelable`.
- `frontend/src/stores/downloadQueue.ts` already listens to `download-queue-updated`, but only stores the queue snapshot.
- `frontend/src/App.vue` already renders a floating download button while the queue is active. The button is not currently clickable and has no expanded queue UI.
- Generated Wails bindings currently include `CancelDownload`, `GetDownloadQueueState`, and `QueueModDownload`, but no retry API.
- The frontend must not reconstruct retry requests from partial queue display fields; the backend owns version resolution, target resolution, dependency queuing, and API-key behavior.

## Requirements

- The floating download button must toggle an expanded queue panel when clicked.
- The panel must show individual queue items with useful status, title/file name, version/platform context, and per-item actions.
- Running and pending queue items must expose cancel actions when `cancelable` is true.
- Failed and canceled queue items must remain visible long enough to retry from the panel.
- Retrying a failed or canceled item must re-enqueue the original backend job data through a backend-owned retry path.
- Queue state updates must continue to flow through the existing `download-queue-updated` event.
- The queue summary badge must continue to count active work only: running plus pending.
- Empty inactive queue state should hide or close the panel unless retryable failed/canceled items are still present.
- User-facing labels must be localized in the existing zh/en i18n structure.
- The implementation must preserve the app/core layering: Wails-specific runtime code stays in `app.go`; reusable queue logic stays in `core/downloader` and `core/appcore`.

## Out of Scope

- Persisting queue history across app restarts.
- Showing byte-level download progress or speed.
- Parallel downloads.
- Bulk cancel or bulk retry controls.
- Long-term successful download history.

## Acceptance Criteria

- [x] Clicking the floating download button opens and closes a queue panel.
- [x] Running and pending items are visible in the panel and can be canceled individually.
- [x] A canceled running or pending item becomes a retryable item instead of disappearing immediately.
- [x] A failed item becomes retryable and can be requeued from the panel.
- [x] Retrying an item assigns it a fresh queue id, moves it back into active queue state, and removes the stale failed/canceled history row.
- [x] Queue events update the panel without a manual refresh.
- [x] Search-result button loading state still reflects queued/running project keys.
- [x] Frontend TypeScript builds with regenerated or synchronized Wails bindings.
- [x] Relevant Go tests cover queue state, cancel history, and retry behavior.

## Notes

- This is a cross-layer change touching core downloader state, appcore/Wails bindings, Pinia store, app shell UI, and generated frontend types.
