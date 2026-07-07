# Design

## Boundary

The change belongs primarily in `core/downloader`, with app/frontend behavior reusing the existing download queue API. The Wails adapter and Pinia queue store should not need a new cancellation endpoint if the downloader exposes analysis as a normal queue item.

## Queue Model

Today a job is inserted into `pending` only after dependency hydration and dependency queueing finish. Move the requested install into the queue earlier, then let the queue runner perform pre-download work as part of the active job:

1. Normalize and resolve the requested install target.
2. Resolve the requested version enough to create a queue job.
3. Enqueue the main job immediately.
4. In the runner, hydrate required dependencies and enqueue missing required dependencies before downloading the main job.
5. Check `ctx.Err()` after each analysis boundary and before enqueueing dependency or main download work.

This gives `CancelDownload` a current queue item to cancel while analysis is happening.

## Data Flow

- `QueueModDownload` should return `Queued: true` once the install attempt is accepted into the queue.
- `runDownloadQueue` remains the owner of the job context and cancel function.
- `downloadModJob` should include dependency hydration and dependency queueing before file download, or delegate to a helper called from the runner with the same job context.
- Recursive dependency queueing must preserve the existing `visited` map behavior to avoid cycles and duplicate jobs.

## Cancellation Semantics

- User cancellation keeps using `CancelDownload(id)`.
- If cancellation is requested during dependency analysis, the job context becomes canceled.
- The runner treats `context.Canceled` the same way as current running download cancellation: consume the cancel request and append a retryable canceled item.
- Helpers that may do work after a provider/database call must check `ctx.Err()` before enqueueing any follow-up jobs.

## Compatibility

- Queue item JSON shape stays compatible.
- Existing statuses can remain `running`/`pending` unless implementation finds a low-risk need for an `analyzing` status. The acceptance criteria require cancelability, not a new status contract.
- Existing frontend queue UI should work if the item is exposed through `DownloadQueueState`.

## Risks

- Moving dependency queueing into the runner changes when dependencies are enqueued relative to the main job. Preserve current order by placing dependencies ahead of the main file download for that active job.
- Provider refresh functions do not accept context today, so cancellation may not interrupt an in-flight provider call immediately. The required behavior is to prevent post-cancel enqueue/download side effects once control returns.
