# 架构梳理与改进计划

> 基线日期：2026-10-25
> 现状验证：根模块 `go test ./...`、`core/` 子模块 `go test ./...`、`go vet ./...`、`gofmt -l .` 全部通过；前端 `npm run test` 18 项通过。本文只给出改进方案，不含代码改动。

---

## 1. 结论摘要

架构方向是清晰的：Vue → Wails Adapter → `core/appcore.Service` → 领域包，`core` 不依赖 Wails，事件通过 adapter 翻译。当前的主要问题不是“分层错了”，而是**分层内部缺少收敛**：

1. **业务状态实际存放在包级全局变量里**，`Service` 不是组合根，无法真正拥有生命周期。
2. **Wails 适配层存在重复契约与隐式初始化**，新增一个设置项要改 7~8 个位置。
3. **两个超大编排文件**（`appcore/service.go` 2017 行、前端 `Manage.vue` 1474 行）承载了过多职责。
4. **CI 只构建不验证**，测试、lint、vet、race 均未进入流水线。

建议优先做 Phase 0（低风险收敛）与 Phase 1（契约统一），Phase 2（文件拆分）在前两步稳定后做纯搬运，Phase 3（运行时实例化，即去全局化）按需推进。

---

## 2. 架构现状

### 2.1 分层与调用方向

```text
Vue views / Pinia stores / composables
        │  frontend/wailsjs 生成绑定
        ▼
root App（app.go / app_contracts.go）
        │  typed request、原生对话框、Wails EventsEmit
        ▼
core/appcore.Service（工作流编排，无 Wails 依赖）
        ├── providers   平台 API、搜索、元数据缓存
        ├── downloader  队列、依赖预检、文件传输
        ├── modbridge   线上元数据 ↔ 本地 JAR 桥接
        ├── minecraft   启动器识别、JAR 解析、版本发现
        ├── storage     SQLite 用户数据 + gob/zstd 元数据缓存
        ├── configs     TOML 配置
        └── global      进程级内存索引
```

### 2.2 量化基线

| 项目 | 现状 |
| --- | --- |
| Wails 适配层 | `app.go` 约 590 行，71 个 `App` 方法 |
| 核心服务 | `core/appcore/service.go` 2017 行，82 个导出方法，覆盖搜索/收藏/设置/下载/本地实例 5 大域 |
| 领域包规模 | `downloader/download.go` 1664 行；`storage/mods.go` 1562 行；`storage/userdb.go` 1364 行；`modbridge/modbridge.go` 1232 行；`minecraft/modparser.go` 1069 行 |
| 前端视图 | `Manage.vue` 1474 行（script 855 行）；`Favorites.vue` 1035 行；`App.vue` 745 行；`plugins/i18n.ts` 840 行 |
| 状态位置 | `global/*` 4 个文件、`storage` 包、`downloader` 包、`providers` 后台任务均为包级单例 |
| 测试 | 后端 21 个包有测试且通过；前端仅 3 个工具函数测试，无组件测试 |
| CI | 仅 `wails build` + 打包发布，无测试/lint/vet 步骤 |

### 2.3 生命周期现状

`appcore.Service` 自身只保存 `ctx`、`config`、`options`、文件监听句柄（`core/appcore/contracts.go:204-212`），所有业务状态通过 `Startup` 写入包级变量：

- `downloader.Configure(...)`、`configureSimpleMode(...)`、`modbridge.ConfigurePreferredReleaseType(...)`、`minecraft.SetJijDetectionEnabled(...)`（`core/appcore/service.go:83-93`）
- `global.SetMinecraftDir(...)`（`core/appcore/service.go:94-97`）
- `storage.OpenAt(...)`（`core/appcore/service.go:99-104`）
- `providers.StartBackgroundTasks()`（`core/appcore/service.go:67`）
- `Shutdown/Close` 反向停止这些全局单例（`core/appcore/service.go:139-158`）

这意味着 `appcore.New` 并不是真正的构造函数，`Startup` 是全局副作用脚本；两个 `Service` 实例（GUI + CLI、并行测试）无法共存。

---

## 3. 问题清单

### P0-1 Wails 适配层存在隐式初始化路径

**证据**

