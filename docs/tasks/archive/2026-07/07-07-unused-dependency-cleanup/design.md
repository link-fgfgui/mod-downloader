# Unused dependency cleanup design

## Boundaries

- `core/minecraft` owns pure local JAR parsing, including loader-specific dependency declarations.
- `core/appcore` owns selected-instance scan orchestration and reuses existing local mod path validation/deletion APIs.
- `core/modbridge` or an appcore-local helper owns convergence between local scan data and online metadata already attached to `ModInfo`.
- `app.go` exposes adapter methods and maps core structs to Wails bindings. It must not parse JARs or delete files directly.
- Frontend owns Manage page actions, dialogs, snackbar messaging, and Settings UI.

## Data Flow

```
JAR files
  -> minecraft.ScanVersionMods / ParseModZipReader
  -> ModInfo with local IDs, SHA1, path, enabled, JiJ, local required dependency IDs
  -> appcore refresh/enrichment using cached or resolved platform metadata
  -> unused dependency scan builds selected-instance dependency graph
  -> Wails scan result
  -> Manage review dialog
  -> existing ApplyLocalModBatchOperation(delete) for confirmed cleanup
```

## Data Contracts

Add local dependency metadata to `structs/minecraft.ModInfo`:

```go
type LocalModDependency struct {
    ModID string `json:"modId"`
    Type  string `json:"type,omitempty"` // required for cleanup-relevant deps
}
```

`ModInfo` should carry `Dependencies []LocalModDependency` or an equivalently named field. The field is JAR-derived and must survive enrichment without replacing online metadata fields.

Expose a scan result shape from core/appcore through Wails:

```go
type UnusedDependencyScanRequest struct {
    ExcludedPaths []string `json:"excludedPaths,omitempty"`
}

type UnusedDependencyCandidate struct {
    Path       string   `json:"path"`
    FileName   string   `json:"fileName"`
    ModIDs     []string `json:"modIds"`
    Name       string   `json:"name,omitempty"`
    OnlineName string   `json:"onlineName,omitempty"`
    Evidence   []string `json:"evidence"`
}

type UnusedDependencyScanResult struct {
    Candidates []UnusedDependencyCandidate `json:"candidates"`
}
```

`ExcludedPaths` supports the post-delete workflow: deleted files should not be treated as dependents when the scan runs after the selected instance refresh.

## Dependency Graph Rules

- Strong install identities are top-level `ModInfo.ID` values only.
- JiJ metadata remains informational; do not count `JijMods` as installed top-level dependencies or dependents.
- A dependency edge is `dependent file -> required mod ID`.
- Disabled files do not count as dependents.
- A candidate file may declare multiple top-level mod IDs. It is considered used if any enabled non-excluded file requires any of those IDs.
- Self-dependencies and dependency IDs declared by the same file should not keep that file alive.

## Candidate Classification

The scan should be conservative:

- Positive library evidence includes normalized online category/tag `library`.
- Local evidence may include dependency-style mod IDs or metadata conventions only if implemented with tests and no broad false-positive rule.
- Ordinary mods with no library/dependency evidence are not candidates just because nothing depends on them.
- Result evidence strings should be stable, localized in the frontend where possible, or represented as reason codes if that proves cleaner during implementation.

## Settings

Add a persisted preference in `core/configs.Preferences`, defaulting to enabled:

```go
AutoScanUnusedDependencies *bool `toml:"auto_scan_unused_dependencies" json:"auto_scan_unused_dependencies"`
```

Expose a normalized boolean through `appcore.SettingsView` and `app.go.SettingsView`. Save via the existing Settings save pattern, either by adding a focused save method or extending an existing settings request if that fits better during implementation.

Manual scan is not gated by this setting.

## Manage UI

- Add a scan button near Refresh in the Manage header, using an icon button or icon+text action consistent with existing controls.
- Disable scan while a refresh, scan, or batch operation is in flight.
- After successful manual scan:
  - zero candidates: show localized snackbar;
  - candidates: open a review dialog.
- After successful delete:
  - keep existing delete success behavior;
  - if auto scan is enabled, run scan with the deleted selected paths excluded;
  - show a cleanup prompt only if candidates exist.
- Candidate cleanup confirmation calls the existing batch delete method with candidate paths.

## Compatibility

- Existing local mod scanning and online metadata enrichment stay compatible. New dependency fields are additive JSON fields.
- Existing frontend display remains valid if `dependencies` is absent.
- Existing config files default to auto scan enabled.
- Since Wails method/type surface changes, regenerate `frontend/wailsjs/go/*` bindings.

## Rollback

- The scan feature is additive. If UI issues appear, hide the scan action and automatic prompt while keeping parser changes and tests.
- If dependency parsing proves too noisy, keep the Wails API returning empty candidates until parser rules are tightened.
