# SQLite user data storage

## Goal

Persist user-owned mod state in SQLite while keeping TOML limited to app configuration and keeping the existing platform metadata cache as rebuildable cache data. Users must be able to choose where cache/runtime data is stored.

## Background

- `core/database` currently stores platform metadata, pinned mod versions, favorite lists, and favorite mods together in `mods.gob.zst`.
- `core/configs.RuntimeConfig` already has `cache_dir` / `MOD_DOWNLOADER_CACHE_DIR`; `appcore.Service.cachePath()` honors it, but the Settings UI does not expose it.
- The public pin and favorites APIs are already used by Wails, `appcore`, `httpserver`, `modbridge`, and Vue stores. Their behavior should remain stable while the storage backend changes.
- TOML config must continue to store settings only: API keys, theme/animation/Minecraft dir/runtime cache settings. It must not become the storage for pins, favorites, or other user collections.

## Requirements

- R1: Add SQLite storage for user-owned mod data, including pinned mod versions, favorite lists, and favorite mods.
- R2: Keep platform metadata and version cache data in the existing gob/zstd cache file so API cache behavior remains rebuildable and separate from user data.
- R3: Preserve existing database package APIs used by `appcore`, `httpserver`, `modbridge`, and Wails adapters unless a signature change is necessary.
- R4: Migrate legacy pinned mods and favorites from an existing `mods.gob.zst` cache into SQLite on open when legacy records are present.
- R5: Do not store pins or favorites in `mod-downloader.toml`; keep TOML responsibilities unchanged except for the existing runtime cache-dir setting.
- R6: Expose cache directory selection through the Settings flow, persisting the chosen directory to `runtime.cache_dir` and reopening storage at the new path.
- R7: Keep env/runtime overrides working: explicit `Runtime.CachePath` wins, then `Runtime.CacheDir`, then TOML `runtime.cache_dir`, then the default cache location.
- R8: Preserve normalizations and edge cases: platform/mod ID/loader lowercasing, stable pinned list sort, favorite list sort, duplicate favorite upsert behavior, list delete cascade, and returned-copy behavior.

## Acceptance Criteria

- [x] Pins survive close/reopen through SQLite and are not stored in active gob state.
- [x] Favorite lists and mods survive close/reopen through SQLite, including rename, duplicate update, delete cascade, sort order, and returned-copy behavior.
- [x] Existing legacy gob data for pins/favorites is migrated to SQLite without discarding platform metadata cache.
- [x] Existing TOML config load/save tests still pass and no pin/favorite data is written to TOML.
- [x] Settings view displays cache directory information and lets users choose/reset a custom cache directory.
- [x] Changing cache directory persists to TOML and causes backend storage to reopen at the resolved path.
- [x] Wails bindings and frontend Settings store/view are updated for the new settings fields/methods.
- [x] Core and app tests pass for affected packages; frontend build passes after binding changes.

## Out of Scope

- Replacing the platform metadata cache with SQLite.
- Adding multi-profile sync or cloud storage.
- Designing a full database migration framework beyond the SQLite schema required for this task.
- Moving Minecraft directory, API keys, theme, or animation settings out of TOML.
