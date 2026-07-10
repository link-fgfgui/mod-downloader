# Storage Guidelines

> Database patterns and conventions for this project.

---

## Overview

This project uses two local storage files with different lifecycles:

- `mod-metadata.tmp` — gob/zstd serialized platform metadata cache.
- `mod-favs.sqlite` — SQLite user-owned data store.

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
// storage/storage.go
const cacheVersion = 5 // Bump on breaking changes

type cacheState struct {
    Version int // Must match cacheVersion

    // Platform metadata (persisted)
    ModProjects map[string]models.ModProject
    ModVersions map[string]models.ModVersion
    
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
| **Platform metadata** | Persistent gob cache (`mod-metadata.tmp`) | High (API rate limits, 200ms+) | Low (immutable per SHA1) | `ModProject`, `ModVersion` |
| **User-owned mod state** | SQLite (`mod-favs.sqlite`) | User data, not rebuildable | User-driven | `PinnedMod`, `FavoriteList`, `FavoriteMod` |
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
// storage/storage.go
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
stored in `mod-metadata.tmp` or `mod-downloader.toml`.

#### 2. Signatures

```go
const (
    CacheFileName    = "mod-metadata.tmp"
    UserDataFileName = "mod-favs.sqlite"
)

func OpenAt(cachePath string) error
func Close()
func UserDataPathForCachePath(cachePath string) string

type RuntimeOptions struct {
    CachePath       string
    CacheDir        string
    DefaultCacheDir string
}

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

SQLite schema owner: `core/storage/userdb.go`.

#### 3. Contracts

- `OpenAt(cachePath)` opens `mod-metadata.tmp` at `cachePath` and SQLite at
  `UserDataPathForCachePath(cachePath)`.
- `Runtime.CachePath` is a precise cache-file override. `Runtime.CacheDir` and
  TOML `runtime.cache_dir` resolve to `filepath.Join(cacheDir, CacheFileName)`.
- `Runtime.DefaultCacheDir` is an adapter-provided fallback only. It must be
  lower priority than explicit runtime cache settings and TOML/env
  `runtime.cache_dir`, and higher priority than `storage.DefaultCachePath()`.
- The Wails GUI sets `Runtime.DefaultCacheDir` to the process working
  directory so GUI defaults colocate with `pwd/mod-downloader.toml`; CLI callers
  leave it empty so core falls back to the OS temp directory.
- `mod-downloader.toml` stores configuration only. Pins, favorites, and other
  user collections must not be serialized into TOML.
- `cacheState` contains platform cache data only. Do not retain user-data fields
  solely to decode an older gob layout.
- Platform cache data remains gob-backed and rebuildable. Do not move platform
  metadata to SQLite unless a task explicitly owns that migration.
- `OpenAt` opens only the current cache and user-data filenames. It does not
  discover, rename, import, or delete files from older layouts.
- SQLite stores normalized key fields: lower-case `platform`, `mod_id`, and
  `mod_loader`; trimmed `minecraft_version`; empty strings instead of NULL for
  favorite scope fields so unique constraints behave like the old composite
  keys.
- Favorite categories are stored as JSON text in `categories_json`; callers
  still receive `[]string`.

#### 4. Validation & Error Matrix

- Empty cache path -> `OpenAt` returns an error.
- Empty `Runtime.DefaultCacheDir` -> ignore it and continue to
  `storage.DefaultCachePath()`.
- Missing gob cache -> start with an empty platform cache.
- Missing SQLite file -> create parent directory, create schema, then continue.
- SQLite open/schema failure -> `OpenAt` returns an error; callers must not
  silently continue with user-data writes disabled.
- Empty pin key fields -> no row written.
- Favorite mod for a missing list -> no row written.
- Duplicate favorite mod key -> update existing row while preserving ID and
  creation time when caller leaves them empty.

#### 5. Good/Base/Bad Cases

- Good: pin Sodium for `modrinth/sodium/1.21.1/fabric`; it persists in
  `mod-favs.sqlite` and `ResolveVersions` continues to read through
  `storage.GetPinnedMod`.
- Good: change `runtime.cache_dir`; service saves TOML config and reopens
  storage at `<cache_dir>/mod-metadata.tmp` plus `<cache_dir>/mod-favs.sqlite`.
- Good: Wails startup sets `Runtime.DefaultCacheDir` to `os.Getwd()` so an
  unset GUI cache preference resolves to `<pwd>/mod-metadata.tmp`.
- Base: CLI/appcore callers with no cache override and no default cache dir use
  `<os.TempDir()>/mod-downloader/mod-metadata.tmp`.
- Bad: passing the GUI working directory as `Runtime.CacheDir`; that overrides a
  saved `runtime.cache_dir` and ignores the user's explicit preference.
- Bad: writing favorite lists to `mod-metadata.tmp`, because cache version bumps or
  cache deletion would destroy user data.
- Bad: adding pin/favorite fields to `mod-downloader.toml`; TOML is config, not
  collection storage.

#### 6. Tests Required

- SQLite persistence tests for pins and favorite lists/mods across close/reopen.
- Favorite tests for create, rename, delete cascade, duplicate upsert, sort
  order, missing list, and returned-copy behavior.
- Appcore settings test for `runtime.cache_dir` save plus storage reopen.
- Wails adapter test that unset GUI cache preference resolves to
  `<pwd>/mod-metadata.tmp`.
- Wails adapter test that configured `runtime.cache_dir` overrides the GUI
  working-directory default.
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

Wrong:

```go
// GUI default passed as an explicit override; this wins over saved config.
appcore.New(appcore.Options{Runtime: appcore.RuntimeOptions{CacheDir: wd}})
```

Correct:

```go
// GUI default is lower priority than user config/env.
appcore.New(appcore.Options{Runtime: appcore.RuntimeOptions{DefaultCacheDir: wd}})
```

### Scenario: Favorite Lists Persistent Collections

#### 1. Scope / Trigger

Use this pattern for user-owned collections of platform mods, such as named
favorite lists. These records are persistent user data, not local JAR scan
cache or platform metadata cache, so they belong in SQLite (`mod-favs.sqlite`).
Schema changes belong in `core/storage/userdb.go`.

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
storage.UpsertPinnedMod(storage.PinnedMod{Platform: platform, ModID: modID})
```

Correct:

```go
storage.UpsertFavoriteMod(storage.FavoriteMod{
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
    Mods          []storage.FavoriteMod `json:"mods"`
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
func (s *Service) AddFavoriteListReference(parentListID, childListID string) storage.FavoriteListRef
func (s *Service) RemoveFavoriteListReference(parentListID, childListID string) bool
func (s *Service) ListFavoriteListRefs(parentListID string) []storage.FavoriteListRef
func (s *Service) ListFavoriteContents(listID string) storage.FavoriteListContents
```

#### 3. Contracts

- Bulk add deduplicates target list IDs and validates every target list before
  writing rows for that target.
- Copied `FavoriteMod` rows must clear `ID`, `CreatedAt`, and `UpdatedAt`
  before upsert so a copied row never reuses the source row's primary key.
- Duplicate membership in a target list is an update, not a second row.
- Whole-list copy reads `storage.ListFavoriteContents`, so live referenced
  child-list mods are copied concretely into the target list.
- Favorite-list packwiz export also reads `storage.ListFavoriteContents`; the
  archive includes direct mods plus recursively referenced child-list mods,
  using the storage layer's cycle protection and duplicate precedence.
- Whole-list copy from a list to itself is skipped and reported; selected-mod
  bulk add may target the source list because it is idempotent.
- Reference add/remove delegates cycle prevention and persistence to
  `storage.CreateFavoriteListRef` / `DeleteFavoriteListRef`.
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
- Good: Export a root list referencing a child that references a grandchild;
  the packwiz archive contains mods from all three lists once.
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
    storage.UpsertFavoriteMod(mod) // reuses source ID and hides aggregate failures
}
```

Correct:

```go
copied := mod
copied.ID = ""
copied.ListID = targetListID
copied.CreatedAt = 0
copied.UpdatedAt = 0
storage.UpsertFavoriteMod(copied)
```

### Scenario: Favorite List Organization APIs

#### 1. Scope / Trigger

Use this pattern when the UI needs to organize favorite lists with groups,
manual ordering, pinning, or custom icons. SQLite owns persistence in
`core/storage`; app-independent orchestration belongs in `core/appcore`; Wails
methods in `app.go` remain thin delegates for generated frontend bindings.

#### 2. Signatures

Core service methods:

```go
func (s *Service) UpdateFavoriteListMetadata(list storage.FavoriteList) storage.FavoriteList
func (s *Service) ReorderFavoriteLists(ids []string) bool
func (s *Service) ListFavoriteGroups() []storage.FavoriteGroup
func (s *Service) CreateFavoriteGroup(name string) storage.FavoriteGroup
func (s *Service) RenameFavoriteGroup(id, name string) storage.FavoriteGroup
func (s *Service) DeleteFavoriteGroup(id string) bool
func (s *Service) ReorderFavoriteGroups(ids []string) bool
```

Wails adapter methods expose the same signatures and must only call the matching
service method.

#### 3. Contracts

- `UpdateFavoriteListMetadata` edits grouping, icon fields, pinning, and
  `sortOrder`; it does not rename the list.
- Renaming still goes through `RenameFavoriteList` so name validation remains
  separate from display metadata.
- `ListFavoriteLists` returns pinned lists before unpinned lists, preserving
  manual order inside those sections.
- Deleting a favorite group clears `groupId` from assigned lists rather than
  deleting those lists.
- Frontend icon customization stores `iconKind="mdi"` with an MDI value, or
  `iconKind="project"` with a project slug and best-effort `iconUrl`.
- Drag reorder writes ordered ID slices through `ReorderFavoriteLists` or
  `ReorderFavoriteGroups`; the frontend reloads lists after a successful write.

#### 4. Validation & Error Matrix

- Empty group name -> no group is created or renamed.
- Missing favorite list ID in metadata update -> returns an empty list.
- Missing group/list ID during reorder -> ignored by the database reorder loop.
- Delete missing group -> returns `false` and leaves lists unchanged.
- Project slug icon lookup miss -> keep the slug and render a fallback icon.

#### 5. Good/Base/Bad Cases

- Good: UI pins a list by calling `UpdateFavoriteListMetadata` with the existing
  list plus `Pinned=true`, then reloads `ListFavoriteLists`.
- Good: UI deletes a group and reloads lists; former members appear ungrouped.
- Base: UI reorders only the visible pinned section; pinned grouping still keeps
  those lists before unpinned lists.
- Bad: Frontend sorts pinned lists once locally and assumes persistence without
  calling `ReorderFavoriteLists`.
- Bad: Frontend stores group metadata in browser-only state instead of SQLite.

#### 6. Tests Required

- Appcore test for group create/rename/delete and list metadata update.
- Appcore test for group and list reorder returning persisted order through
  `ListFavoriteGroups` / `ListFavoriteLists`.
- Frontend build after Wails binding regeneration.
- Existing Download/Manage add-to-favorite flows must continue to call the
  store `addDrafts` path.

#### 7. Wrong vs Correct

Wrong:

```ts
favoritesStore.lists.sort((a, b) => Number(b.pinned) - Number(a.pinned))
// No backend write; order is lost after reload.
```

Correct:

```ts
await favoritesStore.reorderLists(visibleListIds)
await favoritesStore.loadLists()
```

### Scenario: Favorite Version / Modloader Migration

#### 1. Scope / Trigger

Use this pattern when migrating a favorite list's resolved contents to a
different Minecraft version and modloader. The workflow spans SQLite favorite
contents, cached platform project/version metadata, `core/appcore` service
contracts, Wails bindings, and frontend conflict UI. Preview and apply semantics
belong in `core/appcore`; Wails adapters only delegate.

#### 2. Signatures

Core service request/response types:

```go
type FavoriteMigrationRequest struct {
    SourceListID     string `json:"sourceListId"`
    TargetListID     string `json:"targetListId"`
    MinecraftVersion string `json:"minecraftVersion"`
    ModLoader        string `json:"modLoader"`
    IgnoreConflicts  bool   `json:"ignoreConflicts,omitempty"`
}

