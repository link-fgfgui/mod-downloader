# Download Queue Controls Design

## Architecture

The existing queue remains owned by `core/downloader`. The app shell only renders the current projection and calls backend actions by item id.

Data flow:

`core/downloader` queue state -> `appcore.EventDownloadQueueUpdated` -> `app.go` Wails event `download-queue-updated` -> `frontend/src/stores/downloadQueue.ts` -> `frontend/src/App.vue` queue panel.

Actions flow:

`App.vue` -> Pinia store -> Wails binding -> `app.go` -> `appcore.Service` -> `core/downloader`.

## Backend Contract

Extend `DownloadQueueItem` with action flags and error context:

- `retryable bool`: true for failed or canceled history items.
- `reason string`: populated for failed and canceled items when available.

Add a backend retry method:

- `downloader.RetryDownload(ctx context.Context, id string, events ...Events) bool`
- `appcore.Service.RetryDownload(id string) bool`
- `App.RetryDownload(id string) bool`

Retry uses a backend-retained `downloadJob` snapshot. It must not ask the frontend to provide a reconstructed `ModDownloadRequest`.

## Queue State Model

The queue keeps:

- current running job
- pending jobs
- bounded retryable history for failed and canceled jobs

Statuses:

- `running`: active current job, cancelable
- `pending`: waiting job, cancelable
- `failed`: completed with an error, retryable
- `canceled`: canceled by user, retryable

Successful jobs are removed from the visible queue state. Failed/canceled history is memory-only and bounded to avoid unbounded growth.

`DownloadQueueState.Active` should remain true while active work exists or retryable items exist, so the floating button remains available for retry. `Pending` and `Running` remain active-work counters only.

## Cancellation Semantics

Cancel pending:

- remove it from pending
- append a `canceled` retryable history item
- emit queue state

Cancel running:

- call the job cancel func
- mark that job as user-canceled so `runDownloadQueue` can append `canceled` history after `downloadModJob` returns `context.Canceled`
- emit queue state immediately and again after the running job clears

Canceled downloads should not emit `download-failed`.

## Retry Semantics

Retry failed/canceled:

- find the retryable history item by id
- remove that history entry
- enqueue a copy of its saved job
- assign a fresh id through the existing enqueue path
- emit queue state

If the id does not exist or is not retryable, return false. Retrying should preserve the original resolved version, target directory, instance id, mod loader, Minecraft version, project metadata, and CurseForge API key.

## Frontend Design

`frontend/src/stores/downloadQueue.ts` becomes the owner of queue actions:

- import `CancelDownload`, `RetryDownload`, and `GetDownloadQueueState`
- expose `activeCount`, `hasVisibleItems`, `cancel(id)`, and `retry(id)`
- keep event-driven refresh/update behavior

`frontend/src/App.vue` replaces the inert floating icon with a toggleable panel anchored near the button:

- show count badge for active work
- list visible items compactly
- use icon buttons for cancel/retry with tooltips
- close the panel when no visible items remain
- avoid nested cards; use a single surface/menu-style panel

Localization keys live under `download.queue`.

## Compatibility

Wails bindings must be regenerated or manually synchronized if `wails generate module` is unavailable. Adding fields to `DownloadQueueItem` is backward-compatible for current frontend usage. Adding `RetryDownload` is a new Wails method and requires generated JS/TS bindings.

## Rollback

The change is localized. Rollback removes the retry API, the retryable history state, queue store actions, and the expanded panel, restoring the previous floating button.
