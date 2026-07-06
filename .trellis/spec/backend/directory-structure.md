# Directory Structure

> How backend code is organized in this project.

---

## Overview

mod-downloader is now split into three Go-facing repositories:

- `mod-downloader` — Wails v2 desktop app shell, Vue 3 frontend, and Wails adapter code.
- `mod-downloader-core` — reusable Go core module, checked out in this repo as the `core/` git submodule.
- `mod-downloader-cli` — standalone CLI repo that depends on the same core module through its own `core/` submodule.

The main app module keeps `replace github.com/link-fgfgui/mod-downloader-core => ./core` in `go.mod` for local development. CF/MR mod metadata types live in `github.com/link-fgfgui/mod-downloader-core/models` as the single source of truth.

---

## Directory Layout

```
mod-downloader/
├── main.go                      # Wails app bootstrap
├── app.go                       # Wails-exposed API methods (App struct)
├── core/                        # git submodule: github.com/link-fgfgui/mod-downloader-core
│   ├── appcore/                 # UI-independent service boundary shared by Wails and CLI
│   ├── httpserver/              # Extension HTTP bridge, adapter-neutral event callback
│   ├── models/                  # Canonical data types (single source of truth)
│   ├── structs/                 # Request/response structs + Minecraft manifest types
│   ├── providers/               # CurseForge/Modrinth platform abstraction layer
│   ├── database/                # BoltDB persistence
│   ├── downloader/              # Download queue + state machine
│   ├── modbridge/               # Cross-domain bridge
│   ├── global/                  # Global clients and in-memory indexes
│   ├── configs/                 # Config load/save
│   ├── minecraft/               # Minecraft JAR parser and launcher layouts
│   └── logging/                 # Structured logger wrapper
├── frontend/                    # Vue 3 + Pinia frontend (Wails-generated bindings in wailsjs/)
├── go.mod                       # requires mod-downloader-core and replaces it with ./core
└── .gitmodules                  # pins the local core/ submodule
```

```
mod-downloader-cli/
├── cliapp/                      # CLI command definitions and output formatting
├── cmd/mod-downloader-cli/       # CLI binary entrypoint
├── core/                        # git submodule: github.com/link-fgfgui/mod-downloader-core
└── go.mod                       # requires mod-downloader-core and replaces it with ./core
```

---

## Module Organization

### Layered data flow (mod metadata, inside `core/`)

```
[CF/MR SDK] → providers (SDK→models converters) → models (canonical types)
                    ↓                                  ↑
              database (caches models.*)         structs (request/response; consumes models)
                    ↓                                  ↑
              downloader (consumes models.*)      appcore (UI-independent service)
                    ↓
              modbridge (cross-domain: version resolution, install status, SHA1↔platform bridge)
               ↙       ↘
         global      minecraft
   (local mods +     (local JAR
    JAR mem cache)    parsing)
```

**Boundary constraint**: `minecraft` (local analysis) and `providers` (platform analysis) must NOT import each other. Their convergence point is `modbridge`. Dependency direction is unidirectional: `downloader → modbridge → {providers, database, global, minecraft}`.

### Scenario: UI-Independent Core Service And CLI Adapters

#### 1. Scope / Trigger

Use this pattern whenever a workflow must be available from both the Wails UI
and a non-UI caller such as the CLI. The trigger is any app lifecycle, config,
instance, search, pinning, download, or local-mod workflow that would otherwise
be duplicated between `app.go` and command code.

#### 2. Signatures

Core service:

```go
svc := appcore.New(appcore.Options{
    ConfigOverrides: appcore.ConfigOverrides{MinecraftDir: dir, HasMinecraftDir: true},
    OnEvent: func(event appcore.Event) { ... },
})
err := svc.Startup(ctx)
defer svc.Close()    // CLI / tests: close DB without saving transient overrides
defer svc.Shutdown() // Wails: persist selected minecraft dir and close DB

svc.SearchMods(req)
update := svc.SearchModsCollect(req)
result := svc.QueueModDownload(req)
waited := svc.InstallModAndWait(ctx, req)
versions := svc.GetVersions()
mods, err := svc.LocalMods(instanceKey)
```

Adapters:

