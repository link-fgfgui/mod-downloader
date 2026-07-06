# Design

## Boundary

The compatibility target is the sibling CLI repository at
`../mod-downloader-cli` running against the main app's current core submodule
commit, `56f8e8b`. The main app repository is used as the source of truth for the
desired core state because it contains the recent core updates under review.

## Compatibility Strategy

1. Establish a baseline by recording the app core commit and the CLI core
   commit.
2. Move the CLI's local `core/` submodule to the target core commit for the
   purpose of build/test validation.
3. Run CLI compile and test checks to surface concrete incompatibilities.
4. For each incompatibility:
   - Prefer CLI-side changes when the CLI is adapting to additive core features,
     renamed exported fields, or stricter service behavior.
   - Prefer core-side changes when the target core commit removed or changed an
     exported API that should remain stable for both app and CLI consumers.
5. Validate the selected side with focused tests first, then full package
   checks.

## Contracts To Preserve

- `mod-downloader-cli/go.mod` must keep:
  `replace github.com/link-fgfgui/mod-downloader-core => ./core`.
- CLI code must not import Wails runtime.
- Shared data types remain in `models` or existing `structs`; no transition
  aliases or re-export files should be introduced.
- CLI runtime overrides are command-scoped and must not persist through service
  shutdown.

## Rollback

If the target core commit exposes a regression that should be fixed in core,
keep the CLI submodule move separate from source edits so the incompatible
commit can be reverted or amended independently. If a CLI-side adaptation is
wrong, revert only the CLI source change and keep the test output as evidence.
