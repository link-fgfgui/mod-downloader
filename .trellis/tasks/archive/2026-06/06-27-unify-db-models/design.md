# Design: Unify database layer with models types

## Type Mapping

| Old (database pkg) | New (models pkg) | Field changes |
|---|---|---|
| `ModPlatform` | `models.ModProject` | `Name→Title`, `McmodURL→IconURL`, added `Icon`/`Downloads`/`ID` |
| `ModPlatformVersion` | `models.ModVersion` | `VersionID` stays (was separate from `ID`; in ModVersion both exist) |
| `database.ModDependency` | `models.ModDependency` | `DependencyType` JSON tag `dependencyType→type` |

## cacheState Changes

```go
type cacheState struct {
    Version                int
    JarMetadataVersion     string
    ModPlatforms           map[platformKey]models.ModProject      // was ModPlatform
    PlatformAssociations   map[string]PlatformAssociation         // unchanged
    PlatformVersions       map[versionKey]models.ModVersion       // was ModPlatformVersion
    PlatformVersionScopes  map[versionScopeKey]storedVersionScope  // unchanged
    PinnedMods             map[pinnedModKey]PinnedMod              // unchanged
    JarMetadata            map[string][]structs.ModInfo            // unchanged
    PlatformVersionKeyByID map[string]versionKey                   // unchanged (lookup index)
}
```

## API Signature Changes (database pkg)

| Function | Before | After |
|---|---|---|
| `UpsertModPlatform` | `(ModPlatform) error` | `(models.ModProject) error` |
| `GetModPlatform` | `(string,string) (ModPlatform,bool)` | `(string,string) (models.ModProject,bool)` |
| `GetModPlatformBySlug` | `(string,string) (ModPlatform,bool)` | `(string,string) (models.ModProject,bool)` |
| `TouchModPlatform` | unchanged (only uses platform+projectID+timestamp) | same but writes into ModProject |
| `SetPlatformVersions` | `(string,string,[]ModPlatformVersion) error` | `(string,string,[]models.ModVersion) error` |
| `SetPlatformVersionSnapshot` | `(string,string,[]ModPlatformVersion,int64,[]ModPlatformVersionScope) error` | `(string,string,[]models.ModVersion,int64,[]ModPlatformVersionScope) error` |
| `GetPlatformVersions` | `(string,string) ([]ModPlatformVersion,error)` | `(string,string) ([]models.ModVersion,error)` |
| `GetLatestProjectBySHA1` | `(string,string) (ModPlatformVersion,bool)` | `(string,string) (models.ModVersion,bool)` |
| `SetVersionDependencies` | `(string,[]ModDependency) error` | `(string,[]models.ModDependency) error` |
| `GetVersionDependencies` | `(string) ([]ModDependency,error)` | `(string) ([]models.ModDependency,error)` |

## gob Compatibility

gob 按字段名匹配。`ModPlatform.Name` → `ModProject.Title` 意味着旧 gob 数据中的 `Name` 字段无法反序列化到 `Title`。

策略：递增 `cacheVersion` 到 2。`loadCacheState` 在读取后检查 version，如果不匹配则丢弃旧状态，返回 `newCacheState()`。缓存是可丢失的加速层，丢弃是安全的。

## Internal Helper Adjustments

- `copyVersion` → 改为操作 `models.ModVersion`
- `copyDependencies` → 改为操作 `models.ModDependency`
- `savePlatformVersion` → 内部直接操作 `models.ModVersion`，使用 `VersionID` 字段（ModVersion 有 `VersionID` 字段）
- `normalizeDependencies` → 操作 `models.ModDependency`，注意字段名差异：`DependencyType` (不变)

## providers 层清理

1. 删除 `bridge.go`
2. `cache.go`：`GetProjectByID` 直接返回 database 的 `models.ModProject`，不再转换；`StoreProject` 直接传入
3. `modprovider.go`：`saveProjectVersionsSnapshot` 直接传 `[]models.ModVersion` 给 database；删除 `projectVersionResultsToDB` / `dbProjectVersionsToResults` / `projectDependenciesToDB` / `dbDependenciesToResults`

## Risks

- gob 序列化的类型名变了（`database.ModPlatform` → `models.ModProject`），旧文件反序列化会失败 → `cacheVersion` 升级自动处理
- `ModVersion.ID` 和 `ModVersion.VersionID` 在 database 内部有不同语义（`ID` 是内部 UUID，`VersionID` 是平台版本号）→ 需要在 `savePlatformVersion` 中保持这个区分
