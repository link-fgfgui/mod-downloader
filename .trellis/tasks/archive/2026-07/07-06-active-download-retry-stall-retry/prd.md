# Active download retry and stall retry

## Goal

Improve resilience for complex network conditions where an active download can stall with zero effective progress. Users can retry a currently running download directly from the queue panel, and the backend can automatically retry a stalled download before surfacing it as failed.

User request: "考虑到下载中的文件可能因为复杂网络环境速度归零，把这两个功能都实现一下，用trellis task".

## Confirmed Facts

- The previous task added `RetryDownload(id)` for failed/canceled retry history.
- Current running queue items are `cancelable` but not retryable in the frontend panel.
- `RetryDownload(id)` currently only searches failed/canceled retry history and returns false for running IDs.
- Actual network downloads use `grab.NewClient()`, `grab.NewRequest(...)`, `client.Do(req)`, and `resp.Err()`.
- `grab.Response` exposes `BytesComplete()`, `IsComplete()`, `Cancel()`, and `Err()`, which can support stall detection.
- Download jobs already retain backend-owned resolved version, target directory, instance ID, project metadata, mod loader, Minecraft version, and CurseForge API key.

## Requirements

- Running download items must expose a retry action in the queue panel.
- Clicking retry on a running item must cancel the current transfer attempt and requeue the same backend job with a fresh queue ID.
- Running retry must not require the frontend to reconstruct a `ModDownloadRequest`.
- Running retry should not create a user-visible canceled history row; it is a restart action.
- Backend downloads must detect stalled transfers where downloaded byte count stops changing.
- A stalled transfer must be retried automatically up to a bounded number of attempts.
- Automatic stall retries must not emit `download-failed` or create failed history until retry attempts are exhausted.
- When automatic retry attempts are exhausted, the final item must become failed and manually retryable through the existing failed history path.
- Existing user cancel behavior must remain distinct from retry/stall cancellation.
- The implementation must keep Wails runtime code in the app adapter and queue/download logic in `core/downloader`.

## Defaults

- Stall threshold: 30 seconds without byte progress after the transfer begins.
- Automatic stall retry count: 2 additional attempts after the initial attempt.

## Acceptance Criteria

- [x] Running queue rows show a retry button.
- [x] Retrying a running row cancels the active attempt, assigns a fresh queue ID, and requeues without showing an intermediate canceled item.
- [x] Failed/canceled manual retry behavior from the previous task still works.
- [x] A transfer whose byte count does not advance for the stall threshold is retried automatically.
- [x] Automatic stall retries stop after the bounded retry count and then produce a failed, retryable queue item.
- [x] User cancel still produces a canceled, retryable queue item and does not auto-retry.
- [x] Relevant downloader unit tests cover running retry and stall retry decision behavior.
- [x] Go tests and frontend build/lint pass.

## Notes

- This task is a cross-layer queue action change plus core downloader stall handling.
