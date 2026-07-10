# Outbound User-Agent

## Scenario: Versioned Provider And Download Requests

### 1. Scope / Trigger

Use this contract when creating provider clients or HTTP requests that fetch
installable mod files through `appcore` and `downloader`.

### 2. Signatures

```go
type appcore.Options struct { Version string }
func (s *appcore.Service) UserAgent() string
func downloader.QueueModDownloadWithUserAgent(ctx context.Context, req structs.ModDownloadRequest, apiKey, userAgent string, events ...downloader.Events) structs.ModDownloadResult
func downloader.InstallOptionalDependenciesWithUserAgent(ctx context.Context, id, apiKey, userAgent string, events ...downloader.Events) []structs.ModDownloadResult
```

### 3. Contracts

- The normalized format is `mod-downloader/<version>`; an empty version uses
  `mod-downloader/dev`.
- The Wails adapter passes its embedded build version through
  `appcore.Options.Version`.
- Modrinth API, CurseForge API, direct file downloads, required dependencies,
  optional dependencies, and discard fetches use the same UA.
- Legacy downloader entry points remain available and use the `dev` UA.
- CurseForge requests also preserve `Accept: application/json` and
  `x-api-key`.

### 4. Validation & Error Matrix

- Empty/whitespace version -> `mod-downloader/dev`.
- Missing CurseForge API key -> CurseForge client remains disabled.
- CurseForge non-200 response -> close the body and return its status error.
- Empty custom downloader UA -> compatibility fallback, not an empty header.

### 5. Good/Base/Bad Cases

- Good: a `v1.2.3` build sends `User-Agent: mod-downloader/v1.2.3` for API and
  file requests.
- Base: tests and local callers that omit a version send
  `mod-downloader/dev`.
- Bad: set a literal `mod-downloader` separately in each transport; versioning
  then drifts and dependency downloads are easily missed.

### 6. Tests Required

- Assert version normalization and the configured Modrinth client UA.
- Assert CurseForge transport sets UA, Accept, and API key without mutating the
  caller request.
- Assert standard-library file-transfer requests and discard fetches send the
  versioned UA.
- Run core and consuming-app tests, build, and vet.

### 7. Wrong vs Correct

```go
// Wrong: unversioned and local to one request path.
req.Header.Set("User-Agent", "mod-downloader")

// Correct: pass the service identity through every download job.
req.Header.Set("User-Agent", job.UserAgent)
```
