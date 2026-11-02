# TODO5 — 发布就绪收尾计划（Phase 4 修订版）

> **基线日期**：2026-11-02（在 `~/Downloads/next_development_plan.md` 2026-11-01 版基础上重新校准）
> **基线状态**：根模块 `go test ./...`、`(cd core && go test ./...)`、`go build . && go build -tags server .`、`go vet ./...`、
> 前端 `npx vitest run`（8 文件 / 25 用例）、`npm run build` 全绿；两仓库工作区干净。
> **验收方式**：遵循 `AGENTS.md` —— 无头 server + Playwright MCP，自动化基线四件套 + 姊妹仓库联动。
>
> 本计划取代 `next_development_plan.md` 中已被后续重构改变的前提；该文件的 Phase 2 桌面人工点检清单继续有效（§8 转载）。

---

## 0. 基线重校（相对 2026-11-01 计划的增量）

### 已完成（原计划列为待办，现已落地，勿重复做）

- [x] **契约与投影重构**：App 业务方法收敛为 **67 个**并通过（不再是原计划所述"70→71 待补、73!=72 失败"）。
      `GetLocalModView` 返回 `core/structs/view.LocalModView`；Manage 分组、Jij 去重、兼容判定、规范搜索文本
      全部在 `core/appcore/localmodgroups.go`（`BuildLocalModGroups`）；前端 `lib/modGroupDisplay.ts` 只做展示投影。
- [x] **TODO4 两项缺陷**：视图偏好持久化（`hooks/useManageViewMode.ts` + 单测 + `docs/specs/frontend/manage-view-mode-preference.md`）；
      Dialog 末帧闪烁（`fill-mode-forwards` 覆盖 dialog / alert-dialog / dropdown-menu / tooltip + spec）。
- [x] `components/ui/select.tsx` 已删除。
- [x] **CLI 姊妹仓库**（其架构计划项）：P1-1 jar 解析统一（`jarmeta.go` 已删）、P1-2 verify/upgrade 走 core
      （`appcore.VerifyLocalMods` 等）、P1-3 目标探测下沉（`target.go` / `resolver.go`）、P2-5 Makefile。

### 前提变化（原计划数字/路径作废，以本节为准）

1. 契约基线是 **67**；每新增一个 `Save*` 绑定方法，`app_contract_test.go` 的期望数 +1（见 §3.1 → 68）。
2. 内置 CF Key 符号名是 `configs.DefaultCurseForgeAPIKey`（**Forge 大写 F**，`core/configs/defaults.go:14`，
   注释已写明 ldflags 注入方式）；原计划写的 `DefaultCurseforgeAPIKey` 拼写无效。
3. **旧仓 `../mod-downloader` 已删除并归档为 `../archived-legacy-mod-downloader.tar.gz`**：
   `AGENTS.md` §5/§6 中所有指向旧仓 `docs/specs/` 的引用已成死链。**已决策（2026-11-02）：不从 tar 包提取旧
   spec 迁移**——旧规范大概率基于 Wails v2 / Vue 技术栈已不再适用；直接把引用改指本仓 `docs/specs/`（见 §7）。
4. `frontend` 同时存在两套 `cn`：13 个文件 `import { cn } from "cn"`（npm 包，**无 tailwind-merge**）、
   11 个文件用 `@/lib/utils`（clsx + twMerge）。这不只是冗余依赖，是行为不一致的隐患（§5.2）。

### 新增待办（原计划未含）

- 设置自动保存窄竞态：第一次保存请求在途时再次切换同一开关，响应返回会把更新后的草稿覆盖回旧值
  （`guiv3-test-work/rerun-ui-notes.md` 已知未修；既有 C17 用例 5×60ms 防抖路径正常）。见 §3.2。

---

## 1. P0-A 编译期注入：版本号与内置 CF Key

**目标文件**：`Taskfile.yml`（顶层 vars）、`build/Taskfile.yml:265`（server）、`build/linux/Taskfile.yml:62`、
`build/windows/Taskfile.yml`、`build/darwin/Taskfile.yml` 的 `BUILD_FLAGS`、`build/config.yml`、`frontend/package.json`。

- [ ] 顶层 Taskfile 定义 `APP_VERSION`（优先 git describe/tag，回退 `dev`）与 `DEFAULT_CF_API_KEY`（优先环境变量/secrets，回退空串）。
- [ ] 各平台生产 `BUILD_FLAGS` 的 `-ldflags="-w -s"` 统一改为同时注入：
      `-X main.appVersion={{.APP_VERSION}} -X github.com/link-fgfgui/mod-downloader/core/configs.DefaultCurseForgeAPIKey={{.DEFAULT_CF_API_KEY}}`
      （注意模块路径是 `github.com/link-fgfgui/mod-downloader/core/configs`，勿用旧 `mod-downloader-core`）。
