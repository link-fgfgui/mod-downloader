# Local mod bulk operations

## Goal

Local Mod Management should support practical multi-select file operations so users can enable, disable, invert enabled state, and delete selected local mod jars without leaving the app.

## Background

- `frontend/src/views/Manage.vue` already renders grouped local mods through `VirtualList` and exposes multi-select actions for copying names and IDs.
- `frontend/src/components/VirtualList.vue` already supports single selection, Ctrl/Cmd multi-select, Shift range select, Ctrl/Cmd+A select all, Escape clear, selected item projection, and a floating action bar.
- `core/minecraft/modparser.go` already treats both `.jar` and `.jar.disabled` as mod jars, strips either suffix for display, and derives `enabled` from the file name.
- `core/appcore/service.go` currently exposes local mod scanning and selected-instance refresh, but no local mod file mutation API.
- `app.go` is the Wails adapter layer. Reusable local mod mutation logic belongs in `core/appcore`, with Wails binding methods in `app.go`.

## Requirements

- Add batch enable and disable operations for selected local mod groups in the Manage page.
- Add an alternate right-click behavior on the enable/disable batch button that inverts each selected jar's enabled state.
- Add a batch delete operation for selected local mod groups.
- The left-click enable/disable primary action must show `Disable` and disable all selected jars when any selected jar is currently enabled; it must show `Enable` and enable all selected jars only when every selected jar is disabled.
- Operations must target the selected local instance only and mutate the actual jar files represented by selected rows.
- Enabling a selected disabled jar must rename `*.jar.disabled` to `*.jar`; disabling a selected enabled jar must rename `*.jar` to `*.jar.disabled`.
- Invert must enable disabled selected jars and disable enabled selected jars in one batch.
- Delete must remove selected jar files from disk after user confirmation.
- After any successful mutation, the selected instance's mod list must refresh and the multi-select state must clear.
- UI text must be localized in both Chinese and English.
- Batch operations must handle grouped rows that represent multiple declared mods in one jar without duplicating file operations.
- Failed operations must not silently pretend success; the user should see an error state or message and the list should be refreshed to reflect disk state.

## Acceptance Criteria

- [ ] Selecting one or more rows in Local Mod Management shows batch controls for copy names, copy IDs, enable/disable, delete, and deselect.
- [ ] Left-clicking the enable/disable batch control disables all selected jars when any selected jar is enabled, and enables all selected jars when every selected jar is disabled.
- [ ] Right-clicking the enable/disable batch control inverts enabled state for each selected file and suppresses the browser context menu.
- [ ] Enabled selected jars are disabled by renaming from `.jar` to `.jar.disabled`; disabled selected jars are enabled by renaming from `.jar.disabled` to `.jar`.
- [ ] Delete asks for confirmation before removing files; confirming deletes selected jar files and cancelling leaves disk unchanged.
- [ ] Grouped rows perform one disk operation per unique file path even when a jar declares multiple mods.
- [ ] After successful enable, disable, invert, or delete, the Manage page refreshes from the selected instance and clears selection.
- [ ] If a selected file is missing, outside the selected instance's mods directory, or would collide with an existing target file name, the operation returns an error and does not partially corrupt unrelated files.
- [ ] `go test ./...` passes in the app repo after Wails adapter changes.
- [ ] `go test ./...` passes in `core/` after core service changes.
- [ ] `npm run build` passes in `frontend/` after frontend changes and generated bindings are updated if Wails signatures change.

## Out of Scope

- Undo/restore for deleted files.
- Moving deleted files to trash instead of permanent deletion.
- Per-row enable/disable buttons outside the multi-select action bar.
- Editing mod metadata or changing how local mods are grouped.
