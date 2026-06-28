# Database Guidelines

> Database patterns and conventions for this project.

---

## Overview

This project uses **BoltDB** (embedded key-value store) for caching platform mod metadata. The cache is serialized as gob-compressed data (`mods.gob.zst`).

**Key principles**:
- Persistent cache is for **expensive-to-fetch, low-churn data** (platform API responses)
- High-churn data (local JARs) belongs in memory-only caches (see `global/jarcache.go`)
- Cache schema changes require `cacheVersion` bumps to trigger rebuilds

---

## Cache Schema Evolution

### Pattern: Cache Version Bumps for Breaking Changes

**Problem**: When the `cacheState` struct changes (fields added/removed), old gob-serialized cache files cause deserialization errors on app startup.

**Solution**: Increment `cacheVersion` constant when making breaking changes to `cacheState`. Old caches are discarded and rebuilt.

**Implementation**:
```go
// database/database.go
const cacheVersion = 3 // Bump on breaking changes

type cacheState struct {
    Version int // Must match cacheVersion

    // Platform metadata (persisted)
    ModProjects      map[string]models.ModProject
    ModVersions      map[string]models.ModVersion
    PinnedMods       map[string]PinnedMod
    
    // Local JAR metadata removed in v3 (moved to global.jarCache)
    // JarMetadata map[string][]structs.ModInfo  // ❌ Removed
}

func (db *Database) Load() error {
    // ... deserialize ...
    if state.Version != cacheVersion {
        log.Info("Cache version mismatch, rebuilding")
        return db.initializeEmpty() // Discard old cache
    }
}
```

**When to bump**:
- ✅ Adding/removing fields from `cacheState`
- ✅ Changing field types (e.g., `string` → `[]string`)
- ✅ Renaming fields
- ❌ Adding fields to nested structs that are already versioned independently (e.g., `models.ModVersion` has its own schema)

**Migration strategy**: This project uses **discard-and-rebuild** (no incremental migrations). Platform data refetches from APIs; local data rescans from disk.

---

## Data Separation by Lifecycle

### Convention: Persistent vs Memory-Only Caches

**Rule**: Match cache strategy to data source characteristics.

| Data Type | Cache Location | Rebuild Cost | Churn Rate | Example |
|-----------|---------------|--------------|------------|---------|
| **Platform metadata** | Persistent (`mods.gob.zst`) | High (API rate limits, 200ms+) | Low (immutable per SHA1) | `ModProject`, `ModVersion` |
| **Local JAR metadata** | Memory (`global.jarCache`) | Low (~5ms parse per JAR) | High (user adds/removes mods) | Parsed `ModInfo` from JARs |

**Example - Wrong (before refactor)**:
```go
// ❌ Local JAR metadata in persistent cache
type cacheState struct {
    JarMetadata map[string][]structs.ModInfo // Rebuilt on every scan!
}

// Problem: Serialize 100 JARs worth of metadata → slow save
// Problem: Deserialize on startup → slow load
// Problem: Cache invalidation logic complex
```

**Example - Correct (after refactor)**:
```go
// ✅ Platform data persisted
// database/database.go
type cacheState struct {
    ModProjects map[string]models.ModProject // API data, expensive
    ModVersions map[string]models.ModVersion  // API data, expensive
}

// ✅ Local data in memory
// global/jarcache.go
var jarCache = make(map[string][]structs.ModInfo) // Cheap to rebuild
```

**Decision matrix**:
- **Expensive + low-churn** → Persistent cache
- **Cheap + high-churn** → Memory-only cache
- **Expensive + high-churn** → Need more sophisticated strategy (not in this project yet)

---

## Query Patterns

### Pattern: Composite Keys for Multi-Platform Data

Platform mod data is indexed by composite keys: `<platform>:<id>` (e.g., `modrinth:AANobbMI`, `curseforge:12345`).

**Helper functions** (in `models/models.go`):
```go
func ProjectKey(platform, projectID string) string {
    return platform + ":" + projectID
}

func ParseProjectKey(key string) (platform, projectID string, ok bool) {
    parts := strings.SplitN(key, ":", 2)
    if len(parts) != 2 { return "", "", false }
    return parts[0], parts[1], true
}
```

**Usage**:
```go
// Store
key := models.ProjectKey("modrinth", "AANobbMI")
db.cacheState.ModProjects[key] = project

// Lookup
project, ok := db.cacheState.ModProjects[key]
```

**Why composite keys**: Supports multiple platforms (CurseForge, Modrinth) without ID collisions. Platform-agnostic caller code.

---

## Migrations

No incremental migrations. Cache schema changes trigger full rebuilds via `cacheVersion` bumps (see [Cache Schema Evolution](#cache-schema-evolution)).

---

## Naming Conventions

### Struct Fields
- BoltDB keys: `camelCase` (Go convention)
- JSON tags: `camelCase` (matches frontend expectations)
- Gob serialization: uses field names directly

### Map Keys
- Platform composite keys: `<platform>:<id>` format
- SHA1 hashes: lowercase hex string (40 chars)

---

## Common Mistakes

### Mistake: Persisting High-Churn Data

**Symptom**: App startup/shutdown slow, cache file grows unbounded

**Cause**: Caching data that changes frequently or is cheap to rebuild (e.g., local JAR metadata)

**Fix**: Move to memory-only cache (`global` package) or don't cache at all

**Prevention**: Ask "Is this expensive to fetch?" and "Does it change often?" before adding to persistent cache. See [Data Separation by Lifecycle](#convention-persistent-vs-memory-only-caches).

### Mistake: Forgetting to Bump Cache Version

**Symptom**: App crashes on startup with gob deserialization errors after code changes

**Cause**: Changed `cacheState` struct without incrementing `cacheVersion`

**Fix**: Bump `cacheVersion` constant in `database/database.go`

**Prevention**: Add `cacheVersion` bump to PR checklist when touching `cacheState`

### Mistake: Mixing Domain Concerns in Cache

**Symptom**: Platform API code needs to know about local JAR structure, or vice versa

**Cause**: Storing cross-domain bridging data in the wrong cache (e.g., local→platform associations in platform cache)

**Fix**: Keep domain caches pure. Cross-domain queries go through `modbridge` package.

**Prevention**: Each cache owns one data source. Platform cache = API data only. Local cache = JAR data only. Bridge package handles convergence.
