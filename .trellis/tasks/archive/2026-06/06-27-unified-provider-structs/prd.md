# PRD: Abstract Platform-Agnostic Unified Provider Structs

## Background

当前 `providers/modprovider.go` 中 `curseForgeProvider` 和 `modrinthProvider` 各自用平台 SDK 类型直接构造 `SearchModResult` / `ProjectVersionResult`，存在几个问题：

1. **结构冗余**：`structs/search.go` 里的 DTO 和 `database/mods.go` 里的存储结构字段高度重叠但不统一
2. **provider 接口过胖**：`Search`/`ListVersions` 返回完整对象列表，无法做到"先拿 ID 列表，需要时再查详情"
3. **转换逻辑重复**：两个 provider 的 `xxxToSearchResult` / `xxxToProjectVersionResult` 模式一致，只是数据源不同
4. **前端 DTO 和内部结构不一致**：展示字段（Icon、Platform）混在通用数据里

## Goals

1. **统一数据结构**：以 `database/mods.go` 的 `ModPlatform` / `ModPlatformVersion` 为基础，扩展出一套完整的平台无关结构，覆盖 cf/mr API 实际用到的所有字段
2. **精简 modProvider 接口**：查询方法返回 ID 列表（或轻量引用），不返回完整对象；实际数据通过统一结构存取
3. **前端直接消费统一结构**：替换现有 `SearchModResult` / `ProjectVersionResult`，前端 DTO 直接用新结构
4. **消除 structs/search.go 与 database/mods.go 的结构重复**

## Scope

### In Scope

- 定义平台无关的统一 Project / Version / Dependency 结构（替换 `SearchModResult` + `ProjectVersionResult` + `ProjectDependency` + `ModPlatform` + `ModPlatformVersion` + `ModDependency`）
- 重构 `modProvider` interface：Search/ExactSearch 返回 project ID 列表；ListVersions 返回 version ID 列表
- provider 实现（cf/mr）负责：SDK 类型 → 统一结构，并存入内存/DB
- 调用层（service.go、downloader、app.go）通过 ID 查统一结构获取详情
- 前端 DTO 直接使用新统一结构的 JSON 序列化
- 保持所有现有功能不变（搜索、版本列表、下载、pin、按钮状态）

### Out of Scope

- 新增 API 功能
- 前端 Vue 组件逻辑变更（仅 TS 类型跟随 wails generate 更新）
- database.go 的存储引擎变更

## Acceptance Criteria

- [ ] `structs/search.go` 中 `SearchModResult`、`ProjectVersionResult`、`ProjectDependency` 被统一结构替换
- [ ] `database/mods.go` 中 `ModPlatform`、`ModPlatformVersion`、`ModDependency` 与新统一结构对齐或直接复用
- [ ] `modProvider` 接口的 Search/ListVersions 返回 ID 列表而非完整对象
- [ ] cf/mr provider 实现只做 SDK → 统一结构的映射
- [ ] `go build ./...` 通过
- [ ] `go test ./...` 通过
- [ ] 前端 `wails generate module` 后 TS 类型与新结构匹配

## Constraints

- Wails v2 桌面应用，Go struct 的 JSON tag 直接决定前端自动生成的 TS 类型
- 前端通过 `frontend/wailsjs/go/models.ts` 自动生成消费 Go struct
- `database/mods.go` 的 gob 序列化需要兼容（或做一次迁移）
