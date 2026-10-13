# Implement Plan: Unify on models package

## Pre-flight

- [ ] Confirm clean git status: `git status` (commit or stash any unrelated changes first).
- [ ] Confirm baseline build is green: `go build ./... && go vet ./... && go test ./...`.

## Step 1 — Wire new conversion functions into active flow (`providers/modprovider.go`)

Edit `providers/modprovider.go` only. Do not touch other files yet.

- [ ] In `curseForgeProvider.ExactSearch` (line ~58): change `p.modToSearchResult(mod)` → `p.modToModProject(mod)`.
- [ ] In `curseForgeProvider.Search` (line ~99): change `p.modToSearchResult(mod)` → `p.modToModProject(mod)`.
- [ ] In `curseForgeProvider.ListVersions` (line ~135): change `p.fileToProjectVersionResult(file)` → `p.fileToModVersion(file)`.
- [ ] In `modrinthProvider.Search` (line ~180): change `p.searchHitToSearchResult(hit)` → `p.searchHitToModProject(hit)`.
- [ ] In `modrinthProvider.ListVersions` (line ~210): change `p.versionToProjectVersionResult(version)` → `p.versionToModVersion(version)`.
- [ ] In `modrinthProvider.searchExactMod` (line ~353): change `p.projectToSearchResult(project)` → `p.projectToModProject(project)`.
- [ ] Delete the 8 legacy conversion function definitions:
  - `modToSearchResult` (line ~323)
  - `searchHitToSearchResult` (line ~356)
  - `projectToSearchResult` (line ~382)
  - `versionToProjectVersionResult` (line ~535)
  - `setFileResultFields` (line ~576)
  - `fileToProjectVersionResult` (line ~591)
  - `dependenciesFromVersion` (line ~609)
  - `dependenciesFromFile` (line ~639)
- [ ] Keep the 8 new conversion functions (`modToModProject`, `searchHitToModProject`, `projectToModProject`, `fileToModVersion`, `versionToModVersion`, `setModVersionFileFields`, `dependenciesFromVersionToModDeps`, `dependenciesFromFileToModDeps`).
- [ ] **Validation gate**: `go build ./providers/...` must succeed. (Type references still use `appstructs.SearchModResult` which is an alias — still valid at this point.)

## Step 2 — Delete `providers/model.go` and `providers/model_test.go`

- [ ] Delete `providers/model.go`.
- [ ] Delete `providers/model_test.go` (duplicate of `models/models_test.go`).
- [ ] **Validation gate**: `go build ./providers/...` will FAIL — unqualified `ModProject` / `ModVersion` / `ModDependency` / `ProjectKey` / `ParseProjectKey` / `VersionKey` / `ParseVersionKey` in `providers/modprovider.go`, `providers/cache.go`, `providers/service.go` no longer resolve. This is expected; Step 3 fixes it.

## Step 3 — Fix providers package references

In `providers/modprovider.go`, `providers/service.go`, `providers/cache.go`:

- [ ] Add `"mod-downloader/models"` import if not present (modprovider.go already has it).
- [ ] Replace unqualified `ModProject` → `models.ModProject` (in type contexts only — NOT in function names like `modToModProject`).
- [ ] Replace unqualified `ModVersion` → `models.ModVersion` (same caveat).
- [ ] Replace unqualified `ModDependency` → `models.ModDependency`.
- [ ] Replace `ProjectKey(` → `models.ProjectKey(`, `ParseProjectKey(` → `models.ParseProjectKey(`, `VersionKey(` → `models.VersionKey(`, `ParseVersionKey(` → `models.ParseVersionKey(`.
- [ ] Replace `appstructs.SearchModResult` → `models.ModProject` (modprovider.go has ~63 of these; service.go has ~9).
- [ ] Replace `appstructs.ProjectVersionResult` → `models.ModVersion`.
- [ ] Replace `appstructs.ProjectDependency` → `models.ModDependency`.
- [ ] Remove `"mod-downloader/structs"` / `appstructs` import from `providers/modprovider.go` and `providers/service.go` IF no longer used. `appstructs.SearchModsRequest` / `appstructs.SearchModsUpdate` are still used → keep the import.
- [ ] **Validation gate**: `go build ./providers/...` must succeed. `go test ./providers/...` must pass.

## Step 4 — Update `structs/search.go`

- [ ] Delete the three type alias declarations (lines 21-28):
  ```go
  type SearchModResult = models.ModProject
  type ProjectVersionResult = models.ModVersion
  type ProjectDependency = models.ModDependency
  ```
- [ ] In the same file, update field/parameter types that referenced the aliases:
  - `SearchModsUpdate.Results []SearchModResult` → `[]models.ModProject`
  - `ModDownloadRequest.Result SearchModResult` → `models.ModProject`
  - `DownloadStatesRequest.Results []SearchModResult` → `[]models.ModProject`
