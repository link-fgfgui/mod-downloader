# Favorite SQL storage and references design

## Schema

Add schema version 2 in `core/database/userdb.go`.

New `favorite_lists` columns:

- `group_id TEXT NOT NULL DEFAULT ''`
- `icon_kind TEXT NOT NULL DEFAULT ''`
- `icon_value TEXT NOT NULL DEFAULT ''`
- `icon_url TEXT NOT NULL DEFAULT ''`
- `pinned INTEGER NOT NULL DEFAULT 0`

New tables:

```sql
CREATE TABLE IF NOT EXISTS favorite_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    sort_order INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS favorite_list_refs (
    id TEXT PRIMARY KEY,
    parent_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
    child_list_id TEXT NOT NULL REFERENCES favorite_lists(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    sort_order INTEGER NOT NULL,
    UNIQUE(parent_list_id, child_list_id),
    CHECK(parent_list_id <> child_list_id)
);
```

Indexes:

- `favorite_lists(pinned, sort_order, created_at)`
- `favorite_lists(group_id, sort_order, created_at)`
- `favorite_groups(sort_order, created_at)`
- `favorite_list_refs(parent_list_id, sort_order)`
- `favorite_list_refs(child_list_id)`

## Recursive Reference Resolution

Use `WITH RECURSIVE` from the requested root list:

- Anchor row is the root list.
- Recursive rows follow `favorite_list_refs.parent_list_id -> child_list_id`.
- Track `path` and `depth` to avoid infinite loops even if corrupt data exists.
- Join reachable list IDs to `favorite_mods`.
- Prefer depth 0 direct rows when duplicate favorite keys exist.

Cycle check before insert:

- Given proposed `parent -> child`, run a recursive query from `child`.
- If `parent` appears in descendants, reject the insert.

## Go Boundary

Database package owns SQL and returns value copies:

- `CreateFavoriteGroup`
- `UpdateFavoriteGroupName`
- `DeleteFavoriteGroup`
- `ListFavoriteGroups`
- `ReorderFavoriteGroups`
- `UpdateFavoriteListMetadata`
- `ReorderFavoriteLists`
- `CreateFavoriteListRef`
- `DeleteFavoriteListRef`
- `ListFavoriteListRefs`
- `ListFavoriteContents`

Do not put recursive graph logic in frontend or Wails adapter code.
