# AGENTS

本文件面向在本仓库工作的 AI 助手与人类开发者，给出架构边界、首要分层规则、开发与验证命令、
编码红线，以及继承自 `../mod-downloader` 的经验沉淀（docs）机制。

---

## 1. 项目概述与架构边界

本仓库是 **Wails v3 桌面应用**（搜索、下载、管理 Minecraft 模组及其依赖）。`core/` 是一个
独立 Go 模块（`github.com/link-fgfgui/mod-downloader/core`，`go.mod` 经
`replace => ./core` 注入），**同一份 core 也被 TUI 仓库 `../mod-downloader-cli` 通过
symlink `core -> ../mod-downloader-gui-v3/core` 引用**。因此本仓库存在 **GUI（React）
与 TUI（tview，位于姊妹仓库）两个前端**，共享同一条 Go 领域层。

```text
React 前端 (pages/, stores/, hooks/, lib/)          ← 渲染、本地 UI 缓存、输入体验
        │  ① 生成式绑定 frontend/bindings/（App 方法 RPC）
        │  ② Wails 事件总线（线缆名 = appcore EventKind.Name()）
Root App 适配器 (main.go / main_server.go, app.go,
                app_contracts.go, app_events.go)    ← 装配、绑定透传、原生对话框
        ▼
core/host    UI 无关运行时：core 生命周期 + HTTP 扩展服务器 + 类型化事件分发
        ▼
core/appcore.Service 业务 facade（进程级单例）
        ├── providers    Modrinth / CurseForge 集成
        ├── downloader   下载队列、分块、重试与停滞处理
        ├── modbridge    在线元数据 ↔ 本地 mod
        ├── minecraft    实例/启动器探测、jar 解析、JIJ 依赖分析
        ├── storage      SQLite（收藏、置顶、统计）+ gob+zstd 元数据缓存
        ├── configs      TOML 配置、边界常量与规范化
        ├── structs      跨前端共享投影类型（数据契约，唯一来源）
        └── models       规范域模型（单一事实源）
```

- 姊妹布局：`../mod-downloader-cli` 必须与本仓库同级存在，否则其 `core` symlink 断裂。
- 所有域结构体只定义在 `core/models` / `core/structs`；适配器与前端不得复制定义。
- core 启动失败必须 fail-fast：`GetStartupStatus` 返回类型化失败原因，前端 bootstrap
  先查它再挂载 store，不得用静默空结果掩盖。

---

## 2. 首要规则：不要在 core 之外写 UI 框架无关的逻辑

由于 **GUI 与 TUI 共享同一份 core**，任何“换一个 UI 框架（React ↔ tview）依然成立”的
逻辑，都必须**只有一份 Go 实现，且住在 core**。这是本仓库最重要的分层红线。

### 2.1 归类判据

改任何逻辑前先回答两个问题：

1. **换到另一个 UI 框架，这条规则还成立吗？**
2. **它会写回 core 状态，或两套 UI 必须表现一致吗？**

任一为“是” → 写进 core，通过 `appcore.Service` 方法或 `core/structs` 投影类型暴露，
`app.go` 只透传，前端只消费。两者都是“否” → 才允许留在前端（或仓库根适配层）。

| 属于 core（GUI/TUI 必须一致） | 允许留在前端 |
| :--- | :--- |
| 搜索表达式解析、排序、分组、兼容性判定 | 分页、焦点、抽屉、选区、表单草稿 |
| 设置边界常量与 clamp 规范化 | 动画时长表现、GSAP/FLIP、主题样式 |
| 收藏排序/作用域过滤、置顶分区 | tooltip 文案、i18n、拼音首字母输入增强 |
| 下载队列状态推进、分块/重试规则 | 命令式布局、每帧格式化（基于 core 已算好的值） |
| 事件 kind、线缆名、生命周期、HTTP 服务器 | 直接渲染投影 |

### 2.2 为什么这么严（历史教训）

- 旧仓库 `../mod-downloader`（Wails v2、单前端时代）沉淀的红线见第 5 节；进入
  GUI+TUI 双前端时代后，重复实现的问题被实测放大（详见
  `../mod-downloader-cli/docs/architecture/core-gui-tui-adaptation-plan.md`）：
  - 同一组“Manage 分组 / Jij 去重 / 兼容性判定 / 搜索文本”曾有三份实现
    （core、TUI `uistate`、GUI `lib/*.ts`），行为漂移后靠下沉 core（Phase 5）收敛；
  - 设置边界常量三份、收藏排序两份（且都漏了 `created_at` 平局规则）；
  - 三套事件枚举靠手写线缆名保持一致。
