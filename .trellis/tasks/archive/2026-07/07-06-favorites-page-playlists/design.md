# Favorites page playlist management design

## Architecture

Add a new favorite-list data model in `core/database` instead of overloading the existing `PinnedMod` key. Existing pins are a download-version override mechanism; favorite lists are a user collection mechanism with list names and membership.

Expose favorite-list operations through `core/appcore.Service`, `app.go`, and generated Wails bindings:

- list favorite lists
- create favorite list
- rename favorite list
- delete favorite list
- list items in a favorite list
- add favorite item
- remove favorite item
- move or copy items between lists only if included in the final scope

Frontend owns page composition in a new `frontend/src/views/Favorites.vue` plus a Pinia store, likely `frontend/src/stores/favorites.ts`.

Add-to-favorites entry points exist in both:

- `Download.vue` / `SearchResultList.vue`, using search result `models.ModProject` identity directly
- `Manage.vue`, using local `structs.ModInfo` only after online metadata enrichment has produced a valid `onlinePlatform` and `onlineProjectId`

## Data Model

Proposed records:

- `FavoriteList`: `id`, `name`, `createdAt`, `updatedAt`, `sortOrder`, optional `system`
- `FavoriteMod`: `id`, `listId`, `platform`, `modId`, `versionId`, `minecraftVersion`, `modLoader`, optional display fields copied from cached project metadata

The unique key for list membership should be `listId/platform/modId/minecraftVersion/modLoader`, with `versionId` treated as optional item metadata. This allows the same mod to exist in different lists and preserves current version-scope behavior.

For Download-origin items, derive `platform` and `modId` from `models.ModProject`. For Manage-origin items, derive `platform` and `modId` from `ModInfo.onlinePlatform` and `ModInfo.onlineProjectId`; unresolved local jar metadata (`id`, `name`, file path, or SHA1 alone) is not sufficient to create a favorite item.

## Compatibility

Keep `PinnedMod` APIs unchanged for download behavior. Favorites must be independent from pinned-version records:

- do not migrate existing `PinnedMod` data into favorite lists
- do not show pinned records as a built-in favorite list
- keep `/unpin` and the current `pinnedMods` store responsible for pinned-version override management
- build Favorites as independent collections backed by their own database records and Wails APIs

## Frontend UX

The Favorites page should use a work-focused playlist layout:

- left rail: favorite lists with create/rename/delete controls
- main panel: selected list header and actions
- main list: `VirtualList` rows for favorite mods
- empty states for no lists, no selected list, and selected list with no mods

Rows should follow existing mod-list visual language from `SearchResultList.vue` and `Manage.vue`, with icons, platform/version chips, and compact actions.

Manage should expose an add-to-favorites bulk/action button through `VirtualList` selection. The action is disabled when the selected row or selection contains any unresolved local mod. Mixed selections must disable the whole action rather than adding only the valid subset. A tooltip or disabled-state copy should make the reason clear without allowing invalid favorites to be persisted.

## Local Metadata Refresh

Reuse the existing SHA1-to-online-project enrichment path used by Manage icons. The implementation should remove the need for repeated manual refreshes by centralizing refresh/enrichment behavior so both selected-version changes and explicit refreshes:

- scan local jars
- apply cached online metadata synchronously when available
- asynchronously resolve missing hashes
- update selected-version state
- emit `selected-version-changed` after async metadata changes

The current evidence suggests `RefreshSelectedVersionMods()` has the async callback path, while `SelectVersion()` does not run the same enrichment path. First-load listener registration ordering should also be checked so the frontend does not miss the async update event.

## Boundaries

- Wails runtime imports stay in `app.go`.
- Reusable domain/storage logic belongs under `core/`.
- Generated `frontend/wailsjs` files must be regenerated after public Wails API changes.
- i18n strings must be added in both `zh` and `en`.

## Risks

- Gob cache schema changes require `cacheState.normalize()` compatibility and tests for old data.
- Overloading `PinnedMod` would couple UI collections to download resolver behavior; the chosen plan avoids that coupling.
- `VirtualList` selection is index-based, so stores should replace list arrays carefully and let `VirtualList` clear stale selections when the selected list changes.
- Manage-origin favorite eligibility depends on online metadata availability; avoid falling back to local jar IDs because those may not map to a real online project.