type FavoriteMigrationPreview struct {
    SourceListID string                      `json:"sourceListId"`
    TargetListID string                      `json:"targetListId"`
    Matched      []FavoriteMigrationMatch    `json:"matched"`
    Conflicts    []FavoriteMigrationConflict `json:"conflicts"`
    Errors       []string                    `json:"errors,omitempty"`
}

type FavoriteMigrationApplyResult struct {
    Applied bool                        `json:"applied"`
    Preview FavoriteMigrationPreview    `json:"preview"`
    Result  FavoriteBulkOperationResult `json:"result"`
}
```

Core service methods:

```go
func (s *Service) PreviewFavoriteListMigration(req FavoriteMigrationRequest) FavoriteMigrationPreview
func (s *Service) ApplyFavoriteListMigration(req FavoriteMigrationRequest) FavoriteMigrationApplyResult
```

Wails adapter methods must use the same request/result types and delegate
directly to the service.

#### 3. Contracts

- Preview reads `storage.ListFavoriteContents(sourceListID)` so direct rows and
  live referenced child-list rows are resolved the same way as whole-list copy.
- Preview never writes favorite rows.
- Apply always re-runs preview before writing, so stale UI previews cannot be
  applied blindly.
- Project lookup prefers cached SQLite/gob platform metadata by
  `platform/modId`, then `platform/slug`, and only falls back to provider lookup
  if the cache misses.
- Version matching uses `providers.ListMatchingProjectVersions` so existing
  provider cache, sorting, and loader/version filtering rules remain the source
  of truth.
- Matched target rows preserve display metadata and replace only target scope
  fields: `listId`, `minecraftVersion`, `modLoader`, and `versionId`.
- Apply writes through `AddFavoriteModsToLists`, not direct
  `storage.UpsertFavoriteMod`, so copied IDs/timestamps and aggregate
  added/updated/skipped accounting stay consistent with other favorite copy
  workflows.

#### 4. Validation & Error Matrix

- Missing source/target list ID -> preview `Errors`, apply writes nothing.
- Missing Minecraft version or modloader -> preview `Errors`, apply writes
  nothing.
- Project lookup miss -> conflict reason `project not found`.
- No matching target-scope version -> conflict reason
  `matching version not found`.
- Conflicts with `IgnoreConflicts=false` -> apply writes nothing and returns
  `Applied=false`.
- Conflicts with `IgnoreConflicts=true` -> apply writes matched rows and reports
  conflict count as skipped.

#### 5. Good/Base/Bad Cases

- Good: Preview a list with a live child reference; matched/conflict rows include
  resolved child-list entries without duplicating rows.
- Good: Apply a partial migration with `IgnoreConflicts=true`; matching mods are
  upserted into the target list and missing target versions are skipped.
- Base: Preview an all-conflict migration; no rows are written and the UI can
  show conflict reasons before asking the user to ignore.
- Bad: Frontend loops over preview matches and calls `AddFavoriteMod` directly,
  bypassing conflict handling and bulk result accounting.
- Bad: Migration code hand-rolls version filtering instead of using provider
  matching, causing different sort/version semantics from search and install.

#### 6. Tests Required

- Appcore test for all-match preview/apply, asserting preview writes no favorite
  data and apply writes target-scope rows with matched `versionId`.
- Appcore test for partial conflict: apply without ignore writes nothing, apply
  with ignore writes matched rows and skips conflicts.
- Appcore test for all-conflict migration writing nothing.
- Appcore test for missing project lookup producing a conflict instead of a
  panic.
- Appcore test for source contents from a referenced favorite list.

#### 7. Wrong vs Correct

Wrong:

```go
preview := s.PreviewFavoriteListMigration(req)
for _, match := range preview.Matched {
    storage.UpsertFavoriteMod(match.Target) // bypasses ignore-conflict rules and ID reset semantics
}
```

Correct:

```go
preview := s.PreviewFavoriteListMigration(req)
if len(preview.Conflicts) > 0 && !req.IgnoreConflicts {
    return FavoriteMigrationApplyResult{Preview: preview}
}
return s.ApplyFavoriteListMigration(req) // re-previews and writes through bulk copy semantics
```

---

### Scenario: Historical Usage Counters

#### 1. Scope / Trigger

Use this pattern for the home dashboard's lifetime operation totals. The
counters are user-owned history and therefore live in SQLite
(`mod-favs.sqlite`), not in the rebuildable metadata cache, TOML settings, or a
frontend snapshot. A newly created current-schema database starts every counter
at zero. This feature does not authorize scanning or backfilling older data.

#### 2. Signatures

SQLite schema owner: `core/storage/userdb.go`.

```sql
CREATE TABLE IF NOT EXISTS usage_stats (
    key TEXT PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0 CHECK(value >= 0),
    updated_at INTEGER NOT NULL
);
```

```go
type UsageStatKey string

