# Dynamic Download Chunk Sizing

## Goal

Make ranged downloads derive chunk boundaries from the remote file size and the requested segment count, so `file_concurrency` produces a predictable number of balanced segments instead of operating over a fixed 4 MiB chunk size.

## Background

- `filetransfer.DefaultChunkSize` is currently fixed at 4 MiB in `core/downloader/filetransfer/types.go`.
- `StdlibBackend.Download` learns the total size from a `Range: bytes=0-0` probe before calling `rangesForSize` in `core/downloader/filetransfer/stdlib.go`.
- `Request.Concurrency` currently controls active workers. The fixed chunk size independently determines the number of queued ranges.
- Large ranged downloads persist one file per range and resume those parts by index, so changing range boundaries changes the compatibility of incomplete temporary parts.
- Adaptive mode starts with `Request.Concurrency` workers and can only add workers while unclaimed ranges remain.

## Requirements

- For a known-size, range-capable download, calculate balanced byte ranges from the total size and intended segment count.
- Cover every byte exactly once, preserve range ordering, and handle files smaller than the requested segment count without empty ranges.
- Preserve direct streaming fallback when size is unknown or range requests are unsupported.
- Preserve cancellation, progress tracking, atomic merge, and completed-download behavior.
- Do not add a third-party transfer dependency.
- Define adaptive-concurrency behavior explicitly before implementation.

## Compatibility

- Existing incomplete ranged temporary parts may have boundaries based on the previous fixed 4 MiB chunk size. They must not be mistaken for valid parts under a new layout.
- Existing public Go API and configuration keys should remain stable unless the final design requires a documented semantic change.

## Acceptance Criteria

- Tests prove that a known file size is divided into balanced, contiguous, non-overlapping ranges based on the requested segment count.
- Tests cover uneven division, a file smaller than the requested count, one segment, and invalid inputs.
- Existing small-memory, large-temp, resume, direct-download, cancellation, and progress tests continue to pass or are updated to preserve their contracts.
- `go test -race ./downloader/...` passes from `core/`.
- `go test ./...`, `go build ./...`, and `go vet ./...` pass from `core/`.

## Out Of Scope

- Replacing the standard-library HTTP backend.
- Benchmarking against or reproducing NDM internals.
- Changing queue-level concurrent file downloads.

## Resolved Design Decision

- Dynamic sizing uses a segment pool larger than the initial worker count when adaptive mode is enabled. Workers start at `Request.Concurrency` and may ramp up over the precomputed balanced ranges. Non-adaptive mode uses the requested concurrency as the segment count.
