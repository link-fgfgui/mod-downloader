# Design

## Ownership And Configuration

Add `SimpleMode bool` to `configs.Preferences` with the existing TOML, JSON, and environment tag conventions. The zero value is the backward-compatible default. Expose the value through `appcore.SettingsView`, the app adapter `SettingsView`, generated Wails bindings, the Pinia settings store, and a dedicated localized Settings switch.

Use a dedicated `SaveSimpleModeSettings` request/method rather than folding this preference into network settings. Saving persists the preference, applies the runtime policy, clears disabled reminder state, emits any required queue-state refresh, and returns the canonical settings snapshot.

## Runtime Policy

`modbridge` owns the process-wide simple-mode flag because it owns the remote mod ID resolution boundary. Store the flag atomically and expose narrowly scoped configuration/query functions. `appcore.Service` applies it during startup and immediately after a settings save.

The central invariant is: while simple mode is enabled, `VersionModIDs` returns no IDs before consulting memory, database, or remote sources. This deliberately ignores old caches. `cachedRemoteModIDs` also checks the flag immediately before invoking the HTTP parser so work that was queued behind the concurrency gate cannot start after the mode changes.

Enabling the mode clears pending backfill entries. Backfill enqueue, drain execution, precise install status, and incompatible analysis all short-circuit while the flag is set. A parse already inside `parseRemoteModJarForIDs` is not canceled and completes its existing cache-entry lifecycle, including waking duplicate waiters. Those waiters re-check simple mode and do not expose the completed result while the mode remains enabled.

## Feature Behavior

`DownloadStates` returns `defaultDownloadButtonState` for every result in simple mode without scanning ID caches or scheduling backfills. The frontend therefore retains ordinary download buttons without needing a second copy of mode policy in the download store.

The downloader checks the same runtime policy before dependency preflight:

- mark the main job's dependency phase complete;
- skip dependency metadata hydration;
- do not resolve or enqueue required dependencies;
- do not create optional reminders;
- do not detect or archive incompatible dependencies;
- return an empty batch incompatible analysis;
- reject optional-dependency installation from a reminder created before the toggle.

When simple mode is enabled, downloader runtime configuration clears existing optional reminders under the queue lock and publishes the updated queue state. The normal single/batch file queue remains available.

`downloadModJob` and `tryHardlinkInstall` continue calling `VersionModIDs`; the central guard returns no IDs, selecting their established temporary-file/local-parse paths. Locally parsed primary IDs may still be persisted. They are local results, cause no remote metadata request, and become useful again if normal mode is restored.

## Cross-Layer Flow

1. Settings loads `simpleMode` through `GetSettings`.
2. The user toggles the switch; the Pinia store calls `SaveSimpleModeSettings`.
3. The Wails adapter maps the request to `appcore.Service`.
4. The service persists `preferences.simple_mode` and applies the runtime mode to `modbridge` and downloader reminder state.
5. Search state and download preflight consult the core flag; the UI receives ordinary download states and no dependency reminders.
6. Disabling the mode re-enables cache reads, remote fallbacks, backfills, and dependency behavior without a data migration.

## Concurrency And Transition Rules

The mode flag must be checked both before remote cache ownership is created and after concurrency capacity is acquired. If the second check rejects work, the owner must complete the cache entry with a disabled/error result and close its ready channel so duplicate callers cannot deadlock.

No active HTTP context is canceled on enable. This matches the approved transition contract and limits the change to preventing new network work. Switching back to normal mode allows future callers to reuse successfully completed cache data or retry a skipped entry normally.

## Compatibility And Rollback

Missing configuration fields decode to `false`; existing config files require no migration. Normal mode remains the default and retains all current behavior.

Rollback removes the setting/bindings and runtime guards. No cache or database cleanup is necessary because simple mode does not alter the persisted mod ID schema or delete cached IDs.

## Risks

- A guard only at the UI or `VersionModIDs` entry would allow queued owners to start later; the pre-parser re-check is required.
- Clearing reminder snapshots without blocking their stored install API would leave an alternate invocation path; both state and action must be gated.
- Duplicating simple-mode state in frontend, downloader, and modbridge could drift. The authoritative runtime query remains in `modbridge`; UI state is only the persisted setting projection.
- Tests that modify process-wide mode must restore normal mode to avoid cross-test contamination.
