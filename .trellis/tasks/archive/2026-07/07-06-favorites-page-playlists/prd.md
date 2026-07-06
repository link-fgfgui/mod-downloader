# Favorites page playlist management

## Goal

Add a dedicated Favorites page that lets users manage multiple named mod favorite lists, similar to NetEase Cloud Music playlist management: users can create lists, select a list, see the mods in that list, and manage list membership from a dense virtualized mod list.

The feature should add a user-facing collection workflow while keeping the existing pinned-version management separate. Pinned versions remain a download-version override mechanism; Favorites are independent user-created mod collections.

## Background And Evidence

- The app is a Wails + Vue 3 + Vuetify desktop app with routes in `frontend/src/router/index.ts` and navigation in `frontend/src/components/SideBar/SideBar.vue`.
- The shared virtualized list component is `frontend/src/components/VirtualList.vue`; it already supports selectable rows, a floating bulk action bar, infinite-load footer hooks, and row slots.
- Search results already reuse `VirtualList` through `frontend/src/components/SearchResultList.vue`.
- Local installed mods already reuse `VirtualList` directly in `frontend/src/views/Manage.vue`.
- The current `/unpin` page in `frontend/src/views/Unpin.vue` manages a single flat set of pinned mod versions with `v-data-table`; it does not support multiple named lists.
- The current Pin backend stores `database.PinnedMod` records keyed by `platform`, `modId`, `minecraftVersion`, and `modLoader` in `core/database/mods.go`; there is no playlist/folder/list ID, list name, list order, or mod order.
- Existing pinned-version APIs are exposed through `app.go` as `GetPinnedModVersion`, `PinModVersion`, `ListPinnedMods`, and `UnpinMod`, and are consumed by `frontend/src/stores/downloadSearch.ts` and `frontend/src/stores/pinnedMods.ts`.
- Pinned versions currently influence download selection in `core/modbridge/modbridge.go`; favorite lists must not accidentally remove that behavior unless explicitly planned.
- Local mod online display metadata already exists on `structs.ModInfo` as `onlineName`, `onlinePlatform`, `onlineProjectId`, `onlineSlug`, `iconUrl`, and `categories`, and `Manage.vue` uses `iconUrl` for the displayed icon.
- `core/appcore.Service.RefreshSelectedVersionMods()` enriches local mods from cached SHA1 metadata, emits `selected-version-changed`, then asynchronously resolves missing hashes through `providers.ResolveProjectsByHashes()` and emits `selected-version-changed` again when online metadata changes.
- `core/appcore.Service.SelectVersion()` currently refreshes local mods and emits `selected-version-changed`, but does not run the same online metadata enrichment path as `RefreshSelectedVersionMods()`.
- The frontend `minecraft` store listens for `selected-version-changed`, but first-load lifecycle ordering may allow `Manage.vue` to call `RefreshSelectedVersionMods()` before the store listener is registered, which can explain why icons sometimes require one or two manual refreshes.
- User-facing text is centralized in `frontend/src/plugins/i18n.ts` for Chinese and English.

## Requirements

- R1. Add a Favorites page reachable from the sidebar and router.
- R2. Users can create multiple named favorite lists.
- R3. Users can select a favorite list from a left-side or playlist-style list selector and see that list's mods.
- R4. The mod list inside the selected favorite list must reuse `VirtualList`, not `v-data-table`.
- R5. The Favorites page should support common playlist-like management actions: create list, rename list, delete list, remove mods from the selected list, and bulk actions through `VirtualList` selection.
- R6. Favorite list items must preserve enough mod identity to download/inspect a mod later: platform, mod/project ID, optional pinned version ID, Minecraft version, and mod loader.
- R7. Existing pinned-version behavior must remain compatible with current download flows.
- R8. Favorite lists must not integrate or migrate existing pinned-version records; `/unpin` remains responsible for managing pinned download-version overrides.
- R9. Add Chinese and English i18n strings for new nav/page/actions/errors.
- R10. Add backend tests for favorite list persistence and core service behavior.
- R11. Add focused frontend validation for store/list behavior if the repository test setup supports it; otherwise document manual validation.
- R12. Users can add mods to a favorite list from Download search results.
- R13. Users can add mods to a favorite list from Manage local mod rows only when the local mod resolves to valid online metadata.
- R14. Manage's add-to-favorites action must be disabled for invalid/unresolved local JARs.
- R15. Reuse the existing local-mod online metadata enrichment path for Manage favorite eligibility and fix the delayed icon/metadata refresh behavior so users do not need one or two manual refreshes before resolved icons appear.
- R16. In Manage bulk selection, if any selected row lacks valid online metadata, the entire add-to-favorites action is disabled; do not silently add a valid subset.

## Acceptance Criteria

- [ ] Sidebar includes a Favorites entry that opens the new Favorites page.
- [ ] A user can create at least two favorite lists with distinct names and switch between them without losing list contents.
- [ ] The selected favorite list renders its mods through `VirtualList` and supports row selection and bulk clear/remove behavior.
- [ ] Deleting a list requires confirmation and removes only that list's membership, not unrelated lists or existing pinned-version state.
- [ ] Renaming a list persists after reload.
- [ ] Download search result rows can be added to a selected favorite list.
- [ ] Manage rows with valid `onlinePlatform` and `onlineProjectId` can be added to a selected favorite list.
- [ ] Manage rows without valid online metadata show a disabled add-to-favorites action when selected.
- [ ] Manage bulk selection disables the add-to-favorites action when at least one selected row lacks valid online metadata.
- [ ] Manage local mod icons and online metadata update automatically after hash resolution without requiring repeated manual refreshes.
- [ ] Existing pinned-version download behavior remains covered by tests and still passes.
- [ ] Existing pinned records remain manageable through the existing pinned/unpin flow and are not shown as favorite-list contents.
- [ ] `go test ./...` passes in the app repo after Wails adapter changes.
- [ ] `go test ./...` passes in `core/` after database/service changes.
- [ ] Frontend build or type check passes after new Wails bindings are generated.

## Out of Scope

- Cloud sync across devices.
- Sharing/exporting favorite lists.
- Drag-and-drop reordering unless it falls out naturally from the chosen data model.
- Replacing the download search experience.
- Removing the pinned-version concept from the download resolver.
- Showing existing pinned records as a built-in default favorite list.
- Adding unresolved local JARs to favorite lists.
