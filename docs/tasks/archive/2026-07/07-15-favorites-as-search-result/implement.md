# Implementation Plan

1. [x] Read the frontend pre-development and cross-layer/state guidance, then inspect the current Favorites and Download store contracts.
2. [x] Add a `showResults`/`setResults`-style action to `downloadSearch` that accepts `models.ModProject[]`, resets search-only state, and refreshes download states.
3. [x] Add the Favorites context-menu item, conversion helper, route navigation, and localized labels in `zh` and `en`.
4. [x] Verify empty lists, scoped filtering, metadata mapping, and state refresh behavior through focused tests or type-safe code paths.
5. [x] Run `npm run lint` and `npm run build` from `frontend/`; review the diff for unrelated changes.
