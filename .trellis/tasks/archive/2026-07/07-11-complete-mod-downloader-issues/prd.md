# Complete mod-downloader issues

## Goal

Complete every unfinished item in the Microsoft To Do list named exactly
`mod-downloader`, using the dependency and priority order recorded in
`issues.md`, while preserving one Trellis task as the single planning and
verification record.

## Background

- Microsoft To Do currently reports 29 unfinished top-level tasks in the exact
  `mod-downloader` list. The similarly named list marked "do not read" is out of
  scope.
- `issues.md` maps those tasks to I01-I29, includes unfinished checklist items,
  and records dependency chains and recommended implementation batches.
- The work crosses the Vue/Vuetify frontend, Pinia stores, the Wails adapter,
  and the Go `core` module. Public Wails contract changes require regenerated
  bindings.
- All 29 items are owned by this task. No Trellis child tasks will be created.

## Requirements

### R1: Backend contracts and shared foundations

- Complete I01, I02, I03, I09, and I24.
- Favorite-list visibility and selection must follow the active Minecraft
  version and mod-loader tuple.
- Downloads/API settings must have explicit defaults, validation ranges, and
  live runtime reconfiguration behavior.
- Preserve defaults `file_concurrency = 4`, `concurrent_downloads = 1`, and
  `requests_per_second = 0`; validate them as `1-32`, `1-16`, and `0-100`
  respectively, with the backend as the authoritative validation boundary.
- Logging must expose positive `enabled` semantics without reversing behavior
  for existing installations.
- MCIM mode must not prevent CurseForge search.
- Virtual lists must render immediately after KeepAlive page activation.

### R2: Structural cleanup and shared UI capabilities

- Complete I10, I28, I07, and I25.
- Remove favorite grouping from the frontend, store, and public Wails surface
  without destructively deleting user favorite data.
- Remove Home current-status and bottom-status cards plus obsolete state work.
- Reuse the existing version list and pin flow in the local-mod version dialog.
- Give Download and Manage virtual-list viewports one shared layout contract.

### R3: Dependent functionality

- Complete I05, I21, I22, I20, I11, I14, I17, I18, I15, I16, I06, I12, I13,
  I26, I27, and I29 in the dependency order from `issues.md`.
- Settings auto-save must centralize debounce, pending state, error rollback,
  snackbar reporting, and API-key keep/clear semantics.
- List shortcuts must ignore editable controls and open dialogs.
- Compact Home-card numbers must retain three significant digits at 1K and
  above while the total remains fully localized.
- Usage-stat mutations must push an event through the Wails adapter; Home only
  listens while active and retains its initial fetch.

### R4: Independent page fixes

- Complete I04, I08, I19, and I23, including every unchecked checklist item
  nested under I04 and I08 in `issues.md`.
- Animation mode changes must always clean route opacity/transform state.
- Conflict dialogs must identify the conflicting filename and place the hint on
  its own line.
- Remove only the redundant cache-default field and Unpin filter fields; retain
  their still-valid actions with semantics adjusted to the complete list.

### R5: Source synchronization

- Keep `issues.md` checkboxes aligned with verified implementation progress.
- Mark a Microsoft To Do task complete only after its mapped acceptance checks
  pass. Complete nested checklist items at the same time as their owning work.
- Do not read or mutate any other Microsoft To Do list.

## Acceptance Criteria

- [x] AC1: Every I01-I29 checkbox and nested checkbox in `issues.md` is checked,
  backed by current code and relevant automated or documented UI verification.
- [x] AC2: The exact `mod-downloader` Microsoft To Do list contains no unfinished
  task from the original 29-item scope.
- [x] AC3: Dependency order is respected, with foundation changes verified
  before dependent changes are accepted.
- [x] AC4: `go build ./...`, `go vet ./...`, and `go test ./...` pass at the app
  root; `go test ./...` passes in `core/`.
- [x] AC5: `npm run lint` and `npm run build` pass in `frontend/`.
- [x] AC6: Wails bindings are regenerated and consumed successfully whenever a
  public method or shared payload changes.
- [x] AC7: One and only one Trellis task represents this body of work; it is
  checked, committed, and archived only after the completion audit passes.

## Out Of Scope

- The similarly named Microsoft To Do list whose title says not to read it.
- Destructive removal of favorite storage tables or historical favorite data.
- Unrelated refactors or features not represented by I01-I29.