```go
// Wails adapter keeps frontend-facing method names stable.
func (a *App) SearchMods(req structs.SearchModsRequest)
func (a *App) QueueModDownload(req structs.ModDownloadRequest) structs.ModDownloadResult

// CLI binary entrypoint lives in the sibling mod-downloader-cli repository.
go run ./cmd/mod-downloader-cli <command> [flags]
```

#### 3. Contracts

- `appcore` must not import `github.com/wailsapp/wails/v2/pkg/runtime`.
- `httpserver` must not import Wails runtime; it emits `httpserver.Event`
  through `httpserver.Options.OnEvent`, and `app.go` maps that to
  `runtime.EventsEmit`.
- `cliapp` and `cmd/mod-downloader-cli` live in the `mod-downloader-cli`
  repository and must not import Wails runtime.
- Wails event names such as `search-mods-updated`,
  `download-queue-updated`, and `selected-version-changed` belong in `app.go`
  or another Wails adapter, not in `appcore`.
- `appcore.Event.Kind` is an adapter-neutral signal. Wails maps it to runtime
  events; CLI may ignore it or render progress.
- CLI global overrides (`--minecraft-dir`, `--curseforge-api-key`,
  `--modrinth-api-key`) apply to the current command only. Use `Service.Close`
  for CLI cleanup so transient overrides are not written back on shutdown.
- Persisting config from CLI must go through explicit config commands such as
  `config --set-minecraft-dir`, `config --theme`, or API-key set/clear flags.
- Shared data types remain in `models` and existing `structs` packages. Do not
  add aliases or re-export files for CLI convenience.

#### 4. Validation & Error Matrix

- Empty CLI project without `--platform` and without `platform:project` prefix
  -> command error.
- Empty or missing CLI `--instance` for install -> command error.
- Invalid selected instance key -> service returns an error; Wails may preserve
  panic behavior only at the Wails adapter boundary for frontend compatibility.
- Failed download -> `InstallModAndWait` returns failed events; CLI exits
  non-zero with the failure reason.
- Empty Minecraft root -> version discovery returns no versions, not a panic.
- Wails startup failure while loading release versions -> Wails startup returns
  before starting the extension HTTP server, preserving prior behavior.

#### 5. Good/Base/Bad Cases

- Good: `app.go` opens a Wails directory dialog, passes the chosen path to
  `svc.SetMinecraftDir`, then maps `appcore.EventSelectedVersionChanged` to
  `runtime.EventsEmit(ctx, "selected-version-changed", payload)`.
- Good: `app.go` starts `httpserver.New(httpserver.DefaultAddr,
  httpserver.Options{OnEvent: a.emitHTTPServerEvent})` and owns the Wails event
  bridge; `core/httpserver` only reports adapter-neutral events.
- Base: in the `mod-downloader-cli` repo, `go run ./cmd/mod-downloader-cli
  --minecraft-dir /tmp/mc versions --json` lists instances and then calls
  `svc.Close`; `/tmp/mc` is not persisted.
- Bad: `appcore` imports Wails runtime to emit frontend events directly.
- Bad: `httpserver` imports Wails runtime to emit extension events directly.
- Bad: CLI creates its own Modrinth/CurseForge converters instead of using
  `providers` and `models`.
- Bad: `cliapp` duplicates version-directory parsing instead of calling
  `appcore.GetVersions` / `minecraft.LoadLauncherVersions`.

#### 6. Tests Required

- Core dependency test: from `core/`, `go list -deps ./appcore ./httpserver`
  must not include `github.com/wailsapp/wails/v2/pkg/runtime`.
- CLI dependency test: from the `mod-downloader-cli` repo, `go list -deps
  ./cliapp ./cmd/mod-downloader-cli` must not include
  `github.com/wailsapp/wails/v2/pkg/runtime`.
- CLI JSON test: `config --json` decodes as `appcore.SettingsView` and masks
  API keys.
- CLI version discovery test: `versions --json` with `--minecraft-dir` override
  returns the expected `[]structs/minecraft.VersionInfo`.
- Existing Wails adapter tests must continue to pass without regenerated
  frontend bindings when public method signatures are preserved.

#### 7. Wrong vs Correct

Wrong:

```go
// core/appcore/service.go
import "github.com/wailsapp/wails/v2/pkg/runtime"

func (s *Service) SearchMods(req structs.SearchModsRequest) {
    providers.SearchMods(req, func(update structs.SearchModsUpdate) {
        runtime.EventsEmit(s.ctx, "search-mods-updated", update)
    })
}
```