- `App.service()` 在 `a.core == nil` 时直接 `appcore.New(...)`，但**不调用 `Startup`**（`app.go:69-80`）。此时配置未加载、数据库未打开、provider 未初始化、事件未接线。
- `startup()` 中 `a.core.Startup(ctx)` 失败时只回写 config 后直接 `return`，**没有记录日志**，也不会启动 HTTP server（`app.go:48-67`）。
- `a.config` 在启动、每次 `Save*`、`shutdown` 中反复回写（`app.go:58`、`61`、`78`、`292`~`374`、`439`），但除懒加载分支外几乎不再被读取，属于可删除的镜像状态；并发调用下还是未加锁写入。

**影响**

- 任何在 `startup` 完成前到达的前端调用都会拿到“半初始化”的 Service，错误会被静默吞掉。
- 启动失败不可观测，用户只会看到空列表或按钮无响应。

**建议**

- 删除 `service()` 的懒创建，改为 `mustService()`：未初始化时记录错误并返回明确失败；或保留懒加载但内部调用完整 `startup`。
- `Startup` 失败必须 `logging.Error`，并明确 HTTP server 是否继续启动。
- 删除所有 `a.config = a.core.Config()` 镜像赋值，配置只在 `Service` 内维护。

### P0-2 跨层 DTO 重复且手工映射

**证据**

- `SettingsView` 在两处逐字段重复：`app_contracts.go:16-40` 与 `core/appcore/contracts.go:69-93`。
- 8 个 `Save*SettingsRequest` 同样重复：`app_contracts.go:42-85` 与 `core/appcore/contracts.go:95-133`。
- `settingsViewFromCore` 手工逐字段转换（`app.go:527-553`）。
- 新增一个设置项的改动面：`configs` → `appcore.SettingsView` → `GetSettings` → `Save*` → `app_contracts.SettingsView` → `app.go` 映射 → 前端 store → 设置页，约 8 处。

**影响**

- 字段漏映射时静默变成零值，编译器不会报错；这是典型的“新增设置不生效”来源。

**建议**

- 将前端可见的设置快照与保存请求下沉到 `core/structs`（与现有 request/response 归属一致），`appcore` 与 `app.go` 共用同一类型，删除 `app_contracts.go` 中的重复定义与 `settingsViewFromCore`。
- Wails 支持引用包内类型生成绑定；迁移成本是一次重新生成 `frontend/wailsjs` 并调整前端 import。
- 保留 `ExportFavoritePackwizResult` 这类真正只属于适配层的结果类型。

### P0-3 配置应用逻辑重复且顺序敏感

**证据**

- `downloader.Configure(...)` 出现 3 份拷贝：`service.go:83-89`、`837-843`、`885-891`。
- `SaveMCIMSettings` 只改镜像开关，却顺带重配下载器（`service.go:836-843`）。
- `SaveNetworkSettings` 传给下载器的是原始 config 字段（`service.go:886-890`），而 `Startup` 传给的是 `*Value()` 归一化结果（`service.go:84-88`），依赖“前面已赋值归一化值”这一隐含前提。
- 各站点的附带动作不一致，收敛时极易遗漏：`SaveMCIMSettings` 还调用 `s.configureProviderClients()`（`service.go:836`）；`SaveNetworkSettings` 还调用 `configureRemoteModIDConcurrency` + `s.configureProviderClients()`（`service.go:892-893`）；`Startup` 还调用 `configureRemoteModIDConcurrency` + `s.configureProviderClients()`（`service.go:93,105`）。
- `configureSimpleMode` 存在两种变体：`Startup` 不带 events（`service.go:90`），`SaveSimpleModeSettings` 带 `s.downloadEvents()`（`service.go:813`，只影响 optional reminder 清理语义）。下载事件本身是逐调用传递（`service.go:948-1027`），不存入全局状态——合并**不会**丢进度事件，但 simple-mode 路径的 events 变体必须保留。

**影响**

- 修改一个网络参数时可能漏掉某个全局开关；两份配置来源可能分叉。

**建议**

- 抽出单一 `applyRuntimeConfig()`，动作集必须完整枚举（作为验收清单逐条对照现有 4 个站点核对）：归一化 → 写 config → `downloader.Configure` → `configureSimpleMode` → `modbridge.ConfigurePreferredReleaseType` → `minecraft.SetJijDetectionEnabled` → `configureRemoteModIDConcurrency` → `configureProviderClients()`。
- `providers.StartBackgroundTasks`、`storage.OpenAt` 属于生命周期而非配置应用，保留在 `Startup`，不要并入。
- 所有 `Save*`、`Startup` 只调用它，禁止在各 Save 方法里直接调用上述全局 setter。

