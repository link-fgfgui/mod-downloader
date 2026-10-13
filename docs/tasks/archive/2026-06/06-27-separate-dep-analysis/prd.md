# 分离静态与动态依赖分析

## Goal

将模组管理（Manage 页面）使用的本地 JAR 元数据分析与下载页面使用的平台 API 依赖分析充分隔离，使两套逻辑可以独立演进，同时保留管理页面通过 SHA1 桥接到下载/平台元数据的能力。

## Background

### 两套完全不同的数据源

| 维度 | 管理页面（本地/动态） | 下载页面（平台/静态） |
|------|----------------------|----------------------|
| 数据来源 | 本地 JAR 内嵌元数据（fabric.mod.json, mods.toml 等） | CurseForge / Modrinth API |
| 主索引 | modID（JAR 声明的 ID，可能重复） | projectID（平台唯一） |
| 依赖表达 | JAR 内依赖暂不解析（TODO，预留空位） | `ModVersion.Dependencies []ModDependency` |
| 存储 | 纯内存，不持久化（实时来自本地文件） | 持久化到 `mods.gob.zst` |
| 实时性 | 高（随 mods 目录变更实时扫描） | 缓存 + 15min TTL |
| 使用场景 | 展示已安装模组、启用/禁用 | 搜索 → 选版本 → 递归下载依赖 |

### modID 重复问题（本地侧需处理）

- 同一个 JAR 可能包含多个 modID（nested jars、多 mod jar）
- 不同 JAR 可能声明相同 modID（fork、重新打包等）
- `global.localModFiles[sha1]` 是 `[]LocalModFile`，一个 SHA1 对应多个 mod 条目
- `uniqueModsByID()` 在单个 JAR 内去重；跨 JAR 重复由 `LocalModPathsInInstanceByModID` 全部返回

## Confirmed Decisions

### D1. 聚焦逻辑分离，JAR 依赖解析留 TODO
本任务只做两套分析逻辑的结构隔离，**不**新增 JAR 内依赖声明（fabric.mod.json `depends`、mods.toml `[[dependencies]]`）的解析。为其在 `minecraft/` 包预留扩展位（注释/空函数占位）。

### D2. 两侧通过 SHA1 桥接（维持现状机制）
本地 JAR 与平台 `ModVersion` 的关联继续依赖 SHA1 匹配。不引入新的显式关联表。需要跨域查询时走现有 SHA1 lookup。

### D3. 跨域状态查询提取到独立 `modbridge` 包
将 `downloader/download.go` 中的 `localModButtonStatus`、`localModButtonStatusPrecise`、按钮状态常量（new/installed/update/conflict）及相关交叉查询提取到新的 `modbridge` 包。downloader 与未来 Manage 页面均通过该包获取安装状态。

为消除 `downloader ↔ modbridge` 循环依赖，**版本解析与选中实例应用**（`downloadVersionsForRequest`、`applySelectedInstance`、`findProjectVersionByID` 及实例辅助）一并下沉到 `modbridge`。最终依赖单向：`downloader → modbridge → {providers, database, global, minecraft}`。`ApplySelectedInstance` 将 `targetDir` 作为返回值交回 downloader（modbridge 只计算不消费）。

### D4. 本地 JAR 元数据纯内存，不持久化
- `database.cacheState.JarMetadata` 中**仅服务本地已安装 JAR 的部分**移出持久化层，改为 `global` 包内的纯内存 `map[string][]ModInfo`（`minecraft` 解析后写入 `global` 缓存；现有 `minecraft → global` 依赖方向不变，无循环）。
- 序列化（`mods.gob.zst`）不再包含本地 JAR 元数据。
- 启动/实例切换时通过 `ScanVersionMods()` 实时重建。
- 内存缓存仅用于避免同一 SHA1 在一次会话内重复解析。

### D5. 远程 JAR 解析结果合入平台版本数据持久化
- 下载前 `parseRemoteModJar()`（HTTP Range 解析远程 JAR）得到的 modID 列表，作为 `ModVersion` 的衍生字段随平台版本数据一起持久化（远程文件不可变，SHA1 不变，缓存长期有效）。
- 该路径不再写入本地 JAR 元数据内存缓存，与本地侧彻底分离。

### D6. `applyPlatformMetadata` 不再回写本地存储
- 本地侧存储只保存 JAR 原始解析结果（纯净）。
- 平台名称/版本/描述的合并改为展示层（`modbridge`）按 SHA1 桥接实时合并，不污染本地数据。

## Target Package Structure

| 包 | 职责 | 持久化 |
|---|------|--------|
| `minecraft/` | 本地 JAR 解析 → `ModInfo`；JAR 依赖解析 TODO 预留 | — |
| `global/` | 本地 JAR 元数据内存缓存 `map[sha1][]ModInfo` + 本地模组索引（localModFiles/localModFilePaths） | ✗ 内存 |
| `providers/` | 平台 API 获取+缓存，`ModVersion.Dependencies` + 远程 JAR modID 衍生字段 | ✓ |
| `database/` | 平台版本（含依赖+远程 modID）、PlatformAssociations、PinnedMods | ✓ |
| `downloader/` | 下载编排、依赖递归下载；不再直接调用 `global.LocalModPaths*` | — |
| `modbridge/`（新） | 版本解析（含 pin）、选中实例应用、跨域状态判定、平台元数据展示层合并 | ✗ 运行时 |

边界约束：`minecraft/`（本地分析）与 `providers/`（平台分析）互不导入；交汇点统一收敛到 `modbridge/`。downloader 单向依赖 `modbridge`（版本解析也下沉到此，消除循环依赖）。

## Requirements

- R1: `minecraft/` 本地 JAR 解析逻辑独立，不依赖 `providers/`、`models.ModVersion`、`models.ModDependency`。
- R2: 本地 JAR 元数据缓存为纯内存，移出 `database` 持久化层；`mods.gob.zst` 不再包含该数据。
- R3: 远程 JAR 解析出的 modID 列表作为 `ModVersion` 衍生字段持久化。
- R4: `applyPlatformMetadata` 不回写本地 JAR 元数据；平台信息合并下沉到 `modbridge` 展示层。
- R5: `localModButtonStatus*` 及按钮状态常量、版本解析（`downloadVersionsForRequest` / `applySelectedInstance`）迁移到 `modbridge` 包；downloader 单向调用该包。
- R6: 下载依赖递归（`queueMissingRequiredDependencies` / `hydrateRequiredDependencies`）仅消费平台侧 `ModVersion.Dependencies`，不触碰本地 JAR 分析。
- R7: 为 JAR 内依赖解析预留扩展位（TODO 占位，不实现）。

## Acceptance Criteria

- [ ] `minecraft/` 包不 import `providers/` 与 `models` 的平台依赖类型。
- [ ] `database` 序列化不再包含本地已安装 JAR 元数据；重启后 Manage 页面数据由实时扫描重建。
- [ ] 远程 JAR modID 列表持久化在平台版本数据中，重启后按钮状态判定无需重新 Range 解析。
- [ ] `modbridge` 包包含版本解析、按钮状态判定与平台元数据合并，downloader 单向依赖它（无 downloader↔modbridge 循环）。
- [ ] 现有按钮状态行为（new/installed/update/conflict）与依赖递归下载行为保持不变（回归通过）。
- [ ] 现有测试通过：`minecraft/modparser_test.go`、`database/mods_test.go`、`global/*_test.go`。

## Out of Scope

- JAR 内依赖声明的实际解析（D1，留 TODO）。
- 管理页面新增依赖展示 UI。
- 显式 local↔platform 关联表（D2，维持 SHA1 桥接）。