Correct:

```go
// core/appcore/service.go
func (s *Service) SearchMods(req structs.SearchModsRequest) {
    providers.SearchMods(req, func(update structs.SearchModsUpdate) {
        s.emit(EventSearchModsUpdated, update)
    })
}

// app.go
func (a *App) emitCoreEvent(event appcore.Event) {
    runtime.EventsEmit(a.ctx, searchModsUpdatedEvent, event.Payload)
}
```

### Convention: `models` is the single source of truth

**What**: `github.com/link-fgfgui/mod-downloader-core/models` defines `ModProject`, `ModVersion`, `ModDependency`, and the composite-key helpers (`ProjectKey`, `ParseProjectKey`, `VersionKey`, `ParseVersionKey`). Every other package imports `models` directly — no type aliases, no re-export files.

**Why**: Previously `structs.SearchModResult = models.ModProject` (alias) and `providers/model.go` (re-export) gave the same type three names. This made cross-file search noisy, obscured which package owned the type, and let a parallel "old" conversion path (`modToSearchResult`) coexist with a "new" path (`modToModProject`) — the old path silently dropped the `ProjectID` field, a bug that went unnoticed because the new (correct) path was dead code.

**Example**:
```go
// Good — import models directly
import "github.com/link-fgfgui/mod-downloader-core/models"

func (a *App) ListMatchingProjectVersions(result models.ModProject, mcVersion, modLoader string) []models.ModVersion

// Bad — alias or re-export (removed in 06-27-unify-models-cleanup)
type SearchModResult = models.ModProject   // forbidden: third name for same type
// providers/model.go: type ModProject = models.ModProject  // forbidden: re-export file
```

### Scenario: Online Metadata Display For Local Mods

#### 1. Scope / Trigger

Use this pattern when platform metadata from CurseForge or Modrinth should affect how a locally installed JAR is displayed. Local JAR parsing still owns install identity; platform metadata owns display enrichment.

#### 2. Signatures

Canonical project metadata:

```go
type ModProject struct {
    Title      string   `json:"title"`
    IconURL    string   `json:"iconUrl"`
    Categories []string `json:"categories,omitempty"`
}
```

Local mod display metadata:

```go
type ModInfo struct {
    ID              string   `json:"id"`      // JAR-declared identity
    Name            string   `json:"name"`    // JAR-declared fallback display
    OnlineName      string   `json:"onlineName,omitempty"`
    OnlinePlatform  string   `json:"onlinePlatform,omitempty"`
    OnlineProjectID string   `json:"onlineProjectId,omitempty"`
    OnlineSlug      string   `json:"onlineSlug,omitempty"`
    IconURL         string   `json:"iconUrl,omitempty"`
    Categories      []string `json:"categories,omitempty"`
}
```

#### 3. Contracts

- Provider converters populate `models.ModProject.Categories` from provider-native categories/tags:
  - Modrinth search: `SearchResult.Categories`
  - Modrinth project: `Project.Categories` plus `Project.AdditionalCategories`
  - CurseForge mod: `Mod.Categories`, preferring `Slug` and falling back to `Name`
- `models.NormalizeCategories` lowercases, trims, and deduplicates category strings before they cross package boundaries.
- `modbridge.ApplyProjectMetadataToModInfo` may fill `OnlineName`, `IconURL`, platform/project fields, and `Categories`.
- Display enrichment must not overwrite `ModInfo.ID`, `Name`, `Version`, `SHA1`, `Path`, `Enabled`, or `JijMods`.
- Frontend display should prefer `onlineName || name || id`; technical subtitles may keep JAR-derived IDs/version/file details.
- Every selected-version local-mod refresh path must use the same enrichment pipeline:
  scan local jars, apply cached platform metadata, asynchronously resolve missed
  SHA1s, update `global.SetSelectedVersion`, and emit
  `EventSelectedVersionChanged` after async metadata changes. `SelectVersion`
  and `RefreshSelectedVersionMods` must not diverge.
- Async metadata writeback must verify the selected instance still matches the
  refreshed instance before mutating global selected-version state.

#### 4. Validation & Error Matrix