### P0-4 读接口隐藏失败 / 修改状态

**证据**

- `LookupProjectBySlug` 找不到时返回零值 `models.ModProject{}`（`app.go:142-148`），前端无法区分“未找到”与“网络失败”。
- `LocalModsInDir` 丢弃 enrich 错误：`enriched, _ := enrichModMetadata(mods)`（`service.go:1169`）；且目录不存在时返回 `(nil, nil)`（`service.go:1155-1158`），“路径错误”与“空目录”同样不可区分。
- `GetVersions` 在读取时调用 `ensureSelectedVersion`（`service.go:1088-1095`）；`LocalMods` 内部会触发 `SelectVersion` 与完整刷新（`service.go:1139-1147`）。只读接口会写全局状态、扫描磁盘并发事件。
- `error-handling.md` 明确禁止“把失败隐藏为空成功”。

**影响**

- 前端提示失真；读操作产生隐式副作用，缓存、CLI 只读命令和测试都难以预测。

**建议**

- 为“未找到”设计显式结果：`(models.ModProject, bool)` 或 typed `NotFound` 错误。
- 读接口不写状态：默认实例选择并入 `LoadVersionsFromDisk` 完成路径（行为保持，见 2.3），或改为显式 `EnsureSelectedVersion()`（行为变更，需前端同步）。
- 逐步把新接口统一为 `(result, error)`；`panic` 仅保留在既有导出/对话框路径，并在文档中标注为遗留契约。

### P1-1 God 文件

- `core/appcore/service.go`：搜索、收藏（含跨版本复制约 170 行）、设置、下载、实例刷新全部在一个文件。
- `storage/mods.go` / `storage/userdb.go`：缓存域与用户数据域各自超过 1300 行。
- 前端 `Manage.vue`、`Favorites.vue`、`App.vue`：模板 + 业务逻辑 + 对话框状态混合；`App.vue` 内嵌完整下载队列面板。
- `plugins/i18n.ts` 单文件承载两套语言全部文案。

**建议**：只做“代码搬运”，不改行为。服务按域拆文件并保持方法名；前端抽子组件与 composable；i18n 拆 `locales/zh.ts` / `locales/en.ts`。

### P1-2 前端事件接线分散

**证据**

- 事件名常量分散：`stores/downloadQueue.ts:20-21`、`stores/downloadSearch.ts:437-474`，而 `Manage.vue:1237` 直接使用字面量 `"download-queue-updated"`。
- 多个 store/view 各自 `EventsOn`，没有统一的事件注册与释放入口。

**影响**：重命名事件时容易漏改；重复监听会放大高频进度事件的开销。

**建议**：建立 `frontend/src/utils/events.ts` 作为唯一事件名来源；所有 `EventsOn` 只引用常量；保留现有“按 `id:status` 签名过滤进度”的策略。

### P1-3 质量门禁缺失

**证据**：`.github/workflows/build.yml` 只有构建与发布；未运行 `go test`、`go vet`、`gofmt`、`npm run lint`、`npm run test`。CI 使用 Node 20，而前端测试脚本 `node --test tests/*.test.ts` 依赖 Node ≥22.18/23.6 才默认启用的 TypeScript 类型剥离。另注意 `go test ./...` 会扫到 `frontend/node_modules` 内的 Go 源码（`flatted/golang/pkg/flatted`），CI 接入测试时必须采用 0.1 的排除写法。

**影响**：回归只能靠人工；绑定漂移、类型错误、竞态都可能在发布后才暴露。

### P2-1 全局单例（战略项）

`global`、`storage`、`downloader` 队列、`providers` 后台任务均为进程级单例。当前产品形态是单窗口桌面应用，收益主要在测试隔离与生命周期确定性，而非多实例业务需求。

**建议**：不作为近期目标；若推进，按 `storage` → `providers` → `runtime state` → `downloader queue` 的顺序逐步实例化，每步独立可发布。不要为了迁移新增别名/再导出层（违反 AGENTS.md）。

### P2-2 其他低优先级项

- `app.go:579-589` 的 `loadVersionsFromDisk` / `scanVersionMods` / `scanAllModDirsForHardlinkIndex` 仅被 `app_test.go` 使用，属于测试垫片；`maskKey`（`app.go:284-286`）同理。
- `app_contracts.go:87` 的 `apiKeyKeepSentinel` 仅测试使用。
- `go test ./...` 会扫描 `frontend/node_modules` 中的 Go 包（本地输出可见 `mod-downloader/frontend/node_modules/flatted/golang/pkg/flatted`）；Go 无包排除机制，解决方式随 0.1 采用 `go list ./... | grep -v node_modules`。

