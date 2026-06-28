# Unify database layer with models types

## Goal

让 database 包直接使用 `models.ModProject` / `models.ModVersion` / `models.ModDependency` 作为存储类型，消除 database 包自有的 `ModPlatform` / `ModPlatformVersion` / `database.ModDependency`，以及 providers 包中对应的手动转换层。

## Background

- `models/models.go` 已定义统一类型 `ModProject`、`ModVersion`、`ModDependency`。
- `structs/search.go` 已通过 type alias 将老的 `SearchModResult` / `ProjectVersionResult` / `ProjectDependency` 指向 models。
- `providers/model.go` 对 models 类型做了 re-export（type alias + var alias）。
- **但 database 包仍然维护自己的一套平行类型**：`ModPlatform`、`ModPlatformVersion`、`database.ModDependency`。
- `providers/cache.go` 做 `models ↔ database` 双向逐字段转换，有信息损耗（`Downloads`、`Icon` 丢失，`IconURL` 塞进 `McmodURL`）。
- `providers/bridge.go` 全是死代码（type alias 后 old↔new 转换是恒等操作，且无调用方）。
- `providers/modprovider.go` 有 `projectVersionResultsToDB` / `dbProjectVersionsToResults` / `projectDependenciesToDB` / `dbDependenciesToResults` 做同样的逐字段转换。

## Requirements

1. **database 包使用 models 的统一类型**
   - `ModPlatform` → `models.ModProject`
   - `ModPlatformVersion` → `models.ModVersion`
   - `database.ModDependency` → `models.ModDependency`
   - `cacheState` 中的 map value 类型相应切换
   - 内部 key 类型（`platformKey`、`versionKey` 等）可保留，它们是查找索引
   - `PinnedMod`、`PlatformAssociation`、`storedVersionScope`、`ModPlatformVersionScope` 保留不变（它们在 models 层没有对应物，职责不同）

2. **消除 providers 的转换层**
   - 删除 `providers/bridge.go`（全是死代码）
   - 删除 `providers/cache.go` 中的 `modPlatformToModProject` / `modProjectToModPlatform` / `modPlatformVersionToModVersion` / `modVersionToModPlatformVersion` 及其调用
   - 删除 `providers/modprovider.go` 中的 `projectVersionResultsToDB` / `dbProjectVersionsToResults` / `projectDependenciesToDB` / `dbDependenciesToResults`
   - `providers/cache.go` 的 `GetProjectByID` / `StoreProject` 等直接使用 database 返回的 models 类型

3. **app.go 中 `PinnedMod` 的使用保持不变**（PinnedMod 不在统一范围内）

4. **gob 编码兼容性**
   - 字段重命名（如 `ModPlatform.Name` → `ModProject.Title`）会导致 gob 反序列化旧缓存时丢失数据
   - 可接受方案：首次启动时旧缓存自动失效重建（缓存本质上可丢失）
   - 需要在 `cacheVersion` 常量递增来触发自动重建

## Acceptance Criteria

- [ ] database 包不再定义 `ModPlatform`、`ModPlatformVersion`、`database.ModDependency`
- [ ] database 包的公开 API（`UpsertModPlatform`、`GetModPlatform` 等）接收/返回 `models.ModProject` / `models.ModVersion` / `models.ModDependency`
- [ ] `providers/bridge.go` 已删除
- [ ] `providers/cache.go` 和 `providers/modprovider.go` 中不再有 `database.ModPlatform` / `database.ModPlatformVersion` / `database.ModDependency` 的引用
- [ ] `go build ./...` 通过
- [ ] `go test ./...` 通过
- [ ] `cacheVersion` 递增，确保旧缓存文件自动失效
