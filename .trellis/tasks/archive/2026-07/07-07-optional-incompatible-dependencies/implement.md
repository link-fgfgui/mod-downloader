# Implementation Plan

## Pre-Development Context

- Before code edits, load `trellis-before-dev` for the touched package/layers.
- Read relevant spec indexes:
  - `.trellis/spec/guides/index.md`
  - `.trellis/spec/app/backend/index.md`
  - `.trellis/spec/core/backend/index.md` if editing the `core` submodule.

## Ordered Checklist

1. Backend model/contracts
   - Extend `core/structs/search.go` queue snapshot structs for optional dependency reminders.
   - Add request/result structs for dismiss, clear, and install-reminder actions if needed.
   - Add request/result structs for batch incompatible preflight analysis.
   - Preserve `DownloadQueueState.Active` as an active-download/retryable-history signal, or update every waiter such as `InstallModAndWait` if its semantics change.
   - Regenerate or synchronize Wails frontend models after Go struct/API changes.

2. Dependency classification helpers
   - Replace narrow required-only hydration with dependency hydration that preserves required/optional/incompatible relations.
   - Add helper predicates for required, optional, and incompatible dependency types.
   - Keep provider normalization unchanged unless tests reveal gaps.

3. Optional reminder queue state
   - Add in-memory optional reminder storage to `core/downloader`.
   - Create reminder groups after queueing a main mod with actionable optional dependencies.
   - Filter already installed optional deps with `modbridge.InstallStatusPrecise`.
   - Add dismiss, clear, and install-reminder functions.
   - Emit queue state after reminder mutations.

4. Incompatible analysis
   - Add `modbridge` helper to resolve incompatible dependency relations to installed local paths.
   - Add `BtnStatusIncompatible` and apply it in `DownloadStates`.
   - Add a batch analysis helper/API that returns grouped incompatible conflicts for selected search results without queueing downloads.
   - Ensure unresolved incompatible metadata fails soft.

5. Incompatible install archiving
   - Re-run incompatible analysis at queue/install time.
   - Carry archive targets in `downloadJob`.
   - Archive incompatible paths with existing `.old` helper before final install/hardlink path mutates target mods.
   - Avoid archiving the requested mod itself or already-installed identical SHA1 paths.

6. Appcore/Wails APIs
   - Expose dismiss/clear/install optional reminder actions through `core/appcore.Service` and `app.go`.
   - Preserve event emission through `download-queue-updated`.

7. Frontend state
   - Update `frontend/src/stores/downloadQueue.ts` for reminder counts, actions, and right-click clear.
   - Update `frontend/src/stores/downloadSearch.ts` confirmation statuses and incompatible handling.
   - Update batch download flow to run backend incompatible preflight before queueing and to branch into confirm-all, skip-conflicted, or cancel-all outcomes.

8. Frontend UI/i18n
   - Convert `frontend/src/App.vue` queue panel to tabs while preserving current download item rendering.
   - Add optional reminder group UI with dismiss and install-all actions.
   - Add bell icon behavior when only reminders remain.
   - Add a grouped batch incompatible dialog that shows every selected conflicted mod and its incompatible installed mods/paths.
   - Add zh/en i18n strings for optional reminders and incompatible confirmation.

9. Tests and verification
   - Add Go unit tests for optional reminder creation/dismiss/clear/install path.
   - Add Go unit tests for incompatible status detection and archive-target handling.
   - Add Go unit tests for batch incompatible preflight grouping.
   - Verify frontend batch dialog outcomes: confirm queues all, decline skips conflicted, Esc/dismiss queues none.
   - Update frontend types/build as required.

## Validation Commands

- `go test ./...`
- `cd core && go test ./...`
- `cd frontend && npm run build`
- Run Wails binding generation command used by this repo if generated `frontend/wailsjs` types are stale.

## Risk Areas

- Queue state is global in `core/downloader`; tests must reset new reminder state in `resetDownloadQueueForTest`.
- Recursive dependency queueing uses a visited map; optional install-all must avoid duplicate queues while still allowing independent user actions.
- Incompatible relation resolution may require remote JAR mod-ID parsing; keep it out of search-list blocking paths where possible, or use existing async backfill style.
- `InstallModAndWait` currently waits for `!state.Active`; reminders should be exposed separately so informational messages do not keep CLI/server install waits open.

## Review Gates Before `task.py start`

- Confirm final PRD acceptance criteria match the requested batch-confirmation behavior.
- Ensure complex-task artifacts (`prd.md`, `design.md`, `implement.md`) have no placeholder sections.
