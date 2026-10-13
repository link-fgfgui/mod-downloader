# 实现计划：分离项目元数据与文件列表缓存 TTL

## 实现步骤

### 1. ModProject 增加 CachedAt 字段
- [ ] `models/models.go`：`ModProject` 增加 `CachedAt int64 \`json:"cachedAt"\``

### 2. database 层改动
- [ ] `database/mods.go`：`UpsertModPlatform` 区分完整元数据 upsert 和最小 upsert
  - 当 `p.Title != ""` 时（完整元数据），设置 `p.CachedAt = time.Now().Unix()`
  - 当 `p.Title == ""` 时（最小 upsert，来自版本快照保存），保留 existing 的所有元数据字段，只更新 platform/projectID/slug
- [ ] `database/mods.go`：`SetPlatformVersionSnapshot` 移除 `deleteProjectVersions` 和 `deletePlatformVersionSnapshotScopes` 调用，改为纯 upsert

### 3. providers 层改动
- [ ] `providers/modprovider.go`：增加 `projectMetadataTTL = 30 * 24 * time.Hour`
- [ ] `providers/modprovider.go`：搜索结果转换方法 `modToModProject` / `projectToModProject` 确认完整填充字段
- [ ] `providers/modprovider.go`：新增 `refreshProjectMetadataIfStale(provider, platform, projectIDOrSlug)` 函数
  - 检查 `ModProject.CachedAt`，如超过 30 天则调用 provider 的 `ExactSearch` 刷新
  - 刷新失败时静默忽略
- [ ] `providers/modprovider.go`：在 `listProjectVersionsWithFilter` 中调用 `refreshProjectMetadataIfStale`

### 4. service 层改动
- [ ] `providers/service.go`：`SearchMods` 完成后，将返回的 `ModProject` 列表存入缓存
  - 对每个结果调用 `database.UpsertModPlatform`

### 5. 验证
- [ ] `go build ./...` 编译通过
- [ ] `go test ./...` 测试通过
- [ ] 确认旧缓存文件兼容（CachedAt=0 视为过期）

## 验证命令

```bash
go build ./...
go test ./database/... ./providers/... ./models/...
```
