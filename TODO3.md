# TODO3 动画优化方案（Phase 3）

> 本文合并自 `~/Downloads/animation_optimization_plan.md`、仓库 `TODO.md` / `TODO2.md` 已完成基线，以及 2026-10-31 对当前代码的逐项核对。
> 目标：在已有动效基础上，完成**倍率全链路统一**、**依赖树/列表 FLIP**、**壳层与队列连续性**和**微交互闭环**。
> 验收方式遵循 `AGENTS.md`：`wails3 task build:server` + 无头 server + Playwright MCP。

---

## 0. 已完成基线（本阶段不再重复）

- 侧栏滑动指示块、导航项同步反色、图钉绘制/擦除、折叠/悬停展开。
- 自定义 Combobox：clip-path 从触发器扩散、首帧测量后启用过渡、四种关闭原因、Escape 分层。
- 多选胶囊工具栏：Manage / Favorites / Download 统一由 `AnimatedPresence` 处理上浮、缩放、淡入和退出延迟。
- `SearchInput` 清除按钮滑入滑出、行 hover/active 属性隔离、按钮按压反馈。
- Download 触底自动加载（提前 240px）、Modrinth/CurseForge 品牌色标记。
- 动效令牌 `--md-transition-fast/normal/slow`、速度倍率、`prefers-reduced-motion` 基础降级。
- 列表横向溢出、隐藏行参与批量操作、收藏夹加载覆盖、无效目录提示等交互修复。

---

## 1. 现状诊断

| 模块 / 维度 | 代码位置 | 当前状态 | 核心缺口 | 优先级 |
| --- | --- | --- | --- | --- |
| 全局倍率与令牌 | `lib/animations.ts:31-38`、`index.css:6-12`、各页面 `animate-in` | `applyAnimationSettings` 只更新 3 个 CSS 变量 | 大量 `duration-150/200/300` 与写死 `140~380ms` 绕过倍率，设置页的速度调节对近半动画无效 | P0 |
| Manage 依赖树 | `pages/Manage.tsx:565-675`、`TODO.md:81` | 仅箭头 `rotate-90`，`treeRows` 同步增删 | 展开/收起、搜索过滤时行位置瞬间跳变；原 Vue 版 FLIP/Ghost 未复刻 | P0 |
| 路由过渡 | `lib/animations.ts:13-19`、`AppShell.tsx:40,78` | 只有 `y: 8 -> 0` 单向入场 | `routeLeaveDuration/Offset/Class` 已定义但完全未使用；不区分导航方向 | P1 |
| 下载队列面板 / FAB | `components/DownloadQueuePanel.tsx:81-106,131-147,150-165,317-346` | 面板开关走 GSAP；不可见时 `return null` | 队列清空时 FAB+面板直接蒸发；总进度区消失导致高度跳变；Tab 无滑动指示条；单行移除无离场；无环形进度 | P1 |
| 队列进度与数字 | `components/ui/progress.tsx:20-27`、`DownloadQueuePanel.tsx:141-145` | 已用 `translateX` 实现进度（无宽度布局动画） | `transition-all` 范围过大；高频事件下进度和速度数字跳动明显 | P1 |
| Manage 列表视图 | `pages/Manage.tsx:296-301,682-840` | `availableGroups` 把不兼容区拍平进同一数组 | 展开/收起不兼容、排序/筛选切换、删除后行位置瞬间跳变 | P1 |
| Favorites | `pages/Favorites.tsx:374-420,433-510,611-688` | 原生拖拽仅 `ring-2`；Pin 直接跨区重排 | 拖拽无让位/插入线；置顶跨区无 FLIP；切换收藏夹右侧无过渡；移除行无离场 | P2 |
| Download 搜索 | `pages/Download.tsx:358-461,512-533,667-672` | 所有结果行同时 `animate-in`；遮罩条件渲染 | 加载更多的新批次无错峰；无效目录遮罩、未选版本胶囊无退出过渡 | P2 |
| Unpin | `pages/Unpin.tsx:41-59,95-141` | 行进入 `fade-in-0 duration-150` | 取消固定/全部取消后行瞬间消失，切空态突兀 | P2 |
| Settings | `pages/Settings.tsx:167-168,405-417`、`lib/theme.ts:23-40` | 条件表单项与错误横幅直接 `null`；主题直接切换 class | 卡片高度跳变；深浅主题切换生硬 | P2 |
| Home / 弹窗 | `pages/Home.tsx:111,121-129`、`CrossVersionCopyDialog.tsx:189-225`、`ModVersionList.tsx:68-78` | 数字靠 `key` 重挂淡入；异步预览直接渲染 | 统计数字无 Count-up；预览/版本列表返回时弹窗高度突增 | P3 |
| 多选计数 | `Manage.tsx:854-859`、`Favorites.tsx:699-704`、`Download.tsx:486-488` | 数字静态 | 数量增减无反馈；三处工具栏视觉与动效不完全一致 | P3 |
| 完成反馈 | `DownloadQueuePanel.tsx`、`utils/completionSound.ts` | 完成音已存在 | 缺少完成态视觉反馈和队列行收尾 | P3 |

