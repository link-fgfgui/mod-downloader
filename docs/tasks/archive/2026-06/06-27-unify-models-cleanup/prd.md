# PRD: Unify on models package — remove legacy data structures

## Background

The codebase recently consolidated CF/MR metadata types into the `models` package (`ModProject`, `ModVersion`, `ModDependency`). Two transition layers remain:

1. `structs/search.go` defines type aliases `SearchModResult = models.ModProject`, `ProjectVersionResult = models.ModVersion`, `ProjectDependency = models.ModDependency`.
2. `providers/model.go` re-exports the same types plus `ProjectKey`/`ParseProjectKey`/`VersionKey`/`ParseVersionKey` as package-level identifiers.
3. `providers/modprovider.go` keeps two parallel sets of SDK→struct conversion functions. The "old path" (`modToSearchResult`, `fileToProjectVersionResult`, `versionToProjectVersionResult`, `searchHitToSearchResult`, `projectToSearchResult`, `setFileResultFields`, `dependenciesFromVersion`, `dependenciesFromFile`) is wired into the active Search/ListVersions flow. The "new path" (`modToModProject`, `fileToModVersion`, `versionToModVersion`, `searchHitToModProject`, `projectToModProject`, `setModVersionFileFields`, `dependenciesFromVersionToModDeps`, `dependenciesFromFileToModDeps`) is dead code that is never called.

`providers/bridge.go` (old↔new converter) was already deleted in a prior pass.

## Problem

- The old conversion functions produce values missing the `ProjectID` field (e.g. `modToSearchResult` at `providers/modprovider.go:323` sets `ID`, `Platform`, `Slug` but never `ProjectID`). Because `ModProject.ProjectID` is not `omitempty`, the frontend receives `"projectId":""`. The new functions set `ProjectID` correctly.
- Two parallel conversion paths for the same types is dead-weight cognitive load and a maintenance trap.
- Three names for the same type (`models.ModProject` / `structs.SearchModResult` / `providers.ModProject`) makes cross-file search noisy and onboarding harder.

## Goal

After this task, the `models` package is the single, canonical source for mod metadata types. No aliases, no re-exports, no parallel conversion functions. Every package that needs these types imports `mod-downloader/models` directly.

## In Scope

1. **Delete legacy conversion functions** in `providers/modprovider.go` and wire the new `*ToModProject` / `*ToModVersion` functions into `ExactSearch` / `Search` / `ListVersions` / `searchExactMod`.
2. **Remove type aliases** in `structs/search.go` (`SearchModResult`, `ProjectVersionResult`, `ProjectDependency`). Update all field/parameter/return types in `structs/search.go` itself to use `models.ModProject` / `models.ModVersion` / `models.ModDependency`.
3. **Delete `providers/model.go`** (re-export file).
4. **Delete `providers/model_test.go`** — it is a verbatim duplicate of `models/models_test.go` that only exercises the re-exported names. `models/models_test.go` already covers the same cases against the canonical names.
5. **Update all call sites** in `app.go`, `providers/service.go`, `providers/modprovider.go`, `providers/cache.go`, `downloader/download.go`, `downloader/download_test.go` to reference `models.ModProject` / `models.ModVersion` / `models.ModDependency` / `models.ProjectKey` / `models.ParseProjectKey` / `models.VersionKey` / `models.ParseVersionKey` directly.
6. **Regenerate Wails frontend bindings** (`wails generate module`) so `frontend/wailsjs/go/models.ts` and `frontend/wailsjs/go/main/App.d.ts` reflect any signature changes. Note: because the type aliases are removed, `structs.SearchModResult` references in `App.d.ts` will become `models.ModProject` references — this is expected and matches the frontend store which already imports `models.ModProject`.

## Out of Scope

