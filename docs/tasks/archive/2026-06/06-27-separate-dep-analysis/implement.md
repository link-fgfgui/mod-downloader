# 执行计划：分离静态与动态依赖分析

按依赖方向自底向上推进：先改数据模型与存储，再建 `modbridge` 包，最后改 `downloader` 调用点。每步以 `go build ./...` + `go test ./...` 驱动。

## 阶段 0：基线

- [ ] `go build ./...` 与 `go test ./...` 通过，记录基线（尤其 `database/mods_test.go`、`minecraft/modparser_test.go`）
- [ ] 确认 `models.ModVersion` 当前 JSON 契约（前端 `dependencies` 字段使用情况）

## 阶段 1：数据模型 + 存储调整

- [ ] `models/models.go`：`ModVersion` 新增 `ModIDs []string json:"modIds,omitempty"`
- [ ] `database/database.go`：
  - [ ] 从 `cacheState` 移除 `JarMetadata`、`JarMetadataVersion` 字段
  - [ ] `normalize()` 删除 `JarMetadata` 初始化分支
  - [ ] `migrateLocked()` 删除 jarMetadata 相关分支（如整体空置则移除该方法调用）
  - [ ] `cacheVersion` `2` → `3`
- [ ] `database/mods.go`：
  - [ ] 删除 `GetJarMetadata` / `SetJarMetadata` / `jarMetadataVersion` 常量
  - [ ] 新增 `SetVersionModIDs(platformVersionID string, modIDs []string) error`（参照 `SetVersionDependencies` 的 `PlatformVersionKeyByID` 查找 + 回写模式）
  - [ ] 评估 `copyModInfos` 是否还有引用，无则删除
- [ ] `database/database.go` 若 `structs/minecraft` 导入只为 `JarMetadata`，清理 import
- [ ] 修复 `database/mods_test.go`（`TestCachePinnedModsAndJarMetadata` 拆分/改写：移除 JarMetadata 断言）
- [ ] `go build ./database/...` + `go test ./database/...`

## 阶段 2：global 包持有本地 JAR 内存缓存

- [ ] `global/localmods.go`（或新增 `global/jarcache.go`）：
  - [ ] 新增包级内存缓存 `jarCache map[string][]structs.ModInfo` + `sync.RWMutex`
  - [ ] 新增 `global.GetJarMetadata(sha1) ([]ModInfo, bool)` / `global.SetJarMetadata(sha1, mods) `（含原 `SetJarMetadata` 的 modID 去重/规整逻辑）
- [ ] `minecraft/modparser.go`：
  - [ ] `ParseModJarWithSHA1`：查/写 `global.GetJarMetadata/SetJarMetadata` 取代 `database.GetJarMetadata/SetJarMetadata`
  - [ ] 移除对 `database` 的导入（若仅用于 JarMetadata）；`minecraft → global` 依赖方向已存在，无循环
  - [ ] 在 `ModInfo` 解析处保留 JAR 依赖解析 TODO 注释占位
- [ ] `go build ./global/... ./minecraft/...` + `go test ./global/... ./minecraft/...`

## 阶段 3：新建 modbridge 包

- [ ] 创建 `modbridge/` 包
- [ ] 从 `downloader/download.go` 迁入并改名导出：
  - [ ] 版本解析：`ResolveVersion` / `ResolveVersions` / `FindVersionByID` / `ApplySelectedInstance`（连带 `selectedVersionModsDir`、`versionInstanceID` 等私有辅助）
  - [ ] 状态判定：`InstallStatus`（原 `localModButtonStatus`）、`InstallStatusPrecise`（原 `localModButtonStatusPrecise`）、`DownloadStates`（原 `GetDownloadStates`）
  - [ ] 常量与辅助：`btnStatus*`、`applyButtonStatus`、`defaultDownloadButtonState`、`projectVersionSHA1Set`、`localModPathsForMods`
  - [ ] 桥接：`VersionModIDs(version, modLoader)`（读 `version.ModIDs`，空则 `parseRemoteModJar` + `database.SetVersionModIDs` 回写）
  - [ ] `parseRemoteModJar` 迁入（或保留 downloader 并由 modbridge 调用——按编译依赖方向决定，倾向迁入 modbridge）
  - [ ] 展示合并 `PlatformMetadataForSHA1`（替代 `applyPlatformMetadata` 的即时合并版本，**不回写**）
- [ ] `go build ./modbridge/...`

## 阶段 4：改写 downloader 调用点

- [ ] `download.go` 删除已迁出的函数，改调 `modbridge.*`：
  - [ ] `queueModDownload`：版本解析 → `modbridge.ResolveVersion`；`applySelectedInstance` → `modbridge.ApplySelectedInstance`
  - [ ] `queueMissingRequiredDependencies`：`localModButtonStatusPrecise` → `modbridge.InstallStatusPrecise`
  - [ ] `hydrateRequiredDependencies`：`findProjectVersionByID` → `modbridge.FindVersionByID`
  - [ ] `upsertDownloadedMod`：移除 `applyPlatformMetadata` 回写；解析纯 JAR 写入 `global` 内存缓存 + `global` 索引；可选回写 `version.ModIDs`
  - [ ] 移除 `database.GetJarMetadata/SetJarMetadata` 全部调用
- [ ] `app.go: GetDownloadStates` → `modbridge.DownloadStates`（或经 downloader 薄转发，保持绑定签名不变）
- [ ] 确认 `downloader` 不再导入 `global`（除下载落盘必需）/ 不再直接做跨域状态判定
- [ ] `go build ./...`

## 阶段 5：验证

- [ ] `go test ./...` 全绿
- [ ] 重点回归：
  - [ ] 下载按钮状态（new/installed/update/conflict）行为与基线一致
  - [ ] 依赖递归下载仍正常（required 依赖入队）
  - [ ] pin 版本解析路径不变
  - [ ] 旧缓存文件存在时启动不崩溃（cacheVersion 升级丢弃重建）
- [ ] `wails build`（或前端构建）确认 `ModVersion.ModIDs` 新字段不破坏前端
- [ ] 清理临时文件

## 验证命令

```bash
go build ./...
go test ./...
go test ./database/... ./minecraft/... ./modbridge/... ./downloader/...
```

## 回滚点

- 阶段 1（cacheVersion 升级）后若回滚，需把 `cacheVersion` 改回并恢复 `JarMetadata`，旧缓存自动重建，无数据损坏风险。
- 各阶段独立可编译，按阶段提交便于二分定位。

## 风险文件

- `downloader/download.go`：函数迁移量最大，注意私有辅助连带迁移与导入循环。
- `database/database.go`：`cacheState` 结构变更影响 gob 序列化，务必升 `cacheVersion`。
- `database/mods_test.go` / `minecraft/modparser_test.go`：JarMetadata 相关断言需重写。
