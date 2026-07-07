# Separate GUI and CLI default cache directories

## Goal

Make the Wails GUI use the current working directory as its default cache directory while preserving the core/CLI default of using the OS temp directory.

This keeps GUI data colocated with the app's existing `pwd/mod-downloader.toml` config behavior, without changing the reusable core default that CLI callers rely on.

## Confirmed Facts

- `core/configs.AppConfigPath()` already loads/saves `mod-downloader.toml` from `pwd`.
- `core/database.DefaultCachePath()` currently returns `<os.TempDir()>/mod-downloader/mods.gob.zst`; this is the core default used when no runtime or config cache path is provided.
- Before this task, `core/appcore.Service.cachePath()` honored `Runtime.CachePath` first, then `Runtime.CacheDir`, then config `runtime.cache_dir`, and finally `database.DefaultCachePath()`.
- Before this task, `app.go` created the Wails service without a GUI-specific cache directory, so the GUI fell through to the core temp-dir default when no preference was configured.

## Requirements

- GUI startup must default to a cache directory rooted at the process current working directory when no explicit cache dir/path preference is set.
- CLI/core behavior must continue to default to the OS temp directory through `database.DefaultCachePath()`.
- Explicit runtime cache path, runtime cache dir, TOML `runtime.cache_dir`, and `MOD_DOWNLOADER_CACHE_DIR` behavior must remain intact.
- Public Wails API signatures must not change.
- Cache file naming must continue to use `database.CacheFileName` (`mods.gob.zst`) and the paired SQLite user data file must remain next to it.

## Acceptance Criteria

- [x] Starting the Wails app service with no configured cache dir resolves the settings cache path to `<pwd>/mods.gob.zst`.
- [x] `database.DefaultCachePath()` still resolves under `os.TempDir()` for core/CLI callers.
- [x] A configured cache directory still wins over the GUI default.
- [x] Existing Go tests pass for the app and core packages.

## Notes

- Lightweight task; PRD-only is sufficient.
