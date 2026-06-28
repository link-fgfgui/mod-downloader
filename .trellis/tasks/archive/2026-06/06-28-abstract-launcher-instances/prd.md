# Abstract launcher instance folder handling

## Goal

Refactor launcher instance folder handling so standard `.minecraft` folders and Prism Launcher `instances/` folders are resolved through a shared abstraction. The current behavior must stay compatible while making future launcher support a matter of adding another resolver instead of adding launcher-specific branches to `app.go`, `modbridge`, or other callers.

## Background

- `app.go` currently chooses between `minecraft.IsPrismInstancesDir(mcDir)` plus `loadPrismInstancesVersions(mcDir)` and `loadMinecraftDirVersions(mcDir)`.
- `app.go` owns Prism aggregation behavior even though Prism path and ID helpers live in `minecraft/prism.go`.
- `minecraft/prism.go` owns Prism detection, game directory selection, composite ID helpers, and `VersionDirPath`.
- `modbridge.selectedVersionModsDir` and `app.scanVersionMods` already depend on `minecraft.VersionDirPath`.
- Existing tests cover Prism instance aggregation, composite ID hardlink scanning, version mod scanning, and standard `.minecraft` version path resolution.

## Requirements

- Keep the existing public behavior for standard `.minecraft` folders:
  - One sidebar entry per valid `versions/<versionID>/<versionID>.json`.
  - Version ID/name semantics remain unchanged.
  - Mods path remains `<minecraftDir>/versions/<versionFolder>/mods`.
- Keep the existing public behavior for Prism `instances/` folders:
  - Detect a selected directory as Prism when it contains at least one instance-like child.
  - Treat each Prism instance as one surfaced app instance.
  - Prefer `<instance>/.minecraft` as the game directory and fall back to `<instance>` when no `.minecraft` subfolder exists.
  - Preserve composite IDs in the form `<instanceName>/<versionFolder>` for stable local-mod instance identity and path resolution.
  - Preserve sidebar display name as the Prism instance directory name.
- Move launcher-specific directory scanning and path resolution behind a shared abstraction owned by the `minecraft` package.
- Keep call sites launcher-agnostic where practical, especially `loadVersionsFromDisk`, `scanVersionMods`, hardlink scanning, and mod install target resolution.
- Make adding another launcher require implementing a new resolver/layout, not editing scattered Prism-specific branches.
- Preserve validation behavior: invalid manifests and versions missing Minecraft version or mod loader continue to be skipped.

## Acceptance Criteria

- [ ] `app.go` no longer directly branches on `minecraft.IsPrismInstancesDir` to choose Prism-specific loading logic.
- [ ] Prism-specific aggregation logic is no longer owned by `app.go`; it is represented through a `minecraft` package abstraction.
- [ ] Standard `.minecraft` tests still pass and cover the standard layout through the abstraction.
- [ ] Prism tests still pass and cover detection, game directory fallback, composite IDs, and path resolution through the abstraction.
- [ ] Hardlink index scanning and selected-version mod scanning continue resolving Prism composite IDs correctly.
- [ ] The code compiles and passes `go build ./...`, `go vet ./...`, and `go test ./...`.

## Out of Scope

- Adding support for a new launcher in this task.
- Changing the frontend labels or settings UX for choosing directories.
- Changing persisted local-mod instance IDs or migrating existing cache state.
