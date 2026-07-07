# Favorite advanced operations

## Goal

Add advanced favorite-list operations so users can organize, reuse, migrate, and customize favorite collections without manually rebuilding lists.

## Background

- Current favorite data is user-owned state in SQLite (`user-data.sqlite`) through `core/database/userdb.go`.
- Current `favorite_lists` rows contain `id`, `name`, timestamps, `sort_order`, and `system`.
- Current `favorite_mods` rows contain concrete platform mod membership keyed by `list_id/platform/mod_id/minecraft_version/mod_loader`.
- Existing UI supports creating, renaming, deleting, selecting, refreshing, multi-select removal, and adding mods from Download or Manage views.
- Existing backend can look up projects and matching versions through `appcore.Service.LookupProject*` and `ListMatchingProjectVersions`.

## Requirements

- R1: Users can select multiple mods inside a favorite list and add those selected mods to one or more other favorite lists.
- R2: Users can add/copy one favorite list directly into another favorite list.
- R3: Users can add a favorite-list reference into another favorite list so the target list includes the referenced list dynamically without duplicating all rows.
- R3a: Favorite-list references should be live references resolved through SQLite-backed relationships, not snapshot copies.
- R4: Users can migrate a favorite list from one `minecraftVersion/modLoader` scope to another scope.
- R5: During migration, the app must detect favorite mods that have no matching version in the target scope, show the missing/conflicting mods, and let the user either cancel or ignore conflicts and apply the migration to resolvable mods.
- R6: Users can group favorite lists under named groups.
- R7: Users can drag favorite lists and groups to adjust display order, and the order must persist.
- R8: Users can pin favorite lists so pinned lists appear before unpinned lists while preserving manual order within each section.
- R9: Users can change a favorite list icon by entering either an MDI icon name or a project slug; project slug icons resolve from platform project metadata when available.
- R10: Existing favorite lists and mods must remain readable after the SQLite schema change without user action.
- R11: Existing Download and Manage page "add to favorite" behavior must continue to work.
- R12: The current SQLite storage structure may be changed when it improves reference resolution, ordering, or migration efficiency, as long as existing user data is migrated.

## Acceptance Criteria

- [ ] AC1: From the Favorites page, multi-selected favorite mods can be added to at least one other favorite list; duplicates in a target list update metadata rather than creating duplicate rows.
- [ ] AC2: A whole favorite list can be copied into another list as concrete favorite mod rows.
- [ ] AC3: A whole favorite list can be added to another list as a reference; listing/rendering the target list includes referenced mods and identifies referenced entries clearly enough for removal or inspection.
- [ ] AC4: Reference resolution is implemented through SQLite relationships and recursive queries, with cycles rejected on write and protected against on read.
- [ ] AC5: A migration preview reports target-scope matches and missing/conflicting mods before writes.
- [ ] AC6: Applying migration without `ignoreConflicts` fails when any mod is missing in the target scope.
- [ ] AC7: Applying migration with `ignoreConflicts` writes only resolvable target-scope favorite mods and reports skipped mods.
- [ ] AC8: Favorite groups can be created, renamed, deleted, assigned to lists, reordered, and persisted across app restart.
- [ ] AC9: Favorite lists can be reordered by drag and drop, and the persisted order is reflected by `ListFavoriteLists`.
- [ ] AC10: Pinned lists appear before unpinned lists and remain pinned after restart.
- [ ] AC11: Favorite list icon values can be saved as MDI names or project slug references and are rendered in the Favorites rail.
- [ ] AC12: Existing database tests for favorites still pass, and new tests cover schema migration, references, bulk copy, migration conflict behavior, grouping, ordering, pinning, and icons.
- [ ] AC13: Wails bindings and frontend build are updated when API signatures change.

## Child Tasks

- `07-07-favorite-sql-storage-references`: SQLite schema migration, groups, list metadata, ordering, pinning, icons, and live SQL-backed references.
- `07-07-favorite-bulk-copy-operations`: Backend/appcore/Wails APIs for selected-mod copy, whole-list copy, and reference operations.
- `07-07-favorite-version-migration`: Migration preview/apply for target Minecraft version and modloader with conflict handling.
- `07-07-favorite-advanced-ui`: Favorites page/store/dialog integration for advanced operations.

Implementation order:

1. `07-07-favorite-sql-storage-references`
2. `07-07-favorite-bulk-copy-operations`
3. `07-07-favorite-version-migration`
4. `07-07-favorite-advanced-ui`

## Out Of Scope

- Migrating downloaded/local mod files on disk.
- Syncing favorite data to cloud accounts.
- Moving platform metadata storage from the existing gob cache to SQLite.
- Implementing dependency-aware migration beyond resolving each favorite mod's own matching target version.

## Decisions

- D1: Favorite-list references are live references. Changes to a referenced source list are reflected when reading the parent list.
- D2: The SQLite schema can be reworked for this task, provided existing `favorite_lists` and `favorite_mods` data is migrated without user action.
