# Design

## Architecture

Keep `core/database` as the storage boundary and preserve its public API. Split the implementation behind that API:

- `mods.gob.zst`: rebuildable platform metadata cache (`ModPlatforms`, associations, versions, version scopes, version mod IDs).
- `user-data.sqlite`: user-owned data (`PinnedMod`, `FavoriteList`, `FavoriteMod`).

`database.OpenAt(cachePath)` remains the primary open call. It resolves and opens the gob cache at `cachePath`, then opens SQLite in the same directory using a fixed user-data filename. `database.Close()` saves the gob cache and closes SQLite.

## Data Flow

Startup:

1. `appcore.Service.Startup` loads TOML/env config.
2. `Service.cachePath()` resolves cache path using runtime override, cache dir, TOML cache dir, or default.
3. `database.OpenAt(cachePath)` loads gob platform cache, opens SQLite user DB, ensures schema, and migrates any legacy user rows from gob state.

Runtime:

- Platform lookup/write methods continue to use gob-backed `cacheState`.
- Pin and favorite methods use SQLite queries.
- Callers in `appcore`, `httpserver`, `modbridge`, and Wails remain storage-agnostic.

Settings:

1. `GetSettings` returns the configured cache dir and resolved cache path.
2. `ChooseCacheDir` / save-cache-dir flow persists `config.Runtime.CacheDir` in TOML.
3. The service reopens `database` at the new cache path so the change applies without waiting for restart.

## SQLite Schema

Use a small schema with normalized uniqueness matching the current in-memory keys:

- `schema_migrations(version integer primary key, applied_at integer not null)`
- `pinned_mods(id text primary key, platform text not null, mod_id text not null, version_id text not null, minecraft_version text not null, mod_loader text not null, unique(platform, mod_id, minecraft_version, mod_loader))`
- `favorite_lists(id text primary key, name text not null, created_at integer not null, updated_at integer not null, sort_order integer not null, system integer not null default 0)`
- `favorite_mods(id text primary key, list_id text not null references favorite_lists(id) on delete cascade, platform text not null, mod_id text not null, version_id text, minecraft_version text, mod_loader text, title text, slug text, icon_url text, description text, categories_json text, created_at integer not null, updated_at integer not null, unique(list_id, platform, mod_id, minecraft_version, mod_loader))`

`categories_json` stores the normalized category string slice as JSON. This keeps the schema simple and avoids a join table for UI display metadata.

## Migration

Legacy fields remain in `cacheState` long enough to decode old gob files. After SQLite opens:

- Upsert legacy pins into SQLite.
- Insert legacy favorite lists and mods into SQLite without overwriting already-existing SQLite rows.
- Clear legacy user maps from in-memory cache state so the next gob save no longer contains user data.

Do not bump `cacheVersion` for this migration, because bumping would discard the old gob before user data can be migrated.

## Compatibility

- Existing public pin/favorite methods and JSON shapes remain unchanged.
- Config stays TOML-based and retains `runtime.cache_dir`.
- `Runtime.CachePath` remains a precise test/CLI override. When it is used, the SQLite file lives next to that explicit cache path.
- If SQLite open fails, database open should fail so callers do not silently lose user-data writes.

## Trade-Offs

- SQLite is added only for user data now, not for platform cache. This avoids a broad rewrite of platform cache queries and preserves the current low-risk rebuildable-cache model.
- Keeping legacy fields in `cacheState` is temporary compatibility debt, but it is necessary to migrate existing user data safely.
- `user-data.sqlite` follows the cache directory for now because the existing app has only one runtime data-location setting. A separate user-data directory can be added later if product requirements split "cache" from "profile data".
