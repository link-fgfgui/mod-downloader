# Network Runtime Configuration

## Scenario: Configuring Download And API Concurrency

### 1. Scope / Trigger

Use this contract when adding or changing download workers, per-file range
concurrency, remote JAR metadata parsing, provider HTTP transports, or their
configuration. These settings are loaded by `configs`, applied by
`appcore.Service.Startup`, and consumed by `downloader`, `modbridge`, and the
CurseForge/Modrinth clients.

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
func modbridge.ConfigureRemoteModIDConcurrency(limit int)
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
  reconfigures new downloader jobs, remote JAR metadata parsing, and the shared
  provider limiter. Existing in-flight work is not canceled or resized.
- `concurrent_downloads` is the shared numeric limit for simultaneous file
  downloads and distinct uncached remote JAR mod-ID parses. These consumers
  use separate gates, so one workload does not consume the other's slots.
- Remote JAR parsing acquires its gate only after memory/DB lookup and
  URL/loader cache ownership are resolved. Cache hits and callers coalesced
  behind an existing parse do not consume slots.
- Lowering remote JAR concurrency lets active parses finish and blocks new
  parses until the active count falls below the new limit. Increasing it wakes
  waiters immediately.
- Each remote JAR metadata parse has one 45-second overall deadline covering
  URL resolution, Range size probing, ZIP directory reads, and metadata reads.
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
- Missing or invalid concurrent downloads -> both file downloads and remote
  JAR metadata parsing default to one concurrent task.
- Remote JAR limit reduced below the active count -> active parses continue;
  queued parses wait until enough active work completes.
- Remote JAR parsing exceeds 45 seconds -> the shared per-JAR deadline cancels
  its HTTP work and the gate slot is released.
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
  workers, allows up to three distinct remote JAR metadata parses through a
  separate gate, and gives provider API starts a ten-per-second limiter.
- Base: an existing config without these sections preserves the old four-way
  file transfer, serial file queue, and serial remote JAR parsing behavior.
- Bad: use one semaphore for API calls and file chunks; a large file can then
  starve metadata requests.
- Bad: acquire remote JAR capacity before checking the URL/loader cache;
  duplicate callers can occupy every slot while waiting for one physical JAR.
- Bad: dequeue a dependency-blocked parent into a spare worker before its
  required dependency jobs leave pending/running state.

### 6. Tests Required

- Decode TOML and prefixed environment variables; assert default normalization.
- Save below/above-range settings through appcore; assert the returned view and
  reloaded TOML contain normalized values.
- Configure two concurrent downloads with a blocking backend; assert both are
  running and each request carries the configured file concurrency.
- Configure two remote mod-ID parses with blocking fakes; assert two distinct
  URLs enter, a third waits, and same-URL callers still invoke one parse.
- Reduce remote JAR concurrency while parses are active; assert in-flight work
  continues and a waiter starts only after the active count is below the new
  limit.
- Assert startup and saved network settings pass the normalized
  `concurrent_downloads` value to the remote mod-ID gate.
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
modbridge.ConfigureRemoteModIDConcurrency(cfg.Downloads.ConcurrentDownloadsValue())
request.Concurrency = configuredFileConcurrency()
client.Transport = rateLimitedTransport{limiter: sharedLimiter}
```

## Scenario: Simple Mode Remote Mod ID Gate

### 1. Scope / Trigger

Use this contract when a persisted user preference must disable remote JAR
mod-ID resolution while leaving ordinary file downloads and local JAR parsing
available.

### 2. Signatures

```go
// configs.Preferences
SimpleMode bool `toml:"simple_mode" json:"simple_mode" env:"SIMPLE_MODE"`

func modbridge.ConfigureSimpleMode(enabled bool)
func modbridge.SimpleModeEnabled() bool
func downloader.ConfigureSimpleMode(enabled bool, events ...downloader.Events)
```

### 3. Contracts

- `simple_mode` defaults to `false`; the settings view uses `simpleMode` and
  the environment equivalent is `PREFERS_SIMPLE_MODE` through the preferences
  prefix.
- `appcore.Service.Startup` applies the persisted value before accepting queue
  work; saving the setting applies it immediately and returns the canonical
  settings view.
- While enabled, `VersionModIDs` returns no IDs without reading memory/SQLite
  caches, and search state returns ordinary `new` download buttons without
  queuing backfills.
- The remote parser checks the mode before creating cache ownership and again
  after acquiring the concurrency gate. A waiter rejected after the second
  check closes its cache-entry channel and releases the gate.
- A parser already inside the HTTP function may finish; enabling the mode does
  not cancel its context. New work and queued waiters must not start HTTP I/O.
- Enabling the mode clears pending backfills and downloader optional reminders.
  Required/optional/incompatible dependency preflight and stale optional
  reminder actions are disabled.
- Local parsing after a downloaded or hardlinked file remains enabled and may
  persist primary IDs for later normal-mode use.

### 4. Validation & Error Matrix

- Missing `simple_mode` -> `false`; no config migration is required.
- Simple mode with an in-memory or persisted Mod ID cache -> return no IDs and
  do not consult the cache.
- Simple mode with a queued remote parse -> return the mode-disabled result,
  close waiters, and make zero parser calls.
- Simple mode with an active remote parse -> allow that call to finish; all
  subsequent calls remain disabled until the mode is turned off.
- Simple mode optional reminder action -> return no install results and leave no
  reminder snapshot available to the queue.
- Normal mode -> retain cache reads, remote fallback, backfills, precise status,
  and dependency behavior.

### 5. Good/Base/Bad Cases

- Good: check the mode immediately before and after remote gate acquisition,
  then close an aborted cache entry before returning.
- Base: a normal-mode download with no remote IDs downloads to a temporary path,
  parses locally, and installs normally.
- Bad: only disable the Settings button; a queued backfill or direct core caller
  can still start a remote Range request.
- Bad: keep cached IDs authoritative in simple mode; two users with different
  cache history then see different reduced-mode behavior.
- Bad: clear reminder UI without gating its stored install action.

### 6. Tests Required

- TOML/environment decode and appcore save/reload round trip for `simple_mode`.
- Startup and runtime-save tests proving the core flag changes immediately.
- Cached in-memory IDs, persisted IDs, and backfill queue are ignored in simple
  mode.
- A blocking parser test proving one in-flight call may finish while a queued
  caller is released without starting a second parse.
- Search-state test asserting an enabled ordinary download button in simple mode.
- Dependency/reminder tests asserting no required queueing, optional reminders,
  incompatible analysis, or stale optional action.
- Local-parse download test and normal-mode cache restoration test.
- Run `go test -race ./modbridge ./downloader` plus full core/app checks and
  frontend lint/build after binding changes.

### 7. Wrong vs Correct

Wrong:

```go
if simpleMode {
    return defaultButtonState(result) // UI only; queued backfills still run
}
```

Correct:

```go
for {
    if modbridge.SimpleModeEnabled() {
        return nil, errSimpleMode
    }
    entry := ownRemoteCacheEntry(key)
    release := remoteModIDGate.acquire()
    if modbridge.SimpleModeEnabled() {
        release()
        closeAndDelete(entry)
        return nil, errSimpleMode
    }
    return parseRemoteJar(entry, release)
}
```
