# Network Runtime Configuration

## Scenario: Configuring Download And API Concurrency

### 1. Scope / Trigger

Use this contract when adding or changing download workers, per-file range
concurrency, provider HTTP transports, or their configuration. These settings
are loaded by `configs`, applied by `appcore.Service.Startup`, and consumed by
`downloader` and the CurseForge/Modrinth clients.

### 2. Signatures

```go
type configs.DownloadConfig struct {
    FileConcurrency          int
    ConcurrentDownloads      int
    AdaptiveFileConcurrency  bool
    TargetDownloadRateMiB    float64
    VerifySHA1               bool
}

type configs.APIConfig struct { RequestsPerSecond int }

type downloader.Config struct {
    FileConcurrency         int
    ConcurrentDownloads     int
    AdaptiveFileConcurrency bool
    TargetDownloadRateMiB   float64
    VerifySHA1              bool
}

func downloader.Configure(downloader.Config)
```

Configuration keys:

```toml
[downloads]
file_concurrency = 4
concurrent_downloads = 1
adaptive_file_concurrency = false
target_download_rate_mib = 1.0
verify_sha1 = false

[api]
requests_per_second = 0
```

Environment equivalents are `DOWNLOADS_FILE_CONCURRENCY`,
`DOWNLOADS_CONCURRENT_DOWNLOADS`, `DOWNLOADS_VERIFY_SHA1`, and
`API_REQUESTS_PER_SECOND`.

### 3. Contracts

- Non-positive download values normalize to 4 range chunks per file and 1
  simultaneous file.
- A negative API rate normalizes to 0; 0 means unlimited.
- File concurrency is constrained to 1-32, concurrent downloads to 1-16, and
  provider requests per second to 0-100. Values above a maximum clamp to it;
  download values below the minimum use their compatibility defaults.
- `Service.Startup` applies download settings before accepting queue work.
- `Service.SaveNetworkSettings` persists normalized values and immediately
  reconfigures new downloader jobs and the shared provider limiter. Existing
  in-flight downloads are not canceled or resized.
- Every `filetransfer.Request` receives the configured file concurrency.
- When adaptive file concurrency is enabled, a transfer starts with the
  configured file concurrency and adds one range worker per second while its
  measured throughput is below the normalized 0.1-5 MiB/s target. The number
  of workers is bounded only by the number of file ranges.
- The queue exposes every active job in `DownloadQueueState.Items`, and
  `Running` equals the active job count.
- A parent mod requeued after dependency discovery stays blocked while any of
  its required dependency version keys remain pending or running.
- CurseForge and Modrinth share one standard-library request-rate limiter, so
  `requests_per_second` is their combined start rate. Waiting honors request
  context cancellation.
- File downloads are not API-rate-limited; their bandwidth and concurrency are
  controlled by the download settings.
- When `verify_sha1` is enabled and provider metadata contains a SHA1, the
  downloader verifies the completed temporary file before installation. A
  mismatch removes that file and retries the transfer up to two times; after
  the third mismatch the job fails and no bad file is installed. Missing SHA1
  metadata skips verification.

### 4. Validation & Error Matrix

- Missing config sections -> defaults (4, 1, unlimited API).
- Download value <= 0 -> its default; do not create a zero-worker queue.
- Download value above its maximum -> clamp to 32 or 16 respectively.
- API value below 0 -> 0/no wait; above 100 -> clamp to 100.
- Adaptive target values at or below zero use 1 MiB/s; values below 0.1 or
  above 5 clamp to those limits.
- Request context canceled while rate-limited -> return `ctx.Err()` without
  invoking the base transport.
- Cancel one running download -> cancel only that job and keep other workers
  active.
- SHA1 mismatch -> retry the same download, then return a verification error
  without leaving the mismatched temporary file after retries are exhausted.

### 5. Good/Base/Bad Cases

- Good: `file_concurrency=8`, `concurrent_downloads=3`, and
  `requests_per_second=10` starts up to three files, each with eight range
  workers, while provider API starts share a ten-per-second limiter.
- Base: an existing config without these sections preserves the old four-way
  file transfer and serial queue behavior.
- Bad: use one semaphore for API calls and file chunks; a large file can then
  starve metadata requests.
- Bad: dequeue a dependency-blocked parent into a spare worker before its
  required dependency jobs leave pending/running state.

### 6. Tests Required

- Decode TOML and prefixed environment variables; assert default normalization.
- Save below/above-range settings through appcore; assert the returned view and
  reloaded TOML contain normalized values.
- Configure two concurrent downloads with a blocking backend; assert both are
  running and each request carries the configured file concurrency.
- Use a deliberately slow range server and assert adaptive mode starts an
  additional range worker when the requested target speed is missed.
- Assert a rate-limited request canceled during its wait never reaches the base
  transport.
- Preserve cancellation, retry, stall, dependency ordering, and queue snapshot
  regression tests.
- Enable SHA1 verification with a deliberately incorrect then correct backend;
  assert the retry count, final bytes, and cleanup after exhausted retries.
- Run `go test -race ./downloader/...`, then core and app test/vet/build checks.

### 7. Wrong vs Correct

Wrong:

```go
request := filetransfer.Request{URL: url, Destination: path} // ignores config
client := &http.Client{}                                    // bypasses API limit
```

Correct:

```go
downloader.Configure(downloader.Config{
    FileConcurrency: cfg.Downloads.FileConcurrencyValue(),
    ConcurrentDownloads: cfg.Downloads.ConcurrentDownloadsValue(),
})
request.Concurrency = configuredFileConcurrency()
client.Transport = rateLimitedTransport{limiter: sharedLimiter}
```
