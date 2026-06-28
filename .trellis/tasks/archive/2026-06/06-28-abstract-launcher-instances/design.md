# Design: Launcher Instance Folder Abstraction

## Boundary

The `minecraft` package should own launcher directory layout knowledge because it already owns manifest inspection, local JAR parsing, Prism path helpers, and version directory resolution. Higher-level packages should ask `minecraft` for recognized instances and paths instead of checking launcher-specific markers directly.

`app.go` remains responsible for app state, global cache updates, selected-version behavior, Wails events, and validation policy that is app-specific.

## Proposed Abstraction

Introduce a launcher layout/resolver concept in `minecraft`, for example:

- `type InstanceLayout interface`
  - `Name() string`
  - `Matches(root string) bool`
  - `LoadVersions(root string, loadGameDir func(gameDir string) []VersionInfo) []VersionInfo`
  - `VersionDir(root string, version VersionInfo) string`

or the equivalent using concrete resolver functions if that fits the code better after implementation.

The abstraction should keep these contracts:

- Standard layout:
  - Matches as the fallback.
  - Loads versions from `<root>/versions`.
  - Resolves version directories as `<root>/versions/<folder>`.
- Prism layout:
  - Matches when the root contains at least one Prism-like instance child.
  - Loads each child instance's game directory, selecting the first valid internal version.
  - Rewrites `VersionInfo.ID` to the existing composite ID format.
  - Rewrites `VersionInfo.Name` to the instance folder name.
  - Resolves composite IDs through `<root>/<instance>/{.minecraft or root}/versions/<folder>`.

## Data Flow

1. `app.loadVersionsFromDisk(mcDir)` calls a launcher-agnostic `minecraft.LoadLauncherVersions` or similar helper.
2. The helper selects the first matching non-standard layout, otherwise standard fallback.
3. The helper uses manifest loading logic for ordinary game directories. If manifest validation currently needs app-level policy, pass a callback to keep the policy local to `app.go`; otherwise move the full scan into `minecraft`.
4. `app.scanVersionMods`, hardlink scanning, and `modbridge.selectedVersionModsDir` continue to call one launcher-agnostic path resolver such as `minecraft.VersionDirPath`.

## Compatibility

- Do not change `VersionInfo` fields or JSON shape.
- Do not change Prism composite ID format.
- Do not remove existing exported helpers unless all callers and tests are updated in the same task.
- Keep existing tests as regression tests; add or adjust tests around the new abstraction instead of only testing private implementation details.

## Trade-Offs

- A small interface/function abstraction is justified because current Prism logic is already split across `app.go` and `minecraft/prism.go`, and future launchers would otherwise add more top-level branches.
- Avoid over-modeling launcher metadata that the app does not use yet. The minimum useful contract is: detect root, enumerate app instances, resolve selected instance version directory.

## Rollback

Rollback should be straightforward because behavior-preserving refactor should not change external data. If tests reveal path or ID regressions, revert the abstraction changes and keep existing Prism helpers intact.
