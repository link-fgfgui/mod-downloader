# Technical Design

## Boundaries

- `Favorites.vue` owns the list context-menu action and converts scoped favorite entries into the existing `models.ModProject` shape.
- `downloadSearch` Pinia store receives a dedicated action for displaying externally supplied results and refreshing their download states.
- Vue Router navigates to the existing `/download` route; no route schema or backend contract changes are required.
- Existing `SearchResultList` and Download-page actions remain the single presentation and interaction path.

## Data Flow

1. User selects “Use as search results” from a favorite list context menu.
2. Favorites view ensures the list contents are loaded for the active Minecraft version/mod loader.
3. Favorite entries are mapped to `models.ModProject`, using the platform-qualified favorite Mod ID as the stable result ID and preserving display metadata.
4. The download-search store replaces its current result snapshot, resets remote-search pagination/loading state, and refreshes `GetDownloadStates` for the active target tuple.
5. The view navigates to `/download`; the existing Download view renders the imported snapshot and its normal actions.

## Compatibility and Trade-offs

- Imported results are a local snapshot, so they do not make a remote search request or expose remote pagination. A subsequent normal text search replaces the snapshot through the existing `runSearch` flow.
- The favorite list's existing scope filtering is authoritative; only items matching the active Minecraft version and loader are imported.
- The action should use the current store state rather than adding a second result-list component or backend endpoint.

## Failure and Empty State

- If the list has no scoped items, the store receives an empty result array and the Download page renders its existing empty state.
- If download-state refresh cannot produce states, existing disabled-state behavior applies; importing results itself must still complete and navigate.
