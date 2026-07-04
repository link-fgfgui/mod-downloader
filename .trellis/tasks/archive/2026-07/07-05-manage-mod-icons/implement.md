# Implementation: 管理页面模组显示图标

## Checklist

### 1. ModInfo 增加 IconURL 字段
- [ ] `structs/minecraft/modinfo.go`: `ModInfo` 加 `IconURL string \`json:"iconUrl,omitempty"\``

### 2. providers 层新增 hash→project 查询
- [ ] `providers/modprovider.go`: 新增 `modrinthProvider.resolveProjectsByHashes(hashes []string) (map[string]models.ModProject, error)`
  - 调用 `client.VersionFiles.GetFromHashes(hashes, "sha1")`
  - 收集去重 `ProjectID`
  - 调用 `client.Projects.GetMultiple(projectIDs)`
  - 用现有 `projectToModProject` 转换
  - 建立 sha1→ModProject 映射返回
- [ ] `providers/service.go`: 导出 `ResolveProjectsByHashes(hashes []string) map[string]models.ModProject`
  - 调用 `modrinthProvider{}.resolveProjectsByHashes`
  - 每个结果 `database.UpsertModPlatform(project)` 入库
  - 返回 sha1→ModProject

### 3. app.go 同步 + 异步 icon 填充
- [ ] 新增辅助函数 `enrichModIcons(mods []mcstructs.ModInfo) (enriched []mcstructs.ModInfo, missedSHA1s []string)`
  - 遍历 mods，对非空 SHA1 调 `modbridge.PlatformMetadataForSHA1`
  - 命中：设 `mod.IconURL = project.IconURL`
  - 未命中：收集 sha1
- [ ] 修改 `RefreshSelectedVersionMods` 流程：
  - `scanVersionMods` 后调 `enrichModIcons` 同步填充缓存命中的 icon
  - 将带缓存 icon 的结果立即返回并 emit
  - 有 missedSHA1s 时启 goroutine 异步补充：
    - `providers.ResolveProjectsByHashes(missedSHA1s)`
    - 重新对 mods 执行 SHA1→iconUrl 映射
    - 更新 version.Mods，更新 global state
    - `runtime.EventsEmit(selectedVersionChangedEvent, version)`

### 4. 前端条件渲染 icon
- [ ] `frontend/src/views/Manage.vue`: `<template #prepend>` 改为条件渲染
  - 有 `group.primary.iconUrl` 时 `<v-img :src="...">`
  - 否则 `<v-icon icon="mdi-package-variant">`
  - 参照 `SearchResultList.vue` 的写法

## Validation

- [ ] `go build ./...` 编译通过
- [ ] 前端 `npm run build` 无报错
- [ ] 手动验证：管理页面有缓存数据的 mod 立即显示 icon
- [ ] 手动验证：无缓存的 mod 刷新后异步补充 icon

## Rollback

回退所有修改的 4 个文件即可恢复原状，无数据库 schema 变更。
