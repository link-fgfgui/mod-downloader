# Technical Design

## Boundaries

This task keeps existing ownership boundaries:

- `frontend/src/components/` owns reusable list, version-selection, tooltip,
  and action-surface behavior.
- `frontend/src/stores/` owns shared UI state, active target scope, settings
  persistence state, and Wails call orchestration.
- Vue views own page-specific composition and lifecycle registration.
- `app.go` adapts core services to Wails and emits runtime events. It does not
  own reusable domain logic.
- `core/configs`, `core/appcore`, `core/providers`, `core/downloader`, and
  `core/storage` own validation, runtime configuration, provider requests,
  downloading, and persistence respectively.

## Shared Virtual List

`VirtualList.vue` remains the single selection and keyboard owner. It will:

1. Re-measure its Vuetify virtual scroll on mount, KeepAlive activation, and
   material item/viewport changes using a next-tick plus animation-frame
   scheduler that coalesces duplicate work.
2. Register itself with a small active-list coordinator while its page is
   active. Only one visible list receives forwarded shortcuts.
3. Ignore shortcuts originating in inputs, textareas, selects,
   contenteditable elements, or while a dialog overlay is active.
4. Prevent browser text selection only for an actual Shift range-selection
   pointer gesture.

Download and Manage will use the same flex contract (`min-height: 0`, one
scroll owner, no fixed viewport subtraction). Multi-select actions remain view
slots but use icon buttons with hover/focus tooltips. Manage locally tracks
scroll activity to defer row tooltip opening until pointer motion or one second
after scrolling stops.

## Favorites

Favorite list scope is stored on list creation and returned through existing
storage/service contracts. The store derives visible lists from the active
Minecraft/mod-loader tuple and clears selection when it leaves scope.

Grouping is removed top-down from rendering, interactions, Pinia actions,
Wails methods, service methods, and generated bindings. Existing group tables
and historical group IDs are left intact for non-destructive compatibility;
the application simply stops exposing or using them. Remaining lists use a
single ordered collection and one menu state shared by context-click and the
three-dot activator.

The existing `ModVersionList` is the provider-version source for both replacing
a local version and pinning. A reusable dialog action/composable keeps these as
separate commands. Favorites icon activation calls the same pin flow, subject
to whether the referenced item has provider project metadata.

Drag feedback uses one handle, one `effectAllowed`/`dropEffect` contract, a
custom preview, and an explicit insertion marker. Creation updates local state
once and reconciles without replacing the whole array or resetting scroll.

## Settings And Configuration

The public settings DTO is extended with downloads and API fields. Core config
normalization owns defaults and validates:

| Field | Default | Range | Runtime effect |
| --- | ---: | ---: | --- |
| `file_concurrency` | 4 | 1-32 | New file-transfer jobs |
| `concurrent_downloads` | 1 | 1-16 | Download queue worker limit |
| `requests_per_second` | 0 | 0-100 | Shared provider limiter; 0 disables |

The settings store becomes the only persistence coordinator. It keeps a last
confirmed snapshot, debounces edits, exposes pending/saving/error state, and
rolls back failed fields before showing one snackbar. API-key keep/clear
sentinels are normalized before diffing. Directory pickers remain explicit
actions; save buttons are removed.

`logging.enabled` becomes the canonical positive field. Loading accepts legacy
`logging.disabled` only when `enabled` is absent, inverts it once, and saves the
canonical key. Environment compatibility follows the same precedence. Missing
dependency-cleanup configuration defaults to false while explicit true/false
round-trips unchanged.

MCIM request configuration is traced through `SaveMCIMSettings`, client
configuration, and provider search. Mirror configuration may change metadata
base URLs but must not remove CurseForge from the provider registry or alter
required auth/query parameters.

## Home And Usage Events

Home removes current-status and bottom-status surfaces and their obsolete
subscriptions. The six usage cards format their values with three significant
digits only at 1K and above; the total keeps localized full precision.

Core usage-stat mutation points report a typed change to the app adapter. The
adapter emits one Wails event without importing Wails into core. Home performs
an initial fetch, subscribes only while activated, coalesces bursts, and removes
the manual refresh action.

## Independent Fixes

- Route animation hooks always restore opacity and transform in completion,
  cancellation, deactivation, and animation-mode changes. GSAP is labeled
  experimental and element animations are added only through the same cleanup
  lifecycle.
- Conflict payloads carry a display filename derived at the backend boundary;
  the dialog lays out the hint on a separate line.
- Settings removes the redundant cache-default input but retains actual path,
  picker, and reset actions.
- Unpin removes three filters and applies bulk unpin to the full loaded list.

## Compatibility And Rollback

- Storage migrations must be additive or non-destructive. Old favorite group
  data remains readable by older binaries.
- Legacy logging configuration is read-compatible; writing uses the new key.
- Runtime setting changes affect new work and do not cancel in-flight jobs.
- Each batch is independently testable and revertible before dependent batches
  begin. Microsoft To Do completion is delayed until verification, so rollback
  never leaves external task state falsely complete.

## Generated Contracts

Any changed public `App` method or shared Go payload requires `wails generate
module`. Generated `frontend/wailsjs` output is committed with its consumers
and verified by the frontend build.
