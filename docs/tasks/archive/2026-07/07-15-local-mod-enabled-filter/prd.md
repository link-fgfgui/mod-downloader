# Filter local mods by enabled state

## Goal

Let users narrow the local Mod management list by enabled state without losing
the existing text-search workflow.

## Background

- The local Mod list and its search field are owned by
  `frontend/src/views/Manage.vue`.
- Each displayed group exposes its state through `group.primary.enabled`.
- This is UI-only filtering and does not require a backend or Wails contract
  change.

## Requirements

- Add an enabled-state combobox immediately after the local Mod search field.
- Provide the options in this display order: Enabled, Disabled, All.
- Default the filter to All so existing page behavior is preserved.
- Combine the enabled-state filter with the existing text search; a row must
  satisfy both active filters.
- Apply enabled-state changes immediately. Preserve the existing text-search
  debounce behavior.
- Localize all new visible text in Chinese and English.
- Keep the controls usable on narrow layouts without text overflow or overlap.

## Acceptance Criteria

- [x] With All selected, the list contains every group allowed by the current
  text search.
- [x] With Enabled selected, every displayed group has
  `group.primary.enabled === true`.
- [x] With Disabled selected, every displayed group has
  `group.primary.enabled === false`.
- [x] Text search and enabled-state filtering work together in either order.
- [x] The selector appears directly after the search field and remains usable
  on both desktop and narrow page widths.
- [x] Chinese and English labels are present for the selector and all options.
- [x] Frontend lint and production build pass.

## Out Of Scope

- Persisting the selected filter across navigation or application restarts.
- Backend filtering, API changes, or changes to Mod enable/disable operations.

## Notes

This is a lightweight task; no separate design or implementation-plan artifact
is required.
