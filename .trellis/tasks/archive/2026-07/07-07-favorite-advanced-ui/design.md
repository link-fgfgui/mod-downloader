# Favorite advanced UI design

## Store

Extend `frontend/src/stores/favorites.ts` with:

- groups state and load/reorder actions
- list metadata update actions
- reference actions
- bulk selected-mod copy action
- whole-list copy action
- migration preview/apply actions
- contents loading that can include referenced entries

## Favorites View

Keep the app as an operational tool:

- Dense rail with grouped lists and small icon buttons.
- Drag handles as icon buttons.
- Menus for secondary list actions.
- Dialogs for icon, group, copy/reference, and migration.
- No landing/marketing content.

## Migration Dialog

- Target Minecraft version and modloader controls.
- Preview button or automatic preview after valid target.
- Table/list showing matched rows and conflict rows.
- Disabled apply when conflicts exist unless `ignoreConflicts` is checked.

## Icons

- `mdi` mode renders a `v-icon`.
- `project` mode renders `iconUrl` when resolved, otherwise fallback `mdi-package-variant`.
