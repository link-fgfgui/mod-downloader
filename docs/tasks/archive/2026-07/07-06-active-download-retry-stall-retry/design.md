# Active Download Retry And Stall Retry Design

## Architecture

The queue remains owned by `core/downloader`. The frontend calls the existing `RetryDownload(id)` action for both running jobs and retryable history jobs. The backend decides whether the ID refers to the current running job or to failed/canceled history.

## Running Retry

Extend downloader queue state with a `restartRequested` ID set.

Flow:

1. `RetryDownload(id)` checks retryable history first, preserving existing behavior.
2. If no history row matches and the current job ID matches, mark `restartRequested[id]`.
3. Call the current job cancel function.
4. `runDownloadQueue` observes `context.Canceled`.
5. If the job ID was restart-requested, clear the marker and enqueue a copy of the same job with a fresh ID.
6. Do not append canceled history and do not emit failed events.

Running queue items should set `Retryable: true` so the frontend can render the retry action. Pending items remain cancel-only.

## Stall Auto-Retry

Add bounded retry state to `downloadJob`:

- `AutoAttempt int`: current automatic attempt number.

Downloader constants:

- `downloadStallTimeout = 30 * time.Second`
- `maxDownloadStallRetries = 2`
- `downloadProgressPollInterval = 1 * time.Second`

Network download helper:

- Run `grab` with a child context derived from the job context.
- Poll `resp.BytesComplete()` until completion.
- Each time bytes increase, reset the last-progress timestamp.
- If bytes do not increase for `downloadStallTimeout`, cancel only the child attempt context and return a sentinel stall error.

Queue loop:

- If `downloadModJob` returns the sentinel stall error and `AutoAttempt < maxDownloadStallRetries`, requeue the same job with `AutoAttempt + 1`.
- Do not create failed history and do not emit `download-failed` during automatic retries.
- If attempts are exhausted, handle the stall error like any other failure.

## Error And Cancellation Separation

Use separate markers/errors for:

- User cancel: `cancelRequested`
- User running retry: `restartRequested`
- Automatic stall retry: sentinel error returned by the download helper

This prevents a user cancel from being auto-retried and prevents a running retry from appearing as canceled.

## Frontend

The existing queue panel already renders retry buttons for `item.retryable`. The backend will mark running items `retryable: true`, so no new Wails method is needed. The tooltip text remains "Retry download".

## Compatibility

`DownloadQueueItem.Retryable` already exists in generated bindings. Wails bindings should not need regeneration unless Go exported signatures change. Frontend build still validates the queue item contract.

## Rollback

Rollback removes running-item `Retryable`, `restartRequested`, stall constants/helper logic, and auto-retry branches in `runDownloadQueue`.