- 上述问题已经通过 `core/host` + `appcore.EventKind.Name()` + `core/structs` 下沉收敛。
  **新增逻辑时禁止重演**：想把它写进 `app.go`、`stores/*.ts` 或 `lib/*.ts` 之前，
  先按 2.1 判据归类。

### 2.3 依赖方向（机械可检查）

1. `core/` 不得导入任何适配器包；适配器（仓库根 main 包、TUI `internal/...`）可以导入 `core/*`。
2. 事件 kind 与线缆名只在 `core/appcore` 定义一次（`EventKind.Name()`，kebab-case，
   已由契约测试锁定），适配器不得再定义枚举或第二套名字。
3. 订阅 core 事件只允许 `core/host` 的 `host.On / host.OnEvent` 类型化 handler；
   `any` 断言只允许在 `core/host` 内出现一处。
4. `app.go` 不持有生命周期：启动/关闭/HTTP/事件都归 `core/host`；根目录只放
   Wails 装配、绑定透传与原生对话框。`app_contracts.go` 只收适配器特有类型
   （如原生对话框返回值）。
5. 新增 GUI 能力的标准路径是 **core 加 API/投影类型 → `app.go` 绑定 → 前端消费**；
   如果第 1 步做不到（纯 TUI 也需要同一规则），说明这条逻辑本来就不该写在前端。

---

## 3. 构建、测试与绑定

### Go

```bash
go build . && go build -tags server .   # 两种形态（窗口/无头服务器）都必须可编译
go vet ./...
go test ./...                           # 根目录契约与版本测试
(cd core && go test ./...)              # core 是独立模块，必须单独跑
```

- `app_test.go` / `app_contract_test.go` 锁定 App 绑定面（`expectedAppMethods`）：
  新增/删除 `App` 方法时必须同步更新契约测试并使其通过。
- core 是共享层：**任何 core 公共 API 或事件变化，都要到 TUI 仓库验证**
  （见第 6.4 提交前自检）。

### 前端

```bash
cd frontend
npm run build      # tsc 类型检查 + Vite 生产构建
npx vitest run     # 单元测试
```

- Go `App` 方法签名或 core 结构变化后，用 `wails3 generate bindings`
  （等价于 `build/Taskfile.yml` 的 `generate:bindings`，含 `-clean=true -ts`）重新生成
  `frontend/bindings/`；生成在 `server,production` tag 下分析同一套代码。
- 前端只消费 bindings 生成的类型，禁止手写平行 interface。

---

## 4. UI 验证

所有 UI 检查一律使用 Playwright MCP 浏览器工具。禁止使用 `spectacle`、`xwininfo`、
`uinput` 或其他桌面截图/输入自动化：通过无头 Wails 服务器的浏览器页面来检查应用，
更快、可脚本化，并能拿到 DOM 级断言。

### 无头服务器工作流

```bash
wails3 task build:server
./bin/mod-downloader-server
```

服务器默认监听 `http://127.0.0.1:9245`，可用 `MOD_DOWNLOADER_SERVER_PORT` 覆盖。
无头服务器同样受全局单实例约束（绑定 `127.0.0.1:18801`）；已有 GUI/TUI 实例运行时，
先关闭旧实例再启动，否则会以 `instance.ErrAlreadyRunning` 退出。详见
`docs/specs/core/single-instance.md`。
用 Playwright MCP 导航、检查可访问性树、点击、输入、截图。无头服务器提供的是
内嵌的生产前端加真实 Go bindings，因此 API 调用与事件都跑在真实后端上。

需要源码级 React 诊断时，先起无头服务器，再启动带 dev-server 代理的 Vite：

```bash
cd frontend
WAILS_SERVER_URL=http://127.0.0.1:9245 npm run dev -- --port 9246
```

用 Playwright 打开 `http://127.0.0.1:9246/`。这样既保留真实 bindings，又能让 Vite
提供未压缩模块与 React 错误栈。

裸跑 `cd frontend && npm run dev` 不提供 Wails bindings；端到端 UI 验证必须走无头服务器。

---

## 5. 编码标准与反模式

以下红线继承自 `../mod-downloader/docs/specs/backend/quality-guidelines.md`，
在本仓库依旧适用：

### Go

1. **禁止 type alias 过渡层**：不要写 `type SearchModResult = models.ModProject`，
   直接导入并使用 `core/models.*` / `core/structs.*`。
2. **禁止 re-export 文件**：任何包不得转口 core 已导出的类型。
3. **替换即删除**：用新实现替换转换/规则代码时，在同一批提交里删掉旧实现，
   不留 dead path。
