# Provider API Source

## Scenario: Switching Between Official APIs And MCIM

### 1. Scope / Trigger

Use this contract when changing provider client construction, persisted source
preferences. MCIM is an optional mirror for Modrinth and CurseForge metadata
APIs only; it is never enabled implicitly and never handles mod files.

### 2. Signatures

```go
type configs.Preferences struct { MCIMEnabled bool }
type appcore.SaveMCIMSettingsRequest struct { MCIMEnabled bool }
func (s *appcore.Service) SaveMCIMSettings(req SaveMCIMSettingsRequest) SettingsView
func (a *App) SaveMCIMSettings(req SaveMCIMSettingsRequest) SettingsView
```

Persistence and environment contracts:

```toml
[preferences]
mcim_enabled = false
```

The environment equivalent is `PREFERS_MCIM_ENABLED`.

### 3. Contracts

- The default is the official Modrinth and CurseForge sources.
- Enabling MCIM uses `https://mod.mcimirror.top/modrinth/v2/` for Modrinth
  and rewrites CurseForge API requests under
  `https://mod.mcimirror.top/curseforge/`.
- MCIM CurseForge requests do not require an API key. The official CurseForge
  client remains disabled when its key is empty.
- The source switch preserves the versioned User-Agent and the shared API rate
  limiter.
- File downloads always use the exact provider-supplied URL so Modrinth and
  CurseForge can attribute downloads and creator revenue correctly. Downloader
  config has no MCIM/file-source flag and must not rewrite CDN hosts.
- Saving the setting persists it and immediately reconfigures provider API
  clients; an application restart is not required.
- The Wails settings view exposes `mcimEnabled`; the settings page owns only
  draft/loading UI state.

### 4. Validation & Error Matrix

- Missing preference -> official sources.
- MCIM enabled with empty CurseForge key -> construct the mirror client.
- MCIM disabled with empty CurseForge key -> set the CurseForge client to nil.
- Any provider download URL -> pass it unchanged to file transfer validation.
- Context canceled during API rate-limit wait -> do not invoke either source.

### 5. Good/Base/Bad Cases

- Good: the user enables MCIM, saves once, and the next metadata search uses
  MCIM while the selected file still downloads from the official CDN.
- Base: an existing config has no `mcim_enabled` field and continues using the
  official providers.
- Bad: rewrite official CDN URLs to MCIM; platform download counts and creator
  revenue attribution can be lost.
- Bad: require a CurseForge API key while MCIM is active; the mirror endpoint
  works without it.

### 6. Tests Required

- Config tests for default false, TOML true, and `PREFERS_MCIM_ENABLED`.
- CurseForge transport test for URL/query rewriting, no key, caller request
  immutability, and versioned UA.
- Provider construction test for MCIM on/off and Modrinth `BaseURL`.
- Downloader tests assert file-transfer requests retain provider URLs.
- Service persistence test that toggles on and off without restart.
- Regenerate Wails bindings, then run frontend lint/build and core/app
  test/vet/build checks.

### 7. Wrong vs Correct

Wrong:

```go
request.URL.Host = "mod.mcimirror.top" // rewrites official file CDN
```

Correct:

```go
if useMCIM {
    modrinthClient.BaseURL = mcimModrinthURL
    curseForgeTransport.apiBaseURL = mcimCurseForgeBaseURL
}
request.URL = version.DownloadURL // official provider URL, unchanged
```