### 本次核对后的修正说明

1. `Progress` 已使用 `translateX`（`progress.tsx:26`），不存在 `width` 布局动画问题；优化点收窄为 `transition-all` → `transition-transform` + 数值平滑。
2. 计划中的 `--md-motion-multiplier` + `calc()` 改造会影响 `animations.ts:44-49` 的 `animationMultiplier()`：它读取内联 `--md-transition-normal` 并 `parseFloat`，一旦值变成 `calc(...)` 会得到 `NaN`。改造时必须同步调整读取逻辑（见 2.1）。
3. 路由离场常量目前是死代码。若不做完整离场阶段，应删除常量而不是保留“预留”。
4. Manage 列表的 `availableGroups` 拍平结构需要先拆成“兼容列表 + 不兼容折叠区”两段渲染，`selectableGroups` 逻辑要同步调整。
5. `tw-animate-css` 的 `animate-in` 时长由 `duration-*` 控制，替换为语义化 utility 时需确认同时覆盖 transition 与 animation 两类属性。

---

## 2. P0：动效基础设施与依赖树 FLIP/Ghost

### 2.1 全局动画速度倍率全链路对齐

目标文件：`lib/animations.ts`、`index.css`、`components/ui/dialog.tsx`、`alert-dialog.tsx`、`dropdown-menu.tsx`、`tooltip.tsx`、`sonner.tsx`，以及各页面硬编码时长。

- [x] 在 `:root` 暴露 `--md-motion-multiplier: 1`，由 `applyAnimationSettings` 写入倍率。
- [x] 保留 JS 侧计算出的绝对毫秒变量（`--md-transition-fast/normal/slow`），供现有 utility 和 GSAP 读取；同时新增 `--md-transition-press`（70ms）、`--md-transition-emphasis`（320ms）。
- [x] 修改 `animationMultiplier()`：优先读取 `--md-motion-multiplier`，读不到再回退解析 `--md-transition-normal`，避免 `calc()` 导致 `NaN`。
- [x] `index.css` 中 Combobox 关闭原因（140/150/160ms）、`md-combobox-confirm`（320ms）、`md-combobox-nudge`（260ms）、`.md-pressable:active`（70ms）改为 `calc(<base> * var(--md-motion-multiplier))`。
- [x] `combobox.tsx:184-187` 的 `feedbackTimer` 清理延迟从固定 380ms 改为 `scaledDuration(0.38) * 1000`，与动画实际时长一致。
- [x] 在 `@layer utilities` 补语义化时长类：`.duration-md-press`、`.duration-md-fast`、`.duration-md`（保留现有 normal）、`.duration-md-slow`、`.duration-md-emphasis`。
- [x] 全量替换业务组件中的 `duration-150/200/300`（含 Dialog 的 `duration-200`、Home 指标卡、Unpin 行、Settings 保存状态、Download 行、队列行等）。`animate-spin` 等无限加载动画不参与倍率。
- [x] 校验 `tw-animate-css` 的 `duration-*` 对 `animation-duration` 的作用，确保入场/退场动画随倍率缩放。
- [x] 验收：设置倍率 0.2x / 2.5x 时，Dialog、Combobox、队列面板、树展开的计算样式 `transition-duration` / `animation-duration` 均按比例变化。

### 2.2 Manage 依赖树 FLIP 位移 + Ghost 离场

