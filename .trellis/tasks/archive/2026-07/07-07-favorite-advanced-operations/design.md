# Favorite advanced operations design

## Scope

This task extends the existing favorite-list feature across:

- `core/database`: SQLite schema, persistence contracts, and database tests.
- `core/appcore`: adapter-neutral favorite operations and migration preview/apply logic.
- `app.go`: Wails method delegation only.
- `frontend/src/stores/favorites.ts`, `frontend/src/views/Favorites.vue`, and related dialogs/i18n: user workflows and typed Wails calls.

## Data Model

Extend `database.FavoriteList` with list metadata:

```go
type FavoriteList struct {
    ID          string `json:"id"`
    GroupID     string `json:"groupId,omitempty"`
    Name        string `json:"name"`
    IconKind    string `json:"iconKind,omitempty"`    // mdi | project
    IconValue   string `json:"iconValue,omitempty"`   // mdi-* or project slug
    IconURL     string `json:"iconUrl,omitempty"`     // resolved project icon when available
    Pinned      bool   `json:"pinned,omitempty"`
    CreatedAt   int64  `json:"createdAt"`
    UpdatedAt   int64  `json:"updatedAt"`
    SortOrder   int    `json:"sortOrder"`
    System      bool   `json:"system,omitempty"`
}
```

Add new persisted record types:

```go
type FavoriteGroup struct {
    ID        string `json:"id"`
    Name      string `json:"name"`
    CreatedAt int64  `json:"createdAt"`
    UpdatedAt int64  `json:"updatedAt"`
    SortOrder int    `json:"sortOrder"`
}

type FavoriteListRef struct {
    ID           string `json:"id"`
    ParentListID string `json:"parentListId"`
    ChildListID  string `json:"childListId"`
    CreatedAt    int64  `json:"createdAt"`
    UpdatedAt    int64  `json:"updatedAt"`
    SortOrder    int    `json:"sortOrder"`
}
```

The reference table should be the authoritative relationship table. It is not a
serialized list of child IDs on `favorite_lists`, because SQL joins and
recursive CTEs can then resolve the graph directly.

Add richer list rendering/result types in `appcore` or `database`:

```go
type FavoriteListContents struct {
    ListID string `json:"listId"`
    Mods   []FavoriteModEntry `json:"mods"`
    Refs   []database.FavoriteListRef `json:"refs,omitempty"`
}

type FavoriteModEntry struct {
    database.FavoriteMod
    SourceListID   string `json:"sourceListId,omitempty"`
    SourceListName string `json:"sourceListName,omitempty"`
    Referenced     bool   `json:"referenced,omitempty"`
}
```

## SQLite Schema

Keep idempotent `ensureSchema`, then add explicit lightweight migrations because `CREATE TABLE IF NOT EXISTS` does not add columns to existing tables:

- `schema_migrations` remains the applied-version ledger.
- Migration 2:
  - Add `group_id`, `icon_kind`, `icon_value`, `icon_url`, and `pinned` to `favorite_lists`.
  - Create `favorite_groups`.
  - Create `favorite_list_refs` with `parent_list_id` and `child_list_id` foreign keys to `favorite_lists(id)`.
  - Add useful indexes on group/list/order fields and ref parent/child fields.

If implementation shows that `favorite_mods` needs better ordering or
reference performance, add columns rather than overloading existing keys:

- `sort_order INTEGER NOT NULL DEFAULT 0` for item-level drag sorting.
- `source_kind TEXT NOT NULL DEFAULT 'direct'` only if direct mod rows and
  future non-mod row kinds must share one table. The preferred first design is
  separate `favorite_mods` and `favorite_list_refs` tables.

Ordering rules:

- `ListFavoriteGroups`: `sort_order ASC, created_at ASC, name ASC`.
- `ListFavoriteLists`: `pinned DESC, sort_order ASC, created_at ASC, name ASC`.
- Grouped UI can then nest by `groupId`; backend ordering remains deterministic.

Reference rules:

