# Build Version

## Scenario: Injecting The Application Version

### 1. Scope / Trigger

Use this contract for release builds, version display, diagnostics, and any
outbound request that identifies the application version.

### 2. Signatures

```go
var appVersion string
func currentAppVersion() string
func (a *App) GetAppVersion() string
```

Build command:

```bash
export APP_VERSION=v1.2.3
wails build -ldflags "-X main.appVersion=${APP_VERSION}"
```

Production builds run from `.github/workflows/build.yml`. The workflow derives
`APP_VERSION` from the tag or commit and uses one `-ldflags` argument for
both `main.appVersion` and the default CurseForge key stored in the
`DEFAULT_CF_API_KEY` GitHub Actions secret.

### 3. Contracts

- `APP_VERSION` is a build input; the binary receives it through Go `-X`.
- An empty or whitespace-only injected value normalizes to `dev`.
- CI uses a tag name for tag builds and a short commit SHA otherwise, unless
  `APP_VERSION` is explicitly set.
- CI reads the default CurseForge key from `secrets.DEFAULT_CF_API_KEY`; the
  workflow contains only the secret reference, not the key value.
- `GetAppVersion` returns the normalized embedded value and has generated Wails
  bindings.

### 4. Validation & Error Matrix

- No ldflags -> `dev`.
- `-X main.appVersion=` -> `dev`.
- Whitespace around an injected version -> trimmed value.
- Public Wails method changed without regenerated bindings -> frontend contract
  is stale and the change is incomplete.

### 5. Good/Base/Bad Cases

- Good: release tag `v1.2.3` produces `GetAppVersion() == "v1.2.3"`.
- Base: local `wails dev` reports `dev`.
- Bad: read `APP_VERSION` with `os.Getenv` at runtime; release binaries then
  lose their identity when launched outside the build shell.

### 6. Tests Required

- Unit test normalization of an injected value and the empty fallback.
- Build once with `go build -ldflags '-X main.appVersion=<value>'`.
- Run `wails generate module`, frontend build, app tests, build, and vet after
  changing the exposed version API.

### 7. Wrong vs Correct

```go
// Wrong: runtime environment, not an embedded build identity.
version := os.Getenv("APP_VERSION")

// Correct: overwritten by the linker at build time.
var appVersion = "dev"
version := currentAppVersion()
```