const (
    UsageStatDownloadsCompleted UsageStatKey = "downloads_completed"
    UsageStatModsEnabled        UsageStatKey = "mods_enabled"
    UsageStatModsDisabled       UsageStatKey = "mods_disabled"
    UsageStatModsDeleted        UsageStatKey = "mods_deleted"
    UsageStatFavoritesAdded     UsageStatKey = "favorites_added"
    UsageStatPackwizExports     UsageStatKey = "packwiz_exports"
)

type UsageStats struct {
    DownloadsCompleted int64 `json:"downloadsCompleted"`
    ModsEnabled        int64 `json:"modsEnabled"`
    ModsDisabled       int64 `json:"modsDisabled"`
    ModsDeleted        int64 `json:"modsDeleted"`
    FavoritesAdded     int64 `json:"favoritesAdded"`
    PackwizExports     int64 `json:"packwizExports"`
}

func IncrementUsageStat(key UsageStatKey, delta int64) error
func GetUsageStats() UsageStats
func (s *Service) GetUsageStats() storage.UsageStats
func (a *App) GetUsageStats() storage.UsageStats
```

#### 3. Contracts

- Counters are cumulative successful-object counts across application runs,
  persisted by an atomic SQLite UPSERT (`value = value + delta`).
- `downloads_completed` increments once only when a new mod file reaches its
  final installation path. Already-installed versions, existing target files,
  failures, cancellations, and skipped work do not increment it.
- `mods_enabled`, `mods_disabled`, and `mods_deleted` increment by the number of
  local files whose requested filesystem operation succeeded. Batch requests
  count each successful file, not each button press.
- `favorites_added` increments only for newly inserted favorite memberships.
  Updating an existing membership does not increment it; bulk operations use
  their returned `Added` count.
- `packwiz_exports` increments once after an export ZIP is written
  successfully. Preview, validation failure, and write failure do not count.
- Missing rows read as zero. A fresh database therefore returns a zero-valued
  `UsageStats` without eagerly inserting six rows.
- The homepage reads these counters through `App.GetUsageStats`; it must not
  derive lifetime totals by scanning favorites, local JARs, downloads, logs, or
  caches.
- Do not probe old schemas, backfill existing operations, import logs, scan old
  files, or add compatibility/migration code. Historical counting begins when
  the current schema and counter hooks are present.

#### 4. Validation & Error Matrix

- Unknown `UsageStatKey` -> ignore it and perform no write.
- `delta <= 0` -> ignore it and perform no write.
- SQLite unavailable during a direct increment -> return the storage error.
- Counter persistence fails after a filesystem operation already succeeded ->
  log the storage error; do not undo the completed user operation.
- Counter read fails or storage is not ready -> return zero-valued
  `UsageStats`; the dashboard remains usable.
- Counter row contains an unrecognized key -> ignore it when constructing the
  typed response.

#### 5. Good/Base/Bad Cases

- Good: enabling two disabled JARs and deleting one other JAR adds `2` to
  `mods_enabled` and `1` to `mods_deleted`, then the values survive close/open.
- Good: adding one new favorite and updating one existing favorite adds only
  `1` to `favorites_added`.
- Base: opening a fresh database returns six zero-valued fields and the home
  dashboard shows zero cumulative operations.
- Bad: using the current number of favorite rows as `favorites_added`; deleting
  a favorite would rewrite history.
- Bad: scanning an existing database, local mods, cache, or logs to invent
  totals for installs that predate this feature.

#### 6. Tests Required

- Storage test increments every valid key, closes/reopens SQLite, and asserts
  exact persisted totals.
- Storage test asserts unknown keys and non-positive deltas create no rows and
  change no totals.
- Downloader test asserts a newly installed file emits one completion event
  and rerunning against an existing target emits none.
- Local-mod service test asserts exact successful enable, disable, and delete
  object counts for batch operations.
- Favorite service tests assert only new memberships count, including bulk
  `Added` versus updated rows.
- Packwiz service test asserts one increment only after a successful export.
- Regenerate Wails bindings and run the frontend build after changing
  `UsageStats` fields or `App.GetUsageStats`.
- Do not add legacy-data migration, backfill, or compatibility tests unless a
  separate task explicitly requires that behavior.

#### 7. Wrong vs Correct

Wrong:

```go
// Reconstructs fake history from current state and silently adds migration
// behavior that was never requested.
stats.FavoritesAdded = int64(len(storage.ListFavoriteMods(listID)))
```

Correct:

```go
_, existed := favoriteModKeySet(storage.ListFavoriteMods(mod.ListID))[favoriteModIdentityKey(mod)]
_, written, err := storage.UpsertFavoriteMod(mod)
if err == nil && written && !existed {
    _ = storage.IncrementUsageStat(storage.UsageStatFavoritesAdded, 1)
}
```

---

### Scenario: Minecraft Release Manifest Cache

#### 1. Scope / Trigger

Use this cache for the official Minecraft release ID list loaded during GUI
startup. The upstream `version_manifest_v2.json` is about 272 KB and changes
infrequently; fetching it on every process start adds avoidable network traffic
and makes startup depend on Mojang availability. The parsed release IDs are
rebuildable metadata and belong in `mod-metadata.tmp`.

#### 2. Signatures

```go
type cacheState struct {
    MinecraftReleases   []string
    MinecraftReleasesAt int64
}

