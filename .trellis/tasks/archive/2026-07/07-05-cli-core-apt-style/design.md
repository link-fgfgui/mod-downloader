# Design: CLI apt-style core support

## Boundaries

- Keep all reusable behavior inside `core/`.
- Do not add Wails runtime imports to `appcore`, `httpserver`, or lower layers.
- Keep shared request/response types in existing `structs` and `models`
  packages.
- Preserve GUI selected-instance behavior as fallback behavior, not as a hard
  dependency for CLI flows.

## Configuration And Startup

Add `configs.LoadEnv()` as an explicit environment-only loader. Keep
`configs.Load()` as the GUI-compatible TOML-plus-env loader.

Add `appcore.RuntimeOptions` to `appcore.Options`:

```go
type RuntimeOptions struct {
    WorkDir          string
    TargetModsDir    string
    MinecraftVersion string
    ModLoader        string
    CacheDir         string
    CachePath        string
    NoConfigFile     bool
}
```

`Service.Startup` uses `configs.LoadEnv()` when `NoConfigFile` is true. It opens
the database with `database.OpenAt` using `CachePath`, or `CacheDir` joined with
the standard cache file name, or `database.DefaultCachePath()`. It does not set
`global.MinecraftDir` unless a config/override actually provides a Minecraft
root.

## Cache Path

Add `database.DefaultCachePath() (string, error)` returning:

```go
filepath.Join(os.TempDir(), "mod-downloader", "mods.gob.zst")
```

Add `database.OpenAt(path string) error`. `database.Open()` becomes a
compatibility wrapper around `DefaultCachePath` and `OpenAt`. `cacheDB.save`
creates the parent directory before writing.

## Install Target Resolution

Extend `structs.ModDownloadRequest` with:

```go
TargetDir  string `json:"targetDir,omitempty"`
InstanceID string `json:"instanceId,omitempty"`
```

Add `modbridge.ResolveInstallTarget(req)` that:

1. Uses explicit `TargetDir` when present.
2. Uses explicit `InstanceID` when present.
3. Derives `InstanceID` as `cli:` plus the absolute cleaned target path when
   explicit target dir is present and instance ID is absent.
4. Requires Minecraft version and mod loader for explicit target installs.
5. Falls back to `ApplySelectedInstance` when no explicit target dir is present.

Downloader install flow calls `ResolveInstallTarget`. Dependency request
construction copies target fields from the parent request.

## Local Mods In Directory

Add `Service.LocalModsInDir(dir, minecraftVersion, modLoader string)`:

- Clean and validate the explicit directory.
- Derive instance ID from the directory using the same CLI identity convention.
- Clear that instance from the local index.
- Scan the directory directly with `minecraft.ScanModsDir`.
- Enrich icons the same way selected-instance local mods do.

This avoids `global.MinecraftDir` and selected-version state for CLI listing.

## Compatibility

- Existing `ApplySelectedInstance`, `LocalMods`, and selected-version APIs remain
  available for GUI callers.
- Existing request shapes continue to work because new fields are optional.
- `Shutdown()` remains the persistent GUI save path; CLI callers should use
  `Close()`.
