# Local mod bulk operations implementation plan

## Checklist

1. Read required specs before coding:
   - `.trellis/spec/guides/index.md`
   - `.trellis/spec/guides/cross-layer-thinking-guide.md`
   - `.trellis/spec/app/backend/index.md`
   - `.trellis/spec/core/backend/index.md`
   - shared backend guideline files referenced by those indexes
2. Add core service request type and `ApplyLocalModBatchOperation` behavior.
3. Add tests for enable, disable, invert, delete, deduplication, missing file, path traversal/outside-root, and rename collision.
4. Add Wails adapter method in `app.go`.
5. Regenerate Wails bindings so `frontend/wailsjs/go/main/App.*` exposes the new method.
6. Update `Manage.vue` to add batch enable/disable/invert/delete controls, loading state, confirmation, success selection clear, and failure feedback.
7. Update `frontend/src/plugins/i18n.ts` with Chinese and English strings.
8. Run formatters and validation.

## Validation Commands

- `go test ./...`
- `(cd core && go test ./...)`
- `(cd frontend && npm run build)`
- Regenerate Wails bindings with the project-standard Wails command before frontend build if Go Wails signatures change.

## Risk Points

- File paths may be relative because scanner can emit paths relative to `.minecraft` when `pathRoot` is set.
- Windows path case and separator handling should use `filepath` helpers.
- Rename collision must not overwrite an existing jar.
- Delete is destructive and must require frontend confirmation.
- Grouped rows can contain multiple declared mods but should map to one unique jar path.

## Rollback

- Revert the new Wails method, core service mutation function/tests, generated bindings, and Manage page action controls together.
- Existing scan-only Local Mod Management behavior should remain intact after rollback.