- Empty provider categories -> `nil` / omitted `categories`, not a placeholder.
- Duplicate categories with different casing -> one lowercase category.
- SHA1 maps to only partial platform metadata without title/icon/categories -> treat as a display miss and allow hash resolution to try fuller metadata.
- No platform metadata -> keep existing JAR-derived display fallback.
- Instance changed while async hash resolution is in flight -> drop the async
  metadata update.

#### 5. Good/Base/Bad Cases

- Good: Modrinth categories `["library", "Magic"]` become `[]string{"library", "magic"}` and render as category chips.
- Base: Local-only JAR with no platform match still displays `ModInfo.Name` and fallback icon.
- Bad: Replacing `ModInfo.ID` with a platform slug; this breaks install status and conflict detection.

#### 6. Tests Required

- Provider converter tests assert unified categories are populated for Modrinth and CurseForge.
- Enrichment tests assert online display fields are applied while JAR-derived identity and JiJ fields are preserved.
- Service tests or focused regressions assert selected-version refresh paths
  share enrichment behavior and emit updated selected-version state after async
  metadata changes where practical.
- Frontend build/type check must pass after adding response fields consumed by Vue.

#### 7. Wrong vs Correct

Wrong:

```go
info.Name = project.Title
info.ID = project.Slug
```

Correct:

```go
info = modbridge.ApplyProjectMetadataToModInfo(info, project)
// UI uses info.OnlineName || info.Name || info.ID
```

Wrong:

```go
// SelectVersion scans jars but skips cached/async online metadata enrichment.
version = s.refreshVersionMods(version, mcDir)
global.SetSelectedVersion(version)
s.emit(EventSelectedVersionChanged, version)
```

Correct:

```go
// Selection and explicit refresh share one enrichment/event path.
return s.refreshAndSelectVersionMods(version, mcDir), nil
```

---

## Naming Conventions

### Files

- `snake_case.go` for multi-word files (e.g. `localmods.go`).
- `<topic>.go` + `<topic>_test.go` pairs (e.g. `models.go` / `models_test.go`).

### Types & functions

- Canonical types in `models/` use the `Mod` prefix: `ModProject`, `ModVersion`, `ModDependency`.
- SDK→struct converter functions are named `sdkTypeToCanonicalType`: `modToModProject`, `fileToModVersion`, `versionToModVersion`, `searchHitToModProject`. The canonical-type suffix matches the `models` type name exactly — do NOT name converters after aliases (e.g. `modToSearchResult` is forbidden; `SearchResult` is no longer a type name).
- Sort/filter helpers are named after the canonical type: `sortModVersions`, not `sortProjectVersionResults`.

---

## Design Decisions

### Decision: Bridge Package for Cross-Domain Convergence

**Context**: We had two completely independent data sources that needed to interact:
- **Local analysis** (`minecraft` package): Parses JAR files from disk, extracts mod metadata (modID, version, name)
- **Platform analysis** (`providers` package): Fetches mod metadata from CurseForge/Modrinth APIs

These needed to bridge via SHA1 hash matching for features like "show install status for this platform mod" or "find platform metadata for this local JAR."

**Options Considered**:
1. Let `minecraft` import `providers` and `database` for cross-domain queries
2. Let `providers` import `minecraft` for local status checks
3. Create a neutral bridge package that imports both

**Decision**: We chose Option 3 (bridge package `modbridge`) because:
- Allows both domains to evolve independently
- Prevents circular dependencies
- Single responsibility: cross-domain queries only
- Makes the convergence point explicit and auditable

**Implementation**:
```go
// modbridge/modbridge.go
package modbridge

import (
    "github.com/link-fgfgui/mod-downloader-core/minecraft"  // local JAR parsing
    "github.com/link-fgfgui/mod-downloader-core/providers"  // platform API
    "github.com/link-fgfgui/mod-downloader-core/database"   // persistence
    "github.com/link-fgfgui/mod-downloader-core/global"     // local mod index
)

// InstallStatus checks if a platform ModVersion is installed locally
func InstallStatus(version models.ModVersion, instanceID string) string {
    // Bridge: version.SHA1 → global.LocalModPaths (SHA1 lookup)
    localPaths := global.LocalModPathsInInstance(version.SHA1, instanceID)
    if len(localPaths) > 0 {
        return "installed"
    }
    return "new"
}
```

**Dependency direction**: `downloader → modbridge → {providers, database, global, minecraft}` (unidirectional, no cycles).

