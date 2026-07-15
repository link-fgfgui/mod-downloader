# Connector compatibility design

## Boundaries

- `core/structs/minecraft` owns the cross-layer snapshots for an installed instance and a parsed local mod.
- `core/minecraft` owns JAR metadata source detection and loader aggregation.
- `core/appcore` owns Connector detection, the non-persistent active loader view, and selected-version events.
- `app.go` only exposes the toggle method through Wails; it does not reproduce Connector rules.
- `frontend/src/stores/minecraft.ts` remains the single Vue owner of the selected instance/tuple projection.
- `VersionChoose.vue` owns the switch command; `Manage.vue` owns presentation-only compatibility grouping.

This remains one task rather than a parent/child tree because the parser contract, selected-version state, and UI grouping form one cross-layer behavior and cannot be validated independently without temporary incompatible contracts.

## Data Contracts

Extend `structs.VersionInfo` with transient fields:

```go
ActualModLoader    string `json:"actualModLoader,omitempty"`
ConnectorAvailable bool   `json:"connectorAvailable,omitempty"`
ConnectorVirtual   bool   `json:"connectorVirtual,omitempty"`
```

- `ActualModLoader` is the loader discovered from launcher metadata and is always Forge or NeoForge for a supported Connector instance.
- `ModLoader` remains the active tuple consumed by search, provider matching, favorites/pins, download requests, dependency restoration, and local management.
- A real snapshot has `ModLoader == ActualModLoader` and `ConnectorVirtual == false`.
- A virtual snapshot keeps the same `ID`, `Name`, `MinecraftVersion`, physical path, and `Mods`, but has `ModLoader == "fabric"` and `ConnectorVirtual == true`.
- These fields are never written to configuration or storage. A version reload reconstructs them from launcher metadata and resets to the real snapshot.

Extend `structs.ModInfo` with normalized `Loaders []string`. It records loaders proven by metadata inside that physical JAR. Empty means unknown and must not be treated as incompatible.

## Local JAR Parsing

The current requested-loader parser remains authoritative for remote/download identity checks. Local scanning gains an all-loader mode:

1. Inspect `fabric.mod.json`, `META-INF/neoforge.mods.toml`, and `META-INF/mods.toml` in one pass.
2. Annotate every parsed top-level `ModInfo` with the parser's normalized loader.
3. Prefer the actual instance loader's record for display/dependency fields when the same mod ID is declared by multiple metadata formats.
4. Merge duplicate case-insensitive mod IDs and union their loader sets in stable order. Keep JIJ entries attached as display/dependency metadata; do not promote JIJ IDs into Connector detection.
5. Key the in-memory JAR cache by SHA1 plus normalized parse scope/preferred loader so an earlier Forge, NeoForge, Fabric, or all-loader parse cannot poison another scope.

`ScanModsDir` and `ScanModFile` use all-loader parsing while still indexing the physical file under the active instance tuple. Downloader and remote-JAR callers continue using requested-loader-only parsing.

## Connector State Flow

`appcore` normalizes each discovered version with `ActualModLoader = ModLoader`. After any full or incremental local-mod refresh it reconciles the selected snapshot:

1. Connector is available only when the actual loader is Forge/NeoForge and a top-level `ModInfo` has case-insensitive ID `connector` with `Enabled == true`.
2. If unavailable, force the real snapshot and clear `ConnectorVirtual`.
3. If available and the current snapshot is virtual, preserve Fabric during a local-mod refresh.
4. Explicit version selection and `RefreshVersions` always start from the real snapshot.

Add `Service.ToggleConnectorLoader() (VersionInfo, error)` and the matching `App` method. It rejects an empty selection or unavailable Connector, toggles only `ModLoader`/`ConnectorVirtual`, updates the selected-version cache, and emits the existing `selected-version-changed` event. It does not rescan, create a directory, or persist state.

Because the first default instance is not currently scanned until Manage opens, `minecraftStore.start()` performs the existing guarded `ensureSelectedModsLoaded()` once after version initialization. This makes the sidebar switch discoverable before visiting Manage without adding a second scanning path.

## Frontend Flow

`VersionChoose.vue` shows one text-and-icon tonal switch button only when `selectedVersion.connectorAvailable` is true. Its label states the destination loader (`Fabric` or the actual host), it is disabled while instance work is in progress, and clicking calls a store action backed by `ToggleConnectorLoader`.

`Manage.vue` keeps physical-file grouping, then unions `group.strong[].loaders` and partitions filtered groups:

- compatible/main: loader set is empty or contains the active `selectedModLoader`;
- incompatible: non-empty loader set does not contain the active loader.

The rendered sequence is compatible groups, one fold/unfold control row, then incompatible groups only while expanded. The existing `VirtualList` gains an item-selectability predicate so the control row cannot enter Ctrl+A, range selection, or action counts. Incompatible rows reuse the same row and action slots as normal rows. The fold resets closed when instance identity or active loader changes.

Search and enabled-state filters run before partitioning, so the folded count and expanded rows reflect the current filter. If only incompatible results remain, the fold control is shown instead of the generic empty state.

Add matching Chinese and English labels for switching and the incompatible section.

## Compatibility And Failure Behavior

- Instances without an enabled Connector retain their existing tuple and UI.
- Disabled `.jar.disabled` Connector metadata may remain visible in Manage but cannot enable the switch.
- Removing/disabling Connector during Fabric view restores the actual loader in the same selected-version update.
- Multi-loader JARs stay in the main list for either declared loader.
- Unknown loader metadata stays in the main list.
- Forge and NeoForge remain distinct host loaders; Connector does not make Forge-only and NeoForge-only mods mutually compatible.
- No database schema, config migration, new event name, or extra disk instance is introduced.

## Rollback

Rollback removes the transient struct fields, scoped JAR-cache key, all-loader local parser, toggle service/API/binding, store/button changes, and Manage partition row. There is no persisted data to migrate back. Because `core/` is a submodule, rollback must keep the core commit and parent submodule pointer synchronized.

## Validation

- Parser tests: specific-loader behavior remains unchanged; all-loader parsing classifies Fabric/Forge/NeoForge, merges a multi-loader ID, and isolates cache scopes.
- Appcore/global tests: enabled versus disabled Connector, Forge/NeoForge host eligibility, toggle round trip, event/cache consistency, removal fallback, and reset on selection/version reload.
- Adapter/bindings: regenerate Wails bindings after the public method and shared fields change.
- Frontend: lint/build plus focused logic coverage where the existing test setup permits; manually verify fold order, filtering, actions, and switch labels at narrow and desktop layouts.
