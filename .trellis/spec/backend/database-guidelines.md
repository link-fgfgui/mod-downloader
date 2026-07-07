# Database Guidelines

> Database patterns and conventions for this project.

---

## Overview

This project uses two local storage files with different lifecycles:

- `mods.gob.zst` — gob/zstd serialized platform metadata cache.
- `user-data.sqlite` — SQLite user-owned data store.

**Key principles**:
- Persistent cache is for **expensive-to-fetch, low-churn data** (platform API responses)
- User-owned state belongs in **SQLite**, not in rebuildable cache files
- High-churn data (local JARs) belongs in memory-only caches (see `global/jarcache.go`)
- Platform cache schema changes require `cacheVersion` bumps to trigger rebuilds

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
| **Platform metadata** | Persistent gob cache (`mods.gob.zst`) | High (API rate limits, 200ms+) | Low (immutable per SHA1) | `ModProject`, `ModVersion` |
| **User-owned mod state** | SQLite (`user-data.sqlite`) | User data, not rebuildable | User-driven | `PinnedMod`, `FavoriteList`, `FavoriteMod` |
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
- **Expensive + low-churn platform data** → Persistent gob cache
- **User-owned app state** → SQLite
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

### Scenario: SQLite User Data Store

#### 1. Scope / Trigger

Use this pattern when adding user-owned mod state that must survive cache
rebuilds, such as pinned mod versions, favorite lists, favorite memberships, or
future user collections. This data is not platform metadata and must not be
stored in `mods.gob.zst` or `mod-downloader.toml`.

#### 2. Signatures

```go
const (
    CacheFileName    = "mods.gob.zst"
    UserDataFileName = "user-data.sqlite"
)

func OpenAt(cachePath string) error
func Close()
func UserDataPathForCachePath(cachePath string) string

func UpsertPinnedMod(p PinnedMod) error
func GetPinnedMod(platform, modID, mcVersion, modLoader string) (PinnedMod, bool)
func DeletePinnedMod(platform, modID, mcVersion, modLoader string) error
func ListPinnedMods() []PinnedMod

func CreateFavoriteList(name string) (FavoriteList, error)
func UpdateFavoriteListName(id, name string) (FavoriteList, bool, error)
func DeleteFavoriteList(id string) (bool, error)
func ListFavoriteLists() []FavoriteList
func UpsertFavoriteMod(mod FavoriteMod) (FavoriteMod, bool, error)
func DeleteFavoriteMod(listID, platform, modID, mcVersion, modLoader string) (bool, error)
func ListFavoriteMods(listID string) []FavoriteMod
```

SQLite schema owner: `core/database/userdb.go`.

#### 3. Contracts

- `OpenAt(cachePath)` opens `mods.gob.zst` at `cachePath` and SQLite at
  `UserDataPathForCachePath(cachePath)`.
- `Runtime.CachePath` is a precise cache-file override. `Runtime.CacheDir` and
  TOML `runtime.cache_dir` resolve to `filepath.Join(cacheDir, CacheFileName)`.
- `mod-downloader.toml` stores configuration only. Pins, favorites, and other
  user collections must not be serialized into TOML.
- Legacy gob fields for `PinnedMods`, `FavoriteLists`, and `FavoriteMods`
  remain only to decode and migrate old cache files.
- Platform cache data remains gob-backed and rebuildable. Do not move platform
  metadata to SQLite unless a task explicitly owns that migration.
- SQLite stores normalized key fields: lower-case `platform`, `mod_id`, and
  `mod_loader`; trimmed `minecraft_version`; empty strings instead of NULL for
  favorite scope fields so unique constraints behave like the old composite
  keys.
- Favorite categories are stored as JSON text in `categories_json`; callers
  still receive `[]string`.

#### 4. Validation & Error Matrix

- Empty cache path -> `OpenAt` returns an error.
- Missing gob cache -> start with an empty platform cache.
- Missing SQLite file -> create parent directory, create schema, then continue.
- SQLite open/schema failure -> `OpenAt` returns an error; callers must not
  silently continue with user-data writes disabled.
- Legacy favorite item without an existing list -> skip that orphan during
  migration instead of failing startup.
- Empty pin key fields -> no row written.
- Favorite mod for a missing list -> no row written.
- Duplicate favorite mod key -> update existing row while preserving ID and
  creation time when caller leaves them empty.

