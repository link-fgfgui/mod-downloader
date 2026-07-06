# Favorites page playlist management implementation plan

## Checklist

- [x] Keep Favorites separate from old pinned-version compatibility; do not integrate or migrate pinned records.
- [x] Add favorite-list and favorite-item types to `core/database`.
- [x] Extend `cacheState` with favorite-list maps and normalize/intern support.
- [x] Add database CRUD functions and tests for create, rename, delete, list, add item, remove item, persistence, and stable sorting.
- [x] Add `appcore.Service` methods that validate and normalize inputs.
- [x] Expose Wails adapter methods in `app.go`.
- [x] Regenerate Wails frontend bindings.
- [x] Add a Pinia favorites store for lists, selected list, items, loading states, and pending mutations.
- [x] Add reusable add-to-favorites dialog/menu logic that can be opened from both search results and Manage rows.
- [x] Add Download search result actions for adding one or more results to a selected favorite list.
- [x] Add Manage row/bulk actions for adding only online-resolved local mods to a selected favorite list.
- [x] Disable Manage add-to-favorites actions when any selected local mod lacks valid `onlinePlatform` and `onlineProjectId`.
- [x] Refactor local mod scan/enrichment so `SelectVersion()` and `RefreshSelectedVersionMods()` share the same cached enrichment, async hash resolution, state update, and `selected-version-changed` emission behavior.
- [x] Verify first-load Manage lifecycle does not miss async metadata update events from the enrichment callback.
- [x] Add `Favorites.vue` using `VirtualList` for selected-list items.
- [x] Update router and sidebar navigation.
- [x] Add zh/en i18n strings.
- [x] Keep `/unpin` as the pinned-version management page, unless later explicitly removed in a separate task.
- [x] Run backend and frontend validation.

## Validation Commands

- `go test ./...`
- `cd core && go test ./...`
- `cd frontend && npm run build`

## Risky Files

- `core/database/database.go`
- `core/database/mods.go`
- `core/appcore/service.go`
- `app.go`
- `frontend/wailsjs/go/main/App.d.ts`
- `frontend/wailsjs/go/main/App.js`
- `frontend/wailsjs/go/models.ts`
- `frontend/src/router/index.ts`
- `frontend/src/components/SideBar/SideBar.vue`
- `frontend/src/components/SearchResultList.vue`
- `frontend/src/views/Download.vue`
- `frontend/src/views/Manage.vue`
- `frontend/src/views/Unpin.vue`
- `frontend/src/views/Favorites.vue`
- `frontend/src/plugins/i18n.ts`

## Review Gate Before Start

- PRD has no blocking open questions.
- Design confirms Favorites and pinned-version management are separate.
- Implementation scope keeps `/unpin` separate from the new Favorites page.