---

## 4. 分阶段改进计划

### Phase 0：低风险收敛（P0）

目标：不改变任何前端行为与 Wails 方法名。

- [x] **0.1 CI 质量门禁**
  - 新增独立测试 job，只在 ubuntu 单平台运行（不复用三平台构建矩阵，控制成本）：`(cd core && go test ./...)`、`go vet ./...`（两处）、`gofmt -l` 校验、根模块测试。
  - 根模块测试写法：`go test $(go list ./... | grep -v node_modules)`——`frontend/node_modules/flatted/golang` 含 Go 源码，裸 `./...` 会把它带入流水线（见 P2-2）。
  - 新增前端 `npm ci` + `npm run build` + `npm run lint` + `npm run test`。
  - Node 升级到 24：`node --test` 的 TS 类型剥离要 22.18+/23.6+ 才默认启用，选 22 LTS 有次版本陷阱。
  - 竞态检测先以 `(cd core && go test -race ./...)` 运行，允许先 nightly 再转必跑。
- [x] **0.2 适配层生命周期清理**
  - 删除 `a.config` 镜像赋值；`startup` 失败记录日志并明确 server 启动策略（建议一并记录 `startupErr` 供诊断）。
  - `App.service()` 去掉无 `Startup` 的懒创建，改为显式错误。实现口径：71 个 `App` 方法大多无 `error` 返回，不要逐个改签名；未就绪时 `logging.Error` + `panic` 明确原因（如 `core not initialized: startup failed`），经 Wails 转为前端 Promise rejection，由前端全局错误处理器呈现——该模式已在对话框路径（`ExportFavoriteListPackwizZip`）使用。
- [x] **0.3 配置应用单一入口**
  - 抽 `applyRuntimeConfig()`，替换 `Startup`/`SaveSimpleModeSettings`/`SaveMCIMSettings`/`SaveNetworkSettings` 中的重复 setter；动作集按 P0-3 清单逐条对照（含 `configureRemoteModIDConcurrency` 与 `configureProviderClients`，不只是 `downloader.Configure`）。
  - `providers.StartBackgroundTasks`/`storage.OpenAt` 留在 `Startup`，不并入。
- [x] **0.4 前端事件常量统一**
  - 新建事件名模块，替换 `Manage.vue:1237` 字面量与各 store 内局部常量。
- [x] **0.5 测试垫片归位**
  - 将仅测试使用的函数移入 `app_test.go` 或删除包装；清理 `apiKeyKeepSentinel`。

**验收**：测试与构建全绿；无 Wails 绑定签名变化；无行为差异。

### Phase 1：契约统一（P0）

- [x] **1.1 设置 DTO 单一来源**
  - `SettingsView` 与 `Save*Request` 移入 `core/structs`，`appcore` 与 `app.go` 共用。
  - 删除 `app_contracts.go` 重复定义与 `settingsViewFromCore`。
- [x] **1.2 设置保存去样板化**
  - 内部统一为“读取 → 修改 → 归一化 → `applyRuntimeConfig()` → 返回新快照”的一条路径；对外方法名保持不变。
- [x] **1.3 错误契约**
  - “未找到”与“失败”分离；移除 `LocalModsInDir` 的 `_` 错误丢弃（`service.go:1169`），并将目录不存在返回 `(nil, nil)`（`service.go:1155-1158`）一并改为显式错误。
  - 新接口一律 `(result, error)`；旧 `panic` 路径标注遗留，不扩大使用。
- [x] **1.4 生成绑定与前端适配**
  - `wails generate module` → 更新前端 import → `npm run build`。

**验收**：全仓只有一份 `SettingsView` 定义；新增设置项只需改 `configs`、DTO、前端三处；前端类型检查通过。

### Phase 2：结构拆分（P1，纯搬运）

- [x] **2.1 后端拆分**
  - `service.go` → `service_search.go` / `service_favorites.go` / `service_settings.go` / `service_downloads.go` / `service_localmods.go` / `service_runtime.go`；方法名与签名不变。
  - `storage/mods.go` 按“平台项目 / 版本快照 / 本地元数据 / releases”拆；`userdb.go` 按“收藏夹 / 固定 / 统计”拆。
  - 视痛点再拆 `modbridge`（安装状态 / 元数据富化 / 回填 / 冲突检测）。