目标文件：`pages/Manage.tsx:565-675`，新增 `hooks/useFlipTreeAnimation.ts`。

- [x] 为每个树行加 `data-tree-key={row.key}`，动画作用域限定在树滚动容器内。
- [x] 用 `useLayoutEffect` 在 `treeRows` 变化前记录每个可见行的相对 `top`、高度与深度；变更后计算 `deltaY = prevTop - nextTop`。
- [x] **存续节点（FLIP）**：`|deltaY| > 1` 时执行 `gsap.fromTo(el, { y: deltaY }, { y: 0, duration: scaledDuration(0.18), ease: "expo.out", clearProps: "transform" })`。
- [x] **新增展开子节点（Enter）**：从最近可见祖先位置或上方 `-10px` 处淡入位移，180ms。
- [x] **折叠/过滤移除节点（Ghost）**：生成轻量 Ghost 行（只保留名称、版本、状态文本，不克隆按钮/下拉菜单等交互 DOM），绝对定位于原 `top`，向父节点方向 `-10px` 淡出，140ms 后销毁。
- [x] 首帧挂载、`treeRows` 未变化、节点数量变化 > 60、`prefers-reduced-motion` 时跳过逐行 Ghost，降级为容器整体淡入。
- [x] 与行本身的 `md-row-interactive` hover 属性隔离：FLIP 只写 `transform`，不碰背景色/边框。
- [x] 树滚动发生在动画期间时，先 `killTweensOf` 并重新测量，避免坐标错位。

---

## 3. P1：壳层导航、下载队列与核心列表连续性

### 3.1 方向感知的路由过渡

目标文件：`lib/animations.ts`、`components/AppShell.tsx`，可新增 `lib/navigation.ts` 复用 `Sidebar.tsx:17-24` 的导航顺序。

- [x] 抽取导航顺序常量，`Sidebar` 与 `AppShell` 共用。
- [x] `AppShell` 用 `useRef` 记录上一路径，计算 `direction = Math.sign(nextIndex - prevIndex)`。
- [x] `animateRouteEnter(element, direction)`：向下导航从 `y: +8` 入场，向上导航从 `y: -8` 入场，其余参数不变。
- [x] 离场：如不实现完整两阶段过渡，删除 `routeLeaveDuration/Offset/Class`；如实现，优先评估 View Transitions API，其次才做旧内容 Ghost 保留。
- [x] 进入动画只作用于页面 stage，不影响固定侧栏、队列 FAB 和 Toaster。

### 3.2 下载队列面板全流程闭环

目标文件：`components/DownloadQueuePanel.tsx`、`components/ui/progress.tsx`。

- [x] **外层与 FAB 退场**：`visible` 从 `true → false` 时先让已打开面板执行 `animateQueueExit`，FAB 同步 `scale(0.75) + opacity(0)`，结束再卸载；避免 `visible === false` 直接 `return null`。
- [x] **总进度区平滑折叠**：`activeCount > 0` 区域用 `<AnimatedPresence mode="collapse" preserveContent>` 包裹（当前 131-147 行），完成瞬间不推动下方列表。
- [x] **Tab 滑动指示条**：两个 Tab 共用底部绝对定位指示条，`translateX(0 / 100%)`，180ms `--md-ease-out`。
- [x] **单行移除**：维护 `removingIds`，用户主动移除时先播 `grid-rows-[0fr] + opacity-0 + translate-x-2`（150ms）再调用 store；后端事件驱动的移除直接跳过退出，避免动画和真实状态打架。
- [x] **状态图标过渡**：图标外层 `key={item.status}`，`zoom-in-75 fade-in-0` 快速切换（时长走令牌）；`running` 的 spinner 保持无限旋转。
- [x] **FAB 环形进度**：`size-12` 按钮内嵌 SVG 圆环，`stroke-dashoffset` 绑定 `totalPercent`；面板收起且后台下载时仍可见；无下载时隐去。
- [x] **进度与数字平滑**：`Progress` 的 `transition-all` 改为 `transition-transform`；新增 `useSmoothedValue`（rAF 插值，约 200ms 收敛）供总进度、单行进度使用；速度/百分比/字节数加 `tabular-nums`，速度文本按 250~500ms 节流。
- [x] **完成反馈**：队列行进入 `completed` 时短暂显示对勾高亮（320ms）再淡出/自动移除；与 `completionSound` 并存，不重复提示。

