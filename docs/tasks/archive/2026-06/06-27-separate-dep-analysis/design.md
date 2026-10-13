# 设计：分离静态与动态依赖分析

## 1. 架构目标与分层

将两套数据源/分析逻辑沿"本地 JAR 分析"与"平台 API 分析"切开，新增 `modbridge` 包作为两者唯一的交汇点。隔离后包依赖单向：

```
app.go
  ├─ downloader  ──┐
  └─ modbridge ◄───┘        downloader → modbridge（单向，消除当前的循环风险）
        │
        ├─ providers   (平台 API 获取 + 缓存，含 Dependencies)
        ├─ database    (持久化：PlatformVersions 含依赖与派生 ModIDs；不再持久化 JAR 元数据)
        ├─ global      (本地模组内存索引：localModFiles / localModFilePaths)
        └─ minecraft   (本地 JAR 解析 → ModInfo；解析结果写入 global 内存缓存)
```

关键约束：`minecraft`（本地分析）与 `providers`（平台分析）互不导入；两者只在 `modbridge` 中通过 SHA1 / modID 关联。

## 2. 包职责划分

| 包 | 职责 | 本次变化 |
|----|------|----------|
| `minecraft/` | 本地 JAR 解析（fabric.mod.json / mods.toml → `ModInfo`）；解析后写入 `global` 内存缓存 | JAR 依赖解析留 TODO 占位 |
| `global/` | 本地模组内存索引，按 instanceID / modID / SHA1 查询；新增本地 JAR 元数据内存缓存 | 接管原 `database.JarMetadata` 的缓存职责，改为纯内存 |
| `providers/` | CurseForge / Modrinth 版本获取（含 `Dependencies`）与缓存 | 不变 |
| `database/` | 持久化 `PlatformVersions`（含依赖 + 新增派生 `ModIDs`）、`PlatformAssociations`、`PinnedMods` | 移除 `JarMetadata` 字段与持久化；新增版本 `ModIDs` 读写 |
| `downloader/` | 下载队列编排、依赖递归下载 | 不再直接读 `global` / `database.JarMetadata`；状态判定与版本解析改调 `modbridge` |
| `modbridge/`（新） | 版本解析（含 pin）、选中实例应用、安装状态判定（new/installed/update/conflict）、SHA1↔平台版本桥接、展示层平台元数据合并 | 从 `downloader` 提取 |

## 3. 数据存储策略

| 数据 | 位置 | 持久化 | 刷新时机 |
|------|------|--------|----------|
| 平台版本 + 依赖 | `database.PlatformVersions` | ✓ `mods.gob.zst` | API 拉取 / 15min TTL |
| 平台版本派生 modID 列表 | `database.PlatformVersions[].ModIDs`（新字段） | ✓ 随版本持久化 | 首次需要时惰性解析远程 JAR 后回写 |
| 本地 JAR 解析结果 | `global` 包内存 map | ✗ 仅内存 | 实例扫描 / mods 目录变更 |
| 本地文件索引 | `global.localModFiles/localModFilePaths` | ✗ 仅内存 | 实例扫描 |
| 跨域桥接计算 | `modbridge` 运行时 | ✗ 按需 | 调用时 |

### 3.1 本地 JAR 元数据：去持久化（场景 1）

`database.JarMetadata`、`SetJarMetadata`、`GetJarMetadata`、`copyModInfos`、`JarMetadataVersion` / `jarMetadataVersion` 迁移逻辑全部从 `database` 移除。改由 `global` 包持有纯内存缓存：

```go
// global 包内
var jarCacheMu sync.RWMutex
var jarCache = map[string][]structs.ModInfo{} // key: file sha1
```

`minecraft.ParseModJarWithSHA1` 改为查/写 `global` 的该内存缓存（`global.GetJarMetadata` / `global.SetJarMetadata`）。现有依赖方向 `minecraft → global` 不变（`ScanVersionMods` 已调用 `global.UpsertLocalMod`），无循环风险。进程重启后缓存为空，由 `ScanVersionMods` 重新填充——符合"本地数据实时、不跨会话持久化"的约束。

### 3.2 远程 JAR 解析：归并到平台版本（场景 2，方案 C）

远程 JAR（HTTP Range 解析，开销大、内容随 SHA1 不变）解析出的 modID 列表作为 `ModVersion` 的派生字段持久化：

```go
type ModVersion struct {
    // ... 现有字段 ...
    Dependencies []ModDependency `json:"dependencies,omitempty"`
    ModIDs       []string        `json:"modIds,omitempty"` // 新增：远程 JAR 解析出的 modID（去重、小写规整）
}
```

惰性填充流程（取代原 `metadataForProjectVersionWithResult` + `SetJarMetadata`）：

1. `modbridge` 需要某版本的 modID 时，先读 `version.ModIDs`
2. 为空则 `parseRemoteModJar(version.DownloadURL, modLoader)` 得到 modID 列表
3. 通过新 `database.SetVersionModIDs(platformVersionID, modIDs)` 回写到该 `PlatformVersions` 条目
4. 返回 modID 列表供状态判定使用

