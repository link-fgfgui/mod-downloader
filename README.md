# Mod Downloader

Minecraft 模组下载与实例管理器。Wails v3 桌面应用（React 18 + Tailwind v4 前端，Go 后端），
支持 Modrinth / CurseForge 搜索下载、本地实例 Mod 管理（依赖树、批量启停/卸载、未使用依赖标记）、
收藏夹、跨版本复制与 packwiz 导出。

## 架构

```text
React 前端 (pages/, stores/, components/)      ← 渲染、本地 UI 缓存、输入体验
        │ ① 生成式绑定 frontend/bindings/（App 方法 RPC）
        │ ② Wails 事件总线（线缆名 = appcore EventKind.Name()）
Root App 适配器 (main.go, app.go, app_*.go)    ← Wails 装配、绑定透传、原生对话框
        ▼
core/host    UI 无关运行时：生命周期 + HTTP 扩展服务器 + 类型化事件分发
        ▼
core/appcore.Service 业务 facade（进程级单例）
        ├── providers / downloader / modbridge / minecraft
        └── storage / configs / structs / models
```

- `core/` 是独立 Go 模块（`github.com/link-fgfgui/mod-downloader/core`），同时被姊妹仓库
  `../mod-downloader-cli`（CLI/TUI）通过 symlink 引用；GUI 与 TUI 共享同一份领域层。
  任何“换 UI 框架仍成立”的逻辑必须住在 `core`。详见 [AGENTS.md](AGENTS.md)。
- 领域规范与经验沉淀在 [docs/specs/](docs/specs/)（core / adapter / frontend / guides）。

## 环境要求

- Go 1.26+（core 模块最低 1.24）
- Node.js 24+ 与 npm
- Wails v3 CLI：`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`
- Linux 需要 `libgtk-3-dev libwebkit2gtk-4.1-dev`

## 开发与构建

```bash
wails3 dev                              # 桌面开发模式（前端热更新）
wails3 task build                       # 生产桌面构建 → bin/
wails3 task package                     # 平台安装包（deb/rpm/appimage、nsis/msix、dmg）
wails3 task build:server                # 无头服务器构建 → bin/mod-downloader-server
wails3 task common:generate:bindings    # Go 绑定面变化后重新生成 frontend/bindings/
wails3 task clean:artifacts             # 清理 bin/、dist 与本地运行产物
```

测试与检查：

```bash
go build . && go build -tags server .   # 两种形态都必须可编译
go vet ./... && go test ./...
(cd core && go vet ./... && go test ./... && go test -race ./...)
cd frontend && npm run lint && npm run test && npm run build
cd ../mod-downloader-cli && go build ./... && go vet ./... && go test ./...   # core 变化后联动
```

## 无头验收（Playwright）

```bash
wails3 task build:server
./bin/mod-downloader-server          # http://127.0.0.1:9245（MOD_DOWNLOADER_SERVER_PORT 可覆盖）
```

用浏览器打开该地址即可操作真实后端（生产前端 + 真实 bindings）。无头服务器同样受单实例约束
（绑定 `127.0.0.1:18801`），已有 GUI/TUI 实例运行时需先关闭。源码级调试可另起 Vite：

```bash
cd frontend && WAILS_SERVER_URL=http://127.0.0.1:9245 npm run dev -- --port 9246
```

## 配置

配置文件为工作目录下的 `mod-downloader.toml`（首次运行自动生成）。常用键：

```toml
[preferences]
  theme = "dark"                        # dark / light / system
  language = "system"                   # system / zh / en
  minecraft_dir = "~/.minecraft"
  simple_mode = false
  animation_duration_multiplier = 1.0   # 全局动效速度倍率
  mcim_enabled = false                  # CurseForge 元数据走 MCIM 镜像
  enable_jij_detection = true           # Jar-in-Jar 子模组检测
  hardware_acceleration = true          # Linux WebKitGTK GPU 策略（重启生效）
  require_ctrl_for_multi_select = false # 开启后普通点击单选，Ctrl/Cmd 多选

[downloads]
  file_concurrency = 4
  concurrent_downloads = 1
  adaptive_file_concurrency = false
  target_download_rate_mib = 1.0
  verify_sha1 = false

[api]
  requests_per_second = 0               # 0 为不限速

[keys]
  curseforge_api_key = ""
  modrinth_api_key = ""

[logging]
  enabled = true
  max_size_mb = 6
  max_backups = 2
```

环境变量覆盖：`CF_API_KEY`、`MODRINTH_API_KEY`、`THEME`、`LANGUAGE`、`MINECRAFT_DIR`、
`MOD_DOWNLOADER_CACHE_DIR`、`MOD_DOWNLOADER_SERVER_PORT` 等（见 `core/configs/structs.go` 的
`env:` 标签）。构建期注入（release 构建）：

```bash
APP_VERSION=v1.2.3 DEFAULT_CF_API_KEY=... wails3 task build
```

`APP_VERSION` 默认取 `git describe`；`DEFAULT_CF_API_KEY` 仅在用户未配置 Key 时作为内置回退。

## 许可证

- 主程序（`core/` 之外）：GPL-3.0-or-later，见 [LICENSE](LICENSE)。
- `core/`：LGPL-3.0-or-later WITH LGPL-3.0-linking-exception，见
  [core/LICENSE](core/LICENSE) 与 [core/LICENSE.exception](core/LICENSE.exception)。
  弱许可 + 链接例外是“适配器可替换（GUI/TUI）”这一架构决策的许可证化体现。
- 每个 Go 源文件在头部声明 SPDX 标识。
