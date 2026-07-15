# Connector Compatibility

## Scenario: Transient Fabric View For Connector Instances

### 1. Scope / Trigger

Use this contract when changing installed-version selection, local JAR parsing/cache keys, selected-instance refresh, loader-scoped download behavior, or Manage compatibility grouping. An enabled top-level mod ID `connector` on a Forge/NeoForge instance allows one physical instance to expose its real host tuple and a transient Fabric tuple.

### 2. Signatures

```go
type VersionInfo struct {
    ID                 string
    Name               string
    MinecraftVersion   string
    ModLoader          string
    ActualModLoader    string
    ConnectorAvailable bool
    ConnectorVirtual   bool
    Mods               []ModInfo
}

type ModInfo struct {
    ID      string
    Loaders []string
    Enabled bool
}

func ParseModZipReader(r *zip.Reader, sourceName, modLoader string) []ModInfo
func ParseLocalModZipReader(r *zip.Reader, sourceName, preferredLoader string) []ModInfo
func (s *appcore.Service) ToggleConnectorLoader() (VersionInfo, error)
func (a *App) ToggleConnectorLoader() (VersionInfo, error)
```

The frontend calls generated `ToggleConnectorLoader(): Promise<VersionInfo>` and consumes the existing `selected-version-changed` event.

### 3. Contracts

- `ActualModLoader` is normalized launcher metadata. `ModLoader` is the active tuple used by providers, downloads, pins/favorites, and local workflows.
- A virtual view keeps the same ID, name, Minecraft version, mods, and directory; only `ModLoader="fabric"` and `ConnectorVirtual=true` differ.
- Connector is available only for an enabled top-level case-insensitive ID `connector` whose actual loader is `forge` or `neoforge`. Disabled JARs and JIJ IDs do not activate it.
- Connector fields are process memory only. Version reload and explicit instance selection restore `ModLoader=ActualModLoader`; local-mod refresh preserves Fabric only while Connector remains enabled.
- Requested-loader parsing stays strict for remote/download identity. Local scans inspect Fabric, Forge, and NeoForge metadata, prefer the actual loader's record for duplicate IDs, and union normalized `ModInfo.Loaders`.
- JAR cache identity is SHA1 plus parse scope/preferred loader. A requested Forge result must never satisfy a local all-loader or Fabric lookup.
- Manage treats a physical group as incompatible only when its non-empty loader union excludes the active loader. Empty/unknown and multi-loader groups remain in the main list; incompatible groups form a default-collapsed tail.

### 4. Validation & Error Matrix

| Condition | Behavior |
| --- | --- |
| No selected instance | Toggle returns `no selected version` |
| No enabled Connector or unsupported host | Toggle returns `connector unavailable` |
| Enabled Connector on Forge/NeoForge | Toggle real host to Fabric and back |
| Connector disabled/removed during Fabric view | Same refresh restores actual loader and hides switch |
| Explicit reselect/version reload | Restore actual loader without persistence |
| JAR declares multiple loader metadata files | Return one case-insensitive mod ID with unioned loaders |
| Loader metadata unknown | Keep the local group in the main Manage list |

### 5. Good/Base/Bad Cases

- Good: a NeoForge instance with enabled Connector switches to Fabric while retaining the same target mods directory; Fabric provider versions then match and install into that directory.
- Good: a universal JAR with Fabric and NeoForge metadata stays in the main list under both views.
- Base: a disabled `connector.jar.disabled` remains manageable as a local file but does not show the switch.
- Base: a normal Fabric/Forge/NeoForge instance preserves prior selection and list behavior.
- Bad: create a second disk instance ID or persist the virtual loader in TOML/SQLite.
- Bad: key parsed local metadata only by SHA1, because the first loader-specific parse then poisons later scopes.
- Bad: classify unknown loader metadata as incompatible.

### 6. Tests Required

- Parser: strict requested-loader behavior, all-loader Fabric/Forge/NeoForge detection, duplicate-ID merge order, loader union, and cache-scope isolation.
- Appcore: enabled/disabled Connector, Forge/NeoForge eligibility, toggle round trip, selected cache/event consistency, physical identity preservation, reselect reset, and removal fallback.
- Wails: regenerate bindings after method or shared-field changes.
- Frontend: lint/type-check/build; verify switch destination labels, tuple propagation, compatible-first order, default fold, filtered counts, and existing actions after expansion.
- Shared `VirtualList`: section/control rows must be excluded from click selection, range selection, Ctrl+A, and action counts.

### 7. Wrong vs Correct

Wrong:

```go
// Reuses a loader-specific result for every later interpretation of this JAR.
global.SetJarMetadata(sha1, mods)
virtual.ID = selected.ID + "-fabric" // invents a second physical instance
```

Correct:

```go
global.SetJarMetadata(sha1, "loader:"+loader, mods)
virtual := selected
virtual.ModLoader = "fabric"
virtual.ConnectorVirtual = true // ID and physical path stay unchanged
```
