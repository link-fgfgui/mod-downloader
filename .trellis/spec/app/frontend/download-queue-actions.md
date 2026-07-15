# Download Queue Actions

## Scenario: Remove A Download Queue Item

### 1. Scope / Trigger

Use this contract when changing download queue cancel, retry, or history-removal actions across Vue, Wails, appcore, and `core/downloader`.

### 2. Signatures

```go
func (a *App) RemoveDownload(id string) bool
func (s *Service) RemoveDownload(id string) bool
func downloader.RemoveDownload(id string, events ...downloader.Events) bool
```

```ts
RemoveDownload(id: string): Promise<boolean>
downloadQueueStore.remove(id: string): Promise<boolean>
```

### 3. Contracts

- The queue ID is the existing `DownloadQueueItem.id`; no alternate identity is introduced.
- Removable items are `retryable` entries with exact status `failed` or `canceled`, plus jobs still in `pending`.
- A pending removal deletes the job directly and must not append a canceled retryable record. If a worker has already moved the job to `current`, removal returns `false` and must not invoke its cancel function.
- Removal changes in-memory queue history only. It never deletes destination, partial-download, or cache files.
- The retry button keeps left-click retry behavior and removes failed/canceled items on context-menu. The cancel button keeps left-click cancellation behavior and removes pending items on context-menu.
- Context-menu removal executes without confirmation and prevents the native menu only for removable states. Running-item context menus remain unaffected.
- Successful removal emits one queue-state update and returns `true`. The store refreshes after success, matching cancel and retry actions.

### 4. Validation & Error Matrix

| Input/state | Result | Queue event |
| --- | --- | --- |
| Matching failed/canceled retryable ID | Remove item, return `true` | Once |
| Matching pending ID | Remove directly without canceled history, return `true` | Once |
| Empty/whitespace ID | Return `false` | None |
| Missing ID | Return `false` | None |
| Unsupported retryable status | Return `false` | None |
| Running ID, including a former pending job | Return `false`; do not cancel | None |

### 5. Good/Base/Bad Cases

- Good: right-click the retry icon for a failed/canceled row, or the cancel icon for a pending row; only that row disappears and no job is enqueued.
- Base: left-click the same icon; existing retry behavior creates a fresh queue ID and enqueues the job.
- Base: left-click a pending row's cancel icon; the job moves to canceled retryable history.
- Bad: implement pending removal through `CancelDownload`; the supposedly deleted row immediately returns as canceled history.
- Bad: trust the UI snapshot and cancel a job that has already moved from pending to running.

### 6. Tests Required

- Downloader success tests cover failed, canceled, and pending items; assert only the matching item is removed, stale progress is cleared, and exactly one snapshot retains other items.
- The pending test asserts no canceled retryable history is created.
- Downloader rejection tests cover empty, missing, unsupported retryable status, and running IDs and assert no event is emitted.
- Regenerate Wails bindings after changing the public `App` method, then run frontend lint/type-check/build and app/core tests.

### 7. Wrong vs Correct

Wrong:

```go
CancelDownload(ctx, id, events...)
```

Correct:

```go
if item.Job.ID == id && (item.Status == "failed" || item.Status == "canceled") {
    downloadQueue.retryable = append(downloadQueue.retryable[:i], downloadQueue.retryable[i+1:]...)
}
// Pending jobs are removed directly under the queue lock; current jobs are untouched.
```
