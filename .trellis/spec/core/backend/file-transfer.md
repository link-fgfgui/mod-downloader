# File Transfer

## Scenario: Downloading Mod Files

### 1. Scope / Trigger

Use this contract for HTTP file downloads started by core/downloader. The
implementation is owned by downloader/filetransfer and uses the Go standard
library only; do not add grab, gdl, req, or another download library.

### 2. Signatures

    type Backend interface {
        Kind() BackendKind
        Download(context.Context, Request, *ProgressTracker) (Result, error)
    }

    type Request struct {
        URL, Destination string
        Headers           map[string]string
        TempDir           string
        Concurrency       int
        AdaptiveConcurrency bool
        TargetBytesPerSecond int64
        ChunkSize         int64
        MemoryLimit          int64
        OverwriteExisting    bool
    }

    func (p *ProgressTracker) Snapshot() Progress

### 3. Contracts

- Probe with Range: bytes=0-0 before downloading.
- A valid 206 Content-Range enables concurrent range chunks.
- For non-adaptive ranged downloads, divide the known file size into
  `Request.Concurrency` balanced segments; segment sizes differ by at most one
  byte and files smaller than the requested count do not produce empty ranges.
- Adaptive ranged downloads reserve a larger balanced segment pool, start at
  `Request.Concurrency` workers, and may add workers while queued segments
  remain. `Request.ChunkSize` is retained for API compatibility but does not
  determine range boundaries.
- Files up to 15 MiB keep completed chunks in memory and merge in order.
- Larger files store resumable chunks under
  temp/mod-downloader/md5(layout-version|url|destination|size|segment-count),
  then merge atomically. The layout identity must change when boundaries can
  change so stale parts are never accepted merely because their sizes match.
- A server without range support streams directly to destination.part.
- Adaptive range mode observes completed bytes once per second and adds a
  worker when throughput is below `TargetBytesPerSecond`; it leaves direct
  downloads unchanged.
- Every request receives caller-provided headers and
  Accept-Encoding: identity unless explicitly overridden.
- Progress is atomically queryable and may also invoke a callback.
- The queue polls progress for stall detection and cancels the backend before
  returning errDownloadStalled.

### 4. Validation & Error Matrix

- Empty URL or destination -> validation error before network I/O.
- Existing destination with overwrite disabled -> error.
- Probe error -> direct streaming fallback.
- Non-2xx direct response -> status error.
- Range response other than 206 or wrong byte count -> error; completed temp
  parts remain available for retry.
- Non-positive size or segment count -> no range layout.
- Context cancellation -> stop active requests and return the context error.
- SHA1 verification mismatch -> handled by downloader after a successful
  transfer; the backend returns the downloaded path and does not install it.

### 5. Good/Base/Bad Cases

- Good: a 40 MiB range-capable file is divided into balanced ranges, downloads
  concurrently to layout-scoped MD5 temp parts, resumes partial parts, merges,
  syncs, and renames.
- Base: a small range-capable file uses memory chunks; a non-range server uses
  one direct stream.
- Bad: download directly to the final path or accept a 200 response for a
  requested chunk; either can leave a corrupt file.

### 6. Tests Required

- Range-capable small file: bytes, ordering, headers, progress callback.
- Large file: MD5 temp path, partial chunk resume, cleanup after merge.
- Range arithmetic: uneven division, file smaller than segment count, one
  segment, invalid inputs, and complete non-overlapping byte coverage.
- Layout identity: changing file size or segment count changes the temp path.
- Non-range file: probe plus direct request and final progress.
- Queue integration: versioned UA, selected filename, cancellation, and stall.
- Downloader integration: configured SHA1 verification retries a mismatched
  completed file and removes it after the retry limit.
- Run go test -race ./downloader/... plus core/app full test, build, and vet.

### 7. Wrong vs Correct

Wrong: create a third-party client and depend on opaque progress state.

    resp := grab.NewClient().Do(req)

Correct: use the project-owned transfer contract and queryable progress.

    tracker := filetransfer.NewProgressTracker(onProgress)
    result, err := filetransfer.NewStdlibBackend(nil).Download(ctx, req, tracker)

Wrong: reuse `md5(url|destination)` temp parts after changing range boundaries.

Correct: include the layout version, remote size, and segment count in the temp
identity so only byte-compatible parts can resume.