func GetMinecraftReleaseVersions() ([]string, int64)
func SetMinecraftReleaseVersions(versions []string, updatedAt int64) error
```

The appcore startup path owns refresh policy:

```go
const minecraftReleaseCacheTTL = 24 * time.Hour
func loadMinecraftReleaseVersions() error
```

#### 3. Contracts

- A non-empty cache timestamped within 24 hours is copied into `global` and
  startup performs no manifest request.
- A missing or expired cache triggers one official manifest request. Successful
  parsed release IDs update both `global` and the metadata cache.
- If refresh fails and any stale cache exists, log a warning, use the stale
  list, and allow startup to continue.
- If refresh fails and no cache exists, return the error because no release
  choices are available.
- Stored values are trimmed, empty values removed, duplicates removed while
  preserving order, and read methods return copies.
- This data is not user-owned and must not be placed in SQLite or TOML.
- The fields are part of the current rebuildable gob schema at
  `cacheVersion=6`. Old caches are discarded; do not add import, backfill, or
  compatibility code.

#### 4. Validation & Error Matrix

- Empty versions or `updatedAt <= 0` -> ignore the cache write.
- Fresh non-empty cache -> zero network calls.
- Expired cache + successful fetch -> replace the cached list and timestamp.
- Expired cache + fetch error -> return stale list and no startup error.
- Empty cache + fetch error -> return the fetch error.
- Cache write failure after successful fetch -> log warning and keep the
  in-memory list usable for the current process.

#### 5. Good/Base/Bad Cases

- Good: restart twice within a day; the second startup reads release IDs from
  `mod-metadata.tmp` and sends no request to Mojang.
- Good: start offline with yesterday's cache; version selection remains usable.
- Base: a fresh cache fetches the manifest once, parses release entries, and
  persists them on cache close.
- Bad: store only in `global`; every process restart downloads the same 272 KB
  manifest.
- Bad: fail the entire app startup when refresh fails despite a valid stale
  list being available.

#### 6. Tests Required

- Storage test trims/deduplicates values, persists across close/reopen, and
  verifies returned slices cannot mutate stored state.
- Appcore test seeds a fresh cache and asserts the fetch function is not called.
- Appcore test seeds an expired cache, returns a fetch error, and asserts stale
  fallback plus nil startup error.
- Appcore test then returns a fresh list and asserts cache contents and timestamp
  are replaced.
- Run full core and app test/vet/build checks after changing startup behavior.

#### 7. Wrong vs Correct

Wrong:

```go
versions, err := minecraft.FetchMinecraftReleaseVersions() // every startup
if err != nil {
    return err
}
```

Correct:

```go
cached, updatedAt := storage.GetMinecraftReleaseVersions()
if len(cached) > 0 && updatedAt >= time.Now().Add(-24*time.Hour).Unix() {
    global.SetMinecraftReleaseVersions(cached)
    return nil
}
```

---

### Scenario: Local Metadata Negative Cache

#### 1. Scope / Trigger

Use this cache when local JAR SHA1/fingerprint enrichment successfully queries
a provider but receives no match. Without a negative cache, every Manage-page
refresh repeats the same bulk requests for unpublished or private mods. This is
rebuildable provider metadata, so it belongs in `mod-metadata.tmp`, not SQLite.

#### 2. Signatures

```go
type cacheState struct {
    LocalMetadataMisses map[localMetadataMissKey]int64
}

