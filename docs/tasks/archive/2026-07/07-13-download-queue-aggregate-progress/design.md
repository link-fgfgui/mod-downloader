# Download Queue Aggregate Progress Design

## Architecture

`core/downloader` remains the single owner of queue progress. The transfer layer supplies atomic byte snapshots, the downloader samples and aggregates them, and the existing queue event carries the resulting projection to Vue.

```text
provider file size / transfer ProgressTracker
                    |
                    v
core/downloader progress registry
                    |
                    v
structs.DownloadQueueState -- appcore event --> Wails event
                    |
                    v
Pinia downloadQueue store --> App.vue queue panel
```

No frontend timer, polling loop, or independent speed calculation is added. The existing `download-queue-updated` event remains the only live update path.

## Shared Contracts

Extend the canonical version model with provider metadata:

```go
type ModVersion struct {
    // existing fields
    FileSize int64 `json:"fileSize"`
}
```

`curseForgeProvider.fileToModVersion` maps `File.FileLength`. `modrinthProvider.setModVersionFileFields` maps `File.Size` when present. A missing or non-positive value means unknown; existing cached JSON remains compatible and can gain an authoritative size when the transfer starts.

Extend queue projections:

```go
type DownloadQueueState struct {
    // existing fields
    BytesComplete  int64 `json:"bytesComplete"`
    TotalBytes     int64 `json:"totalBytes"`
    BytesPerSecond int64 `json:"bytesPerSecond"`
}

type DownloadQueueItem struct {
    // existing fields
    BytesComplete int64 `json:"bytesComplete"`
    TotalBytes    int64 `json:"totalBytes"`
}
```

All values are non-negative in public projections. `TotalBytes == 0` means unknown. Pending and running items receive progress fields; retryable failed/canceled items keep zero values and are not rendered as active transfers.

## Downloader State

Add a queue-owned, mutex-protected progress registry keyed by queue item ID. Each entry tracks current bytes, authoritative total bytes, latest bytes/second, and whether the contribution is completed. The registry contains active jobs and successful contributions retained for the current queue cycle.

Lifecycle:

1. When the first job enters an idle queue, start a new progress cycle and clear stale progress entries.
2. On enqueue, create an entry with `BytesComplete=0` and `TotalBytes=ModVersion.FileSize` when positive.
3. When transfer sampling obtains a positive total, replace provider size with the transfer total; an unknown transfer total never overwrites a known provider size.
4. On successful completion, clamp completed bytes to total for a known-size job, set speed to zero, emit the final snapshot, and retain the entry while other pending/running jobs remain.
5. On failure, user cancellation, dependency-analysis requeue, explicit restart, or bounded automatic retry, remove or replace the old attempt's contribution so abandoned bytes are not double-counted.
6. After the last pending/running job leaves, clear cycle entries. Retryable history may keep the queue surface visible, but aggregate progress and speed remain hidden because active count is zero.

Required dependencies discovered during preflight are added through the same enqueue path. Their newly known total legitimately expands the denominator. A retry receives a fresh queue ID and a fresh attempt contribution.

## Sampling And Events

Reuse the ticker already owned by `doDownloadWithStallTimeout`. On each tick:

- read one `ProgressTracker.Snapshot()`;
- compute non-negative `delta bytes / elapsed seconds` for that job;
- update the queue progress entry under the queue mutex;
- release the mutex before emitting `download-queue-updated`;
- continue using the same byte snapshot for stall detection.

Emit a final sample when the backend returns so short downloads still reach 100% even if they finish before the first tick. A tracker reset caused by SHA1 retry starts a fresh attempt sample and may reset that job's contribution. Event frequency remains bounded to approximately one event per active file per second plus lifecycle events; transfer chunk callbacks do not emit Wails events.

`currentDownloadQueueState` calculates:

- per-item bytes from the item's registry entry;
- aggregate completed and total bytes by summing entries with `TotalBytes > 0`;
- aggregate speed by summing latest speeds only for currently running IDs.

Unknown-size entries are excluded from both aggregate byte sums. Arithmetic clamps completed bytes into `[0, total]` and guards against negative deltas, invalid totals, resets, and overflow before projecting a percentage in Vue.

## Frontend Design

Update the Pinia snapshot type and `App.vue` leave-time clone to include all three aggregate fields. The generated Wails model remains the runtime contract.

Within the existing queue header, show an unframed aggregate row while `pending + running > 0`:

- a compact total progress bar;
- transferred/total bytes and percentage when `totalBytes > 0`;
- an indeterminate bar when no active task has a known total;
- aggregate speed formatted as `B/s`, `KiB/s`, `MiB/s`, or `GiB/s`.

For each pending/running row, add a stable-height progress region below metadata. Known totals use a determinate linear progress bar and a compact percentage; unknown totals use an indeterminate bar without a fabricated percentage. Failed/canceled rows retain reason and action layout without an inactive progress bar.

Use Vuetify progress components and existing theme tokens. Keep the 420px panel and mobile width constraint, use no nested cards, and ensure the action column remains stable. Add localized labels under the existing `download.queue` zh/en keys.

## Compatibility

- Added JSON fields are backward-compatible for current consumers; absent fields resolve to zero.
- Existing stored version snapshots without `fileSize` remain readable.
- Public Wails method signatures and event names do not change, but generated TypeScript models must be regenerated because shared structs change.
- CLI and other `core` consumers receive additional fields without taking a dependency on Wails.
- Queue control, completion-sound, retry history, dependency ordering, and leave-snapshot ownership remain unchanged.

## Risks And Mitigations

- **Deadlock or event reentrancy:** never invoke callbacks or emit events while holding `downloadQueue`'s mutex.
- **Progress regression from lifecycle churn:** retain successful contributions until active drain; explicitly replace abandoned/retried attempts.
- **Incorrect speed after reset:** clamp negative byte deltas to zero and reset sampling baselines per transfer attempt.
- **Double counting concurrent retries:** register and remove contributions atomically around queue ID replacement.
- **Short downloads missing progress:** emit a final tracker snapshot before removing the running item.
- **Unknown or stale provider size:** treat transfer totals as authoritative and exclude still-unknown entries from aggregate sums.
- **UI leave flicker:** copy new aggregate fields into `visibleQueueSnapshot` and clear only through the existing after-leave hook.

## Rollback

Rollback removes the new model and queue fields, provider size mappings, downloader progress registry/sampling callback, generated model fields, and Vue progress presentation. No database migration or persisted queue state needs reversal. Because `core/` is a submodule, rollback must keep the core commit and parent submodule pointer synchronized.
