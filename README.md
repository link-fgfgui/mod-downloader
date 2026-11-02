# Mod Downloader

Minecraft mod downloader and instance manager. A Wails v3 desktop app (React 18 +
Tailwind v4 frontend, Go backend) for Modrinth / CurseForge search and download,
local instance mod management (dependency tree, batch enable/disable/uninstall,
unused-dependency detection), favorites, cross-version copy, and packwiz export.

## Architecture

```text
React frontend (pages/, stores/, components/)   ← rendering, local UI cache, input
        │  ① generated bindings in frontend/bindings/ (App method RPC)
        │  ② Wails event bus (wire names = appcore EventKind.Name())
Root App adapter (main.go, app.go, app_*.go)     ← Wails wiring, binding pass-through, native dialogs
        ▼
core/host    UI-agnostic runtime: lifecycle + HTTP extension server + typed event dispatch
        ▼
core/appcore.Service  business facade (process-wide singleton)
        ├── providers / downloader / modbridge / minecraft
        └── storage / configs / structs / models
```

- `core/` is a standalone Go module (`github.com/link-fgfgui/mod-downloader/core`).
  The sibling CLI/TUI repository `../mod-downloader-cli` consumes it through a
  symlink, so both frontends share one domain layer. Any logic that still holds
  after swapping the UI framework must live in `core` — see [AGENTS.md](AGENTS.md).
- Durable specifications live in [docs/specs/](docs/specs/) (`core/`, `adapter/`,
  `frontend/`).

## Requirements

- Go 1.26+ (the `core` module requires 1.24)
- Node.js 24+ and npm
- Wails v3 CLI: `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0`
- Linux: `libgtk-3-dev libwebkit2gtk-4.1-dev`

## Build and test

```bash
wails3 dev                              # desktop dev mode with frontend hot reload
wails3 task build                       # production desktop build → bin/
wails3 task package                     # platform installer (deb/rpm/appimage, nsis/msix, dmg)
wails3 task build:server                # headless server build → bin/mod-downloader-server
wails3 task common:generate:bindings    # regenerate frontend/bindings/ after Go binding changes
wails3 task clean:artifacts             # remove bin/, dist and local runtime artifacts
```

Verification baseline:

```bash
go build . && go build -tags server .   # both window and headless forms must compile
go vet ./... && go test ./...
(cd core && go vet ./... && go test ./... && go test -race ./...)
cd frontend && npm run lint && npm run test && npm run build
cd ../mod-downloader-cli && go build ./... && go vet ./... && go test ./...   # after core changes
```

## Headless verification (Playwright)

```bash
wails3 task build:server
./bin/mod-downloader-server          # http://127.0.0.1:9245 (override with MOD_DOWNLOADER_SERVER_PORT)
```

Open that URL in a browser to drive the real backend (production frontend plus
real bindings). The headless server is also bound by the global single-instance
guard (`127.0.0.1:18801`); close any running GUI/TUI instance first. For
source-level React debugging, start Vite against the running server:

```bash
cd frontend && WAILS_SERVER_URL=http://127.0.0.1:9245 npm run dev -- --port 9246
```

## Configuration

The app reads `mod-downloader.toml` in the working directory (created on first
run). Common keys and their defaults:

```toml
[preferences]
  theme = "dark"                        # dark / light / system
  language = "system"                   # system / zh / en
  minecraft_dir = "~/.minecraft"
  simple_mode = false
  animation_duration_multiplier = 1.0   # global motion speed multiplier
  mcim_enabled = false                  # CurseForge metadata through the MCIM mirror
  enable_jij_detection = true           # Jar-in-Jar sub-mod detection
  hardware_acceleration = true          # Linux WebKitGTK GPU policy (restart required)
  require_ctrl_for_multi_select = false # plain click selects one row; Ctrl/Cmd adds

[downloads]
  file_concurrency = 4
  concurrent_downloads = 16
  adaptive_file_concurrency = false
  target_download_rate_mib = 1.0
  verify_sha1 = false

[api]
  requests_per_second = 0               # 0 = unlimited

[keys]
  curseforge_api_key = ""
  modrinth_api_key = ""

[logging]
  enabled = true
  max_size_mb = 6
  max_backups = 2
```

Environment overrides: `CF_API_KEY`, `MODRINTH_API_KEY`, `THEME`, `LANGUAGE`,
`MINECRAFT_DIR`, `MOD_DOWNLOADER_CACHE_DIR`, `MOD_DOWNLOADER_SERVER_PORT`, and
others declared by the `env:` tags in `core/configs/structs.go`. Release builds
can inject values at link time:

```bash
APP_VERSION=v1.2.3 DEFAULT_CF_API_KEY=... wails3 task build
```

`APP_VERSION` defaults to `git describe`; `DEFAULT_CF_API_KEY` is only used as a
fallback when the user has not configured a key.

## License

- Main program (outside `core/`): GPL-3.0-or-later, see [LICENSE](LICENSE).
- `core/`: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception, see
  [core/LICENSE](core/LICENSE) and [core/LICENSE.exception](core/LICENSE.exception).
  The weaker license plus linking exception is the licensing expression of the
  replaceable-adapter (GUI/TUI) architecture.
- Every Go source file carries an SPDX header.
