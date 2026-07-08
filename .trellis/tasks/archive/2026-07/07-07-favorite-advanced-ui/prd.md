# Favorite advanced UI

## Goal

Expose the advanced favorite operations in the Vue/Wails UI with efficient list organization and clear conflict handling.

## Requirements

- Show favorite groups, pinned lists, list icons, and persisted order in the Favorites page rail.
- Support drag reorder for groups and lists.
- Add list actions for pin/unpin, icon change, group assignment, concrete copy, live reference, migration, rename, and delete.
- Add selected-mod action for copying selected favorite mods to other lists.
- Add migration preview/apply dialog with conflict display and ignore-conflicts option.
- Preserve existing add-to-favorites flows from Download and Manage.

## Acceptance Criteria

- [x] Favorites rail renders groups, pinned lists, custom icons, and current selected list correctly.
- [x] Dragging groups/lists persists order and survives reload.
- [x] Selected favorite mods can be added to another list from the Favorites page.
- [x] Whole-list copy and live-reference actions are available from list menus.
- [x] Migration dialog previews matches/conflicts before apply.
- [x] Existing Download/Manage add-to-favorite dialogs still work.
- [x] `npm run build --prefix frontend` passes.

## Dependencies

- Requires the backend APIs from the SQL storage/reference, bulk copy, and migration child tasks.
