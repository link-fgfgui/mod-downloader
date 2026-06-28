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

---

## Examples

- Well-organized canonical-type package: [models/models.go](file:///home/link/Documents/go_proj/mod-downloader/models/models.go)
- Converter functions following the naming convention: [providers/modprovider.go](file:///home/link/Documents/go_proj/mod-downloader/providers/modprovider.go) (`modToModProject`, `fileToModVersion`, etc.)
- Bridge package for cross-domain convergence: [modbridge/modbridge.go](file:///home/link/Documents/go_proj/mod-downloader/modbridge/modbridge.go)
