# Optional and Incompatible Dependency Handling Design

## Boundaries

- `providers` continues to normalize platform dependency metadata into `models.ModDependency`.
- `modbridge` owns install-status and compatibility analysis because it already bridges platform versions, remote mod IDs, and local installed files.
- `downloader` owns queue/reminder state and the final install-time action that archives local jars.
- `appcore` and `app.go` expose backend queue/reminder APIs to Wails without duplicating business logic.
- `frontend/src/App.vue` owns the floating queue panel UI; `downloadQueue` Pinia owns the live queue snapshot and message actions.
- `downloadSearch` and `Download.vue` own search-result confirmation state.

## Backend Contracts

Extend download queue structs with structured non-download reminders instead of encoding reminder text into `DownloadQueueItem`.

Proposed additions:

- `DownloadQueueState.OptionalReminders []OptionalDependencyReminder`
- `DownloadQueueState.MessageCount int` or a derived frontend count from `OptionalReminders`
- Keep `DownloadQueueState.Active` as the active-download/retryable-history signal unless all callers are updated. `InstallModAndWait` currently waits for `!state.Active`, so optional reminders must not make CLI/server wait paths hang.
- `DownloadQueueItem.Kind` only if a single mixed item list is preferred; otherwise keep downloads and reminders separate.
- `OptionalDependencyReminder`:
  - `id`
  - `mainProject` / `mainTitle`
  - `mainVersionId`
  - `minecraftVersion`
  - `modLoader`
  - `dependencies []OptionalDependencyCandidate`
- `OptionalDependencyCandidate`:
  - `projectId`
  - `platform`
  - `title`
  - `versionId`
  - `status`
  - `disabled`
  - `reason`
  - enough request data to call the normal queue path, or a backend reminder action ID that resolves to stored request data.

Add queue actions:

- `DismissOptionalDependencyReminder(id string) bool`
- `ClearOptionalDependencyReminders() bool`
- `InstallOptionalDependencies(reminderID string) ModDownloadResult` or a batch result type
- `AnalyzeBatchIncompatibleConflicts(req BatchDownloadRequest) BatchIncompatibleAnalysis` or equivalent service method for preflight batch conflict analysis

The install action should resolve stored dependency requests server-side and call `queueModDownload` for each dependency. Do not let the frontend reconstruct requests from display-only fields.

## Optional Dependency Flow

1. `QueueModDownload` resolves the selected version.
2. Dependency metadata is hydrated for `required`, `optional`, and `incompatible` relation types. Current `hydrateRequiredDependencies` is too narrow for this task.
3. Required dependencies keep using `queueMissingRequiredDependencies`.
4. Optional dependencies are converted into reminder candidates after filtering out already installed candidates with `InstallStatusPrecise`.
5. A reminder group is appended to queue state when at least one actionable optional candidate remains.
6. Queue state emission includes reminders so the floating button and panel update without a manual refresh.
7. Dismissing a reminder removes only that reminder group.
8. Installing reminder dependencies calls the normal backend queue path, preserving all compatibility checks and existing download events.

## Incompatible Dependency Flow

Add a modbridge analysis helper that resolves the requested version's `incompatible` dependencies to local installed paths:

- Resolve target dependency project/version using the parent platform and `dependencyDownloadRequest`-equivalent request construction.
- Resolve target mod IDs from the dependency version via existing `VersionModIDs` logic when possible.
- Find local installed paths through `LocalModPathsForModIDs`.
- Return a structured result with status, conflicting paths, and display metadata.

Search-list status:

- Add a new button status, for example `BtnStatusIncompatible = "incompatible"`.
- `DownloadStates` should apply this status when incompatible installed paths are detected.
- Button color should be warning/error-toned and icon should make the warning visible.
- `downloadSearch.confirmStatuses` must include `incompatible`.
- Confirmation dialog copy should branch for `update`, `conflict`, and `incompatible`.

Batch preflight:

- The batch download button should call a backend batch-analysis API before queueing any selected item.
- The analysis returns one entry per selected mod with incompatible conflicts, including the selected mod title/project key and every incompatible installed mod/path that would be renamed.
- If no conflicts are returned, the frontend queues the full batch immediately.
- If conflicts are returned, the frontend shows one confirmation dialog with the complete grouped conflict content.
- Confirm queues the full selected batch and marks conflicted entries as confirmed/bypassed so the normal install path performs archive/rename.
- Decline queues only selected mods that did not appear in the conflict analysis result.
- `Esc`, close, or dialog dismissal cancels this batch click entirely and queues nothing.

Install-time behavior:

- `queueModDownload` must re-run incompatible analysis before queueing the requested mod.
- If incompatible local paths are present and the request was not confirmed/bypassed, return a skipped result that the frontend uses to show confirmation. If the current app contract cannot pass confirmation state cleanly, the frontend warning state can own confirmation and backend can treat any queued request as confirmed after re-running analysis.
- The final download job should carry `IncompatibleLocalPaths []global.LocalModFilePath` or equivalent archive targets.
- Before installing the requested mod, archive those paths through the same `archiveSupersededModJars` helper used by mod-ID conflict replacement.

## Frontend UI

Queue panel:

- Add tabs: downloads and optional dependencies.
- Downloads tab preserves the current list/actions for running, pending, failed, and canceled items.
- Optional tab shows reminder groups. Each group has the main mod, candidate optional dependencies, a dismiss action, and an install-all action.
- The floating button remains visible when `queue.active` or optional reminders exist; the frontend should not rely on `queue.active` alone after reminders are added.
- If active download count is zero and optional reminders exist, use a bell icon and right-click clears optional reminders.
- Existing watch that closes the panel when no visible items remain must consider optional reminders.

Search confirmation:

- Keep existing left-click/right-click convention.
- Extend the confirmation dialog to render incompatible-specific title/body/confirm copy.
- Add a batch-incompatible confirmation dialog or extend the existing dialog to support grouped conflicts and three outcomes: confirm all, skip conflicted, cancel.
- Add zh/en i18n labels for optional tab, reminder actions, incompatible confirmation, and incompatible status.

## Compatibility and Rollback

- Required dependency recursion must be regression-tested because dependency hydration changes.
- Queue snapshot JSON is Wails-facing; regenerate/sync frontend bindings after struct changes.
- Optional reminders are session-only; rollback is clearing in-memory reminder state.
- Incompatible archive action reuses `.old` suffix naming, so rollback for a user is manually renaming files back, matching current replacement behavior.

## Trade-offs

- Keeping optional reminders inside queue state avoids a separate notification system and satisfies the floating-button behavior, but it expands queue semantics beyond downloads.
- Resolving dependency display metadata opportunistically avoids blocking installs on provider lookup failures.
- Single right-click remains a fast path, while batch download gets one explicit preflight dialog so bulk operations do not silently rename incompatible jars.