- No behavior changes to search/download logic. This is a pure rename + dead-code-removal refactor.
- No changes to `models/models.go` field definitions or JSON tags.
- No changes to `database/` — it already consumes `models.*` directly.
- No changes to `structs/minecraft/` (Minecraft version manifest types, unrelated).
- No frontend `.vue` / `.ts` source edits beyond what wails binding regeneration produces. The frontend already uses `models.ModProject` / `models.ModVersion` (see `frontend/src/stores/downloadSearch.ts`).
- No changes to `providers/cache.go`'s API surface — it already uses `models.*` via the `providers.ModProject` alias; only its internal references need updating if `providers/model.go` is deleted. (Verify: `cache.go` uses unqualified `ModProject`/`ModVersion`, so it needs the same `models.` prefix treatment.)

## Acceptance Criteria

- [ ] `go build ./...` succeeds with no errors.
- [ ] `go vet ./...` succeeds with no errors.
- [ ] `go test ./...` passes (all existing tests green).
- [ ] No occurrence of `SearchModResult`, `ProjectVersionResult`, or `ProjectDependency` anywhere in `*.go` files (verified by `grep -rn 'SearchModResult\|ProjectVersionResult\|ProjectDependency' --include='*.go' .` returning zero matches).
- [ ] `providers/model.go` and `providers/model_test.go` are deleted.
- [ ] `structs/search.go` no longer contains the three type alias declarations.
- [ ] The legacy conversion functions (`modToSearchResult`, `searchHitToSearchResult`, `projectToSearchResult`, `fileToProjectVersionResult`, `versionToProjectVersionResult`, `setFileResultFields`, `dependenciesFromVersion`, `dependenciesFromFile`) are deleted from `providers/modprovider.go`.
- [ ] The new conversion functions (`modToModProject`, `searchHitToModProject`, `projectToModProject`, `fileToModVersion`, `versionToModVersion`, `setModVersionFileFields`, `dependenciesFromVersionToModDeps`, `dependenciesFromFileToModDeps`) are called by the active `ExactSearch` / `Search` / `ListVersions` / `searchExactMod` flows.
- [ ] Frontend bindings regenerated: `frontend/wailsjs/go/main/App.d.ts` references `models.ModProject` / `models.ModVersion` instead of `structs.SearchModResult` / `structs.ProjectVersionResult` where the Go signatures changed.
- [ ] Manual smoke check: `ProjectID` is populated in search results (the old `modToSearchResult` left it empty; the new `modToModProject` sets it). This is verifiable by confirming the new functions are wired, no runtime test required.

## Risks

- **Wails binding regeneration** changes the TS import paths in `App.d.ts`. The frontend store already imports from `models`, so this should be a no-op or improvement. Verify the frontend still type-checks (`npm run build` in `frontend/`).
- **`providers/cache.go`** uses unqualified `ModProject` / `ModVersion` / `models.ModProject` mixed — need to ensure all references get the `models.` prefix after `model.go` is deleted.
- **Test coverage**: deleting `providers/model_test.go` loses no coverage because `models/models_test.go` tests the same functions against the canonical names.

## Files Touched (expected)

| File | Action |
|------|--------|
| `providers/modprovider.go` | Delete legacy fns; wire new fns; rename type refs to `models.*` |
| `providers/service.go` | Rename type refs to `models.*` |
| `providers/cache.go` | Rename type refs to `models.*` (verify) |
| `providers/model.go` | **Delete** |
| `providers/model_test.go` | **Delete** |
| `structs/search.go` | Remove 3 aliases; update internal refs to `models.*` |
| `app.go` | Rename type refs to `models.*` |
| `downloader/download.go` | Rename type refs to `models.*` |
| `downloader/download_test.go` | Rename type refs to `models.*` |
| `frontend/wailsjs/go/main/App.d.ts` | Regenerated by `wails generate module` |
| `frontend/wailsjs/go/main/App.js` | Regenerated by `wails generate module` |
| `frontend/wailsjs/go/models.ts` | Regenerated (no content change expected) |
