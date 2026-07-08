# Unused dependency cleanup implementation plan

## Checklist

1. Parser model
   - Add local dependency struct/field to `core/structs/minecraft`.
   - Update Fabric parser to read required dependency IDs from `fabric.mod.json`.
   - Update Forge/NeoForge TOML parser to read mandatory/required dependencies.
   - Filter pseudo-dependencies and unresolved placeholders.
   - Add parser tests for required, optional, pseudo, multi-mod, and JiJ cases.

2. Core scan service
   - Add request/result structs in `core/structs` or `core/appcore` following existing API ownership patterns.
   - Implement selected-instance scan in `core/appcore`.
   - Reuse current refresh/enrichment path so online categories/tags are available.
   - Build dependency graph from enabled, non-excluded top-level files.
   - Classify candidates conservatively and include evidence.
   - Add service tests with temporary mod fixtures.

3. Settings preference
   - Add config field and normalization helper in `core/configs`.
   - Expose normalized value in `appcore.SettingsView` and `app.go.SettingsView`.
   - Add save method/request if needed for the toggle.
   - Add config/appcore tests for default and saved values.

4. Wails adapter and bindings
   - Add `App.ScanUnusedDependencies` or equivalent delegating method.
   - Add settings save adapter if needed.
   - Run Wails binding generation after public API changes.

5. Frontend state and UI
   - Update `frontend/src/stores/settings.ts` for the new toggle.
   - Update `frontend/src/views/Settings.vue` with a switch.
   - Update `frontend/src/views/Manage.vue` with manual scan action, loading state, scan result dialog, delete-triggered scan, and candidate cleanup confirmation.
   - Add Chinese and English i18n keys.

6. Verification
   - Run focused parser/appcore tests while implementing.
   - Run `go test ./...` from `core/`.
   - Run `go test ./...` from repo root.
   - Run frontend build/type-check command from `frontend/`.
   - Check generated Wails bindings are committed with source changes.

## Validation Commands

```bash
cd core && go test ./...
go test ./...
cd frontend && npm run build
```

If Go API signatures change, run the repository's Wails binding generation command before frontend build.

## Risk Points

- Loader dependency schemas differ. Keep parser rules narrow and test the exact required-dependency forms used by Fabric, Forge, and NeoForge.
- False-positive cleanup suggestions are worse than missed cleanup opportunities. Candidate classification must require explicit library/dependency evidence.
- Do not bypass `ApplyLocalModBatchOperation` for cleanup deletion; it already owns selected-instance path validation.
- Wails binding drift can break frontend builds; regenerate bindings after adding methods or structs.
- Disabled mods do not count as dependents; implement scans against enabled top-level mods only.

## Review Gate Before Start

- Verify PRD, design, and implementation plan still match after any user answer.
- Only then run `task.py start` and move into Phase 2.
