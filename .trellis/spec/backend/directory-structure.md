# Directory Structure

> How backend code is organized in this project.

---

## Overview

mod-downloader is a Wails v2 desktop app (Go backend + Vue 3 frontend). The backend is a single Go module (`mod-downloader`) organized into packages by layer. CF/MR mod metadata types live in `models` as the single source of truth.

---

## Directory Layout

```
mod-downloader/
├── main.go                      # Wails app bootstrap
├── app.go                       # Wails-exposed API methods (App struct)
├── models/                      # Canonical data types (single source of truth)
│   ├── models.go                #   ModProject, ModVersion, ModDependency + composite-key helpers
│   └── models_test.go
├── structs/                     # Request/response structs + minecraft manifest types
│   ├── search.go                #   SearchModsRequest, ModDownloadRequest, SearchModsUpdate, ...
│   └── minecraft/               #   Minecraft version manifest types (unrelated to mod metadata)
├── providers/                   # CF/MR platform abstraction layer
│   ├── modprovider.go           #   modProvider interface + CF/MR implementations + SDK→models converters
│   ├── service.go               #   Facade: SearchMods, ListMatchingProjectVersions, ...
│   └── cache.go                 #   Higher-level DB access (GetProjectByID, StoreVersion, ...)
├── database/                    # BoltDB persistence (cache snapshots, associations, pins)
│   ├── database.go              #   cacheState, load/save, copy helpers
│   └── mods.go                  #   ModPlatform/Version/Association/Pinned CRUD
├── downloader/                  # Download queue + state machine
│   └── download.go
├── modbridge/                   # Cross-domain bridge: version resolution, install status, SHA1↔platform mapping
│   └── modbridge.go
├── global/                      # Global singletons (CF/MR SDK clients, local mods, in-memory JAR metadata cache)
├── configs/                     # Config load/save
├── minecraft/                   # Minecraft JAR parser, version manifest fetcher
├── logging/                     # Structured logger wrapper
└── frontend/                    # Vue 3 + Pinia frontend (Wails-generated bindings in wailsjs/)
```

---

## Module Organization

### Layered data flow (mod metadata)

```
[CF/MR SDK] → providers (SDK→models converters) → models (canonical types)
                    ↓                                  ↑
              database (caches models.*)         structs (request/response; consumes models)
                    ↓                                  ↑
              downloader (consumes models.*)      app.go (Wails API; consumes models + structs)
                    ↓
              modbridge (cross-domain: version resolution, install status, SHA1↔platform bridge)
               ↙       ↘
         global      minecraft
   (local mods +     (local JAR
    JAR mem cache)    parsing)
```

**Boundary constraint**: `minecraft` (local analysis) and `providers` (platform analysis) must NOT import each other. Their convergence point is `modbridge`. Dependency direction is unidirectional: `downloader → modbridge → {providers, database, global, minecraft}`.

### Convention: `models` is the single source of truth

**What**: `mod-downloader/models` defines `ModProject`, `ModVersion`, `ModDependency`, and the composite-key helpers (`ProjectKey`, `ParseProjectKey`, `VersionKey`, `ParseVersionKey`). Every other package imports `models` directly — no type aliases, no re-export files.

**Why**: Previously `structs.SearchModResult = models.ModProject` (alias) and `providers/model.go` (re-export) gave the same type three names. This made cross-file search noisy, obscured which package owned the type, and let a parallel "old" conversion path (`modToSearchResult`) coexist with a "new" path (`modToModProject`) — the old path silently dropped the `ProjectID` field, a bug that went unnoticed because the new (correct) path was dead code.

