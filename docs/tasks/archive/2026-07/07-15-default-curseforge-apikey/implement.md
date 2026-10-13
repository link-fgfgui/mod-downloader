# Implement: 编译期默认 CurseForge API Key

## Checklist

1. **configs 默认变量 + helper**
   - 文件：`core/configs/defaults.go`（或挂在 `structs.go` 旁的新文件）
   - 增加 `DefaultCurseforgeAPIKey`、`EffectiveCurseforgeAPIKey`
   - 单测：空配置回退、非空优先、两边空白 trim

2. **appcore 改用 effective key**
   - `configureProviderClients`
   - `QueueModDownload` / `InstallModAndWait` / `InstallOptionalDependencies*`
   - `GetSettings` 的 has/mask
   - grep 确认无其它出站路径仍用裸配置 key
   - 单测：空配置 + 注入默认 → CF 可配；用户 key 优先；Save 清除不写默认进 config

3. **GitHub Actions build**
   - 更新 `.github/workflows/build.yml`
   - 使用 `secrets.DEFAULT_CF_API_KEY`，不在仓库或日志中保存真实值
   - 保留 tag/commit 派生的 `APP_VERSION`
   - 通过一个 `wails build -ldflags "..."` 同时注入 version 与 key

4. **验证**
   - `go test ./core/configs/... ./core/appcore/...`
   - 可选：`go test ./...`（app 根）
   - 检查 workflow 中的 Wails build 命令和 secret 引用

5. **不改**
   - `.gitignore`
   - 不新增本地构建 wrapper
   - 前端 UI 文案（effective 已使 hasKey 正确，无需前端改）

## Validation Commands

```bash
go test ./core/configs/ ./core/appcore/
go test ./...
# optional if wails available:
# DEFAULT_CF_API_KEY='test-key' APP_VERSION=dev wails build -ldflags "-X main.appVersion=${APP_VERSION} -X github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey=${DEFAULT_CF_API_KEY}"
```

## Risky Files

- `core/appcore/service.go` — 出站 key 多处
- `.github/workflows/build.yml` — Secret 引用、ldflags 引号与 `$` 字符

## Rollback Points

- After step 1: pure additive, safe
- After step 2: behavior change only when default non-empty
- After step 3: build path only

## Gate Before `task.py start`

- [x] prd.md converged
- [x] design.md present
- [x] implement.md present
- [ ] implement.jsonl / check.jsonl curated
- [ ] user review / approve start