**Extensibility**: Future cross-domain features (e.g., "find all platform mods for this local JAR", "suggest updates for local mods") go in `modbridge`.

### Decision: Launcher Directory Layouts Live in `minecraft`

**Context**: The app supports both a standard `.minecraft` directory and a
Prism Launcher `instances/` directory. Both produce the same app-facing shape:
`[]structs/minecraft.VersionInfo` plus a selected version directory whose
`mods/` child is scanned or used as an install target.

**Decision**: Launcher-specific directory detection, instance aggregation, and
version-directory resolution belong in `github.com/link-fgfgui/mod-downloader-core/minecraft`. App-facing
code calls `minecraft.LoadLauncherVersions` and `minecraft.VersionDirPath`
instead of branching on launcher markers.

**Signatures**:
```go
type GameDirVersionLoader func(gameDir string) []structs.VersionInfo

func LoadLauncherVersions(root string, loadGameDir GameDirVersionLoader) []structs.VersionInfo
func VersionDirPath(root string, version structs.VersionInfo) string
```

**Contracts**:
- `LoadLauncherVersions` receives the user-selected root and a callback that
  loads ordinary game directories containing `versions/`.
- The callback owns manifest validation policy; launcher layouts only decide
  which game directories should be passed to it.
- Standard `.minecraft` is the fallback layout: load from `root`.
- Prism `instances/` is selected when `root` contains a Prism-like child.
- Prism entries keep the existing composite ID format
  `<instanceName>/<versionFolder>` and display name `<instanceName>`.
- `VersionDirPath` is the single resolver used by scanning, hardlink indexing,
  and install-target lookup.

**Validation & Error Matrix**:
- Empty root or nil loader -> `nil` versions.
- Root read failure in a launcher layout -> `nil` versions for that layout.
- Empty version ID/name or unknown composite form -> empty version directory.
- Invalid manifests are skipped by the game-dir loader, not by launcher layout
  code.

**Good/Base/Bad Cases**:
- Good: add a new launcher by adding another layout implementation in
  `minecraft`; callers stay launcher-agnostic.
- Base: standard `.minecraft` with `versions/<id>/<id>.json` still works
  through the fallback layout.
- Bad: adding `if minecraft.IsSomeLauncherDir(...)` branches in `app.go`,
  `modbridge`, or downloader code.

**Tests Required**:
- Standard fallback calls the game-dir loader exactly once with the selected
  root.
- Prism aggregation preserves composite IDs, instance display names, `.minecraft`
  subfolder preference, and root fallback.
- `VersionDirPath` resolves both standard IDs and Prism composite IDs.

### Decision: Memory Cache for High-Churn Data, Persistent Cache for Immutable Data

**Context**: We had two types of cached metadata:
1. **Local JAR metadata**: Changes frequently (user adds/removes mods), sourced from disk
2. **Platform mod metadata**: Changes rarely (remote files are immutable per SHA1), sourced from APIs

Both were stored in the same persistent cache (`mods.gob.zst`), causing:
- Serialize/deserialize overhead on every Manage page load
- Cache invalidation complexity (when to rebuild local vs platform data)
- Slow startup when scanning 100+ local JARs

**Options Considered**:
1. Keep everything in persistent cache, optimize serialization
2. Move local JAR metadata to memory-only cache
3. Move platform metadata to external database (e.g., SQLite)

**Decision**: We chose Option 2 (memory-only for local, persistent for platform) because:
- **Local JARs**: High-churn data that's cheap to rebuild (parse JAR takes ~5ms)
- **Platform data**: Low-churn data that's expensive to fetch (API rate limits, 200ms+ per request)
- Separation matches data lifecycle and cost profile

**Implementation**:
```go
// global/jarcache.go - memory-only cache
package global

var (
    jarCache   = make(map[string][]structs.ModInfo) // SHA1 → modInfos
    jarCacheMu sync.RWMutex
)

func GetJarMetadata(sha1 string) ([]structs.ModInfo, bool) {
    jarCacheMu.RLock()
    defer jarCacheMu.RUnlock()
    mods, ok := jarCache[sha1]
    return mods, ok
}

func SetJarMetadata(sha1 string, mods []structs.ModInfo) {
    jarCacheMu.Lock()
    defer jarCacheMu.Unlock()
    jarCache[sha1] = mods
}
```

