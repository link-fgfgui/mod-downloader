# Favorite SQL storage and references

## Goal

Build the SQLite storage foundation for advanced favorites: list metadata, groups, persisted ordering, pinning, icons, and live favorite-list references resolved by SQL.

## Requirements

- Add SQLite migrations that upgrade existing `user-data.sqlite` files without losing current `favorite_lists` or `favorite_mods`.
- Extend favorite list storage with group assignment, icon metadata, pinned state, and stable sort fields.
- Add favorite group storage with CRUD and persisted ordering.
- Add a `favorite_list_refs` relationship table for live list-to-list references.
- Resolve referenced list contents through SQLite recursive CTEs, not frontend recursion.
- Reject reference cycles on insert and guard reads with visited/depth protection.
- Keep existing favorite database APIs working for current callers.

## Acceptance Criteria

- [x] Existing databases with schema version 1 open successfully and receive the new schema.
- [x] Existing favorite lists and mods remain readable after migration.
- [x] Favorite groups can be created, renamed, deleted, listed, and reordered.
- [x] Favorite lists can save group, icon, pinned, and sort metadata.
- [x] Direct list references can be added and removed.
- [x] Self-references and indirect cycles are rejected.
- [x] Listing a favorite list can include live referenced mods using SQL recursive CTE traversal.
- [x] Resolved contents dedupe duplicate mods, preferring direct rows over referenced rows.
- [x] `cd core && go test ./database` passes with new tests.

## Dependencies

- No child-task dependency. This is the first implementation task.
