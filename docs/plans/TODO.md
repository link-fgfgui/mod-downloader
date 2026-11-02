# TODO

## 已完成

- [x] 迁移 `core` 业务模块
- [x] 迁移 70 个后端方法
- [x] 迁移 9 个事件
- [x] Wails v2 适配层切换为 Wails v3
- [x] 添加后端契约测试
- [x] 生成 Wails v3 TypeScript bindings
- [x] 安装 React、shadcn/ui、GSAP、Zustand、i18n 依赖
- [x] 初始化 shadcn/ui 基础组件
- [x] 迁移中英文文案和纯 TS 工具
- [x] 建立 minecraft、search、queue、settings、favorites、pins、stats stores

## 本阶段迁移

- [x] 搭建 AppShell、六路由、侧栏和启动失败页
- [x] 迁移 Home 页面
- [x] 迁移 Unpin 页面
- [x] 迁移 Settings 页面
- [x] 迁移 Download 页面
- [x] 迁移 Manage 页面
- [x] 迁移 Favorites 页面
- [x] 完成全局下载队列面板
- [x] 接入 GSAP 动画入口，本阶段仅预留
- [x] 前端 Vitest 测试
- [x] Go 两个模块测试
- [x] TypeScript 检查与 Vite 构建
- [x] Wails v3 构建与手工验收

## 手工验收记录

- 通过 Wails v3 server 模式 + Playwright 浏览器验收六个路由、侧栏、版本选择、实例切换、下载队列、收藏夹、固定/取消固定、搜索安装、启停/卸载本地 Mod、设置自动保存与中英文切换；启动失败页由组件测试覆盖。
- 修复验收中发现的问题：Manage 页在未选择实例时 Zustand selector 返回新数组导致的无限更新；`Button`/`Badge` 缺少 ref 转发导致 Tooltip 警告；语言切换后 toast 仍使用旧语言。

## 全功能验收（2026-10-31）

按 `../guiv3-test-todo.md` 完成 Playwright 验收并修复 P0/P1：

- 渲染基线：补齐 `--popover` 令牌；引入 `tw-animate-css`，Radix 弹层/对话框/Toast 背景与进出场正常（深/浅主题截图确认）。
- 功能修复：Manage 隐藏行不再参与批量操作；批量下载双击不再重复入队；“搜索新版本”落地自动执行搜索；API Key 自动/手动保存竞态；配置落盘失败不再假成功（新增 `SettingsView.saveError`）；Unpin 全局取消文案；收藏夹图标清除后被保存复活；引用自/重复引用误报成功；复制目标排除源列表；未选实例的收藏夹空态提示；就绪态扩展 HTTP 错误横幅；server 模式退出应用真正退出；下载状态未就绪时点击安装给出提示。
- 体验/健壮性：收藏夹加载 requestId 防覆盖、首进只加载一次；下载分页跨路由重置；无效目录遮罩提供选择目录按钮；折叠侧栏可键盘展开；引用项不可选中；复制/取消固定/移除等静默操作补 toast 与 catch；Unpin 批量守卫/首屏空表/错误横幅；下载行 metadata 状态 aria；数字输入 clamp；启动失败页新增“重试”。
- 验证基线：`cd frontend && npm run test`（18 passed）、`go test ./...`、`npm run build`（tsc + Vite）、`wails3 task build:server` 全绿。证据在 `../guiv3-test-work/`（`screens/`、`result-*.json`、`server.log`）。
- 尚未覆盖：桌面原生目录/保存对话框、拖拽排序、跨版本复制执行、硬件加速真实效果、完成提示音、扩展通道真实推送；详见验收报告 §2/§4。

## UI 打磨（2026-10-31）

- 侧栏：活动路由改为滑动指示块；滑动过程中经过的导航项同步反色（裁剪覆盖层跟随指示块）；移除导航项 hover 底色，避免 hover 强调与滑动色块重叠闪烁；折叠按钮改为自绘图钉：固定态只显示图钉（无斜线），释放/可折叠时 `\` 沿路径画出，重新固定时擦除；斜线经过图钉处通过同步动画的 mask 留出缺口；导航标签随侧栏宽度淡出/收拢；折叠态图标重新居中。侧栏悬停展开/收起改为全部保持挂载：导航图标不再切换 `justify-center`（整段过渡只位移约 4px），底部版本面板固定 256px 宽并常驻、用透明度/可见性交叉淡化，头部按钮与折叠态图钉常驻淡入淡出，避免快速进出时 DOM 换入换出造成的位置跳动。
- 通知：Toaster 从底部左侧移到右上角。
- Download：搜索结果的“加载更多”改为滚动触底自动加载（IntersectionObserver，提前 240px 预取）；保留加载中、失败重试和无更多结果状态。Modrinth/CurseForge 标记改用各自品牌色（绿/橙，含深浅主题）。
- Manage：列表视图/依赖树视图切换加入滑动指示块；行选中、勾选框和依赖树箭头补充过渡。
- 批量操作条：Manage 与 Favorites 使用折叠高度动画，Download 使用上浮淡入动画；统一由 `components/AnimatedPresence.tsx` 处理进入、退出和卸载延迟。
- 其他动效：下载搜索状态、队列行/标签、队列悬浮按钮图标与徽标、Home 指标卡与数字、Settings 保存状态、不确定进度条、按钮按压反馈。
- 动效时长统一走 `--md-transition-*` / 动画速度倍率；`prefers-reduced-motion` 下关闭过渡但保留加载 spinner。
- 新增配置项 `preferences.require_ctrl_for_multi_select`（默认 `false`）：开启后普通点击只选中当前项，Ctrl/Cmd+点击才增减选择；作用于下载、管理、收藏夹三处列表。仅配置文件生效，未在设置页暴露。

```toml
[preferences]
  require_ctrl_for_multi_select = false