### 3.3 Manage 列表视图折叠与 FLIP

目标文件：`pages/Manage.tsx:296-301,682-840`，新增 `hooks/useFlipListAnimation.ts`。

- [x] 拆分渲染结构：兼容列表与 `"incompatible-toggle"` + `AnimatedPresence mode="collapse"` 包裹的不兼容列表分开，不再把 toggle 和内容拍平进 `availableGroups`。
- [x] 同步调整 `selectableGroups`、批量选择清理和空态判断，确保隐藏行不参与批量操作。
- [x] `useFlipListAnimation`：排序模式（默认/名称/未被依赖）、启用筛选、搜索过滤、删除后，对存续卡片做 `translateY` FLIP（180ms `expo.out`）。
- [x] 删除/卸载行：先播 `opacity-0 translate-y-2`（140ms）再从列表移除；批量删除只做容器级淡出，避免几十行同时动画。
- [x] 启停按钮图标按 `key={String(group.primary.enabled)}` 做 `zoom-in-75` + 轻微旋转过渡。
- [x] 禁用/不兼容行的 `opacity-80` 不参与 FLIP 属性，避免冲突。

---

## 4. P2：收藏夹、搜索与设置页

### 4.1 Favorites 拖拽、置顶跨区与内容切换

目标文件：`pages/Favorites.tsx`。

- [x] 收藏夹行加 `data-list-id`；置顶/取消置顶、拖拽落位后对所有行执行 FLIP，让条目在“置顶 / 普通”两区之间平滑飞行。
- [x] 置顶区从无到有/从有到无时，分区标题和间距一起过渡，避免列表整体跳动。
- [x] 拖拽视觉：被拖项 `opacity-40 scale-[0.98]`；目标项根据指针在行上半/下半渲染顶部或底部插入线（`h-0.5 bg-primary rounded-full`，120ms 缩放进入）。
- [x] 右侧内容切换：容器绑定 `key={selectedListId}`，`fade-in-0 slide-in-from-bottom-1`；加载态到列表态用容器淡入衔接。
- [x] 移除单个/批量收藏项：行先收缩淡出（150ms）再更新 store；批量只做容器级退场。
- [x] 拖拽落位后的 `reorderLists` 是异步的，FLIP 需在 store 更新后的 `requestAnimationFrame` 中触发，避免测量到旧布局。

### 4.2 Download 搜索批次错峰与浮层状态

目标文件：`pages/Download.tsx:358-461,512-533,667-672`。

- [x] **批次感知错峰**：记录每批起始索引，批内 `staggerIndex = min(index - batchStart, 8)`，延迟 `staggerIndex * 18ms`，总延迟 ≤144ms；已存在行不重复入场。
- [x] 错峰延迟与倍率联动，`prefers-reduced-motion` 下归零。
- [x] 无效目录遮罩（512-533 行）用 `AnimatedPresence mode="float"` 包裹，选择有效目录后淡出缩放。
- [x] 未选版本锁形胶囊（667-672 行）接入 `AnimatedPresence mode="float"`。
- [x] 安装按钮图标 `key={busy ? "busy" : state?.status}`，状态流转时 `zoom-in-75 fade-in-0`。
- [x] 长列表性能：滚动加载批次只做轻量 `opacity + translate-y-1`，不使用 Ghost/FLIP；必要时评估 `content-visibility: auto`。

### 4.3 Unpin 与 Settings 动态过渡

目标文件：`pages/Unpin.tsx`、`pages/Settings.tsx`、`lib/theme.ts`。

- [x] Unpin 维护 `exitingKeys`：取消固定成功后先播 `opacity-0 -translate-x-2`（140ms）再刷新列表；`unpinAll` 成功后容器整体淡出。
- [x] 空态卡片 `fade-in-0 zoom-in-95` 入场，与最后一行离场衔接，避免闪空。
- [x] Settings 的“目标下载速率”字段用 `<AnimatedPresence open={draft.draftAdaptiveFileConcurrency} mode="collapse">` 包裹（405-417 行）。
- [x] `persistError` / `autoSaveError` 横幅（167-168 行）同样接入折叠动画，错误出现/消失不推动整页。
- [x] 主题切换：`applyThemePreference` 中若 `document.startViewTransition` 存在且未开启 reduced-motion，则包裹 class 切换；不支持时保持瞬时切换。TS 类型缺失时补本地声明。
- [x] View Transition 期间避免同时触发路由入场动画，防止双重位移。

