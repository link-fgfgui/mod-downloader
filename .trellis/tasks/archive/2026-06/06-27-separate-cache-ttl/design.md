# 设计：分离项目元数据与文件列表缓存 TTL

## 现状分析

### 数据层（database/）
- `ModPlatforms map[platformKey]ModProject`：存项目元数据，但实际只存 platform、projectID、slug 等最小信息
- `ModProject.UpdatedAt`：当前用于**版本快照 TTL** 判断（15min），不是元数据本身的缓存时间
- `PlatformVersions`：存版本/文件数据
- `SetPlatformVersionSnapshot` 先 `deleteProjectVersions` 再写入新版本，存在短暂空白

### Service 层（providers/）
- 搜索（`ExactSearch`/`Search`）直接从 API 获取 `ModProject`，**不缓存完整元数据**
- `saveProjectVersionsSnapshot` 只存最小 `ModProject`（platform、projectID、slug）
- `listProjectVersionsWithFilter` 通过 `getFreshProjectVersionsSnapshot` 检查版本 TTL

## 设计方案

### 1. ModProject 增加元数据缓存时间戳

在 `models.ModProject` 增加 `CachedAt int64` 字段：

```go
type ModProject struct {
    // ... 现有字段 ...
    CachedAt int64 `json:"cachedAt"` // 元数据从 API 获取的时间戳（Unix 秒）
}
```

- Gob 反序列化兼容：旧缓存文件中缺少此字段会默认为 0，视为需要刷新
- 不需要 bump `cacheVersion`

### 2. 两个独立 TTL 常量

```go
// providers/modprovider.go
const (
    projectMetadataTTL         = 30 * 24 * time.Hour  // 30 天
    projectVersionsSnapshotTTL = 15 * time.Minute     // 15 分钟（已有）
)
```

### 3. 搜索时缓存完整元数据

搜索结果包含完整元数据（标题、图标、描述、下载数）。在 service 层搜索完成后，将结果存入 `ModPlatforms`：

- `UpsertModPlatform` 时设置 `CachedAt = time.Now().Unix()`
- 仅当传入的 `ModProject` 包含实质内容（Title 非空）时更新 `CachedAt`
- 最小信息 upsert（只有 platform + projectID + slug，来自版本快照保存）不更新 `CachedAt`

### 4. 元数据过期检查与刷新

新增 `database.GetModPlatformIfFresh(platform, projectID string, ttl time.Duration) (ModProject, bool)`：
- 返回缓存的 `ModProject`，第二个值表示是否在 TTL 内
- 调用方根据返回值决定是否需要从 API 刷新

在 `listProjectVersionsWithFilter` 中：
- 获取版本列表时，顺便检查元数据是否过期
- 如果元数据过期（>30 天），在获取版本列表的同时静默刷新元数据
- 元数据刷新失败时继续使用旧数据

### 5. 版本列表增量更新

修改 `SetPlatformVersionSnapshot`：
- **不再 `deleteProjectVersions`**，直接 upsert 新版本
- `savePlatformVersion` 已经是 upsert 语义（检查 existing），保持不变
- 对于 scope 刷新也不再 `deletePlatformVersionSnapshotScopes`
- 版本范围时间戳（`PlatformVersionScopes`）照常更新

tradeoff：已从平台删除的版本会残留在缓存中。对 Minecraft mod 平台来说这是可接受的，因为版本几乎只增不删。

### 6. `UpdatedAt` 语义不变

`ModProject.UpdatedAt` 继续作为版本快照时间戳使用，与 `CachedAt`（元数据缓存时间）独立。

## 影响范围

| 文件 | 改动 |
|------|------|
| `models/models.go` | 增加 `CachedAt` 字段 |
| `database/mods.go` | `UpsertModPlatform` 区分完整/最小 upsert；新增 TTL 查询函数 |
| `providers/modprovider.go` | 增加 `projectMetadataTTL` 常量；搜索结果缓存；元数据刷新逻辑 |
| `providers/service.go` | 搜索完成后调用 `UpsertModPlatform` 存完整元数据 |

## 兼容性

- 旧缓存文件：`CachedAt = 0` → 视为过期，首次访问时刷新
- 不改变 `cacheVersion`
- 前端无需改动
