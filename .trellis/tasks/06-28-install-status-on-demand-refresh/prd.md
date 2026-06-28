# InstallStatus 按需触发本地与远程元数据刷新

## Goal

让搜索列表按钮状态判定 `modbridge.InstallStatus` 不再只读取 `version.ModIDs` 这一个内存字段，
而是按需触发三层元数据获取（本地实例扫描 → DB 缓存的版本 ModIDs → 远程元数据异步拉取），
使 `conflict` / `update` / `installed` 状态在首次搜索时即可正确显示，而非等到安装时才精确判定。

## Background

### 现状（已通过代码确认）

`modbridge.InstallStatus`（[modbridge.go:106-159](file:///home/link/Documents/go_proj/worktrees/mod-downloader/refresh/mod-downloader/modbridge/modbridge.go#L106-L159)）用于搜索列表渲染，当前判定流程：

1. `ResolveVersions` → `providers.ListMatchingProjectVersions`（已通过 `refreshProjectMetadataIfStale` 异步刷新远程**项目**元数据）
2. SHA1 命中 → `BtnStatusInstalled`
3. 项目版本 SHA1 集合命中 → `BtnStatusUpdate`
4. **冲突判定**：只读 `version.ModIDs`（内存字段）。注释明确写道"Search-list rendering must not range-parse remote JARs; only use already persisted mod IDs"。

### 问题

- `version.ModIDs` 内存字段在搜索结果路径下几乎永远为空：`providers.ListMatchingProjectVersions` 返回的 `ModVersion` 不携带 `ModIDs`，它只在 `VersionModIDs` 惰性解析远程 JAR 后**写入 DB**，并不回写传入 `InstallStatus` 的 version struct。
- 即使 DB 中已有 `database.GetVersionModIDs(version.ID)` 缓存，`InstallStatus` 也不读取，直接跳过冲突判定 → 返回 `BtnStatusNew`，导致用户看到"可下载"但实际已安装同类 mod。
- 本地 mod 索引（`global.LocalModPathsInInstance`）只在实例切换 / 启动时通过 `ScanVersionMods` 重建，`DownloadStates` 不会按需触发刷新。

### 已有的基础设施

- `VersionModIDs`（[modbridge.go:250-284](file:///home/link/Documents/go_proj/worktrees/mod-downloader/refresh/mod-downloader/modbridge/modbridge.go#L250-L284)）已实现三级回退：`version.ModIDs` → `database.GetVersionModIDs` → `parseRemoteModJar`，但 `InstallStatus` 刻意不调用它（避免远程 JAR 解析）。
- `providers.RefreshMatchingProjectVersions`（[service.go:130](file:///home/link/Documents/go_proj/worktrees/mod-downloader/refresh/mod-downloader/providers/service.go#L130)）可强制刷新版本列表。
- `app.RefreshSelectedVersionMods`（[app.go:358](file:///home/link/Documents/go_proj/worktrees/mod-downloader/refresh/mod-downloader/app.go#L358)）→ `refreshVersionMods` → `ScanVersionMods` 可重建本地索引。
- `database.SetVersionModIDs` / `GetVersionModIDs` 已持久化版本 ModIDs。
- 前端 `downloadSearch.ts` 已监听 `download-queue-updated` / `download-failed` / `search-mods-updated` / `extension-mods-accepted` 事件并调用 `refreshDownloadStates()`（[downloadSearch.ts:278-319](file:///home/link/Documents/go_proj/worktrees/mod-downloader/refresh/mod-downloader/frontend/src/stores/downloadSearch.ts#L278-L319)）。

### 旧实现参考（commit 6932197^，modbridge 重构前）

旧版 `localModButtonStatus`（搜索列表）按 **SHA1** 键读 `database.GetJarMetadata(version.SHA1)` 持久化缓存。缓存填充路径：
- 安装时 `metadataForProjectVersionWithResult` 远程解析 JAR → `SetJarMetadata`
- 下载后 `upsertDownloadedMod` → `SetJarMetadata`

异步回填链：下载/队列变化 → `emitDownloadQueueState` 发 `download-queue-updated` 事件 → 前端 `refreshDownloadStates()` → 重新拉 `GetDownloadStates` → 缓存命中 → 状态正确。

重构后 `JarMetadata`（SHA1 键）被移除，改为 `PlatformVersions[].ModIDs`（versionID 键），但 `InstallStatus` 只读 `version.ModIDs` 内存字段，断了回填链。本次任务按当前 versionID 键缓存重建等价机制。

## Requirements

### R1: InstallStatus 读取 DB 缓存的版本 ModIDs

`InstallStatus` 冲突判定前，当 `version.ModIDs` 为空时，读取 `database.GetVersionModIDs(version.ID)`。
此为同步、低开销（内存索引读）操作，不引入远程调用。等价于旧实现的 `database.GetJarMetadata(version.SHA1)` 读取。

### R2: 按需触发本地 mod 索引刷新（仅补首次缺口）

`DownloadStates` 在判定前，若 `global.HasLocalModPathsInInstance(instanceID)` 为 false（即实例从未扫描过），
同步触发一次本地 mod 扫描（复用 `refreshVersionMods` / `ScanVersionMods` 路径）后再判定。
不加 TTL 守卫；已扫描过的实例不重复扫描（与旧实现"实例切换/启动时扫描"语义一致，仅补上首次未扫描的缺口）。

### R3: 按需触发远程版本元数据拉取（异步回填）

当 DB 缓存也缺失 `ModIDs` 时，`InstallStatus` 本次返回 `BtnStatusNew`，但**异步**触发远程 JAR 解析（复用 `VersionModIDs` 路径，写回 `database.SetVersionModIDs`）。解析完成后通过事件通知前端重新拉取 `GetDownloadStates`，下次渲染即可命中 DB 缓存。

此为旧实现异步回填链的等价重建（旧版按 SHA1 键 `JarMetadata`，本次按 versionID 键 `PlatformVersions[].ModIDs`）。需避免重复并发解析同一 version（去重守卫）。

### R4: 不破坏现有按钮状态语义

`new` / `installed` / `update` / `conflict` 四态判定结果与 `InstallStatusPrecise` 在 modIDs 已知时一致；
不引入新的状态值。

### R5: 性能不退化

`DownloadStates` 并发判定 10 条搜索结果时，不能因 R1–R3 导致显著延迟（远程调用必须异步或带 TTL 守卫）。

## Acceptance Criteria

- [ ] `InstallStatus` 在 `version.ModIDs` 为空时读取 `database.GetVersionModIDs(version.ID)`（R1）
- [ ] 选中实例本地 mod 索引为空时，`DownloadStates` 触发一次本地扫描后再判定（R2）
- [ ] DB 缓存缺失时，`InstallStatus` 异步触发远程 JAR 解析并写回 DB；解析完成后发事件通知前端刷新（R3）
- [ ] 同一 version 的并发远程解析被去重守卫拦截，不重复发起（R3）
- [ ] 当 modIDs 已知时，`InstallStatus` 与 `InstallStatusPrecise` 返回相同状态（R4）
- [ ] 新增 / 更新 `modbridge` 单元测试覆盖 R1、R3 去重守卫的关键路径
- [ ] `go test ./...` 通过；`go vet ./...` 无新增告警（R5）
- [ ] `design.md` 与 `implement.md` 完成，覆盖异步回填事件通道与并发守卫

## Out of Scope

- 修改前端 `downloadSearch` store 的 `refreshDownloadStates` 调用时机（复用现有事件监听）
- 修改 `InstallStatusPrecise` 的安装时精确判定逻辑
- 新增持久化字段或更改 `cacheVersion` 版本号
- JAR 内嵌依赖（`depends` / `mods.toml` dependencies）解析

## Open Questions

- 无（O1、O2 已决议：R3 异步回填；R2 仅补首次缺口不加 TTL）。
