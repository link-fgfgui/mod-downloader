# Development History (Archive)

This file replaces the former `TODO.md` through `TODO5.md` plan files. It records
what each phase delivered, the commit that closed it, and what remains. The
original plans are available in git history.

## Phase timeline

| Phase | Scope | Outcome |
| --- | --- | --- |
| `TODO` | Migrate the Wails v2/Vue app to Wails v3 + React | 70 backend methods and 9 events moved into `core`; six routes, stores, bindings, Vitest, and manual acceptance completed |
| `TODO2` | Interaction and animation fixes from the first UI review | Floating multi-select toolbar, row hover isolation, button press feedback, custom Combobox with directional open/close, `SearchInput` clear button, folder-picker fixes |
| `TODO3` | Animation system (Phase 3) | Global motion multiplier, tree/list FLIP + ghost exits, direction-aware route transitions, queue-panel/FAB lifecycle, Favorites drag indicators, Download batch stagger, Unpin/Settings transitions, Home count-up |
| `TODO4` | Two reported defects | Manage List/Tree preference persisted in `localStorage`; Radix exit animation keeps its final frame (no full-dialog flash) |
| `TODO5` | Release readiness (Phase 4) | Link-time version/CF key injection, CI + ESLint gates, multi-select setting and autosave race guard, lazy routes, dependency cleanup, Manage/Favorites page split, test expansion, README/AGENTS cleanup |

## TODO5 delivery record

Completed in five commits (not pushed at the time of writing):

| Commit | Batch |
| --- | --- |
| `5797d60` | Build-time injection, CI/ESLint, screenshot/template asset cleanup |
| `6ff5637` | Settings completion (68th binding, autosave race) and lazy routes |
| `b5e23f0` | Dependency convergence, Manage/Favorites split, test expansion |
| `ccf17d0` | README rewrite and dead-link cleanup |
| `2056a8e` | Archive TODO5 with execution results |

Final baseline at archive time: 68 App business methods; `Manage.tsx` 552 lines,
`Favorites.tsx` 543 lines; frontend tests 12 files / 37 cases; both Go modules and
both build forms green.

## Remaining work

- **Desktop manual checklist** (requires installer artifacts): native
  directory/save dialogs, real downloads and completion feedback, extension HTTP
  channel, cross-version favorite copy, file-manager reveal, WebKitGTK hardware
  acceleration and theme transitions.
- **`core` de-globalization** (far future): instantiate storage/providers/global/
  downloader instead of package-level singletons. Trigger only when multi-instance
  or parallel-test interference appears.
- **Known issue**: during the headless server's initial hardlink scan, the event
  WebSocket may reject connections for roughly 10 seconds; the frontend retries.
  Desktop in-process events are unaffected.

## Removed source plans

`TODO.md`, `TODO2.md`, `TODO3.md`, `TODO4.md`, and `TODO5.md` were consolidated
here on 2026-11-02. Use git history if the full original text is needed.