```go
// database/mods.go - persistent cache (platform data only)
func (db *Database) SetVersionModIDs(platformVersionID string, modIDs []string) error {
    // Persist remote JAR modIDs alongside platform ModVersion
    // (remote files are immutable, cache is long-lived)
}
```

**Cache lifecycle**:
- **Local cache**: Rebuilt on app startup via `ScanVersionMods()`, cleared on instance switch
- **Platform cache**: Persisted across restarts, 15min TTL for freshness

**Why this works**: Local JAR scanning is real-time and cheap; platform API fetches are expensive and rate-limited. Match cache strategy to data source characteristics.

### Decision: Upward Signaling via Callback (Layer Constraint Workaround)

**Context**: `modbridge` sits below the Wails adapter in the dependency graph (`app.go → appcore → downloader → modbridge → {providers, database, global, minecraft}`). `modbridge` must NOT import `wails/runtime` (it would invert the dependency direction and couple a pure-logic package to the Wails runtime). But `modbridge.DownloadStates` sometimes needs to trigger a frontend refresh after an async backfill completes — and only `app.go` can emit Wails events because only it owns `a.ctx`.

**Options Considered**:
1. Let `modbridge` import `wails/runtime` and emit directly (breaks layering, couples logic to runtime)
2. Have `modbridge` return a "needs refresh" flag and let `app.go` poll (brittle, races with async goroutine)
3. Pass a `func()` callback from `app.go` through `downloader` into `modbridge`, invoked when async work finishes

**Decision**: Option 3 (callback through the intermediate layer). `app.GetDownloadStates` calls `appcore.Service.GetDownloadStates`; `appcore` passes an `onBackfillComplete func()` to `downloader.GetDownloadStates`, which transparently forwards it to `modbridge.DownloadStates`. When the callback fires, `appcore` emits `EventDownloadStatesUpdated`; `app.go` maps that adapter-neutral event to `runtime.EventsEmit`. `modbridge` invokes the callback once after all async backfill goroutines finish — never needing to know about Wails.

**Implementation**:
```go
// app.go — owns ctx and maps adapter-neutral core events to Wails
func (a *App) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
    return a.service().GetDownloadStates(req)
}

func (a *App) emitCoreEvent(event appcore.Event) {
    if event.Kind == appcore.EventDownloadStatesUpdated {
        runtime.EventsEmit(a.ctx, downloadStatesUpdatedEvent)
    }
}

// core/appcore/service.go — pure service layer emits adapter-neutral event
func (s *Service) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
    return downloader.GetDownloadStates(req, func() {
        s.emit(EventDownloadStatesUpdated, nil)
    })
}

// core/downloader/download.go — pure passthrough
func GetDownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
    return modbridge.DownloadStates(req, onBackfillComplete)
}

// core/modbridge/modbridge.go — invokes callback after async work, no wails import
func DownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
    // ... sync status decisions ...
    backfill := drainPendingBackfills()
    if len(backfill) > 0 && onBackfillComplete != nil {
        go func() {
            for _, b := range backfill { backfillVersionModIDs(b.version, b.modLoader) }
            onBackfillComplete()
        }()
    }
    return states
}
```

**When to apply**: Any time a lower-layer package (`modbridge`, `providers`, `database`, `httpserver`) needs to signal an adapter after async work, but cannot import `wails/runtime` due to layering. Pass a `func()` callback or `OnEvent` hook from the adapter-neutral service boundary, then let `app.go` perform Wails-specific event emission. The callback must be invoked exactly once after all async work completes; nil-check before invoking.

---

### Decision: Host-Owned JIJ Metadata (`ModInfo.JijMods`)

**Context**: `ParseModZipReader` recursively extracts metadata from both the host JAR's own `mods.toml` / `neoforge.mods.toml` / `fabric.mod.json` declarations and nested jar / jar-in-jar (JIJ) entries. JIJ modIDs are weak references: they describe what a host bundles, not what the host exposes as its own install identity. Flatly returning JIJ entries beside top-level declarations made consumers filter weak refs at every conflict / archive boundary.

**Decision**: `ParseModZipReader` returns only top-level `ModInfo` entries for the parsed JAR. Any JIJ entries carried by that JAR, including recursive JIJ entries, are attached to each top-level `ModInfo` as `JijMods []JijModInfo`. `JijModInfo` intentionally stores only `ID` and `Name`; it is display/diagnostic metadata, not install identity.

