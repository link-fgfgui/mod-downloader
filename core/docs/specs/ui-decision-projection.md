# Shared Decision Projections for Both Frontends

## Decision

Local-mod rules that the GUI and TUI must present identically are computed in
core; adapters only render fields:

- `LocalModGroup.RowKey`: stable row key for a physical jar (non-empty parts of
  `id|sha1|path|fileName`).
- `LocalModGroup.CanReplaceVersion`: whether an online platform and project ID
  exist.
- `LocalModGroup.OnlineProject`: `models.ModProject` built from the local mod's
  online metadata.
- `LocalModGroup.Incompatible` / `SearchText` / `Categories` / `Loaders`:
  existing projections.

Adapters must not assemble row keys or online-project fields themselves. When a
UI state such as "has a selected instance" is needed, add a condition on top of
the core fields instead of reimplementing the field rules.

## Dependency forest

`core/modgraph` is the only implementation that builds the Manage dependency
tree: provider matching, JIJ child nodes, missing leaves, and cycle contraction
all live in core. `LocalModView.Graph` returns a flat node list plus key
references to avoid recursive `children` and cyclic JSON;
`LocalModView.Dependents` indexes reverse dependencies by group `RowKey`. The GUI
only hydrates keys into nodes and adds pinyin search enhancement. The TUI
currently renders a flat list with `BuildLocalModGroups`; when it needs a tree,
it must consume the same projection rather than rebuilding the graph.

## Search expressions

`core/searchexpr` is the only Go implementation of the local-mod search syntax:

```text
expression := branch ( '|' branch )*
branch     := term ( '&' term )*
```

`&` binds tighter than `|`; an empty query and an operator-only query match
everything. The TUI's Manage filter calls `searchexpr.Matches` directly. The GUI
still uses an equivalent TypeScript parser for per-keystroke filtering (to avoid
an RPC on every frame); both sides share the same test cases. When changing the
syntax, update both `core/searchexpr/expression_test.go` and
`frontend/src/utils/searchExpression.test.ts`.

## Version comparison

`core/minecraft` provides `CompareVersions` / `IsVersionDowngrade` /
`HasMajorVersionChange`. The TUI uses them to flag "downgrade or major-version
change" risks in the version list. The GUI's TypeScript comparison must use the
same boundary cases as the Go tests.

## Preserving disabled state during install

When replacing an old jar, `core/downloader` checks whether every replaced file
was `.jar.disabled` and restores the `.disabled` suffix after writing the new
file. Adapters **must not** rename files after installation:

- CLI `upgrade` no longer runs a `KeepDisabled` post-processing step;
- GUI/TUI version replacement gets the same behavior automatically.

## Batch local-mod operations

`ApplyLocalModBatchOperation` is the only entry point for enable/disable/delete.
One-shot CLI commands pass an explicit mods directory through
`LocalModBatchOperationRequest.TargetDir`; the GUI/TUI leave it empty and use the
currently selected instance. The CLI must not call `os.Remove` or `os.Rename` on
jars directly.

## Prohibited

- Do not reimplement these fields/rules in `app.go`, `tviewui`, or
  `frontend/src`.
- Do not keep a separate `.jar.disabled` suffix check for the CLI: use
  `minecraft.IsDisabledModName` / `DisableModPath` / `EnableModPath`.
