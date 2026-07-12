# Frontend Navigation Map

The Vue app is organized by route-level views, Pinia stores, and reusable
components. Wails-generated bindings under `wailsjs/go/` are the only frontend
API surface for the Go adapter.

## Routes And State

- `views/Download.vue` + `stores/downloadSearch.ts` and `downloadQueue.ts`:
  search, version selection, queueing, progress, retry, and cancellation.
- `views/Manage.vue` + `stores/minecraft.ts`: installed mods, local metadata,
  enable/disable/delete, and dependency analysis.
- `views/Favorites.vue` + `stores/favorites.ts`: scoped lists, entries,
  references, migration, and export.
- `views/Unpin.vue` + `stores/pinnedMods.ts`: pinned version removal.
- `views/Settings.vue` + `stores/settings.ts`: persisted preferences and
  runtime network/provider configuration.
- `views/Home.vue`: navigation and aggregate usage statistics.

When tracing a click, start at the view handler, follow the store action, then
the generated binding into the corresponding method listed in
`../ARCHITECTURE.md`. Keep UI-only state in Vue; domain changes belong in
`core/appcore`.