- [ ] Keep the `import "mod-downloader/models"` at the top.
- [ ] **Validation gate**: `go build ./structs/...` will fail because downstream packages still use `appstructs.SearchModResult`. This is expected; Steps 5-6 fix them.

## Step 5 — Update `app.go` and `downloader/`

- [ ] `app.go`: add `"mod-downloader/models"` import. Replace `appstructs.SearchModResult` → `models.ModProject` (1 occurrence at line 82) and `appstructs.ProjectVersionResult` → `models.ModVersion` (1 occurrence at line 82). Keep `appstructs` import — still used for `SearchModsRequest`, `ModDownloadRequest`, `ModDownloadResult`, etc.
- [ ] `downloader/download.go`: add `"mod-downloader/models"` import. Replace all `appstructs.SearchModResult` → `models.ModProject` (~10 occurrences), `appstructs.ProjectVersionResult` → `models.ModVersion` (~14 occurrences), `appstructs.ProjectDependency` → `models.ModDependency` (~3 occurrences). Keep `appstructs` import — still used for `ModDownloadRequest`, `ModDownloadResult`, `DownloadQueueState`, etc.
- [ ] `downloader/download_test.go`: add `"mod-downloader/models"` import. Replace `appstructs.SearchModResult` → `models.ModProject` (1 occurrence), `appstructs.ProjectVersionResult` → `models.ModVersion` (1 occurrence).
- [ ] **Validation gate**: `go build ./...` must succeed. `go vet ./...` must succeed. `go test ./...` must pass.

## Step 6 — Verify no legacy names remain

- [ ] Run: `grep -rn 'SearchModResult\|ProjectVersionResult\|ProjectDependency' --include='*.go' .` — expect ZERO matches.
- [ ] Run: `grep -rn 'providers\.ModProject\|providers\.ModVersion\|providers\.ModDependency\|providers\.ProjectKey' --include='*.go' .` — expect ZERO matches.
- [ ] Confirm `providers/model.go` and `providers/model_test.go` no longer exist.
- [ ] Confirm `structs/search.go` no longer has the three `type … = models.…` alias lines.

## Step 7 — Regenerate Wails frontend bindings

- [ ] Run `wails generate module` from project root. (If wails CLI is not installed: `go run github.com/wailsapp/wails/v2/cmd/wails generate module`.)
- [ ] Verify `frontend/wailsjs/go/main/App.d.ts` now shows `ListMatchingProjectVersions(arg1:models.ModProject,arg2:string,arg3:string):Promise<Array<models.ModVersion>>` instead of `structs.SearchModResult` / `structs.ProjectVersionResult`.
- [ ] Verify `frontend/wailsjs/go/models.ts` still exports `ModProject` / `ModVersion` / `ModDependency` under the `models` namespace.
- [ ] **Validation gate**: `cd frontend && npm run build` (or `npm run build-only`) must succeed — confirms frontend still type-checks against regenerated bindings.

## Step 8 — Final full validation

- [ ] `go build ./...`
- [ ] `go vet ./...`
- [ ] `go test ./...`
- [ ] `cd frontend && npm run build` (or equivalent)
- [ ] Review the full diff: `git diff --stat` and `git diff`. Confirm no unintended changes, no leftover comments referencing deleted types.

## Rollback points

- After Step 1: `git checkout providers/modprovider.go` rolls back the wiring.
- After Step 2-3: `git checkout providers/` rolls back the package.
- After Step 4-6: `git checkout .` rolls back everything (no commits yet).
- The refactor is safe to commit as a single commit once all gates pass.

## Notes for the implementer

- The `appstructs` import alias stays in most files because `SearchModsRequest`, `SearchModsUpdate`, `ModDownloadRequest`, `ModDownloadResult`, `ModVersionPinRequest`, `DownloadQueueState`, `DownloadQueueItem`, `DownloadFailedEvent`, `DownloadStatesRequest`, `ModDownloadButtonState` all live in `structs/search.go` and are NOT being moved.
- Watch for `appstructs.SearchModsUpdate.Results` field — its type changes from `[]SearchModResult` to `[]models.ModProject`. Callers that append to it are fine (alias was transparent), but verify no test does a structural comparison that would break.
- `providers/cache.go` uses `ModProject` / `ModVersion` unqualified in function signatures (lines 10, 20, 34, 57, 73, 77, 88). These all need the `models.` prefix.
- The new conversion functions reference `ProjectKey` (e.g. `modToModProject` line 950: `ID: ProjectKey("curseforge", id)`). After `model.go` is deleted, these become `models.ProjectKey`.