**Example**:
```go
// Good — import models directly
import "mod-downloader/models"

func (a *App) ListMatchingProjectVersions(result models.ModProject, mcVersion, modLoader string) []models.ModVersion

// Bad — alias or re-export (removed in 06-27-unify-models-cleanup)
type SearchModResult = models.ModProject   // forbidden: third name for same type
// providers/model.go: type ModProject = models.ModProject  // forbidden: re-export file
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
    "mod-downloader/minecraft"  // local JAR parsing
    "mod-downloader/providers"  // platform API
    "mod-downloader/database"   // persistence
    "mod-downloader/global"     // local mod index
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
version-directory resolution belong in `mod-downloader/minecraft`. App-facing
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

**Context**: `modbridge` sits below `app.go` in the dependency graph (`downloader → modbridge → {providers, database, global, minecraft}`). `modbridge` must NOT import `wails/runtime` (it would invert the dependency direction and couple a pure-logic package to the Wails runtime). But `modbridge.DownloadStates` sometimes needs to trigger a frontend refresh after an async backfill completes — and only `app.go` can emit Wails events because only it owns `a.ctx`.

**Options Considered**:
1. Let `modbridge` import `wails/runtime` and emit directly (breaks layering, couples logic to runtime)
2. Have `modbridge` return a "needs refresh" flag and let `app.go` poll (brittle, races with async goroutine)
3. Pass a `func()` callback from `app.go` through `downloader` into `modbridge`, invoked when async work finishes

**Decision**: Option 3 (callback through the intermediate layer). `app.GetDownloadStates` creates a closure over `runtime.EventsEmit(a.ctx, ...)`, passes it as `onBackfillComplete func()` to `downloader.GetDownloadStates`, which transparently forwards it to `modbridge.DownloadStates`. `modbridge` invokes the callback once after all async backfill goroutines finish — never needing to know about Wails.

**Implementation**:
```go
// app.go — owns ctx, creates the emitter closure
func (a *App) GetDownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState {
    return downloader.GetDownloadStates(req, func() {
        runtime.EventsEmit(a.ctx, downloadStatesUpdatedEvent)
    })
}

// downloader/download.go — pure passthrough
func GetDownloadStates(req appstructs.DownloadStatesRequest, onBackfillComplete func()) []appstructs.ModDownloadButtonState {
    return modbridge.DownloadStates(req, onBackfillComplete)
}

// modbridge/modbridge.go — invokes callback after async work, no wails import
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

**When to apply**: Any time a lower-layer package (`modbridge`, `providers`, `database`) needs to signal the frontend after async work, but cannot import `wails/runtime` due to layering. Pass a `func()` callback from the layer that owns the Wails context (`app.go`) through any intermediate layers. The callback must be invoked exactly once after all async work completes; nil-check before invoking.

---

### Decision: Strong vs Weak ModID References (`ModInfo.IsJij`)

**Context**: `ParseModZipReader` recursively extracts mod metadata from both the host JAR's own `mods.toml` / `neoforge.mods.toml` / `fabric.mod.json` declarations and from nested jar / jar-in-jar (JIJ) entries. Previously all modIDs were returned as a flat list, so callers could not distinguish "this JAR declares modID X" (strong reference) from "this JAR bundles a nested JAR that declares modID X" (weak reference). This caused false conflict hits and wrong replacement-archive decisions (e.g., installing `jei` archiving `tmrv` because `tmrv` also declares `jei` in its JIJ-bundled child).

**Options Considered**:
1. Return two slices `(primary []ModInfo, jij []ModInfo)` from `ParseModZipReader` — breaks every existing call site.
2. Return a new struct `ParsedJar{Primary, Jij []ModInfo}` — cleaner but same call-site impact, higher change cost.
3. Add `IsJij bool` to `ModInfo` — zero call-site breakage; callers filter via `PrimaryModIDs` helper.

**Decision**: Option 3. `IsJij` uses `omitempty` so `false` values are invisible to JSON serialization and the frontend. Filtering is centralized in `PrimaryModIDs`.

**Signatures**:
```go
// structs/minecraft/modinfo.go
type ModInfo struct {
    // ...existing fields...
    // IsJij is true when this entry originates from a nested jar (JIJ). These
    // are weak references that must NOT participate in install-conflict detection
    // or replacement-archive decisions at the same level as top-level declarations.
    IsJij bool `json:"isJij,omitempty"`
}

// minecraft/modparser.go
// PrimaryModIDs returns lowercased, deduplicated mod IDs from mods where IsJij==false.
// Use this instead of manual ID extraction whenever the result is consumed by
// install-status, conflict detection, version persistence, or archive logic.
func PrimaryModIDs(mods []structs.ModInfo) []string
```