#### 5. Good/Base/Bad Cases

- Good: pin Sodium for `modrinth/sodium/1.21.1/fabric`; it persists in
  `user-data.sqlite` and `ResolveVersions` continues to read through
  `database.GetPinnedMod`.
- Good: change `runtime.cache_dir`; service saves TOML config and reopens
  storage at `<cache_dir>/mods.gob.zst` plus `<cache_dir>/user-data.sqlite`.
- Base: an existing gob cache containing legacy pins/favorites migrates once,
  then the next gob save clears those legacy maps.
- Bad: writing favorite lists to `mods.gob.zst`, because cache version bumps or
  cache deletion would destroy user data.
- Bad: adding pin/favorite fields to `mod-downloader.toml`; TOML is config, not
  collection storage.

#### 6. Tests Required

- SQLite persistence tests for pins and favorite lists/mods across close/reopen.
- Legacy gob migration test that asserts migrated rows are readable from SQLite
  and legacy gob user maps are cleared after close.
- Favorite tests for create, rename, delete cascade, duplicate upsert, sort
  order, missing list, and returned-copy behavior.
- Appcore settings test for `runtime.cache_dir` save plus storage reopen.
- Wails binding regeneration and frontend build after settings API fields or
  methods change.

#### 7. Wrong vs Correct

Wrong:

```go
type cacheState struct {
    PinnedMods map[pinnedModKey]PinnedMod // active storage for user state
}
```

Correct:

```go
func UpsertPinnedMod(p PinnedMod) error {
    d, err := readyUserDB()
    if err != nil {
        return err
    }
    return d.upsertPinnedMod(p)
}
```

Wrong:

```toml
[favorites]
mods = ["modrinth:sodium"]
```

Correct:

```go
// TOML keeps runtime configuration only.
type RuntimeConfig struct {
    CacheDir string `toml:"cache_dir" json:"cacheDir" env:"MOD_DOWNLOADER_CACHE_DIR"`
}
```

### Scenario: Favorite Lists Persistent Collections

#### 1. Scope / Trigger

Use this pattern for user-owned collections of platform mods, such as named
favorite lists. These records are persistent user data, not local JAR scan
cache or platform metadata cache, so they belong in SQLite (`user-data.sqlite`).
Schema changes belong in `core/database/userdb.go`.

#### 2. Signatures

```go
type FavoriteList struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    CreatedAt int64  `json:"createdAt"`
    UpdatedAt int64  `json:"updatedAt"`
    SortOrder int    `json:"sortOrder"`
}

type FavoriteMod struct {
    ID               string   `json:"id"`
    ListID           string   `json:"listId"`
    Platform         string   `json:"platform"`
    ModID            string   `json:"modId"`
    VersionID        string   `json:"versionId,omitempty"`
    MinecraftVersion string   `json:"minecraftVersion,omitempty"`
    ModLoader        string   `json:"modLoader,omitempty"`
    Title            string   `json:"title,omitempty"`
    Slug             string   `json:"slug,omitempty"`
    IconURL          string   `json:"iconUrl,omitempty"`
    Categories       []string `json:"categories,omitempty"`
}
```

#### 3. Contracts

- `PinnedMod` and `FavoriteMod` are separate concepts. Pinned mods affect
  download version resolution; favorites are user collections.
- Favorite membership is keyed in SQLite by
  `listID/platform/modID/minecraftVersion/modLoader`.
- `platform`, `modID`, and `modLoader` are normalized to lowercase. Display
  metadata is copied from platform metadata and may be empty.
- Deleting a favorite list cascades its favorite mods only. It must not touch
  pinned-version records.

#### 4. Validation & Error Matrix

- Empty list name -> no list created.
- Missing list ID or missing platform/mod ID -> no favorite mod persisted.
- Favorite mod for missing list -> no favorite mod persisted.
- Duplicate favorite mod key -> update existing row while preserving ID and
  creation time.

#### 5. Good/Base/Bad Cases

- Good: Add Modrinth Sodium to two different lists; each list owns its own row.
- Base: Add Sodium twice to the same list/version scope; the existing row is
  updated.
- Bad: Reuse `PinnedMod` to represent favorites; this couples UI collections to
  the download resolver.

#### 6. Tests Required