- [ ] `version.go` 的 `currentAppVersion()` 已支持注入，无需改动；确认 dev 构建（`wails3 dev`）仍显示 `dev`。
- [ ] 同步 `build/config.yml` 的 `version` 与 `frontend/package.json` 的 `version`，保证安装包元数据一致。
- [ ] **验收**：生产构建后 `GetAppVersion()` 返回注入版本；未配置用户 CF Key 时 `EffectiveCurseForgeAPIKey` 返回内置 Key。

## 2. P0-B CI 门禁与发布流水线

**目标文件**：新建 `.github/workflows/build.yml`、`frontend/eslint.config.js`、`frontend/package.json`。

- [ ] **ESLint 9 Flat Config**（typescript-eslint + react-hooks 规则），`package.json` 增 `"lint": "eslint ."`，
      修至零告警（现状无 lint 脚本）。
- [ ] **verify job**（Ubuntu 单平台，PR + push 触发）：
  - `gofmt -l`（排除 `frontend/node_modules`）；
  - `cd core && go vet ./... && go test ./... && go test -race ./...`；
  - 根模块 `go vet ./... && go test ./...`；
  - 前端 `npm ci` → `npm run lint` → `npm run test` → `npm run build`；
  - `wails3 task build:server`（Linux 需 `libgtk-3-dev libwebkit2gtk-4.1-dev`）。
- [ ] **build/release matrix job**（Linux / Windows / macOS）：Go stable + Node 24 + wails3 CLI，
      注入 `APP_VERSION` 与 `secrets.DEFAULT_CF_API_KEY` 构建打包；`v*` tag 推送时创建 GitHub Release。
- [ ] **姊妹仓库联动**：`mod-downloader-cli` 无 CI 且其 `core` 是指向本仓库的 symlink——
      CI 需在同一 job 里 checkout 姊妹布局（`mod-downloader-cli` 与本仓库同级）后跑
      `cd ../mod-downloader-cli && go build ./... && go vet ./... && go test ./...`；
      或先在本仓库 CI 只做 GUI + core，CLI 仓库另建流水线（二选一，倾向前者，单点锁定共享 core 契约）。
- [ ] **验收**：PR 上 verify 全绿；打 `v*` tag 后三平台产物出现在 Release。

---

## 3. P1-A 设置收尾

### 3.1 设置页暴露 `require_ctrl_for_multi_select`

现状：字段已在 `core/structs/settings.go` 与 `core/configs/structs.go`，Download/Manage/Favorites 三处列表消费，
但仅 TOML 手改生效，无 UI 与保存链路。

- [ ] `core/structs/settings.go` 新增 `SaveMultiSelectSettingsRequest`；`core/appcore/service_settings.go` 实现
      `SaveMultiSelectSettings`（持久化到 `mod-downloader.toml`，返回更新后的 `SettingsView`）。
- [ ] `app.go` 暴露同名绑定；`app_contract_test.go` 期望数 **67 → 68**；`wails3 task common:generate:bindings` 重生成。
- [ ] `stores/settings.ts` 增加 save 分支；`pages/Settings.tsx` 增开关；`locales/en.ts` / `zh.ts` 同批补文案；
      `settings.test.ts` 补用例。
- [ ] **验收**：开关切换后重启应用仍保留；三处列表的单击/Ctrl 多选行为随开关变化（Playwright 实测两种取值）。

### 3.2 修复设置自动保存窄竞态

现状：`stores/settings.ts` 按 `AutoSaveKind` 各自防抖，但保存响应（`SettingsView`）返回后会直接写回草稿；
在途响应晚于新一次切换返回时，覆盖新草稿。

- [ ] 修复方案沿用收藏夹加载已验证的 **requestId 防覆盖** 模式：每个 `AutoSaveKind` 维护单调递增序号，
      响应仅当序号等于最新时才写回 store/草稿，过期响应丢弃。
- [ ] **验收**：新增单测——发起保存 A → 立即改草稿发起保存 B → A 响应后到，断言草稿仍为 B 的值；
      既有 C17 连点用例不回归。

---

## 4. P1-B 分包与路由懒加载

**目标文件**：`frontend/vite.config.ts`、`frontend/src/App.tsx`。

- [x] `App.tsx` 保留 `HomePage` 静态导入，其余五页 `React.lazy` + `Suspense fallback={null}`
      （stage 高度由 flex 布局决定，空 fallback 不产生高度跳变；路由入场动画作用于常驻的 stage 容器）。
- [x] ~~manualChunks 拆 vendor~~ **决策（2026-11-02）：不配置**。Vite 8 (Rolldown) 对共享模块自动拆分
      （switch/alert-dialog/useFlipListAnimation 等已是独立 chunk）；懒加载后最大 chunk 为 app 394 KB /
      Manage 331 KB，全部低于 500 KB。内嵌桌面场景无浏览器缓存收益，manualChunks/advancedChunks 无增量价值。