func GetLocalMetadataMissCheckedAt(provider, identity string) (int64, bool)
func SetLocalMetadataMisses(provider string, identities []string, checkedAt int64) error
```

Cache key contract:

```text
modrinth:<lowercase SHA1>
curseforge:<decimal uint32 fingerprint>
```

#### 3. Contracts

- A miss is recorded only after that provider request succeeds and omits the
  queried identity. Provider errors and unavailable clients write nothing.
- Miss timestamps use Unix seconds and suppress the same provider/identity for
  24 hours. Expired entries are eligible for a new request.
- Modrinth and CurseForge misses are independent. A Modrinth miss must not
  prevent a CurseForge fingerprint lookup, or vice versa.
- Local metadata resolution is serialized around cache filtering, requests,
  and writes. A concurrent waiter rechecks the cache after the first request
  completes instead of issuing the same request concurrently.
- Successful project/version metadata remains authoritative; a stale negative
  entry never hides a positive cache hit.
- `LocalMetadataMisses` is part of the rebuildable gob cache. Adding it bumps
  `cacheVersion`; old cache data is discarded. Do not write an old-cache
  decoder, importer, backfill, or compatibility branch.

#### 4. Validation & Error Matrix

- Blank provider or identity -> ignore on write and report no hit on read.
- `checkedAt <= 0` -> ignore the write.
- Duplicate/case-varied identities -> normalize and store one entry.
- Recent miss (`checkedAt >= now - 24h`) -> skip only that provider request.
- Expired miss -> perform the provider request again.
- HTTP/decode/provider error -> log the error and leave the identity retryable.
- Cache unavailable -> resolution may continue remotely; log failed miss
  persistence and do not treat it as a provider failure.

#### 5. Good/Base/Bad Cases

- Good: an unpublished JAR queried twice during one day produces one Modrinth
  hash request and one CurseForge fingerprint request total.
- Good: a transient 503 on the first request is retried on the next refresh.
- Base: a cached positive project/version enriches the JAR without consulting
  the negative cache or network.
- Bad: cache one combined "not found" flag when Modrinth misses, thereby
  preventing CurseForge from matching the file.
- Bad: cache errors as misses; a temporary outage would hide metadata for 24
  hours.

#### 6. Tests Required

- Storage test normalizes provider/identity keys and persists timestamps across
  close/reopen.
- Provider test calls an empty successful lookup twice and asserts one HTTP
  request.
- Provider test expires the timestamp and asserts the next call retries.
- Provider test returns a transport error twice and asserts two HTTP requests
  plus no stored miss.
- Run provider/storage race tests and the full core test/vet/build gate.

#### 7. Wrong vs Correct

Wrong:

```go
projects, _ := provider.Resolve(ids)
cacheMisses(allIDs) // network errors and successful matches are both poisoned
```

Correct:

```go
projects, err := provider.Resolve(ids)
if err == nil {
    cacheMisses(unresolved(ids, projects))
}
```

---

### Scenario: Provider Background Tasks And Storage Shutdown

#### 1. Scope / Trigger

Use this lifecycle whenever a provider launches asynchronous cache refreshes
that read or write storage. A bare goroutine can outlive a service and race
`storage.Close`, causing data races and writes against a closing cache.

#### 2. Signatures

```go
func providers.StartBackgroundTasks()
func providers.StopBackgroundTasks()

