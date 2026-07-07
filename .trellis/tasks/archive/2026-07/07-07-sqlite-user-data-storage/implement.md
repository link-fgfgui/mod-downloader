# Implementation Plan

## Steps

1. Add SQLite dependency to `core/go.mod` and top-level `go.mod` as needed through Go tooling.
2. Add SQLite open/schema/migration helpers in `core/database`, keeping `Open`, `OpenAt`, `Close`, and cache path behavior compatible.
3. Move pinned mod functions from gob maps to SQLite queries while preserving normalization and sorting.
4. Move favorite list/mod functions from gob maps to SQLite queries while preserving cascade, duplicate update, sorting, and returned-copy behavior.
5. Keep gob-backed platform cache functions unchanged except for legacy user-data migration and clearing.
6. Extend appcore settings with cache-dir/cache-path view data and save/reopen behavior.
7. Add Wails adapter methods/fields for cache directory selection and reset/save behavior.
8. Update frontend settings store, Settings page, i18n, and generated Wails bindings.
9. Update tests for SQLite persistence, legacy migration, cache-dir resolution/reopen, and settings view behavior.
10. Update README data-file notes if needed.

## Validation Commands

- `go test ./...` from `core/`
- `go build ./... && go vet ./...` from `core/` if dependency/export changes are accepted by the build environment
- `go test ./...` from repo root
- `go build ./...` from repo root
- `npm run build --prefix frontend`

## Risky Files

- `core/database/database.go`
- `core/database/mods.go`
- `core/database/mods_test.go`
- `core/appcore/service.go`
- `app.go`
- `frontend/wailsjs/go/*`
- `frontend/src/stores/settings.ts`
- `frontend/src/views/Settings.vue`
- `frontend/src/plugins/i18n.ts`

## Rollback Points

- If SQLite dependency fails to build cross-platform, keep the public API changes out and revert to the previous gob-only storage before landing.
- If runtime cache-dir reopen introduces instability, keep TOML/env cache-dir resolution and defer live reopen/UI controls to a smaller follow-up.
