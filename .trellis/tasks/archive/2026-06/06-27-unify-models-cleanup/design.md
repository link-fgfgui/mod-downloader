# Design: Unify on models package

## Type identity after refactor

Single source of truth: `mod-downloader/models`.

| Canonical name | Was aliased as | Used in |
|----------------|----------------|---------|
| `models.ModProject` | `structs.SearchModResult`, `providers.ModProject` | app.go, providers/*, downloader/*, structs/search.go (field types) |
| `models.ModVersion` | `structs.ProjectVersionResult`, `providers.ModVersion` | app.go, providers/*, downloader/*, structs/search.go (field types) |
| `models.ModDependency` | `structs.ProjectDependency`, `providers.ModDependency` | providers/modprovider.go, downloader/download.go |
| `models.ProjectKey` / `models.ParseProjectKey` / `models.VersionKey` / `models.ParseVersionKey` | `providers.ProjectKey` etc. | providers/modprovider.go (internal), providers/model_test.go (deleted) |

## Conversion function consolidation

`providers/modprovider.go` currently has 8 legacy conversion functions (active) and 8 new conversion functions (dead). The new functions produce identical types (because of the alias) but set `ProjectID` correctly.

### Strategy: rename new functions to old names, then delete old

Pure deletion of the old functions and rewiring call sites to the new names would work, but the new names (`modToModProject`, `fileToModVersion`, …) are tautological inside the providers package — `ModProject` is the only project type. The cleaner end state is to keep the **short** names (`modToSearchResult` style) but have them return the new, correct field assignments.

**Decision**: Rename the new functions in place to take over the short names, delete the old functions, keep all call sites unchanged for the conversion calls themselves. Only the type references (`appstructs.SearchModResult` → `models.ModProject`) need updating at call sites.

| Delete (old, buggy) | Rename → (new, correct) takes over short name |
|----------------------|------------------------------------------------|
| `modToSearchResult` | `modToModProject` → `modToSearchResult` |
| `searchHitToSearchResult` | `searchHitToModProject` → `searchHitToSearchResult` |
| `projectToSearchResult` | `projectToModProject` → `projectToSearchResult` |
| `fileToProjectVersionResult` | `fileToModVersion` → `fileToProjectVersionResult` |
| `versionToProjectVersionResult` | `versionToModVersion` → `versionToProjectVersionResult` |
| `setFileResultFields` | `setModVersionFileFields` → `setFileResultFields` |
| `dependenciesFromVersion` | `dependenciesFromVersionToModDeps` → `dependenciesFromVersion` |
| `dependenciesFromFile` | `dependenciesFromFileToModDeps` → `dependenciesFromFile` |

**Rationale**: This minimizes diff noise in the active Search/ListVersions flow — the call expressions `p.modToSearchResult(mod)` etc. stay identical. The only change at call sites is the type-reference rename (`appstructs.SearchModResult` → `models.ModProject`), which is mechanical and global.

**Alternative considered**: Keep the `*ToModProject` names and update all call sites. Rejected because it churns the active flow for no semantic gain and the `Mod` prefix is redundant once aliases are gone.

**Wait — revised decision**: The user's intent is "全面清理旧数据结构" (comprehensively clean up old data structures). Keeping the name `modToSearchResult` preserves the legacy vocabulary ("SearchResult") which contradicts the unification goal. The cleaner end state uses `*ToModProject` / `*ToModVersion` names consistently with the canonical type names.

**Final decision**: Delete the old short-named functions. Keep the new `*ToModProject` / `*ToModVersion` names. Update the 8 call sites in the active flow to call the new names. This is a slightly larger diff but a cleaner end state, and the call-site count is small (8 calls across `ExactSearch`/`Search`/`ListVersions`/`searchExactMod`).

### Call-site rewiring (modprovider.go)

| Line | Was | Becomes |
|------|-----|---------|
| 58 | `p.modToSearchResult(mod)` | `p.modToModProject(mod)` |
| 99 | `p.modToSearchResult(mod)` | `p.modToModProject(mod)` |
| 135 | `p.fileToProjectVersionResult(file)` | `p.fileToModVersion(file)` |
| 180 | `p.searchHitToSearchResult(hit)` | `p.searchHitToModProject(hit)` |
| 210 | `p.versionToProjectVersionResult(version)` | `p.versionToModVersion(version)` |
| 353 | `p.projectToSearchResult(project)` | `p.projectToModProject(project)` |

## Type reference rename strategy

Mechanical rename across all `*.go` files:

- `appstructs.SearchModResult` → `models.ModProject`
- `appstructs.ProjectVersionResult` → `models.ModVersion`
- `appstructs.ProjectDependency` → `models.ModDependency`

Within the `providers` package, unqualified `ModProject` / `ModVersion` / `ModDependency` / `ProjectKey` / `ParseProjectKey` / `VersionKey` / `ParseVersionKey` (resolved via the deleted `model.go`) → `models.ModProject` etc.

Within `structs/search.go`, the bare names `SearchModResult` / `ProjectVersionResult` / `ProjectDependency` (used in `SearchModsUpdate.Results`, `ModDownloadRequest.Result`, `DownloadStatesRequest.Results`) → `models.ModProject` / `models.ModVersion` / `models.ModDependency`. The `models` import already exists in that file.

## Import management

After rename:
- `app.go` adds `"mod-downloader/models"` import (already imports `appstructs "mod-downloader/structs"`; keep both — `appstructs` still needed for `SearchModsRequest`, `ModDownloadRequest`, etc.).
- `providers/modprovider.go` adds `"mod-downloader/models"` (already imports it — verify).
- `providers/service.go` adds `"mod-downloader/models"`.
- `providers/cache.go` already imports `"mod-downloader/models"`.
- `downloader/download.go` adds `"mod-downloader/models"`.
- `downloader/download_test.go` adds `"mod-downloader/models"`.
- `structs/search.go` already imports `"mod-downloader/models"`.

Remove unused `"mod-downloader/structs"` / `appstructs` imports only if no other types from `structs` are used in that file. `structs/search.go` keeps its other types (`SearchModsRequest`, `ModDownloadRequest`, etc.), so the `models` import stays but no `structs` self-reference is needed.

## Wails binding regeneration

After Go compiles, run `wails generate module` from the project root. This regenerates:
- `frontend/wailsjs/go/main/App.d.ts` — `ListMatchingProjectVersions(arg1:models.ModProject,…)` instead of `structs.SearchModResult`.
- `frontend/wailsjs/go/main/App.js` — no semantic change.
- `frontend/wailsjs/go/models.ts` — no content change (types already exported as `models.ModProject`).

The frontend store (`frontend/src/stores/downloadSearch.ts`) already imports `models.ModProject` / `models.ModVersion`, so no `.ts`/`.vue` source edits are needed.

## Rollback shape

Pure refactor, no data migration. Rollback = `git revert <commit>`. No database schema change, no config change. The `mods.gob.zst` cache file format is unchanged (it already stores `models.ModProject` / `models.ModVersion`).

## Compatibility

- **Go API**: Public method signatures on `App` change types from `structs.SearchModResult` to `models.ModProject`. Wails regenerates bindings to match. No runtime behavior change.
- **Frontend**: Already uses `models.ModProject`. No source edits.
- **Database**: Unchanged.
- **Cache file**: Unchanged (gob serialization uses the same underlying types).