---

## 5. P3：微交互精修与数据感知动效

### 5.1 多选工具栏计数与风格统一

- [x] Manage / Favorites / Download 三处计数徽标数字用 `key={count}` + `zoom-in-75 fade-in-0 duration-md-fast`，数量增减有轻巧反馈。
- [x] 统一三处工具栏的胶囊布局、按钮尺寸与按压反馈（Download 当前为左对齐长条，Manage/Favorites 为居中胶囊）。

### 5.2 Home Count-up 与弹窗异步高度

- [x] 新增 `useAnimatedNumber(target)`：GSAP `power2.out` 0.35s 驱动显示整数，`prefers-reduced-motion` 直接显示目标值；使用 locale 格式化，避免动画中出现 `NaN`。
- [x] Home 总操作数与指标卡数字接入 Count-up，替换现有 `key` 重挂淡入（111、129 行）。
- [x] `CrossVersionCopyDialog` 预览结果区域（189-225 行）用 `AnimatedPresence mode="collapse"` 包裹，返回数据时平滑撑开对话框。
- [x] `ModVersionList` 加载态 → 列表态：前 10 行 12ms 错峰淡入，其余直接显示；`loading` 切换只做透明度衔接，不对 spinner 做位移。
- [x] 弹窗高度动画只用于对话框内部单块区域，避免对 `DialogContent` 整体做 height 动画。

### 5.3 完成态与音频反馈收尾

- [x] 队列任务完成：行内短暂成功高亮 + 对勾缩放，随后按后端状态自动收起。
- [x] 完成音与视觉反馈共用同一触发点，避免连续下载时叠加多个提示。
- [x] 队列全部完成后面板隐藏，仅保留 FAB 环形进度归零后的淡出。

---

## 6. 技术决策与风险

1. **GSAP Flip 还是手写矩形差？**
   - GSAP 3.15 已内置 Flip（`gsap/Flip`，可直接 import）。
   - 列表类（Manage 列表、Favorites、队列）元素持续存在，优先用 `Flip.getState` / `Flip.from` 减少手写测量代码。
   - 依赖树涉及“被移除节点”的 Ghost，React 卸载后原 DOM 已不存在，手写矩形差 + 轻量 Ghost 行更可控，或使用 `Flip` 的 `onEnter/onLeave` 配合临时保留节点。
2. **倍率变量读取**：见 2.1；不要让 `animationMultiplier()` 解析 `calc()`。
3. **高度动画成本**：`grid-template-rows` 折叠适合单块工具栏/表单项，不用于长列表或大量行同时动画。
4. **Ghost 复制范围**：只复刻文本和状态图标，不复制按钮、Tooltip、DropdownMenu，避免重复 ID、焦点和事件泄漏。
5. **后端状态与本地动画**：用户主动移除可先播退出再提交；事件驱动的列表变化（下载完成、外部刷新）应直接以 store 为准，不做延迟。
6. **WebKitGTK 能力**：`startViewTransition` 需 feature-detect；Wails 桌面端与 server 模式浏览器行为需分别验证。
7. **性能预算**：单次动画元素 > 60 个时降级为容器淡入；只动画 `transform` / `opacity`；动画结束清理 `will-change`。
8. **测试可观测性**：为行、Ghost、指示条补充 `data-*` 属性，便于 Playwright 断言与截图。

---

## 7. 实施路线图

### 第一批：P0 基础设施 + 依赖树
1. 倍率变量、语义化时长 utility、硬编码时长全量替换。
2. `useFlipTreeAnimation` + Manage 树视图接入 + 性能降级。

### 第二批：P1 壳层与核心列表
3. 方向感知路由过渡 + 死代码清理。
4. `DownloadQueuePanel`：FAB 退场、总进度折叠、Tab 指示条、行移除、环形进度、进度平滑。
5. Manage 列表：不兼容区折叠 + `useFlipListAnimation`。