```

补充验证：侧栏滑块 0 → 220px（180ms），经过项反色且 hover 无第二层底色；折叠按钮的 `\` 与 mask 同步在 `stroke-dashoffset` 1 ↔ 0 间用 180ms 画出/擦除（固定态无斜线，释放态带斜线；有斜线的图钉在折叠栏中仍可见）；折叠宽度 256 → 64、图标中心与条目中心对齐；Manage 视图指示块 695/88 → 783/122；三类批量条进入/退出均可见；下载滚动加载实测 20 → 40 → 60 条，底部状态从“正在加载更多…”回到“向下滚动加载更多”；Modrinth/CurseForge 标记实测为绿色/橙色（边框、文字、底色均使用品牌色）；该配置开关两种取值下的单选与 Ctrl 多选行为均实测通过。证据截图在 `../guiv3-test-work/screens-run2/`。

## TODO2 交互与动效修复（2026-10-31 第二轮）

- 多选操作条：Manage、Favorites 改为浮于列表底部的胶囊工具栏，不再插入列表下方；与 Download 统一使用 `AnimatedPresence` 的上浮/缩放/淡入动画，退出时保留计数与按钮直到收起完成。
- 悬停与按压：列表行、依赖树行、收藏项、搜索结果统一使用 `md-row-interactive` 与过渡令牌；选中态和依赖关系色优先于 hover，不再被 hover 底色覆盖。按钮基类补充更明显的 `active:scale`、快速按下/缓出释放；树节点、版本链接、队列标签、搜索清除、图片按钮等自定义点击元素补齐按压反馈。
- 列表稳健性：Manage 列表/依赖树、Favorites 与 Download 结果列表补充 `min-w-0` 和横向溢出裁剪，消除横向滚动条与行宽超界；Download 页仅在当前关键字实际搜索过后显示“无匹配”，避免输入过程中误报。
- 帮助提示：Manage 搜索语法 `?` 只保留 hover/focus 一种交互；阻止 Radix 默认的点击关闭，点击图标后提示保持可见。
- 版本链接：Manage mod 版本下划线只覆盖版本文字，前面的 `·` 分隔符独立显示。
- Combobox：以自定义浮层替换所有 Radix Select 与 datalist。面板使用 portal + 固定定位，自动选择上下方向；首帧完成测量后禁用过渡再启用，展开时用裁剪路径从触发器方向整体扩散（不再逐项错峰），结束位置与内容测量一致，避免收尾闪动。关闭原因区分选新值（向触发器收回 + 触发器高亮）、重复选当前值（轻微回弹 + 触发器收缩）、点击外部（原位淡出缩小）、Escape（轻微远离）。
- 左下角版本区：所有控件宽度约束在侧栏内，长目录路径省略显示；Minecraft 版本改为有限选项的可搜索 Combobox，不能输入任意值；收藏夹列表名使用支持自由输入的 Combobox。侧栏较矮时导航区可滚动且隐藏滚动条，版本区与目录按钮保持完整可见。
- 搜索框：Manage 与 Download 使用带滑入/滑出清除按钮的 `SearchInput`，清空后按钮向右收起并释放内边距。
- 文件夹选择器：`ChooseMinecraftDir` 改为返回 `DirectorySelectionResult`（path/canceled/invalid），取消保持静默，无效目录与对话框错误给出 toast；选择成功后持久化 Minecraft 目录偏好。目录/缓存目录/导出对话框均传入当前窗口与默认目录，缓存目录选择错误也可反馈。重建 Wails bindings。
- 键盘与弹层层级：Combobox 在捕获阶段接管 Escape，弹层打开时不再触发全局返回；Dialog 内有 Combobox 时第一次 Escape 只关闭浮层，第二次才关闭对话框。`AnimatedPresence` 归档退出内容，避免收起过程中计数/按钮提前清空。
- 验证：`cd frontend && npm run test`（18 passed）、`go test ./...`、`npm run build`、`wails3 task build:server` 全部通过；Playwright 实测侧栏 1280×800 与 932×560 无裁切、Combobox 上/下展开与三种关闭原因、搜索清除、帮助提示点击保持、Manage/Favorites 浮层工具栏、Dialog 内 Combobox Escape 分层、列表无横向溢出。证据截图在 `.playwright-mcp/`（`sidebar-picker-final.png`、`manage-tree-final.png`、`final-manage-bar.png`、`final-favorites-bar.png`、`final-fav-dialog.png`）及仓库根目录本轮复测截图（`verify-row-hover.png`、`verify-tree-hover.png`、`verify-version-underline.png`、`verify-combobox-open.png`、`verify-fav-bar.png`、`verify-download-bar.png`、`verify-search-clear.png`、`verify-search-cleared.png`、`verify-search-idle.png`、`verify-search-no-results.png`）。

## 已知限制

- Manage 依赖树保留了展开、筛选和节点操作，节点箭头已有旋转过渡，但未复刻 Vue 版的 FLIP/ghost 过渡动画；本阶段只要求接入 GSAP 入口，路由与下载队列面板已使用。
- 生产构建仍有单个 chunk 大于 500 kB 的 Vite 警告，功能不受影响。
- 浏览器验收使用 Wails server 模式；原生桌面窗口功能（例如文件选择对话框）仍需在桌面构建中验证。
