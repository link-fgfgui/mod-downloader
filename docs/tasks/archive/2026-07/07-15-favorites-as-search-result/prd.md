# Add favorites context menu search result action

## Goal

Allow users to open a favorite list as a ready-to-use set of download search results, so they can continue the existing download-page actions without re-searching each Mod remotely.

## Background

- The Favorites page renders a context menu for each favorite list in `frontend/src/views/Favorites.vue`.
- The Download page renders `SearchResultList.vue` from the `downloadSearch` Pinia store, whose result type is `models.ModProject`.
- Favorite list contents are already filtered by the active Minecraft version and mod loader in `favoritesStore.items`.
- Favorite entries contain the metadata needed to build a `ModProject`-compatible result (`platform`, `modId`, `slug`, `title`, `iconUrl`, `description`, and categories).

## Requirements

1. Add a localized context-menu action titled “作为搜索结果” / “Use as search results” to the favorite-list menu.
2. When invoked, load/use the selected favorite list's currently scoped items and make them the Download page's displayed search results directly; do not issue a remote text search.
3. Preserve each item's platform, project ID, slug, title, icon URL, description, and categories when converting favorite entries to download results.
4. Navigate to the Download page after the action so the standard result-list actions (version inspection, install, add to favorites, batch actions) remain available.
5. Refresh download/install states for the imported results using the current Minecraft target and mod loader, matching normal Download-page behavior.
6. Handle an empty list without errors; the Download page should show its normal empty-results state.
7. Add translations for the new action in both supported locales.

## Acceptance Criteria

- [x] Each favorite-list context menu contains the new localized action.
- [x] Selecting the action shows the selected list's scoped favorite Mods in `SearchResultList` on the Download route without a network search request.
- [x] Imported results retain the source favorite metadata needed by version lookup and download actions, including platform and Mod ID/project ID.
- [x] Download/install button states are refreshed for all imported results and use the active Minecraft version/mod loader.
- [x] Empty favorite lists navigate successfully and render the normal empty-results state.
- [x] Existing favorite menu actions and normal Download-page text search behavior remain unchanged.
- [x] Frontend build and lint pass.

## Out of Scope

- No new backend/Wails API or persistent storage changes.
- No remote search, fuzzy matching, or pagination for the imported list.