**Contracts**:
- `parseNestedJar` marks ALL returned `ModInfo` as `IsJij = true` regardless of recursion depth. A nested JAR's own top-level declaration is still a weak reference from the host's perspective.
- `ParseModZipReader` returns a mix of strong (`IsJij==false`) and weak (`IsJij==true`) entries, deduped by modID (first occurrence wins, preserving `IsJij`).
- `PrimaryModIDs` is idempotent and deduplicated. It NEVER returns JIJ modIDs.

**Validation & Error Matrix**:
- Host JAR with only JIJ entries → `PrimaryModIDs` returns `[]` (empty); no conflict or archive action.
- Host declares same modID in top-level and JIJ → dedup keeps first occurrence; since top-level is processed before `parseNestedJar`, the entry is `IsJij==false`.

**Good/Base/Bad Cases**:
- **Good**: `tmrv.jar` declares `[[mods]] modId="tmrv"` and `[[mods]] modId="jei"` in its own `mods.toml`, plus a JIJ child `childmod`. `PrimaryModIDs` → `["tmrv", "jei"]`. Installing standalone `jei.jar` triggers conflict against `tmrv.jar` correctly.
- **Base**: Standard JAR with one top-level modID and no JIJ. `PrimaryModIDs` → `["mymod"]`. Behavior unchanged.
- **Bad (prevented)**: Without the rule, installing `jei` would look up `childmod` in the local index via `LocalModPathsForModIDs`, find `tmrv` (because `tmrv` bundles `childmod` as JIJ), and try to archive `tmrv` — incorrectly removing a completely unrelated mod.

**Tests Required**:
- `TestForgeModIDStrengthClassification` (`minecraft/modparser_test.go`): assert top-level `[[mods]]` entries → `IsJij==false`; JIJ child entries → `IsJij==true`; `PrimaryModIDs` excludes JIJ IDs.

**Wrong vs Correct**:

```go
// Wrong — extracts all modIDs including JIJ weak refs
modIDs := make([]string, 0, len(mods))
for _, m := range mods {
    if id := strings.TrimSpace(m.ID); id != "" {
        modIDs = append(modIDs, strings.ToLower(id))
    }
}

// Correct — extracts only strong-reference modIDs
modIDs := minecraft.PrimaryModIDs(mods)
```

**Call-site rule**: Every place that extracts modIDs from `ParseModZipReader` / `ParseModJarWithSHA1` results for use in version-persistence, conflict-detection, or archive logic MUST call `minecraft.PrimaryModIDs` — not a manual loop. Grep for `\.ID` on `ModInfo` slices as a signal to check.

**`UpsertLocalMod` guard rule**: Every loop that calls `global.UpsertLocalMod` over `ParseModZipReader` results MUST skip `IsJij==true` entries:
```go
for i := range mods {
    if mods[i].IsJij {
        continue // JIJ weak refs must not enter the local mod index
    }
    // ...set fields and call UpsertLocalMod
}
```
Failure to add this guard causes false `installed`/`conflict` states when the JIJ modID matches a remote version being browsed.

**`FilterFullyCoveredPaths` guard rule**: `archiveSupersededModJars` must always receive the output of `FilterFullyCoveredPaths`, never the raw `LocalModPathsForModIDs` result. Missing this (as was the case in `tryHardlinkInstall` before this fix) silently bypasses the coverage check and can archive JARs that provide additional modIDs the replacement does not cover.

---

## Examples

- Well-organized canonical-type package: [models/models.go](file:///home/link/Documents/go_proj/mod-downloader/models/models.go)
- Converter functions following the naming convention: [providers/modprovider.go](file:///home/link/Documents/go_proj/mod-downloader/providers/modprovider.go) (`modToModProject`, `fileToModVersion`, etc.)
- Bridge package for cross-domain convergence: [modbridge/modbridge.go](file:///home/link/Documents/go_proj/mod-downloader/modbridge/modbridge.go)
