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

3. **`build.sh`**
   - 根目录可执行脚本
   - `DEFAULT_CF_API_KEY` + 单引号 fallback（使用用户现有配置中的 key 字面量，由实现时从构建环境/用户提供写入；**不要**在 commit message 或日志打印）
   - `APP_VERSION` + 透传 `"$@"`
   - `wails build -ldflags "..."`

4. **验证**
   - `go test ./core/configs/... ./core/appcore/...`
   - 可选：`go test ./...`（app 根）
   - 不强制完整 `wails build`（环境可能缺 wails）；至少 `bash -n build.sh`

5. **不改**
   - `.github/workflows/build.yml`
   - `.gitignore`
   - 前端 UI 文案（effective 已使 hasKey 正确，无需前端改）

## Validation Commands

```bash
go test ./core/configs/ ./core/appcore/
go test ./...
bash -n build.sh
# optional if wails available:
# DEFAULT_CF_API_KEY='test-key' APP_VERSION=dev ./build.sh
```

## Risky Files

- `core/appcore/service.go` — 出站 key 多处
- `build.sh` — shell 转义 / `$` in bcrypt-like key

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