**Signatures**:
```go
// structs/minecraft/modinfo.go
type JijModInfo struct {
    ID   string `json:"id"`
    Name string `json:"name"`
}

type ModInfo struct {
    // ...existing fields...
    JijMods []JijModInfo `json:"jijMods,omitempty"`
}

// minecraft/modparser.go
// PrimaryModIDs returns lowercased, deduplicated top-level mod IDs.
// Use this instead of manual ID extraction whenever the result is consumed by
// install-status, conflict detection, version persistence, or archive logic.
func PrimaryModIDs(mods []structs.ModInfo) []string
```

**Contracts**:
- `ParseModZipReader` returns only the parsed JAR's top-level declarations. JIJ declarations are never returned as sibling `ModInfo` entries.
- Recursive JIJ declarations are flattened into `ModInfo.JijMods` on the host's top-level entries, deduped by modID. A nested JAR's own top-level declaration is still weak from the host's perspective.
- `PrimaryModIDs` is idempotent and deduplicated. It reads only `ModInfo.ID` and never reads `JijMods`.

**Validation & Error Matrix**:
- Host JAR with only JIJ entries and no top-level declaration -> `ParseModZipReader` returns `[]`; no conflict or archive action.
- Host declares same modID in top-level and JIJ -> top-level `ModInfo.ID` participates in install identity; the duplicate JIJ entry stays informational under `JijMods`.

**Good/Base/Bad Cases**:
- **Good**: `tmrv.jar` declares top-level `tmrv` and `jei`, plus a JIJ child `childmod`. `ParseModZipReader` returns `tmrv` and `jei`; each has `JijMods: [{ID:"childmod"}]`. `PrimaryModIDs` returns `["tmrv", "jei"]`.
- **Base**: Standard JAR with one top-level modID and no JIJ. `JijMods` is empty and behavior is unchanged.
- **Bad (prevented)**: Two unrelated hosts can both bundle JIJ `lib_x` without being treated as conflicting providers of `lib_x`, because `lib_x` is not written as a top-level `ModInfo.ID`.

**Tests Required**:
- `TestForgeJijModsAreAttachedToTopLevelMods` (`minecraft/modparser_test.go`): assert top-level `[[mods]]` entries are returned as `ModInfo` entries; JIJ child entries are attached under `JijMods`; `PrimaryModIDs` excludes JIJ IDs.

**Wrong vs Correct**:

```go
// Wrong - reads display-only JIJ metadata as install identity
modIDs := make([]string, 0, len(mods))
for _, m := range mods {
    if id := strings.TrimSpace(m.ID); id != "" {
        modIDs = append(modIDs, strings.ToLower(id))
    }
    for _, jij := range m.JijMods {
        modIDs = append(modIDs, strings.ToLower(jij.ID))
    }
}

// Correct - extracts only top-level install identity
modIDs := minecraft.PrimaryModIDs(mods)
```

**Call-site rule**: Every place that extracts modIDs from `ParseModZipReader` / `ParseModJarWithSHA1` results for use in version-persistence, conflict-detection, or archive logic MUST call `minecraft.PrimaryModIDs` — not a manual loop. Grep for `\.ID` on `ModInfo` slices as a signal to check.

**`UpsertLocalMod` rule**: Loops that call `global.UpsertLocalMod` over `ParseModZipReader` results can write every returned `ModInfo`, because returned entries are top-level only. Do not expand `JijMods` into local mod index rows.

**Conflict archive rule**: Archive candidates come from `LocalModPathsForModIDs(PrimaryModIDs(newMods), instanceID)`. Do not apply an additional "fully covered" filter: with host-owned `JijMods`, the local mod index already contains only top-level install identities, so a partial top-level match (old JAR declares `tmrv` and `jei`, new JAR declares `jei`) is a real duplicate-modID conflict and must remain an archive candidate after user confirmation.

---

## Examples

- Well-organized canonical-type package: `core/models/models.go`
- Converter functions following the naming convention: `core/providers/modprovider.go` (`modToModProject`, `fileToModVersion`, etc.)
- Bridge package for cross-domain convergence: `core/modbridge/modbridge.go`