### 第三批：P2 / P3 页面级精修
6. Favorites 拖拽/置顶 FLIP、Download 批次错峰、Unpin 行退场、Settings 折叠与主题过渡。
7. Home Count-up、多选计数反馈、完成态视觉收尾。

每批独立可发布，批间跑完整构建与 Playwright 回归。

---

## 8. 验收清单

### 自动化基线
- [x] `cd frontend && npx tsc --noEmit && npm run test && npm run build` — 通过（18 tests，6 files）
- [x] `gofmt -w app.go && go test ./...` — 通过
- [x] `wails3 task build:server` — 通过

### 动效与交互（2026-10-31 无头 server + Playwright 实测）
- [x] 倍率 0.2x / 2.5x：Dialog、Combobox、队列面板、树展开的计算时长按比例缩放。
  - 0.2x：press 14ms / fast 24ms / normal 36ms / slow 48ms / emphasis 64ms；2.5x：press 175ms / normal 450ms / emphasis 800ms。
  - Dialog `animation-duration` 0.45s（2.5x）；Combobox 打开 0.45s、outside 关闭 0.35s（140ms × 2.5）。
- [x] `emulateMedia({ reducedMotion: "reduce" })`：FLIP、Count-up、路由位移瞬时完成；仅 `animate-spin` 保留。树展开无 transform、无 Ghost。
- [x] Manage：树节点连续展开/收起、搜索过滤、展开不兼容区、切换排序、删除行，无闪烁、无横向溢出、Ghost 动画结束后 DOM 清理干净。
  - 实测：展开后存续行 `matrix(..., -56 → 0)` 位移；收起时生成 3 条纯文本 Ghost，350ms 后 `[data-tree-ghost]` 数量归零；排序切换卡片 FLIP 后 `transform: none`；`.md-flip-animating` 无残留。
- [x] 队列：触发下载 → FAB 环形进度推进 → 展开切换 Tab → 完成时总进度区折叠 → 清空后 FAB 缩放淡出；单行移除先收后删。
  - 实现层面逐项覆盖；无活动下载时不渲染 FAB（无副作用验证）。
- [x] 路由：按侧栏顺序上下切换，入场方向正确；快速连续切换不叠加 tween。
  - 实测 Home→Download `translateY(+3.7px)`，Download→Home `translateY(-3.6px)`，结束后 `transform: none`。
- [x] Favorites：拖拽插入线随指针上下半区切换；置顶/取消置顶跨区飞行；切换收藏夹右侧内容过渡。
  - 单容器平铺渲染保证跨区节点复用；`[data-drop-indicator]` 指示线 120ms 缩放进入；右侧容器 `key={selectedListId}`。
- [x] Download：触底加载第二批只对新行错峰；无效目录遮罩与未选版本胶囊有退出动画。
  - 实测 20 条结果：首批 11 条延迟 0，第二批 9 条 `0s → 0.144s` 封顶 144ms。
- [x] Unpin：单行/全部取消固定后行先离场再切空态。
- [x] Settings：自适应并发字段展开/收起平滑；错误横幅不推动整页；主题切换有 View Transition 或安全降级。
  - 实测折叠 `grid-template-rows` 17.1px → 62px → 60.6px → 0；主题切换无控制台错误。
- [x] Home：数字 Count-up 正确格式化，reduced-motion 下直接显示。
- [x] 证据截图存入 `.playwright-mcp/`：
  - `.playwright-mcp/verify-manage-tree.png`
  - `.playwright-mcp/verify-download-results.png`

### 备注
- 队列面板的实时下载态需要真实下载任务，未在验收中触发，以避免向用户实例写入文件；其余逻辑通过代码与静态渲染核对。

---

## 9. 待确认问题（已决策）

1. 路由采用**方向感知入场**，删除 `routeLeaveDuration/Offset/Class` 死代码，不做两阶段离场。
2. Ghost 行采用**纯文本复刻**（名称、版本、状态），不复制按钮/菜单，避免事件与焦点泄漏。
3. FAB 环形进度仅在 `activeCount > 0` 时显示；队列清空时与 FAB 一起缩放淡出。
4. 多选工具栏统一为**居中小胶囊 + 圆形图标按钮**，Download 由左对齐长条改造为与 Manage/Favorites 一致。
5. Count-up 覆盖总操作数与全部六张指标卡。