- [x] **验收**：`npm run build` 零 ">500 kB chunk" 警告；Playwright 实测六路由首次切换正常、路由入场方向动画不受影响。

---

## 5. P1-C 仓库卫生与依赖清理

### 5.1 静态资源与跟踪产物

- [ ] `git rm` 根目录 16 张验收截图（`verify-*.png`、`sidebar-collapsed*.png`、`tree-hover.png`、`view-toggle-hover.png`），
      `.gitignore` 增加对应模式（截图一律留在 `.playwright-mcp/` 或 `../guiv3-test-work/`）。
- [ ] 删除 `frontend/public/` 模板垃圾（约 712 KB）：`bg-desktop.jpg`、`bg-mobile.jpg`、`Inter-Medium.ttf`、
      `style.css`、`react.svg`、`wails.png`（源码零引用已核实）。
      注：`index.css:89` 声明 `font-family: Inter` 但无 `@font-face`，当前实为系统字体回退——
      若 Inter 是设计意图，改为在 `index.css` 补 `@font-face` 加载该 ttf；否则连同字体一起删并保留回退。二选一，不悬置。
- [ ] 根目录 `mod-favs.sqlite`、`mod-metadata.tmp`、`mod-downloader` 等运行产物已 gitignore；
      Taskfile 增 `clean:artifacts` 任务一键清理。
- [ ] `TODO.md`–`TODO4.md` 归档至 `docs/plans/`（本文件执行完毕后同样归档）。

### 5.2 依赖收敛（`frontend/package.json`）

- [ ] **统一 `cn`**：13 处 `import { cn } from "cn"` 迁移为 `@/lib/utils`（tailwind-merge 语义），随后移除 `"cn": "^0.5.1"` 依赖。
      迁移后需目检受影响 13 个组件的类名合并行为（`cn` 包不合并冲突类，`twMerge` 会）。
- [ ] 移除未使用依赖：`next-themes`（零引用，主题走自研 `lib/theme.ts`）、`@radix-ui/react-select`（`select.tsx` 已删）。
- [ ] **验收**：`grep -rln 'from "cn"' src/` 为空；`npm run build` + vitest 全绿；删除依赖后锁文件更新。

---

## 6. P1-D 大组件拆分与测试扩充

### 6.1 拆分（纯结构搬运，不改行为；红线：单文件 < 800 行）

| 文件 | 原行数 | 现主文件 | 拆分结果 |
| --- | ---: | ---: | --- |
| `pages/Manage.tsx` | 1219 | **552** | `components/manage/`：`ManageFilterToolbar`(256)、`ManageModTreeView`(204)、`ManageModListView`(252)、`ManageSelectionToolbar`(112)、`ManageImpactDialogs`(186) |
| `pages/Favorites.tsx` | 976 | **543** | `components/favorites/`：`FavoriteListSidebar`(220)、`FavoriteModTable`(117)、`FavoriteSelectionToolbar`(88)、`FavoriteListDialogs`(268)、`shared.ts` |

- [x] 拆分完成；两页均低于 800 行红线。**偏离原 450 目标**：再降需要把状态/effects 抽成数据 hook（行为风险大于收益），
      在 <800 红线内保留；后续如需继续收敛，优先抽 `useManageGroups` 数据管线（独立批次）。
- [x] 拆分批次内跑全量回归（自动化基线 + Playwright 抽查 Manage 树/列表 FLIP、Favorites 拖拽插入线）。
- [x] Download.tsx（761）/ Settings.tsx（632）/ DownloadQueuePanel.tsx（571）暂不拆，红线内。

### 6.2 测试扩充（8 文件 / 25 用例 → 12 文件 / 37 用例）

- [x] `components/ui/combobox.test.tsx`：搜索过滤、`allowCustom` 自由输入、键盘导航 + Enter 提交、Escape 分层（Dialog 内首次只关浮层）。
      顺带修复 `DialogOverlay` 未转发 ref（React 18 下 Radix Slot 警告）。
- [x] `components/DownloadQueuePanel.test.tsx`：空队列不渲染、Tab 切换、仅提醒时自动切 Optional、主动移除先退场后调 store。
- [x] `pages/Manage.test.tsx`：状态筛选隐藏行后批量选择被裁剪，恢复筛选后行保持未选中。
- [x] `pages/Favorites.test.tsx`：引用行（只读投影）不可选中。
- [x] `stores/settings.test.ts` 补 §3.1 / §3.2 新用例。

---

## 7. P2-A 文档与规范库迁移

- [x] 重写 `README.md`（当前仍是 Wails3 脚手架模板）：双许可证声明（主程序 GPL-3.0 / core LGPL-3.0 + linking exception）、
      TOML 配置全表（含 `require_ctrl_for_multi_select`、动画倍率、GPU Policy）、环境变量表、
      `wails3 dev` / `wails3 task build` / `build:server` 指南、无头 server + Playwright 验证工作流。
