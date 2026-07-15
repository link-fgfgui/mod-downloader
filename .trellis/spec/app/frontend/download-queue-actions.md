# Download Queue Actions

## Scenario: Remove A Canceled Download Record

### 1. Scope / Trigger

Use this contract when changing download queue cancel, retry, or history-removal actions across Vue, Wails, appcore, and `core/downloader`.

### 2. Signatures

```go
func (a *App) RemoveCanceledDownload(id string) bool
func (s *Service) RemoveCanceledDownload(id string) bool
func downloader.RemoveCanceledDownload(id string, events ...downloader.Events) bool
```

```ts
RemoveCanceledDownload(id: string): Promise<boolean>
downloadQueueStore.removeCanceled(id: string): Promise<boolean>
```

### 3. Contracts

- The queue ID is the existing `DownloadQueueItem.id`; no alternate identity is introduced.
- Only an item in downloader's `retryable` history with exact status `canceled` may be removed.
- Removal changes in-memory queue history only. It never deletes destination, partial-download, or cache files.
- The retry button keeps left-click retry behavior. Its context-menu action removes only a canceled item, prevents the native menu for that item, and executes without confirmation.
- Successful removal emits one queue-state update and returns `true`. The store refreshes after success, matching cancel and retry actions.

### 4. Validation & Error Matrix

| Input/state | Result | Queue event |
| --- | --- | --- |
| Matching canceled retryable ID | Remove item, return `true` | Once |
| Empty/whitespace ID | Return `false` | None |
| Missing ID | Return `false` | None |
| Failed retryable ID | Return `false` | None |
| Pending or running ID | Return `false` | None |

### 5. Good/Base/Bad Cases

- Good: right-click the retry icon for a canceled row; only that row disappears and no job is enqueued.
- Base: left-click the same icon; existing retry behavior creates a fresh queue ID and enqueues the job.
- Bad: remove any `retryable` item without checking status; this silently deletes failed-download recovery history.

### 6. Tests Required

- Downloader success test asserts only the matching canceled item is removed, stale progress is cleared, and the emitted snapshot retains other items.
- Downloader rejection tests cover empty, missing, failed, pending, and running IDs and assert no event is emitted.
- Regenerate Wails bindings after changing the public `App` method, then run frontend lint/type-check/build and app/core tests.

### 7. Wrong vs Correct

Wrong:

```go
if item.Job.ID == id {
    downloadQueue.retryable = append(downloadQueue.retryable[:i], downloadQueue.retryable[i+1:]...)
}
```

Correct:

```go
if item.Job.ID == id && item.Status == "canceled" {
    downloadQueue.retryable = append(downloadQueue.retryable[:i], downloadQueue.retryable[i+1:]...)
}
```
