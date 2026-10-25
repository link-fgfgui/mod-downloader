# Local Mod Search, Dependency Tree, And Scoped Modals

> Frontend contracts for Manage-page search/tree and favorites scope filtering.

## Search indexing must use an explicit field whitelist

### Don't: reflect over the whole mod group

**Problem**: the old `searchableGroupText` recursively collected every string
field of a mod group.

**Why it's bad**: `mod.path` is an absolute/relative path that contains the
Minecraft version, and `mod.dependencies` always declares a `minecraft` version.
Searching a version (e.g. `1.20.1`) therefore matched *every* mod.

**Instead**: `utils/modSearchText.ts` collects only human-facing text — `id`,
`name`, `onlineName`, `version`, `fileName`, `description`, `categories`, and
each `jijMods[].id/name`. Never include `path`, `dependencies`, booleans, or
internal cache identifiers. Chinese names, categories, and JIJ names also
contribute pinyin initials (`钠离子` → `nlz`) through `pinyin-pro`, so both
plain substrings and initials work.

### Known residual: file names may embed the Minecraft version

`fileName` remains searchable because users legitimately search by jar name.
Many jars look like `xxx-0.5.8-mc1.20.1.jar`, so searching `1.20.1` can still
match those rows. This is intentional; do not "fix" it by removing file-name
search. If it becomes noisy, only fall back to a cleaned file name (strip
`-mc<version>` suffixes) after the raw match misses.

## Search expression syntax

`utils/searchExpression.ts` implements `&` (AND, higher precedence) and `|`
(OR). A query with no operators degrades to the previous substring behavior.
`matchesSearchExpression(text, query)` returns `true` for empty queries and for
queries made only of operators, so a stray `|` never hides every row.

## Dependency tree

`composables/useModDependencyGraph.ts` builds a forest: edges come from required
dependencies, missing dependencies become synthetic leaves, and roots are the
source components of the graph. `flattenModDependencyTree` emits indented rows
for a plain list (no `v-treeview` dependency) with a depth cap and an ancestor
guard against cycles.

### Don't: derive roots from raw in-degree

**Problem**: a mod that is part of a dependency cycle (common with Jar-in-Jar
pairs) has in-degree > 0, so a cycle-only graph produces zero roots.

**Why it's bad**: the tree view renders "no resolvable dependencies" even though
the instance has mods.

**Instead**: condense strongly connected components first (Tarjan) and pick one
representative per source component. Components with no incoming edges from other
components become the roots; the ancestor guard keeps the rendered tree finite.

### Tree filters

The tree obeys the same search/enabled filter as the list: the Manage view
computes the visible key set with `collectMatchingTreeKeys` (matching nodes +
their ancestors + attached synthetic missing leaves) and passes it to
`flattenModDependencyTree({ visibleKeys })`. Active searches also pass
`forceExpand` so matched branches are revealed without manual expansion. Never
render the tree from the unfiltered mod list while filter controls are visible.

### Node status and interactions

`ModTreeNode` carries `isOptional`, `isJij`, `isRoot`, and `isMissing` in
addition to enable state. Status precedence is missing (red) → JIJ (purple) →
disabled (grey) → optional (orange) → enabled (green). Optional dependencies
only become edges when a local provider exists; a missing optional library is
not a broken prerequisite. JIJ sub-mods are synthetic children of their host
with `isJij = true`.

Clicking a row selects it and toggles the branch. The selection highlights the
selected row, its descendants (dependencies) and its ancestors (dependents) via
`collectRelatedTreeKeys`. Right-clicking a real mod node (or the row menu
button) exposes open-file-location, enable/disable, uninstall, and
search-new-version actions; synthetic JIJ/missing nodes only support selection.
`OpenLocalModLocation` accepts only paths that belong to the currently scanned
instance.

### Reverse-dependency analysis

Version replacement risk uses `findReverseDependents` from the same composable,
not a second ad-hoc scan. It applies the graph's required-only filter, excludes
pseudo dependencies (`minecraft`, loaders, `java`), skips disabled mods unless
asked otherwise, and returns the declared version constraints so the tooltip
can say e.g. "Iris requires Sodium 0.5.8". A downgrade **or a major-version
change** is risky when another enabled mod depends on the selected mod.

## Scope-filtered favorites dialog

`AddToFavoriteDialog` filters candidate lists with the exported
`listMatchesScope` from the favorites store, using the first draft's
`minecraftVersion`/`modLoader`. It shows the target scope, and `resolveListId`
blocks reusing a same-named list that belongs to a different scope. This mirrors
the store's own scope getters — never bypass them.

## Unused dependencies are a list sort condition

There is no "scan unused dependencies" button and no separate cleanup dialog.
`ScanUnusedDependencies` runs as a passive list signal:

- Sort mode `unused` triggers a scan and orders flagged groups first; the row
  shows a "Unused dependency" chip with the evidence tooltip.
- `settings.cleanup.autoScan` (`autoScanUnusedDependencies`) keeps the marking
  up to date after refreshes even when another sort mode is active.
- Rows are still deleted through the ordinary selection action bar, so the
  destructive step keeps its existing confirmation.
- The scan is skipped entirely when `enableJijDetection` is off (the backend
  fails closed too); the sort option is shown disabled with an explanation.

Do not reintroduce a modal that deletes every candidate in one click: the list
already exposes per-row and multi-select actions.

## Component boundaries

- `ManageToolbar` owns the header, search expression help, enabled filter, sort,
  and list/tree switch.
- `ManageModList` owns VirtualList rendering, list scroll/tooltip throttling, and
  the selection action bar. Display/selection logic is injected as functions;
  the view owns the underlying state.
- `ManageTreeView` owns graph construction, filtering, selection relation
  highlighting, and the node context menu.
- `ManageVersionDialog` owns version listing, pre-flight, downgrade/major-change
  confirmation, and queueing; the parent only calls `open(group)`.
- `ManageModDialogs` owns delete/disable-impact confirmation rendering.
- `useLocalModSelection` owns batch/selection operation state;
  `useUnusedDependencies` owns the passive unused-dependency scan.
- `DownloadQueuePanel`, `FavoritesCrossVersionCopyDialog`, and
  `FavoritePinDialog` keep the queue and Favorites dialogs out of the root views.
- Locale messages live in `plugins/locales/zh.ts` and `en.ts`; `i18n.ts` only
  wires i18n helpers and the two message bundles.
