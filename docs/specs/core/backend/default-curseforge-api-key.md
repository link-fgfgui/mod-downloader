# Default CurseForge API Key

## Scenario: Compile-Time Default CurseForge API Key

### 1. Scope / Trigger

Use this contract when injecting a release-time CurseForge API key, changing
how settings report whether a key is present, or deciding which key value is
sent on outbound CurseForge metadata / CDN requests.

### 2. Signatures

```go
// core/configs/defaults.go
var DefaultCurseforgeAPIKey string
func EffectiveCurseforgeAPIKey(configured string) string

// core/appcore/service.go
func (s *Service) effectiveCurseforgeAPIKey() string
```

Production build injection is owned by `.github/workflows/build.yml`. It reads
`secrets.DEFAULT_CF_API_KEY` and passes it to the linker together with
`APP_VERSION`. A local equivalent for smoke testing is:

```bash
export DEFAULT_CF_API_KEY='...'
export APP_VERSION=v1.2.3
wails build -ldflags "-X main.appVersion=${APP_VERSION} -X github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey=${DEFAULT_CF_API_KEY}"
```

### 3. Contracts

- Source default of `configs.DefaultCurseforgeAPIKey` is empty. Non-empty values
  come only from linker `-X` or tests.
- Priority (high → low): user-configured `Keys.CurseforgeApiKey` (TOML /
  `KEYS_CF_API_KEY` / UI save / `ConfigOverrides`) → compile-time default → empty.
- Outbound CurseForge usage (provider client, download queue, optional deps)
  must call `EffectiveCurseforgeAPIKey` / `effectiveCurseforgeAPIKey`, never the
  bare config field alone.
- Official CurseForge client enablement uses the **effective** key: empty
  effective key and MCIM off → `SetCurseForgeClient(nil)`.
- `GetSettings` `hasCurseforgeKey` / `curseforgeKeyMask` reflect the **effective**
  key so a binary with only a compile-time default reports the key as set.
- UI clear sets the configured field to empty and **must not** write the
  compile-time default into `mod-downloader.toml`. After clear, effective key
  falls back to the compile-time default when present.
- Release builds obtain the key from the `DEFAULT_CF_API_KEY` GitHub Actions
  secret. The tracked workflow must contain only the secret reference and must
  not print the resolved value.
- Local smoke builds may export `DEFAULT_CF_API_KEY`; values containing `$`
  must be assigned with single quotes before being expanded into `-ldflags`.

### 4. Validation & Error Matrix

| Condition | Result |
|-----------|--------|
| configured non-empty | effective = trimmed configured |
| configured empty, default non-empty | effective = trimmed default |
| both empty, MCIM off | CurseForge client = nil |
| both empty, MCIM on | mirror client constructed (no key required) |
| SaveApiKeys clear (`""`) | config field empty; default not persisted |
| no ldflags | `DefaultCurseforgeAPIKey == ""` (dev behavior unchanged) |

### 5. Good/Base/Bad Cases

- Good: release build injects default; first-run user with empty TOML can use
  official CurseForge.
- Good: user saves a personal key; outbound requests and mask use that key.
- Good: user clears key; TOML stays empty; effective falls back to default.
- Base: `wails dev` without injection behaves as before (no default).
- Bad: assign `s.config.Keys.CurseforgeApiKey = DefaultCurseforgeAPIKey` on
  startup and later `Save` — leaks the default into the user's TOML.
- Bad: read `os.Getenv("DEFAULT_CF_API_KEY")` at runtime for the default —
  release binaries would lose the key outside the build shell.
- Bad: hard-code the key in the workflow or print the resolved secret in CI
  logs.

### 6. Tests Required

- `EffectiveCurseforgeAPIKey`: configured wins, empty falls back, both empty.
- Service: empty config + injected default → client non-nil, settings has key,
  config field remains empty.
- Service: user key wins over default for mask.
- Service: clear persists empty TOML and still reports has-key when default set.
- Workflow: production `wails build` injects both version and
  `secrets.DEFAULT_CF_API_KEY` through one `-ldflags` argument.
- Optional: `go build -ldflags '-X ...DefaultCurseforgeAPIKey=...'` smoke.

### 7. Wrong vs Correct

Wrong:

```go
curseForgeAPIKey := strings.TrimSpace(s.Config().Keys.CurseforgeApiKey)
// misses compile-time default
```

Correct:

```go
curseForgeAPIKey := configs.EffectiveCurseforgeAPIKey(s.Config().Keys.CurseforgeApiKey)
```

Wrong:

```go
// on clear / startup
s.config.Keys.CurseforgeApiKey = configs.DefaultCurseforgeAPIKey
_ = configs.Save(s.config) // persists secret into TOML
```

Correct:

```go
// leave configured field empty; resolve only at use sites
key := s.effectiveCurseforgeAPIKey()
```
