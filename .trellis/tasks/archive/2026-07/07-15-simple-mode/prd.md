# Add Simple Mode

## Goal

Add a persisted, user-visible simple mode that eliminates remote JAR mod ID resolution and presents a predictable reduced download workflow for users who prefer lower metadata traffic over automatic dependency and conflict handling.

## Background

- Remote resolution is centralized in `VersionModIDs`, which reads in-memory and persisted IDs before falling back to HTTP Range parsing of a remote JAR (`core/modbridge/modbridge.go:552`).
- Search-result status rendering schedules asynchronous remote backfills on a cache miss (`core/modbridge/modbridge.go:269`, `core/modbridge/modbridge.go:525`), producing metadata-loading/failed UI states (`frontend/src/stores/downloadSearch.ts:117`).
- Install-time precise status and incompatible-mod analysis use remote-capable ID resolution (`core/modbridge/modbridge.go:287`, `core/modbridge/modbridge.go:333`) and drive update/conflict/incompatible confirmations (`frontend/src/stores/downloadSearch.ts:195`).
- Required-dependency duplicate avoidance and optional-dependency installed-state detection use precise status (`core/downloader/download.go:879`, `core/downloader/download.go:946`).
- Basic downloads already have a no-remote-ID fallback: download or hardlink to a temporary path, parse the local JAR, then perform local replacement decisions (`core/downloader/download.go:681`, `core/downloader/download.go:698`, `core/downloader/download.go:1110`).
- The settings contract crosses `configs.Preferences`, `appcore.SettingsView`, the Wails adapter, generated bindings, the Pinia settings store, and Settings UI (`core/configs/structs.go:149`, `core/appcore/contracts.go:69`, `frontend/src/views/Settings.vue:103`).

## Requirements

- Add a localized Settings switch named "Simple mode" and persist it through the existing preferences configuration. It defaults to disabled for backward compatibility.
- Apply mode changes immediately without restarting the application.
- While enabled, prevent all new synchronous and asynchronous remote mod ID resolution, including search backfills, download preflight, dependency handling, and direct core callers.
- Enforce the policy at the core resolution boundary as well as feature entry points so alternate callers cannot bypass it.
- Ignore in-memory and persisted remote mod ID caches for simple-mode status and preflight behavior. Cache history must not change the reduced feature set.
- Present ordinary downloadable search-result states instead of installed, update, conflict, incompatible, metadata-loading, or metadata-failed states.
- Disable automatic required-dependency installation, optional-dependency reminders/actions, and incompatible-dependency detection. Users manage dependencies manually in simple mode.
- Clear existing optional-dependency reminders when simple mode is enabled and reject stale optional-dependency install actions.
- Keep basic single and batch downloads available. Continue parsing a JAR locally after it has actually been downloaded or hardlinked, including local replacement/archive handling based on that local parse.
- A remote parse that entered the HTTP parsing function before simple mode was enabled may finish. Work waiting for capacity and all later work must observe the new mode before starting a remote request.
- Preserve current status, dependency, conflict, and remote resolution behavior while simple mode is disabled.

## Acceptance Criteria

- [x] The Settings switch is localized in Chinese and English, defaults to off, persists across restart, and takes effect immediately.
- [x] No new remote mod ID HTTP request starts while simple mode is enabled, including work already queued but not yet parsing.
- [x] An already-running remote parse may finish without deadlock; waiters are released or skipped cleanly after the mode changes.
- [x] Existing memory/database mod ID cache contents do not alter simple-mode status or preflight results.
- [x] Search results retain an enabled ordinary download action and never show advanced or remote-metadata states in simple mode.
- [x] Single and batch download paths do not open update/conflict/incompatible confirmations derived from remote mod IDs in simple mode.
- [x] Required dependencies are not auto-enqueued, optional reminders/actions are absent, and incompatible dependencies are not analyzed or archived in simple mode.
- [x] Enabling simple mode clears any existing optional-dependency reminders and prevents their stored actions from being invoked.
- [x] A basic download completes through the existing local-parse fallback and may persist locally parsed IDs for later normal-mode use; those IDs remain ignored while simple mode stays enabled.
- [x] Disabling simple mode restores existing remote resolution and dependent behavior without clearing valid caches.
- [x] Automated tests cover configuration round trips, runtime toggles, central resolution gating, queued/in-flight behavior, backfill suppression, cache ignoring, dependency gating, reminder cleanup, local-parse fallback, and normal-mode compatibility.

## Out of Scope

- Replacing remote mod ID resolution with another resolver.
- Canceling an HTTP Range parse that was already running when the setting changed.
- Clearing valid persisted mod ID caches when entering simple mode.
- Disabling local JAR parsing for downloaded, hardlinked, or otherwise local files.
- Changing normal-mode semantics beyond the new mode gate.
