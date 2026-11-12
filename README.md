# Mod Downloader

**简体中文** | [English](README_EN.md)

Minecraft 模组下载器与实例管理器。基于 Wails v3 开发的桌面应用（React 18 + Tailwind v4 前端，Go 后端），支持 Modrinth / CurseForge 搜索与下载、本地实例模组管理（依赖树分析、批量启用/禁用/卸载、无用依赖检测）、收藏夹、跨版本复制及 packwiz 导出。

## 架构

```text
React 前端 (pages/, stores/, components/)       ← 渲染、本地 UI 缓存、用户交互
        │  ① 自动生成的绑定 frontend/bindings/ (App 方法 RPC)
        │  ② Wails 事件总线 (wire 协议名称 = appcore EventKind.Name())
根 App 适配器 (main.go, app.go, app_*.go)      ← Wails 装配、绑定透传、系统原生对话框
        ▼
core/host    与 UI 无关的运行时：核心生命周期 + HTTP 扩展服务器 + 类型安全事件派发
        ▼
core/appcore.Service  业务门面（进程级单例）
        ├── providers / downloader / modbridge / minecraft
        └── storage / configs / structs / models
```

- `core/` 是一个独立的 Go 模块 (`github.com/link-fgfgui/mod-downloader/core`)。同级的 CLI/TUI 项目 `../mod-downloader-cli` 通过符号链接直接引用它，使两个前端共享同一套领域层代码。任何在更换 UI 框架（如 React ↔ tview）后依然成立的业务逻辑，都必须集中在 `core` 中实现 — 详见 [AGENTS.md](AGENTS.md)。
- 长期技术规范：core 领域规则见 [core/docs/specs/](core/docs/specs/)，adapter/frontend 规则见 [docs/specs/](docs/specs/)。

## 环境要求

- Go 1.26+（`core` 模块要求 1.24）
- Node.js 24+ 与 npm
- Wails v3 CLI：`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`
- Linux：`libgtk-4-dev libwebkitgtk-6.0-dev`（GTK4 + WebKitGTK 6.0；不再使用旧版 GTK3 依赖栈）

## 构建与测试

```bash
wails3 dev                              # 桌面开发模式（支持前端热重载）
wails3 task build                       # 生产环境桌面构建 → bin/
wails3 task package                     # 打包平台安装包（deb/rpm/appimage、nsis/msix、dmg）
wails3 task build:server                # 无头 (Headless) 服务端构建 → bin/mod-downloader-server
wails3 task common:generate:bindings    # Go 绑定变更后重新生成 frontend/bindings/
wails3 task clean:artifacts             # 清理 bin/、dist 及本地运行时生成物
```

基础校验基准：

```bash
go build . && go build -tags server .   # 窗口版和无头版均须通过编译
go vet ./... && go test ./...
(cd core && go vet ./... && go test ./... && go test -race ./...)
cd frontend && npm run lint && npm run test && npm run build
cd ../mod-downloader-cli && go build ./... && go vet ./... && go test ./...   # 修改 core 后校验
```

## 无头服务验证 (Playwright)

```bash
wails3 task build:server
./bin/mod-downloader-server          # 默认监听 http://127.0.0.1:9245（可通过 MOD_DOWNLOADER_SERVER_PORT 覆盖）
```

在浏览器中打开该 URL 即可驱动真实后端（包含生产前端和真实的 Go 绑定）。无头服务端同样受全局单实例锁（`127.0.0.1:18801`）约束，启动前请确保关闭正在运行的 GUI/TUI 实例。如需源码级 React 调试，可在服务端运行状态下启动 Vite 开发服务：

```bash
cd frontend && WAILS_SERVER_URL=http://127.0.0.1:9245 npm run dev -- --port 9246
```

## CurseForge 下载地址回退

CurseForge API 有时不返回文件的直接下载地址。默认构建会根据文件 ID 与文件名重建 `edge.forgecdn.net` CDN 地址；该行为可通过编译标签关闭：

```bash
go build .                                   # 默认启用 CDN 地址回退
go build -tags server .                      # 无头版同样启用
go build -tags no_curseforge_cdn_fallback .  # 严格模式：不重建 CDN 地址
```

启用 `no_curseforge_cdn_fallback` 后，Core 会拒绝处理没有可用下载地址的 CurseForge 模组：

- 下载页保留条目，但下载按钮显示为不可点击的 X，且不会触发元数据重试；
- 本地模组管理不会应用它们的在线元数据，表现为 CurseForge 解析失败，且无法替换版本；
- 显式安装失败时，失败原因会给出官方文件页 `https://www.curseforge.com/minecraft/mc-mods/{slug}/download/{fileId}`。该页面不是 CDN 地址，但会在浏览器中跳转到 CDN。

CI 默认构建不启用严格标签。该编译标签作用于共享 Core；GUI 会消费 Core 返回的 `missingDownloadUrl` 状态并禁用对应下载按钮。

## 配置说明

应用首次启动时会在当前工作目录下生成 `mod-downloader.toml`。常用配置项与默认值：

```toml
[preferences]
  theme = "dark"                        # 主题：dark / light / system
  language = "system"                   # 语言：system / zh / en
  minecraft_dir = "~/.minecraft"
  simple_mode = false
  animation_duration_multiplier = 1.0   # 全局动效速率乘数
  mcim_enabled = false                  # 是否启用 MCIM 镜像获取 CurseForge 元数据
  enable_jij_detection = true           # 是否检测 Jar-in-Jar 内嵌子 Mod
  hardware_acceleration = true          # Linux WebKitGTK GPU 硬件加速策略（修改需重启）
  require_ctrl_for_multi_select = false # 普通点击选中单行；Ctrl/Cmd 增选

[downloads]
  file_concurrency = 4
  concurrent_downloads = 16
  adaptive_file_concurrency = false
  target_download_rate_mib = 1.0
  verify_sha1 = false

[api]
  requests_per_second = 0               # 0 表示不限制请求速率

[keys]
  curseforge_api_key = ""
  modrinth_api_key = ""

[logging]
  enabled = true
  max_size_mb = 6
  max_backups = 2
```

环境变量覆盖：`CF_API_KEY`、`MODRINTH_API_KEY`、`THEME`、`LANGUAGE`、`MINECRAFT_DIR`、`MOD_DOWNLOADER_CACHE_DIR`、`MOD_DOWNLOADER_SERVER_PORT`，以及 `core/configs/structs.go` 中通过 `env:` 标签声明的其他变量。发布构建时可在链接期注入值：

```bash
APP_VERSION=v1.2.3 DEFAULT_CF_API_KEY=... wails3 task build
```

`APP_VERSION` 默认使用 `git describe`；`DEFAULT_CF_API_KEY` 仅在用户未配置 Key 时作为回退值使用。

## 许可证

- 主程序（`core/` 目录之外）：GPL-3.0-or-later，详见 [LICENSE](LICENSE)。
- `core/`：LGPL-3.0-or-later WITH LGPL-3.0-linking-exception，详见 [core/LICENSE](core/LICENSE) 与 [core/LICENSE.exception](core/LICENSE.exception)。
- 每个 Go 源码文件均包含对应的 SPDX 头声明。