- `parent_list_id != child_list_id`.
- Direct duplicates are rejected by `UNIQUE(parent_list_id, child_list_id)`.
- Before inserting a reference, run a SQLite recursive CTE from the proposed child toward descendants; reject if it can reach the parent.
- Listing contents resolves refs with a SQLite recursive CTE over `favorite_list_refs`, bounded by a visited path/depth guard, then joins the reachable list IDs to `favorite_mods`.
- Dedupe resolved mods by the existing favorite key, preferring direct parent-list rows over referenced rows.
- Removing a reference removes only the ref row, not child list contents.

Example query shape:

```sql
WITH RECURSIVE reachable(list_id, depth, path) AS (
    SELECT ?, 0, ? || ','
    UNION ALL
    SELECT refs.child_list_id, reachable.depth + 1, reachable.path || refs.child_list_id || ','
    FROM favorite_list_refs refs
    JOIN reachable ON refs.parent_list_id = reachable.list_id
    WHERE reachable.depth < ?
      AND instr(reachable.path, refs.child_list_id || ',') = 0
)
SELECT favorite_mods.*
FROM favorite_mods
JOIN reachable ON favorite_mods.list_id = reachable.list_id;
```

Use the same pattern with `SELECT 1` to detect whether a proposed child can
reach the proposed parent before inserting a reference.

## Operations

Database-level operations:

- Group CRUD and reorder.
- Favorite list metadata update for group, pinned, icon, and name.
- Favorite list reorder with a batch request.
- Add/copy mods between lists with duplicate upsert semantics.
- Add/remove/list favorite list refs.
- Resolve list contents including refs.

Service-level operations:

- `AddFavoriteModsToLists(sourceListID string, mods []database.FavoriteMod, targetListIDs []string)`.
- `CopyFavoriteListToList(sourceListID, targetListID string)`.
- `AddFavoriteListReference(parentListID, childListID string)`.
- `PreviewFavoriteListMigration(req FavoriteListMigrationRequest)`.
- `ApplyFavoriteListMigration(req FavoriteListMigrationRequest)`.

Migration preview/apply:

- Read resolved favorite mods from the source list, including live references resolved by the SQL CTE.
- For each mod, reconstruct a `models.ModProject` from stored fields and cache lookup where possible.
- Prefer project lookup by `platform + mod_id`; if that fails and slug exists, fall back to slug lookup.
- Use `providers.ListMatchingProjectVersions` through `appcore` to find target-scope versions.
- A match writes a new/updated `FavoriteMod` with target `minecraftVersion`, target `modLoader`, and matched `versionId`.
- A miss is returned as a conflict row.
- `ignoreConflicts=false` aborts all writes when conflicts exist.
- `ignoreConflicts=true` writes resolvable rows and skips misses.

Icon behavior:

- `iconKind=mdi`: validate non-empty `mdi-` prefix on the frontend, normalize on backend.
- `iconKind=project`: save the slug in `iconValue`; service may resolve `iconURL` from cached/lookup platform project metadata. If unresolved, UI falls back to `mdi-package-variant`.

## Frontend Shape

Favorites page additions:

- Rail groups with nested favorite lists.
- Pinned section at the top.
- Drag handles for groups/lists; persist on drop through batch reorder APIs.
- List menu actions: pin/unpin, icon, group assignment, copy to list, reference into list, migrate, rename, delete.
- Item selection action: add selected to other lists.
- Migration dialog: target version/modloader, preview table, conflict warning, ignore-conflicts checkbox, apply button.
- Icon dialog: segmented `mdi` / `project slug`, text input, preview icon/avatar.

Add-to-favorite dialog:

- Allow selecting multiple target lists for advanced copy workflows.
- Preserve existing single-list behavior for Download and Manage callers.

## Compatibility

- Existing SQLite databases open without manual migration.
- Existing rows receive empty group/icon fields and `pinned=false`.
- Existing Wails methods remain where possible; new methods are additive.
- Public Go method/type additions require regenerated `frontend/wailsjs/go/*` bindings.

## Rollback

- SQLite migrations are additive. Code rollback can ignore new columns/tables, but data written into refs/groups/icons will not be visible in old builds.
- If migration-preview logic proves unreliable, ship all non-migration favorite organization features first and gate migration UI behind the service methods being present.
