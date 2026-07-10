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
    FileConcurrency     int
    ConcurrentDownloads int
}

type configs.APIConfig struct { RequestsPerSecond int }

func downloader.Configure(downloader.Config)
```

Configuration keys:

```toml
[downloads]
file_concurrency = 4
concurrent_downloads = 1

[api]
requests_per_second = 0
```

Environment equivalents are `DOWNLOADS_FILE_CONCURRENCY`,
`DOWNLOADS_CONCURRENT_DOWNLOADS`, and `API_REQUESTS_PER_SECOND`.

### 3. Contracts

- Non-positive download values normalize to 4 range chunks per file and 1
  simultaneous file.
- A negative API rate normalizes to 0; 0 means unlimited.
- `Service.Startup` applies download settings before accepting queue work.
- Every `filetransfer.Request` receives the configured file concurrency.
- The queue exposes every active job in `DownloadQueueState.Items`, and
  `Running` equals the active job count.
- A parent mod requeued after dependency discovery stays blocked while any of
  its required dependency version keys remain pending or running.
- CurseForge and Modrinth share one standard-library request-rate limiter, so
  `requests_per_second` is their combined start rate. Waiting honors request
  context cancellation.
- File downloads are not API-rate-limited; their bandwidth and concurrency are
  controlled by the download settings.

### 4. Validation & Error Matrix

- Missing config sections -> defaults (4, 1, unlimited API).
- Download value <= 0 -> its default; do not create a zero-worker queue.
- API value <= 0 -> no wait.
- Request context canceled while rate-limited -> return `ctx.Err()` without
  invoking the base transport.
- Cancel one running download -> cancel only that job and keep other workers
  active.

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
- Configure two concurrent downloads with a blocking backend; assert both are
  running and each request carries the configured file concurrency.
- Assert a rate-limited request canceled during its wait never reaches the base
  transport.
- Preserve cancellation, retry, stall, dependency ordering, and queue snapshot
  regression tests.
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