func (s *Service) Startup(ctx context.Context) error
func (s *Service) Shutdown()
func (s *Service) Close()
```

Internal provider work starts only through:

```go
func startBackgroundTask(task func())
```

#### 3. Contracts

- Provider background metadata refreshes must use `startBackgroundTask`; do not
  use a bare `go refresh...` call.
- The task registry adds to its WaitGroup while holding the same mutex that
  guards its accepting flag. Stop sets accepting false under that mutex before
  waiting, so no `Add` can race `Wait`.
- A separate lifecycle mutex prevents `StartBackgroundTasks` from reopening the
  gate while `StopBackgroundTasks` is still waiting.
- `Service.Startup` opens the accepting gate before any provider API can run.
- `Service.Shutdown` and `Service.Close` stop and drain provider work before
  calling `storage.Close`.
- A task submitted after stop is ignored. The next service startup explicitly
  reopens the gate.
- Task completion must not require the registry mutex; otherwise stop would
  deadlock while waiting.

#### 4. Validation & Error Matrix

- Nil task -> ignore.
- Task registered before stop -> stop waits for completion.
- Task submitted after stop -> reject without starting a goroutine.
- Concurrent stop and registration -> either registration wins and is drained,
  or stop wins and registration is rejected.
- New startup after a completed stop -> reopen and accept tasks.
- Background provider/storage error -> task logs through existing provider
  handling, calls `Done`, and does not block shutdown forever.

#### 5. Good/Base/Bad Cases

- Good: a metadata refresh is reading cache while the window closes; shutdown
  waits, then saves/closes storage.
- Good: a late UI request reaches provider after shutdown began; its refresh is
  not launched.
- Base: no tasks are running, so stop returns immediately.
- Bad: call `go refreshProjectMetadataIfStale(...)`; storage can close while the
  goroutine reads the global cache handle.
- Bad: call `WaitGroup.Wait` while another goroutine may still call `Add`.

#### 6. Tests Required

- Provider test starts blocked work, asserts stop does not return early, then
  releases it and asserts stop completes.
- Provider test submits work after stop and asserts it never runs.
- Appcore race regression runs the favorite migration path that starts a stale
  project refresh, then drains background tasks before storage cleanup.
- Run full providers/appcore race tests plus root/core quality gates.

#### 7. Wrong vs Correct

Wrong:

```go
go refreshProjectMetadataIfStale(provider, platform, project)
storage.Close()
```

Correct:

```go
startBackgroundTask(func() {
    refreshProjectMetadataIfStale(provider, platform, project)
})

