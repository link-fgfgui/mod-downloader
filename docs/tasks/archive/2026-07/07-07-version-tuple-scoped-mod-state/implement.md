# Implementation Plan

## Checklist

- [x] Extend `minecraftStore` with active `selectedMinecraftVersion`, `selectedModLoader`, modloader list, and manual tuple setters that clear `selectedVersion`.
- [x] Update `VersionChoose.vue` to render launcher version, MC version, and modloader controls in the sidebar.
- [x] Remove duplicate MC version/modloader controls from `Download.vue`.
- [x] Synchronize `downloadSearch` tuple state from `minecraftStore` and block state refresh/search/install when tuple data is incomplete where needed.
- [x] Update Manage favorite drafts to use the active tuple consistently.
- [x] Confirm pinned/favorite stores keep full tuple keys and adjust display/filter behavior only if a leak is found.
- [x] Run frontend build/type check.

## Validation

- `npm run build --prefix frontend`
- If frontend build exposes generated binding drift, inspect whether Wails APIs changed; expected result is no binding regeneration needed.

## Rollback Points

- Sidebar state changes are isolated to `minecraftStore` and `VersionChoose.vue`.
- Download page selector removal is isolated to `Download.vue`.
- Search/install tuple use is isolated to `downloadSearch.ts`.
