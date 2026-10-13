# Check core CLI compatibility

## Goal

Verify whether the current core submodule used by the main app remains
compatible with the sibling `../mod-downloader-cli` project, then fix each
confirmed incompatibility on the side that best preserves the shared contract.

The target compatibility point is the main app's current core checkout:
`56f8e8b` (`Merge branch 'wf-favorite-page'`). The CLI currently pins its own
core submodule at `016dfac` (`Add text-based search and direct project lookup
methods`).

## Confirmed Facts

- The main app and CLI are separate Go modules that both replace
  `github.com/link-fgfgui/mod-downloader-core` with their local `./core`
  submodule.
- Recent core commits after the CLI's pinned core include favorite-list
  persistence, local mod batch operations, retryable download queue jobs, active
  download retry/stall handling, animation preferences, and extended download
  state fields.
- The CLI imports shared core packages directly, including `appcore`,
  `database`, `minecraft`, `models`, and `structs`.
- Project guidelines require core changes to be validated inside `core/`, and
  consuming app/CLI checks to be rerun when exported contracts or adapter
  behavior change.

## Requirements

- Detect compile-time and test-time incompatibilities between the CLI codebase
  and core `56f8e8b`.
- For each conflict, decide whether the correct fix belongs in core or in the
  CLI:
  - Fix CLI when the CLI is using an older API shape or making assumptions that
    no longer match the exported core contract.
  - Fix core when a recent change broke an existing shared API contract without
    a justified migration path.
- Keep the app-independent service boundary intact: `appcore`, `httpserver`,
  and CLI code must not depend on Wails runtime.
- Preserve the CLI's existing command surface and JSON/text output semantics
  unless a core contract makes a behavior change unavoidable.
- Update the CLI's core submodule only to a commit that has been validated with
  CLI build/test checks.

## Acceptance Criteria

- [x] `../mod-downloader-cli` builds and tests against core `56f8e8b`.
- [x] `go test ./...` passes from `core/` after any core-side edits.
- [x] `go build ./...` and `go test ./...` pass from `../mod-downloader-cli`.
- [x] Dependency checks keep Wails runtime out of core service packages and CLI
  packages.
- [x] Any source changes are gofmt-clean.
- [x] The final summary identifies each confirmed conflict and which side was
  modified.

## Results

- No compile-time or test-time compatibility conflicts were found after moving
  the CLI core submodule from `016dfac` to `56f8e8b`.
- No core-side or CLI-side Go source edits were required.
- The only compatibility update is the CLI `core` submodule pointer.

## Out of Scope

- Adding new CLI commands for new core features unless needed to restore
  compatibility.
- Regenerating Wails frontend bindings unless core-side changes alter app-facing
  Wails method signatures.
- Publishing or pushing submodule commits.

## Open Questions

None blocking. Repository evidence is sufficient to proceed with compatibility
testing and fixes.