- [x] **修正 `AGENTS.md` 死链**（不提取旧仓）：§5 质量指南来源、§6.2 “开发前查阅”、§6 开头对
      `../mod-downloader/docs/specs/` 的引用，直接改指本仓 `docs/specs/`；旧仓归档仅作历史参考，不再作为规范源。
      红线内容（吞错、type alias、边界常量单源等）在本仓 §5 已有内联版本，检查无遗漏后删除对旧仓的依赖。
- [x] **验收**：文档中命令可复制执行；`docs/specs/*/index.md` 与实际文件一一对应；全仓 grep 无
      `../mod-downloader` 引用残留（归档说明除外）。已核对：仅剩 CLI 姊妹仓库（`../mod-downloader-cli`）
      与本仓目录名 `../mod-downloader-gui-v3` 的有效引用，以及本计划中的历史说明。

## 8. P2-B 桌面端人工实机点检（不阻塞自动化，产出安装包后执行）

> 转载自 `next_development_plan.md` Phase 2，继续有效。

| 验收域 | 关键代码路径 | 点检内容 |
| --- | --- | --- |
| 原生目录/保存对话框 | `app.go` `ChooseMinecraftDir` / `ChooseCacheDir` / `ExportFavoriteListPackwizZip` | 真实窗口模态挂载、中英文标题、取消静默、非法目录警告、合法路径持久化 |
| 真实下载与音画反馈 | `DownloadQueuePanel.tsx`、`utils/completionSound.ts` | 速度平滑、FAB 环形进度、完成音单次触发、SHA1 校验 |
| 扩展 HTTP 通道 | `core/httpserver`、`app.go` | 端口监听、扩展推送入队、端口占用时 `ExtensionServerError` 横幅 |
| 跨版本收藏夹复制 | `CrossVersionCopyDialog.tsx` | 预览与实际复制、目标写入、不兼容统计、界面刷新 |
| 文件管理器定位 | `app.go` `OpenLocalModLocation` | `xdg-open` / `explorer` / `open` 唤起、非法路径拦截 |
| WebKitGTK 硬件加速与主题过渡 | `main.go`、`lib/theme.ts` | GPU Policy Always/Never 启动表现、深浅主题过渡无残影 |

## 9. P2-C `core` 运行时去全局化（远期，按需触发）

触发条件不变：需要同进程多实例或并行测试出现全局单例串扰时启动。演进步骤（storage → providers → global → downloader
实例化）见 `../mod-downloader-cli/docs/architecture-improvement-plan.md` P2-2 与 `next_development_plan.md` §6.2。
在单窗口桌面形态下暂不动。

---

## 10. 执行批次与准出

| 批次 | 内容 | 状态 | 提交 |
| --- | --- | --- | --- |
| 1 | §1 注入 + §2 CI/ESLint + §5.1 截图与模板资源清理 | ✅ 完成 | `5797d60` |
| 2 | §3 设置收尾（68 方法 + 竞态）+ §4 分包懒加载 | ✅ 完成 | `6ff5637` |
| 3 | §5.2 依赖收敛 + §6 拆分与测试扩充 | ✅ 完成（主文件 552/543，未达 450，理由见 §6.1） | `b5e23f0` |
| 4 | §7 文档与规范迁移 | ✅ 完成 | `ccf17d0` |
| 独立 | §8 桌面人工点检 | 待安装包产出后人工执行 | — |
| 远期 | §9 core 去全局化 | 按需评估 | — |

### 执行期间发现并处理的新问题

1. **`DialogOverlay` 未转发 ref**：React 18 + Radix Slot 在 Dialog 内嵌 Combobox 时告警；已改 `forwardRef`（批次 3）。
2. **配置原子写残留**：进程被强杀后 `.<config>.*.tmp` 永久残留；保存前清扫 + 单测 + spec（批次 2）。
3. **Vite 8 是 Rolldown**：`manualChunks` 不适用；懒加载后 Rolldown 自动拆分已满足 <500 kB，不引入 vendor 分包（批次 2）。
4. **无头 server 启动期 WS 竞态**：首次扫描 hardlink 索引期间 `/wails/events` 拒连，前端重试并打出一批
   `ERR_CONNECTION_REFUSED`（约 10 秒窗口）；扫描完成后新页面零错误。仅影响 server 模式首屏事件连接，
   桌面端为进程内事件无此问题。记录待后续排查（可能是 server 启动顺序）。

每批收尾跑 AGENTS.md §6.4 自检：双形态构建、双模块测试、前端三件套、契约/bindings 同步、姊妹仓库联动；
有架构洞见则同批沉淀 `docs/specs/`。