`applyPlatformMetadata` 的**回写行为移除**：不再把平台 Name/Version/Description 覆盖进任何持久化的本地数据。需要展示平台增强信息时，由 `modbridge` 在读取时即时合并（SHA1 → 平台版本 → Title/Version/Description），结果不落地。

## 4. modbridge 包接口（初稿）

从 `downloader/download.go` 迁入并整理：

```go
package modbridge

// 版本解析（原 downloader）
func ResolveVersion(req appstructs.ModDownloadRequest) (models.ModVersion, bool)
func ResolveVersions(req appstructs.ModDownloadRequest) []models.ModVersion
func ApplySelectedInstance(req appstructs.ModDownloadRequest) (out appstructs.ModDownloadRequest, instanceID, targetDir string, ok bool)
func FindVersionByID(versions []models.ModVersion, versionID string) (models.ModVersion, bool)

// 安装状态判定（原 localModButtonStatus*）
func InstallStatus(req appstructs.ModDownloadRequest) string          // 列表用，不解析远程 JAR
func InstallStatusPrecise(req appstructs.ModDownloadRequest) string    // 安装时用，可解析远程 JAR 取 modID
func DownloadStates(req appstructs.DownloadStatesRequest) []appstructs.ModDownloadButtonState

// 桥接：本地文件 ↔ 平台版本
func VersionModIDs(version models.ModVersion, modLoader string) []string // 惰性解析 + 回写
func PlatformMetadataForSHA1(sha1 string) (models.ModProject, models.ModVersion, bool) // 展示层合并用
```

按钮状态常量（`btnStatus*`）、`applyButtonStatus`、`defaultDownloadButtonState` 一并迁入。

`downloader` 侧保留：队列管理、`enqueueDownload`、`downloadModToTarget`、`upsertDownloadedMod`、`queueModDownload` / `queueMissingRequiredDependencies` / `hydrateRequiredDependencies`（依赖递归仍属下载编排，但其内部的状态判定改调 `modbridge.InstallStatusPrecise`，版本解析改调 `modbridge.ResolveVersion*`）。

## 5. 依赖方向与循环消除

当前隐患：`localModButtonStatus*`（拟入 modbridge）依赖版本解析；而 `queueMissingRequiredDependencies`（留 downloader）依赖 `localModButtonStatusPrecise`。

解法（方案 A）：版本解析 + 实例应用 + 状态判定全部下沉到 `modbridge`，`downloader` 单向依赖 `modbridge`。`ApplySelectedInstance` 把 `targetDir` 作为返回值交回 `downloader`（modbridge 只计算不消费）。

## 6. 调用点改写

| 调用点 | 现状 | 改写后 |
|--------|------|--------|
| `app.go: GetDownloadStates` | `downloader.GetDownloadStates` | `modbridge.DownloadStates`（或 downloader 转调） |
| `download.go:272` 状态判定 | `database.GetJarMetadata(version.SHA1)` | `modbridge.VersionModIDs(version, loader)` |
| `download.go:617/626` 远程解析缓存 | `GetJarMetadata` / `SetJarMetadata` | `modbridge.VersionModIDs`（读 `ModVersion.ModIDs` + 惰性回写） |
| `download.go:796` 下载后存储 | `applyPlatformMetadata` + `SetJarMetadata` | 解析纯 JAR → 写 `global` JAR 缓存 + `global` 索引；同时可回写 `version.ModIDs` |
| `modparser.go:320/332` | `database.GetJarMetadata/SetJarMetadata` | `global.GetJarMetadata/SetJarMetadata`（纯内存） |

## 7. 兼容与迁移

- `cacheVersion` 由 `2` → `3`。`ModVersion` 结构变化（+`ModIDs`）且 `JarMetadata` 移除会改变 gob 结构；版本不匹配时 `loadCacheState` 已有逻辑直接丢弃旧缓存重建，安全。
- 旧的 `JarMetadataVersion` / `jarMetadataVersion` 常量与 `migrateLocked` 中相关分支删除。
- 平台版本缓存丢弃后由首次 API 拉取重建；本地数据本就不持久化，无迁移成本。

## 8. 不在本次范围

- JAR 内嵌依赖解析（fabric.mod.json `depends`、mods.toml `[[dependencies]]`）——在 `minecraft` 包留 TODO 占位与扩展点，不实现。
- 显式 `localMod→platformProject` 关联表——继续用 SHA1 桥接。
- 前端展示改动——本次仅后端结构隔离，保持现有 API 行为等价。

## 9. 风险

- **行为等价性**：`InstallStatusPrecise` 的远程 JAR 解析路径从"独立 SHA1 缓存"改为"version.ModIDs 字段"，需保证首次惰性解析与回写后判定结果与现状一致。
- **并发**：`minecraft` 内存 JAR 缓存与 `database.SetVersionModIDs` 回写需各自加锁；`DownloadStates` 已用 goroutine 并发，惰性回写要避免竞态。
- **包提取范围**：迁移 `downloader` 中函数时需连带未列出的私有辅助（`projectVersionSHA1Set`、`localModPathsForMods` 等），编译驱动补全。