- Database tests for create, rename, delete cascade, list sorting, item upsert,
  duplicate update, SQLite persistence after reopen, and returned-copy behavior.
- Service tests that call the appcore favorite methods rather than only database
  functions.

#### 7. Wrong vs Correct

Wrong:

```go
// Favorites accidentally modify download pin behavior.
database.UpsertPinnedMod(database.PinnedMod{Platform: platform, ModID: modID})
```

Correct:

```go
database.UpsertFavoriteMod(database.FavoriteMod{
    ListID: listID,
    Platform: platform,
    ModID: modID,
})
```

### Scenario: Favorite Bulk Operations And Live References

#### 1. Scope / Trigger

Use this pattern when exposing operations that copy favorite mods between lists,
copy one favorite list into another, or manage live favorite-list references.
The storage relationship lives in SQLite, app-independent request handling lives
in `core/appcore`, and Wails only delegates to the service.

#### 2. Signatures

Core service request/response types:

```go
type FavoriteBulkAddRequest struct {
    TargetListIDs []string               `json:"targetListIds"`
    Mods          []database.FavoriteMod `json:"mods"`
}

type FavoriteListCopyRequest struct {
    SourceListID string `json:"sourceListId"`
    TargetListID string `json:"targetListId"`
}

type FavoriteBulkOperationResult struct {
    Added   int      `json:"added"`
    Updated int      `json:"updated"`
    Skipped int      `json:"skipped"`
    Errors  []string `json:"errors,omitempty"`
}
```

Core service methods:

```go
func (s *Service) AddFavoriteModsToLists(req FavoriteBulkAddRequest) FavoriteBulkOperationResult
func (s *Service) CopyFavoriteListToList(req FavoriteListCopyRequest) FavoriteBulkOperationResult
func (s *Service) AddFavoriteListReference(parentListID, childListID string) database.FavoriteListRef
func (s *Service) RemoveFavoriteListReference(parentListID, childListID string) bool
func (s *Service) ListFavoriteListRefs(parentListID string) []database.FavoriteListRef
func (s *Service) ListFavoriteContents(listID string) database.FavoriteListContents
```

#### 3. Contracts

- Bulk add deduplicates target list IDs and validates every target list before
  writing rows for that target.
- Copied `FavoriteMod` rows must clear `ID`, `CreatedAt`, and `UpdatedAt`
  before upsert so a copied row never reuses the source row's primary key.
- Duplicate membership in a target list is an update, not a second row.
- Whole-list copy reads `database.ListFavoriteContents`, so live referenced
  child-list mods are copied concretely into the target list.
- Whole-list copy from a list to itself is skipped and reported; selected-mod
  bulk add may target the source list because it is idempotent.
- Reference add/remove delegates cycle prevention and persistence to
  `database.CreateFavoriteListRef` / `DeleteFavoriteListRef`.
- `app.go` methods are thin Wails-facing delegations only; no persistence or
  copy semantics belong in the adapter.

#### 4. Validation & Error Matrix

- Empty target list IDs or empty mod list -> no writes; skipped count reflects
  skipped mods where applicable.
- Missing target list -> skip that target and append a readable error.
- Invalid favorite mod key -> skip that mod.
- Existing target favorite key -> upsert and increment `Updated`.
- New target favorite key -> upsert and increment `Added`.
- Reference cycle or storage error -> service returns an empty ref and logs the
  error.

#### 5. Good/Base/Bad Cases

- Good: Copy selected Sodium and Lithium rows to two target lists through
  `AddFavoriteModsToLists`; existing target Sodium is updated and target
  Lithium is added.
- Good: Copy a list containing a live child-list reference; the target receives
  concrete rows for both direct and referenced mods.
- Base: Add a live reference through `AddFavoriteListReference`; no favorite mod
  rows are duplicated.
- Bad: Frontend loops over selected mods and calls `AddFavoriteMod` one by one,
  losing aggregate skipped/error reporting and duplicating target validation.
- Bad: Wails adapter rewrites mod IDs or performs copy logic directly.

#### 6. Tests Required

- Appcore test for selected-mod bulk copy: added, updated, skipped missing
  target, and copied row has a different ID than the source.
- Appcore test for whole-list copy using resolved reference contents.
- Appcore test for reference add/list/remove and cycle rejection behavior.
- Wails binding regeneration after adding or changing Wails-visible method
  signatures.

#### 7. Wrong vs Correct