providers.StopBackgroundTasks()
storage.Close()
```

---

## Migrations

Platform cache has no incremental migrations. Cache schema changes trigger full
rebuilds via `cacheVersion` bumps (see [Cache Schema Evolution](#cache-schema-evolution)).

### Legacy Data Migration Policy

- Do not implement legacy data migration unless the task explicitly requires
  it in its requirements or acceptance criteria.
- If legacy migration code is not explicitly required, delete it completely in
  the same change. Do not retain dormant migration helpers, fallback branches,
  old-format fields, or migration-only tests for possible future use.
- A filename rename, storage change, schema change, or format change does not
  imply backward-compatible migration.
- Without an explicit migration requirement, current code reads and writes only
  the current format and paths. Older files remain untouched.
- Do not add legacy filename probes, automatic renames, sidecar moves, old
  format importers, decode-only fields, compatibility branches, or migration
  tests proactively.
- If migration is explicitly required, document the exact source versions,
  destination, conflict behavior, failure behavior, cleanup policy, and tests
  before implementing it.

SQLite user-data schema is created idempotently by `userdb.go`. Schema upgrade
steps are added only when the owning task explicitly requires existing SQLite
databases to be upgraded.

### Scenario: SQLite User-Data Schema Evolution

#### 1. Scope / Trigger

Use this pattern only when a task explicitly requires existing
`mod-favs.sqlite` databases to be upgraded after adding a table, index, or
column for user-owned state. A schema change by itself does not authorize an
upgrade path for old databases. Without that explicit requirement, define only
the current schema for fresh databases and do not add legacy schema probes,
backfills, migration steps, or migration tests.

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
- Keep existing favorite persistence tests passing.

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

**Fix**: Bump `cacheVersion` constant in `storage/storage.go`

**Prevention**: Add `cacheVersion` bump to PR checklist when touching `cacheState`

### Mistake: Mixing Domain Concerns in Cache

**Symptom**: Platform API code needs to know about local JAR structure, or vice versa

**Cause**: Storing cross-domain bridging data in the wrong cache (e.g., local→platform associations in platform cache)

**Fix**: Keep domain caches pure. Cross-domain queries go through `modbridge` package.

**Prevention**: Each cache owns one data source. Platform cache = API data only. Local cache = JAR data only. Bridge package handles convergence.
