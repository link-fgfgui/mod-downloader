# Implementation Plan

1. Add `configs.Language`, default/normalization behavior, config field, and
   parser/config tests in `core/`.
2. Extend `appcore` preferences/settings contracts and implement persisted
   `SaveLanguage` with focused service tests.
3. Extend `app.go` Wails contracts, adapter method, settings mapping, and native
   dialog localization; regenerate Wails frontend bindings.
4. Add frontend language helpers that resolve system language and activate
   vue-i18n before mount.
5. Extend the settings store and Settings view with an auto-saving language
   selector and immediate locale application.
6. Audit Vue/TypeScript user-facing literals, add aligned `zh`/`en` catalog
   keys, and route native dialog calls through the resolved locale.
7. Run formatting, generated-binding checks, focused tests, full Go tests,
   frontend lint, and frontend production build. Re-audit hard-coded UI English
   after the edits.

## Validation Commands

```bash
cd core && go test ./configs ./appcore && go test ./...
go test ./...
cd frontend && npm run lint && npm run build
```

Run `wails generate module` after changing public `App` methods and payloads.

## Risk And Rollback Points

- Preserve the user's current edits in `i18n.ts` and `Settings.vue` while
  modifying those files.
- Mount timing is critical: locale initialization must finish before Vue mounts.
- Generated Wails files must match Go signatures; do not hand-edit them unless
  generation is unavailable and that limitation is reported.
- The `core/` directory is a submodule; app and core diffs must both be reviewed.
