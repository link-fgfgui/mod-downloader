# Implementation Plan

## 1. Shared Dependency Graph

- Refactor local JAR grouping in `core/appcore/unused_dependencies.go` to retain
  direct providers, JIJ providers, required dependencies, and enabled state.
- Add normalized provider-count and projected-state helpers.
- Update unused-dependency scanning to treat enabled JIJ IDs as providers while
  preserving existing library evidence filtering.
- Add focused scanner tests for equal/different conceptual versions, multiple
  providers, disabled JIJ hosts, and post-delete exclusions.

## 2. Cache-Only Mod ID Lookup

- Add a `core/storage` reverse lookup over cached platform versions by mod ID,
  Minecraft version, and loader.
- Return copied, deterministically sorted candidates with joined cached project
  metadata.
- Add hit, miss, incompatible, reopen, and multiple-hit tests.
- Confirm the lookup performs no provider call or cache write.

## 3. Disable Impact API

- Add shared request/result structs in `core/structs/localmods.go`.
- Implement projected disable/invert analysis in `core/appcore` using the shared
  dependency graph.
- Attach optional cache-resolved restore candidates without making a network
  request.
- Add service tests for single, batch, invert, dependent-also-disabled, and
  cancellation-safe analysis behavior.

## 4. Restore Queue API

- Add a backend method accepting the selected cache candidate identity.
- Re-read and validate the candidate against cache and selected instance before
  constructing a download request.
- Queue through the existing downloader path and return its real queued/skipped
  result.
- Add tests for stale, incompatible, missing, and successful candidates.

## 5. Wails And Frontend

- Forward the analysis and restore methods through `app.go`.
- Regenerate Wails bindings after Go contract changes.
- Add one reusable preflight path in `Manage.vue` for row toggle, batch disable,
  and invert actions that contain disable transitions.
- Add a localized warning dialog with affected mods, cancel, continue, and
  cache-hit restore controls.
- Queue a single logical candidate directly and add a platform/project chooser
  for multiple unrelated candidates.
- Ensure cache misses show no active restore control and never initiate search.
- Preserve current delete and cleanup dialogs.

## 6. Documentation And Quality Gate

- Keep `FAQ.md` aligned with final cache ambiguity behavior.
- Run `gofmt` on changed Go files.
- Run `cd core && go test ./... && go build ./... && go vet ./...`.
- Run Wails binding generation when public methods/types change.
- Run `cd frontend && npm run lint && npm run build`.
- Run root `go test ./... && go build ./... && go vet ./...`.
- Review the final diff for generated binding consistency and unrelated changes.

## Risk And Rollback Points

- Keep graph refactoring behavior-covered before connecting it to operations;
  revert to the existing scanner helper shape if evidence filtering regresses.
- Keep cache lookup read-only so rollback cannot require data migration.
- Keep preflight and mutation as separate calls; a frontend rollback restores
  existing operations without changing file-operation semantics.