- [x] **2.2 前端拆分**
  - `Manage.vue` → 表格、筛选工具栏、版本对话框、依赖影响对话框、未使用依赖对话框 + `useLocalModSelection` / `useUnusedDependencies`。
  - `App.vue` 下载队列面板 → `components/DownloadQueuePanel.vue`。
  - `Favorites.vue` 按列表、条目、跨版本复制对话框拆分。
  - `i18n.ts` → `locales/zh.ts` + `locales/en.ts`，公共 key 结构保持不变。
- [x] **2.3 读接口去副作用（保持行为）**
  - `GetVersions`：把缓存命中路径的 `ensureSelectedVersion` 并入 `LoadVersionsFromDisk` 完成路径。缓存被填充必然意味着本进程已完成一次加载与选择，行为保持，属于纯搬运。
  - `LocalMods(versionKey)` 内部调用 `SelectVersion` + 完整刷新，实际是写接口，**无法纯搬运去副作用**。若要前端改为显式 `SelectVersion` + 只读列表调用，属行为变更：必须按 §6 契约阶段规则同步前端与文档、单独排期验收，不得混入本阶段“方法名、行为不变”的验收标准。

**验收**：文件规模显著下降（建议单文件 < 800 行）；方法名、事件名、JSON 字段、测试行为不变。

### Phase 3：运行时实例化（P2，按需）

> 状态：按本计划的设计暂不排期。仅在出现“并行测试不稳定 / 需要多实例 / 生命周期难以排查”时启动；不属于本轮验收范围。

仅在出现“并行测试不稳定 / 需要多实例 / 生命周期难以排查”时启动。

- [ ] **3.1 `storage` 实例化**：包级 `db/userDB` → `*storage.Store`，`Service` 持有并向下传递。
- [ ] **3.2 `providers` 实例化**：后台任务生命周期归 `Registry` 所有。
- [ ] **3.3 `global` 收敛**：先拆分为 `runtime state` / `local index` / `jar cache`，再逐步实例化。
- [ ] **3.4 下载队列实例化**：`downloadQueue` 全局 → `downloader.Queue`，`Service` 持有；测试直接构造队列。
- [ ] **3.5 禁止过渡别名**：不要新增 `type X = Y` 或再导出文件；迁移一步到位。

**验收**：`New + Start` 幂等；同一进程可安全创建多个 `Service`；`go test -race ./...` 稳定通过。

---

## 5. 验收标准（整体）

1. CI 必跑后端测试、vet、gofmt、前端 build/lint/test；`core/` 的 race 测试稳定。
2. 全仓 `SettingsView`、`Save*SettingsRequest` 各只有一份定义；无手工逐字段镜像。
3. `App` 层不存在未初始化的 Service 路径；启动失败必有日志。
4. 读接口不修改全局状态（`LocalMods` 的隐式选择按 2.3 单列为行为变更项，不占纯搬运验收）；未找到与失败可区分。
5. 单文件规模目标：编排层与视图层 < 800 行。
6. 每次改动按 AGENTS.md §5.4 更新对应 `docs/specs/`（尤其是 `directory-structure.md` 与 `quality-guidelines.md`）。

---

## 6. 非目标与执行注意事项

- **不做大爆炸重写**：每个 Phase 独立可回滚，保持 Wails 方法名、事件名、JSON 字段稳定。
- **不改产品行为**：拆分阶段只搬代码；契约阶段的行为变化必须同步前端与文档。
- **`core/` 是子模块**：核心改动需要在 `core` 仓库单独提交，再更新根仓库子模块指针；绑定变更需 `wails generate module`。
- **不为假想需求引入抽象**：Phase 3 只在明确痛点出现时执行；单窗口桌面应用不需要为了“多实例”提前重构。
- **禁止过渡别名/再导出**：与 AGENTS.md §3 一致，迁移要么完成，要么留在原处。

---

## 7. 建议的起步顺序

1. Phase 0.1（CI）——先建立防回归网。
2. Phase 0.2 + 0.3（适配层与配置入口）——改动小、收益直接。
3. Phase 1.1 + 1.2（设置 DTO 与保存路径）——消除最高频的漏字段风险。
4. Phase 2.1（`service.go` 拆分）——在契约稳定后做纯搬运。
5. 其余按痛点推进；Phase 3 暂不排期。