4. **测试边界值从生产常量推导**：写 `MaxLimit + 1`，禁止会随常量演化而过期的魔法数。
5. 错误显式返回并翻译为用户可读文案，禁止吞错或“存而不用”；日志走 `core/logging`。
6. 改过的文件保持 `gofmt` / `go vet` 干净。
7. 每个源文件保留 SPDX 头：core 外 `GPL-3.0-or-later`；core 内
   `LGPL-3.0-or-later WITH LGPL-3.0-linking-exception`。core 用弱许可 + 链接例外，
   正是“适配器可替换（GUI/TUI）”这一架构决策的许可证化体现。

### 前端（React + TypeScript）

1. 弹窗/抽屉等临时 UI 状态必须在关闭与取消路径上彻底重置（不得靠页面重载清状态）。
2. 双端一致的判定一律下沉 core（见第 2 节）；尽量少写 `lib/` helper，现有展示投影
   helper（如 `modGroupDisplay.ts`）不得复刻 core 已有的排序/分组/兼容性规则。
3. i18n 修订时 `locales/en.ts` 与 `locales/zh.ts` 同批更新，不留缺失 key。

---

## 6. 经验沉淀与 docs 机制（移植自 `../mod-downloader`）

经验沉淀使用**轻量、无工具**的纯 Markdown 机制（不引入脚本、状态机、JSONL 清单），
归档在 `docs/specs/`。除本仓库已有的 UI 验证知识外，旧仓库
`../mod-downloader/docs/specs/` 中仍然适用的规范（下载/重试/停滞、连接器兼容性、
代理、存储、错误处理、UI 状态生命周期等）以那边为源；本仓库继续沉淀新增内容时，
优先复用既有文件而不是另起炉灶。

### 6.1 目录结构

```text
docs/specs/
  core/       core 领域规范（存储、下载、网络、兼容性…）
  adapter/    Wails 适配层（绑定面、事件、原生对话框、server 模式）
  frontend/   React UI（状态生命周期、动画、i18n、主题）
  guides/     思维指南（跨层检查清单、复用判断）
```

- 每个子目录配一个 `index.md` 登记所有主题文件；首个文件落地时与目录一并创建。
- core 物理上只存在于本仓库：**core 级规范以本仓库 `docs/specs/core/` 为准**，
  TUI 仓库相关文档应引用这里，而不是复制一份。

### 6.2 开发前查阅（开发前检索）

修改任何既有层或模块之前，先在 `docs/specs/`（包含 `../mod-downloader/docs/specs/`
的对应主题）找到相关规范，确认方案不违反已记录的契约，不重蹈已记录的反模式。

### 6.3 何时沉淀（沉淀触发场景）

凡涉及以下情况，必须当场沉淀：

1. **隐蔽或反复出现的 Bug**：非显而易见的陷阱（静默丢字段、goroutine 泄漏、
   竞态、事件丢失、第三方 API 怪癖）。记录问题、根因、禁止模式（Don't）、
   正确做法（Instead）。
2. **架构与契约边界变更**：新增模块、跨层负载格式变化、SQLite schema 变化、
   `App` 绑定面或 core 事件变化。更新对应 spec 或新建 `docs/specs/<层>/<主题>.md`
   并登记到 `index.md`。
3. **识别出的反模式**：重构中发现的循环依赖、重复实现（尤其违反第 2 节的）、
   状态重复三个来源等。追加到对应层规范或质量指南。
4. **关键特性方案备忘**：非平凡的方案决策（复杂依赖解析、下载分块计算、
   分组/兼容性下沉等），记录关键约束与设计取舍。

### 6.4 提交前 1 分钟闭环自检

完成任务或提交前过一遍：

- [ ] `go build . && go build -tags server .`、`go vet ./...`、`go test ./...` 通过。
- [ ] `(cd core && go test ./...)` 通过。
- [ ] 前端 `npm run build` 与 `npx vitest run` 无错误。
- [ ] 若 `App` 方法或 core 结构变了：bindings 已重新生成，`app_contract_test.go` 已更新。
- [ ] **core 公共 API / 事件变化时，姊妹仓库联动验证**：
      `cd ../mod-downloader-cli && go build ./... && go vet ./... && go test ./...`
- [ ] 本次工作是否修了隐蔽 Bug 或产生了架构洞见？**是**则在 `docs/specs/` 更新/新建
      对应规范，并且该文档变更与代码变更**进入同一批提交**。
- [ ] grep 抽查无“第二实现”复活：例如
      `rg -n "MinFileConcurrency|MinConcurrentDownloads|MaxTargetDownloadRateMiB" --glob '!core/**'`
      在适配器与前端应为 0 命中（边界规则只活在一处）。
