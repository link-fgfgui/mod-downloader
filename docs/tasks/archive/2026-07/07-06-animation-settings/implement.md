# Configure animation behavior implementation plan

## Checklist

- [x] Add config defaults/normalization helpers and preference fields in `core/configs`.
- [x] Extend appcore preference/settings view types and add save method for animation settings.
- [x] Extend Wails adapter types and methods in `app.go`.
- [x] Update Go tests for config loading/normalization and save behavior.
- [x] Regenerate or update Wails frontend bindings.
- [x] Add frontend animation application helper.
- [x] Apply animation settings on app startup.
- [x] Extend settings store and Settings page controls.
- [x] Add i18n labels/messages.
- [x] Update animation CSS for disabled-animation behavior.
- [x] Run verification commands.

## Validation

- `go test ./...`
- `go test ./...` from `core/`
- `npm run build --prefix frontend`

Run `go build ./...` if Wails adapter signatures/imports make it necessary after tests.

## Rollback Points

- Config schema changes are isolated to `core/configs/structs.go` and appcore settings methods.
- Frontend animation behavior is isolated to the new helper plus `animations.css`.
- If Wails code generation is unavailable, update checked-in bindings manually and verify with frontend build.
