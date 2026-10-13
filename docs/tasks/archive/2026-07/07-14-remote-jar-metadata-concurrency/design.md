# Design

## Boundary

`modbridge` owns remote JAR mod-ID resolution, so it will own a process-wide,
dynamically configurable concurrency gate. `appcore.Service` will supply the
normalized `Downloads.ConcurrentDownloadsValue()` during startup and network
settings updates. This preserves the existing dependency direction and keeps
`modbridge` independent of `downloader` and `configs`.

## Data Flow

1. `Service.Startup` normalizes `downloads.concurrent_downloads` and configures
   both the file download queue and the remote mod-ID concurrency gate.
2. `Service.SaveNetworkSettings` persists normalized settings and reconfigures
   both consumers immediately.
3. `cachedRemoteModIDs` performs its existing cache and same-key ownership
   checks first.
4. Only the goroutine responsible for an uncached URL/loader parse acquires the
   global gate, calls `parseRemoteModJarForIDs`, and releases the slot.
5. A drained search backfill batch submits each distinct version concurrently;
   the shared gate bounds actual remote parses, and the completion event fires
   only after all submitted backfills finish.
6. Duplicate callers continue waiting on the existing per-key ready channel
   and therefore do not consume concurrency capacity.

## Concurrency Contract

Use a mutex/condition-variable gate with `active` and `limit` counters. A
condition variable supports runtime increases and decreases without replacing
a semaphore channel. Increasing the limit wakes waiters immediately. Lowering
the limit leaves active parses untouched; waiters remain blocked until
`active < limit` after releases.

The gate defaults to one, matching `DefaultConcurrentDownloads`, for direct
core consumers that do not initialize an `appcore.Service`.

## Timeout

Change `remoteModJarTimeout` from 30 seconds to 45 seconds. The existing
`NewHTTPRangeReaderAtWithTimeout` deadline covers URL resolution, Range size
probing, ZIP directory reads, and metadata reads as one overall per-JAR budget.

## Compatibility And Risk

- No configuration schema or frontend binding changes are required.
- Provider API rate limiting and file chunk concurrency remain independent.
- The main risk is leaking a gate slot on an error path; acquisition will
  return a deferred release function placed immediately before parsing.
- Tests that change the process-wide gate must restore the default limit.

## Rollback

Remove the gate configuration calls and acquisition wrapper, then restore the
timeout constant to 30 seconds. No stored data migration is involved.
