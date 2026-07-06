# Local mod bulk operations design

## Boundaries

- Frontend owns selection UX, confirmation dialogs, loading state, localization, and invoking Wails bindings.
- `app.go` owns Wails adapter methods only.
- `core/appcore` owns validation and file-system mutation so the behavior is testable without Wails runtime imports.
- `core/minecraft` remains the source of truth for `.jar` / `.jar.disabled` interpretation; reuse or mirror its suffix rules through exported helpers if needed.

## Proposed API

Add the request type in `core/structs`:

```go
type LocalModBatchOperationRequest struct {
    Paths  []string `json:"paths"`
    Action string   `json:"action"` // enable | disable | invert | delete
}
```

Expose through Wails:

```go
func (a *App) ApplyLocalModBatchOperation(req appstructs.LocalModBatchOperationRequest) structs.VersionInfo
```

The method returns the refreshed selected version so the frontend can update consistently with `RefreshSelectedVersionMods`.

## File Targeting

- The frontend sends unique file paths from selected groups' `primary.path` values.
- The backend resolves paths against the selected instance's mods directory when paths are relative.
- The backend validates each resolved path stays inside the selected instance's `mods` directory.
- The backend validates the path is either `.jar` or `.jar.disabled`.
- The backend deduplicates paths before mutation.

## Mutation Rules

- `enable`: for each `.jar.disabled`, rename to the same base path ending in `.jar`; leave already enabled `.jar` files unchanged.
- `disable`: for each `.jar`, rename to `.jar.disabled`; leave already disabled `.jar.disabled` files unchanged.
- `invert`: enable disabled files and disable enabled files.
- `delete`: remove each target file.
- If a rename target already exists, return an error instead of overwriting.
- Refresh selected instance mods after the operation attempt so UI state reflects disk state.

## UI Behavior

- Add action bar buttons to `Manage.vue`:
  - Enable or Disable primary action button.
  - Delete button with confirmation dialog.
  - Existing copy and deselect controls remain.
- Right-click on the primary enable/disable button calls `invert` and uses `.prevent`.
- Disable action buttons while a batch operation is in flight.
- Show localized failure feedback when the operation fails.
- Clear selection after successful mutation.

## Primary Action Decision

The primary button labels itself `Disable` when any selected jar is enabled, otherwise `Enable`. Left-click applies that action to all selected jars. Right-click always runs `invert`.

## Compatibility

- Existing scanning behavior remains compatible because `.jar.disabled` already scans as disabled.
- Download/update flows that use local mod indexes will observe refreshed path state after mutation.
- Generated Wails JS/TS bindings must be regenerated after adding the App method.
