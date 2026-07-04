# Design: 管理页面模组显示图标

## Overview

通过 SHA1 反查平台元数据为管理页面的本地模组填充 `iconUrl`。分两层：本地缓存命中立即可用，未命中的走 Modrinth API 异步补充后通知前端刷新。

## Data Flow

```
RefreshSelectedVersionMods
  → scanVersionMods (同步, 返回 []ModInfo)
  → 对每个 mod.SHA1 调 modbridge.PlatformMetadataForSHA1
    → 命中: 填充 iconUrl
    → 未命中: 收集 sha1 列表
  → 返回 VersionInfo (mods 已带缓存命中的 iconUrl)
  → 异步 goroutine:
    → providers.ResolveProjectsByHashes(missedSHA1s)
      → Modrinth: client.VersionFiles.GetFromHashes → 拿到 map[sha1]*Version
      → 从 Version.ProjectID 收集 projectIDs
      → client.Projects.GetMultiple(projectIDs) → 拿到 []*Project (含 IconURL)
      → 转换为 []ModProject, upsert 入数据库
    → 重新执行 SHA1→iconUrl 映射, 更新 VersionInfo.Mods
    → EventsEmit("selected-version-changed", updatedVersion)
      → 前端 store 自动 applySelectedVersion → UI 刷新
```

## Backend Changes

### 1. ModInfo struct 增加字段

`structs/minecraft/modinfo.go`:
```go
type ModInfo struct {
    // ... existing fields ...
    IconURL string `json:"iconUrl,omitempty"`
}
```

### 2. providers 层新增 hash→project 查询

`providers/modprovider.go` 新增方法（和现有 `searchExactMod`, `projectToModProject` 同级）:

```go
func (p modrinthProvider) resolveProjectsByHashes(hashes []string) (map[string]models.ModProject, error)
```

逻辑：
1. `client.VersionFiles.GetFromHashes(hashes, "sha1")` → `map[sha1]*Version`
2. 收集去重的 `ProjectID` 列表
3. `client.Projects.GetMultiple(projectIDs)` → `[]*Project`
4. 用现有的 `projectToModProject` 转换每个 project
5. 建立 `sha1 → ModProject` 映射返回

**不直接调 SDK**：通过已有的 `modrinthProvider` 方法和转换函数来做，和现有搜索流程一致。

### 3. providers/service.go 导出入口

```go
func ResolveProjectsByHashes(hashes []string) map[string]models.ModProject
```

逻辑：
1. 调用 `modrinthProvider{}.resolveProjectsByHashes(hashes)`
2. 对返回的每个 `ModProject` 调用 `database.UpsertModPlatform(project)` 缓存（复用现有的 `cacheSearchResults` 模式）
3. 返回 `sha1 → ModProject` 映射

### 4. app.go 异步补充 icon

在 `RefreshSelectedVersionMods` 中：
1. 同步阶段：`scanVersionMods` 后，遍历 mods 调 `PlatformMetadataForSHA1`，命中的立即填 `iconUrl`
2. 收集未命中的 SHA1 列表
3. 如果有未命中的，启 goroutine：
   - 调 `providers.ResolveProjectsByHashes(missedSHA1s)`
   - 用结果补充 mods 的 iconUrl
   - 更新 global state 和 emit event

## Frontend Changes

### Manage.vue

`<template #prepend>` 部分改为条件渲染：

```vue
<v-avatar color="surface-container-high" rounded="lg" size="48">
    <v-img v-if="group.primary.iconUrl" :src="group.primary.iconUrl" />
    <v-icon v-else icon="mdi-package-variant" color="on-surface-variant" />
</v-avatar>
```

与 `SearchResultList.vue` 的 icon 渲染模式一致。

## Boundaries

- **仅 Modrinth**：CurseForge 的 fingerprint API 使用 Murmur2 而非 SHA1，本期不实现。但本地缓存层（`PlatformMetadataForSHA1`）已覆盖 CurseForge 缓存命中的情况。
- **不阻塞 UI**：API 查询全在异步 goroutine 里，首次打开立即显示缓存命中的 icon + 占位 icon，API 返回后自动刷新。
- **错误静默**：API 查询失败不影响页面，只 log 错误，保持占位图标。
