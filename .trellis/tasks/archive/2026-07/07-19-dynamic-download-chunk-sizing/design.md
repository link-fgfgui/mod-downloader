# Technical Design

## Ownership

The change is confined to `core/downloader/filetransfer`. `StdlibBackend.Download` already probes the remote size and owns ranged transfer selection; `downloader/download.go` continues to pass configured concurrency and adaptive settings without a public API change.

## Range Calculation

Replace fixed-size range generation with a helper that accepts `(size, segmentCount)` and computes `ceil(size / segmentCount)` as the chunk width. It emits contiguous `[start,end]` ranges in index order, clamps the count to at most the number of bytes, and returns no ranges for invalid/empty inputs. The final range ends at `size-1`.

The caller derives the segment pool from `Request.Concurrency`: non-adaptive transfers use that count; adaptive transfers use a bounded multiple of the initial count so the existing ramp-up behavior has queued work. The pool is still capped by file size and does not create empty ranges.

## Resume Compatibility

The temp directory identity includes a layout version, remote size, and segment count in addition to URL and destination. Parts from the old fixed-size layout therefore cannot be mistaken for valid dynamic-layout parts, while retries with the same dynamic layout retain partial resume. Existing per-part expected-size validation remains in place.

## Error And Progress Behavior

No changes to HTTP probing, status validation, cancellation, progress accounting, atomic merge, or direct fallback. Range workers continue to report bytes through `trackingWriter`; adaptive mode continues sampling once per second.

## Test Shape

Unit tests cover range arithmetic independently. Existing backend tests are updated only where they assumed 4 MiB-derived part counts, and must retain ordering, resume, direct fallback, cancellation, and progress coverage.
