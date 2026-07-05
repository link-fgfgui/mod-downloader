# Quality Guidelines

> Code quality standards for backend development.

---

## Overview

Backend code must compile, vet, and test green (`go build ./... && go vet ./... && go test ./...`). Modified files must be gofmt-clean. Beyond that, the conventions below prevent the concrete bugs and debt this project has hit.

---

## Forbidden Patterns

### Don't: Type aliases as a "transition layer"

**Problem**:
```go
// structs/search.go
type SearchModResult = models.ModProject
type ProjectVersionResult = models.ModVersion
```

**Why it's bad**: Gives one type three names (`models.ModProject` / `structs.SearchModResult` / `providers.ModProject` via re-export). Cross-file search becomes noisy, contributors don't know which name to use, and parallel code paths can coexist unnoticed (one path correct, one path buggy). In this project the alias hid a `ProjectID`-missing bug for an extended period.

**Instead**:
```go
// Import models directly everywhere
import "github.com/link-fgfgui/mod-downloader-core/models"
func F(p models.ModProject) ...
```

If a rename is genuinely needed during a migration, land it in one commit and delete the alias immediately — do not leave it as a permanent transition layer.

### Don't: Re-export files

**Problem**:
```go
// providers/model.go — entire file is re-exports
package providers
import "github.com/link-fgfgui/mod-downloader-core/models"
type ModProject = models.ModProject
type ModVersion = models.ModVersion
var ProjectKey = models.ProjectKey
```

**Why it's bad**: Re-exports let package-internal code use unqualified `ModProject` while external code uses `providers.ModProject` or `models.ModProject`. Three names for one type, same trap as aliases above. The re-export file also becomes a maintenance stub that duplicates the source-of-truth list.

**Instead**: Delete the re-export file. Internal code uses `models.ModProject` qualified.

### Don't: Parallel conversion functions for the same target type

**Problem**:
```go
// Two functions producing the same type, one wired in, one dead
func (p curseForgeProvider) modToSearchResult(mod cfSchema.Mod) models.ModProject { ... } // wired, missing ProjectID
func (p curseForgeProvider) modToModProject(mod cfSchema.Mod) models.ModProject { ... }    // dead, correct
```

**Why it's bad**: The dead (correct) path hides the wired (buggy) path. Reviewers see "the new function exists" and assume it's used. The `ProjectID` field silently becomes `""` in production because the wired function never set it.

**Instead**: When migrating a converter, delete the old function in the same change that wires the new one. Never keep both. If a temporary overlap is unavoidable, add a `// TODO(06-27-unify-models-cleanup): delete after wiring` comment and resolve it in the same task.

---

## Required Patterns

### Convention: Canonical type names drive converter names

Converter functions must be named `<sdkType>To<canonicalType>` where `<canonicalType>` matches the `models` package type name exactly.

```go
// Good
func (p curseForgeProvider) modToModProject(mod cfSchema.Mod) models.ModProject
func (p curseForgeProvider) fileToModVersion(file cfSchema.File) models.ModVersion
func (p modrinthProvider) searchHitToModProject(hit *modrinth.SearchResult) models.ModProject

// Bad — name references a deleted alias
func (p curseForgeProvider) modToSearchResult(mod cfSchema.Mod) models.ModProject
```

### Convention: Single source of truth for shared types

Shared data types live in one package (`models`). Other packages import them directly. See [directory-structure.md](./directory-structure.md#convention-models-is-the-single-source-of-truth) for the full contract.

### Convention: Core dependency is local during development

The main Wails app and the standalone CLI both require `github.com/link-fgfgui/mod-downloader-core` and replace it with their local `core/` submodule:

```go
replace github.com/link-fgfgui/mod-downloader-core => ./core
```

Do not point this replace directive at `../mod-downloader-core`; the core repo is intentionally nested as a git submodule in each consumer so clones are self-contained after `git submodule update --init --recursive`.

---

## Testing Requirements

- Every package with logic has `_test.go` files. Run `go test ./...` before reporting done.
- When changing core code, run tests from `core/` as well as from the consuming app or CLI.
- When deleting a duplicate test file (e.g. `providers/model_test.go` was a verbatim copy of `models/models_test.go`), confirm the canonical test file covers the same cases — no coverage loss.

---

## Code Review Checklist

- [ ] `go build ./... && go vet ./... && go test ./...` all pass.
- [ ] Core changes are validated inside `core/` with `go test ./...` and, when practical, `go build ./... && go vet ./...`.
- [ ] Modified files are gofmt-clean (`gofmt -l <files>` returns nothing). Pre-existing dirtiness in untouched files is out of scope.
- [ ] Main app and CLI `go.mod` files replace `github.com/link-fgfgui/mod-downloader-core` with `./core`, not a sibling path.
- [ ] No new type aliases or re-export files introduced.
- [ ] No parallel conversion functions for the same target type — old deleted when new is wired.
- [ ] Converter names match canonical `models` type names (`*ToModProject`, not `*ToSearchResult`).
- [ ] Frontend bindings regenerated (`wails generate module`) when Go API signatures change.
- [ ] `grep` for deleted type/function names returns zero matches.
