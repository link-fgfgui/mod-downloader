# Favorite advanced operations implementation plan

## Checklist

1. Implement child tasks in this order:
   - `07-07-favorite-sql-storage-references`
   - `07-07-favorite-bulk-copy-operations`
   - `07-07-favorite-version-migration`
   - `07-07-favorite-advanced-ui`
2. Read current favorite specs and source files before editing:
   - `.trellis/spec/core/backend/index.md`
   - `.trellis/spec/app/backend/index.md`
   - `.trellis/spec/backend/database-guidelines.md`
   - `core/database/userdb.go`
   - `core/database/mods.go`
   - `core/appcore/service.go`
   - `frontend/src/stores/favorites.ts`
   - `frontend/src/views/Favorites.vue`
3. Extend database types and normalization helpers for groups, list metadata, refs, and migration result/request structs.
4. Add SQLite migration helpers in `core/database/userdb.go` for schema version 2; cover existing DB open/upgrade.
5. Implement database operations:
   - group CRUD/reorder
   - list metadata update/reorder/pin/icon/group assignment
   - bulk favorite mod copy
   - list refs with SQL recursive CTE cycle prevention
   - list contents resolution with SQL recursive CTE traversal and ref dedupe
6. Add database tests for schema upgrade, group ordering, list ordering/pin, icon metadata, copy semantics, ref cycle rejection, ref contents resolution, and cascade behavior.
7. Add appcore operations and tests for bulk add, list copy, list refs, migration preview/apply with conflict handling, and additive existing workflow compatibility.
8. Add Wails adapter methods in `app.go`; regenerate frontend bindings.
9. Extend Pinia favorites store with groups, contents, bulk copy, refs, ordering, pin/icon, and migration actions.
10. Update `Favorites.vue` and supporting dialogs/i18n for advanced operations.
11. Verify UI flows manually through the dev server when frontend changes are complete.

## Validation Commands

- `go test ./...`
- `cd core && go test ./...`
- `cd core && go build ./... && go vet ./...`
- `npm run build --prefix frontend`
- `wails generate module` when Wails API signatures change, then rerun frontend build.

## Risk Points

- Existing SQLite users need real column migrations; `CREATE TABLE IF NOT EXISTS` alone is insufficient.
- Reference resolution must guard against cycles and duplicate mods at both write time and read time; use SQLite CTEs rather than ad hoc frontend recursion.
- Migration preview may call remote provider APIs; UI needs loading/error states and should not write until preview succeeds.
- Wails generated bindings may drift if Go method names or exported structs change without regeneration.
- Frontend drag-and-drop should use stable list/group dimensions to avoid layout shifts.

## Rollback Points

- Land and test database migrations before frontend UI work.
- Keep new Wails methods additive so existing UI behavior can remain functional while advanced UI is incomplete.
- If migration apply is unstable, keep preview/apply methods unexposed in UI until tests are green.
