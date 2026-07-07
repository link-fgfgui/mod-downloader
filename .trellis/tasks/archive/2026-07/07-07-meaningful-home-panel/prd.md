# Add meaningful home panel

## Goal

Replace the bare home page with a useful panel that helps users start the core Mod Downloader workflows from the first screen.

## Confirmed Facts

- `frontend/src/views/Home.vue` currently renders a centered welcome card with no workflow-specific content.
- The app already exposes routes for Home, Download, Manage, Favorites, Unpin, and Settings in `frontend/src/router/index.ts`.
- The sidebar already labels the same destinations through i18n keys in `frontend/src/plugins/i18n.ts`.
- This is a frontend-only UI improvement; no backend API, persistence, or Wails binding changes are required.

## Requirements

- The home page must show a single useful panel instead of the generic welcome card.
- The panel must summarize the app's main workflows: finding/downloading mods, managing local mods, maintaining favorites, reviewing pinned versions, and opening settings.
- The panel must provide direct navigation actions to existing routes.
- The UI must fit the existing Vue + Vuetify style and use existing Material Design icon conventions.
- The content must be localized for both Chinese and English through the existing i18n plugin.
- The layout must be responsive for narrow and desktop viewports without text overlap.

## Out of Scope

- Live statistics, backend data aggregation, or new Wails APIs.
- Changes to sidebar behavior, route definitions, download/search behavior, or local mod management logic.
- New persistent settings or onboarding state.

## Acceptance Criteria

- [x] `frontend/src/views/Home.vue` no longer displays only the generic "Welcome / Mod Downloader" card.
- [x] The home page contains one primary panel with a clear title, concise explanatory copy, and workflow entries for Download, Manage, Favorites, Unpin, and Settings.
- [x] Each workflow entry navigates to its existing route.
- [x] Home page copy is represented in both `zh` and `en` i18n messages.
- [x] The frontend build or type-check passes after the change.