Wrong:

```go
for _, mod := range selected {
    mod.ListID = targetListID
    database.UpsertFavoriteMod(mod) // reuses source ID and hides aggregate failures
}
```

Correct:

```go
copied := mod
copied.ID = ""
copied.ListID = targetListID
copied.CreatedAt = 0
copied.UpdatedAt = 0
database.UpsertFavoriteMod(copied)
```

---

## Migrations

Platform cache has no incremental migrations. Cache schema changes trigger full
rebuilds via `cacheVersion` bumps (see [Cache Schema Evolution](#cache-schema-evolution)).

SQLite user-data schema is created idempotently by `userdb.go`. Legacy
`PinnedMods`, `FavoriteLists`, and `FavoriteMods` decoded from gob cache are
migrated into SQLite on `OpenAt`, then cleared from the active gob state before
the next cache save.

### Scenario: SQLite User-Data Schema Evolution

#### 1. Scope / Trigger

Use this pattern whenever `user-data.sqlite` needs a new table, index, or column
for user-owned state. This includes favorite list metadata, groups, references,
or any future collection data. SQLite schema evolution must preserve existing
user data; unlike the gob platform cache, user data is not discardable.

#### 2. Signatures

Schema owner:

```go
func (s *userStore) ensureSchema() error
func (s *userStore) ensureSchemaV2() error
func (s *userStore) columnExists(table, column string) (bool, error)
```

Open path:

```go
func OpenAt(cachePath string) error // opens cache plus UserDataPathForCachePath(cachePath)
```

#### 3. Contracts

- Base `CREATE TABLE IF NOT EXISTS` statements may create the current full table
  shape for fresh databases.
- Existing databases do not gain new columns from `CREATE TABLE IF NOT EXISTS`;
  every added column must be applied with an explicit `ALTER TABLE ... ADD COLUMN`
  guarded by `PRAGMA table_info`.
- New migrations record an idempotent row in `schema_migrations`.
- New columns on existing user tables must have `NOT NULL DEFAULT ...` whenever
  old rows need to remain readable without backfill.
- New relationship tables must use foreign keys and indexes that match their
  query path.
- Public read queries must select the current full column set after migrations
  have run.

#### 4. Validation & Error Matrix

- Existing SQLite file missing a new column -> add the column before any read
  query that selects it.
- Existing SQLite file already containing the column -> skip `ALTER TABLE`.
- Migration statement fails -> `OpenAt` returns an error; user-data writes must
  not continue against a partially assumed schema.
- Legacy gob user data and SQLite schema migration are independent: gob
  migration inserts rows after SQLite schema creation has completed.

#### 5. Good/Base/Bad Cases

- Good: add `favorite_lists.pinned` through `columnExists("favorite_lists", "pinned")`
  and `ALTER TABLE favorite_lists ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`.
- Base: a fresh install creates `favorite_lists` with all current columns and
  records schema versions in `schema_migrations`.
- Bad: add `pinned` to the `CREATE TABLE IF NOT EXISTS favorite_lists` text only;
  existing databases keep the old table shape and later `SELECT pinned` fails.

#### 6. Tests Required

- Create a v1 SQLite database manually, open it through `OpenAt`, and assert old
  favorite lists/mods remain readable.
- Assert `schema_migrations` contains the new version after open.
- Assert a row from the old schema can write/read the new fields after
  migration.
- Keep existing favorite persistence and legacy gob migration tests passing.

#### 7. Wrong vs Correct

Wrong:

```go
`CREATE TABLE IF NOT EXISTS favorite_lists (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    pinned INTEGER NOT NULL DEFAULT 0
)`
// Existing favorite_lists tables still do not have pinned.
```

Correct:

```go
if ok, err := s.columnExists("favorite_lists", "pinned"); err != nil {
    return err
} else if !ok {
    if _, err := s.db.Exec(`ALTER TABLE favorite_lists ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`); err != nil {
        return err
    }
}
```

---

## Naming Conventions

### Struct Fields
- JSON tags: `camelCase` (matches frontend expectations)
- Gob serialization: uses Go field names directly
- SQLite columns: `snake_case`

### Keys
- Platform composite keys: `<platform>:<id>` format
- SHA1 hashes: lowercase hex string (40 chars)
- SQLite uniqueness for user data must match the old composite-key behavior.

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
