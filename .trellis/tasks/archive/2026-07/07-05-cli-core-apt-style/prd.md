# Adapt core for CLI apt-style requirements

## Goal

Make `core/` usable by the apt-style CLI without requiring a selected GUI
Minecraft instance, a persistent TOML config file, or a repository-local cache
file. The Wails/GUI selected-instance behavior must continue to work as the
compatibility fallback.

Source requirement: `/home/link/Documents/PROJ/mod-downloader-dev/mod-downloader-cli/docs/core-apt-style-todo.md`.

## Confirmed Facts

- `configs.Load()` currently reads `./mod-downloader.toml` and falls back to
  environment variables only when the file is absent.
- `configs.Save()` writes `./mod-downloader.toml`.
- `appcore.Service.Startup` currently calls `configs.Load()`,
  `database.Open()`, and then applies `global.SetMinecraftDir`.
- `database.Open()` currently resolves `mods.gob.zst` under the process working
  directory.
- `structs.ModDownloadRequest` currently has no explicit target directory or
  instance identity.
- `downloader.queueModDownload`, `modbridge.InstallStatus`,
  `InstallStatusPrecise`, and `DownloadStates` currently resolve install state
  through `modbridge.ApplySelectedInstance`.
- `Service.LocalMods` currently refreshes the selected version and therefore
  depends on selected GUI instance state.

## Requirements

- Add an environment-only config loading path for CLI callers and wire service
  runtime options so CLI startup can avoid TOML reads and writes.
- Add runtime options to `appcore.Options` for cache path, target mods
  directory, Minecraft version, mod loader, work dir, and no-config-file mode.
- Add database APIs for caller-controlled cache paths:
  `database.OpenAt(path string)` and `database.DefaultCachePath()`.
- Make the default metadata cache path live under `os.TempDir()` using
  `mod-downloader/mods.gob.zst`, and ensure the parent directory exists before
  saving.
- Extend `ModDownloadRequest` with explicit install target fields:
  `TargetDir` and `InstanceID`.
- Add install-target resolution that prefers explicit request target fields and
  falls back to the selected instance only when explicit fields are absent.
- Preserve `TargetDir` and `InstanceID` when queueing required dependency
  downloads.
- Add a service API to scan local mods in an arbitrary mods directory without
  requiring `global.MinecraftDir` or `global.GetSelectedVersion()`.
- Keep existing selected-instance GUI install and status behavior compatible.

## Out of Scope

- Removing GUI persistent settings APIs such as theme or saved Minecraft
  directory.
- Removing or migrating pinned-mod storage from the metadata cache.
- Changing the gob/zstd cache file format.

## Acceptance Criteria

- [x] CLI callers can construct `appcore.Service` with `NoConfigFile` and avoid
  reading `mod-downloader.toml`.
- [x] CLI callers can open the metadata cache at an explicit cache path, and the
  default cache path is under `os.TempDir()`.
- [x] Installing with `ModDownloadRequest.TargetDir` succeeds without a selected
  GUI instance when Minecraft version and mod loader are provided.
- [x] Dependency install requests keep the same target directory and instance
  identity as their parent install request.
- [x] Local mod scanning can target an arbitrary directory and populates the
  local index under an explicit instance identity.
- [x] Existing selected-instance install/status tests remain green.
- [x] Core validation passes with `go test ./...` from `core/`.

## Notes

- This is an inline Codex workflow, so `implement.jsonl` and `check.jsonl`
  curation is skipped per project Trellis rules.
