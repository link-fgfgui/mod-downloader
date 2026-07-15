# Design: 编译期默认 CurseForge API Key

## Architecture

```
build.sh / wails -ldflags
        │
        ▼
configs.DefaultCurseforgeAPIKey  (linker-injected)
        │
        ▼
configs.EffectiveCurseforgeAPIKey(configured)
        │
        ├── appcore.configureProviderClients  → x-api-key / CF client on/off
        ├── appcore download / optional deps  → downloader key arg
        └── appcore.GetSettings               → hasCurseforgeKey + mask
```

优先级（高 → 低）：

1. 用户配置字段 `Keys.CurseforgeApiKey`（来自 TOML / `KEYS_CF_API_KEY` / UI 保存 / `ConfigOverrides`）
2. `configs.DefaultCurseforgeAPIKey`（编译期）
3. 空 → 官方 CF 关闭（MCIM 仍可按现有逻辑开启）

## Boundaries

| 层 | 职责 |
|----|------|
| `core/configs` | 持有默认变量 + `EffectiveCurseforgeAPIKey`；**不**在 Load/Save 时改写配置字段 |
| `core/appcore` | 所有出站与 settings 视图改用 effective key |
| 根 `build.sh` | 注入 ldflags；可含 fallback 字面量 key |
| app `main` / `version.go` | 不新增 key 变量（与 version 分离，避免 shell 多包混乱） |
| CI / gitignore | 本任务不改 |

## Contracts

### configs

```go
// DefaultCurseforgeAPIKey is empty in source; overwritten by:
// go build -ldflags "-X github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey=..."
var DefaultCurseforgeAPIKey string

func EffectiveCurseforgeAPIKey(configured string) string {
    if k := strings.TrimSpace(configured); k != "" {
        return k
    }
    return strings.TrimSpace(DefaultCurseforgeAPIKey)
}
```

### appcore 调用点

| 位置 | 变更 |
|------|------|
| `configureProviderClients` | `curseForgeAPIKey := configs.EffectiveCurseforgeAPIKey(s.Config().Keys.CurseforgeApiKey)` |
| `QueueModDownload` / `InstallModAndWait` / `InstallOptionalDependencies` | 传 effective key |
| `GetSettings` mask / has | 基于 effective；**不**修改 `s.config.Keys` |
| `SaveApiKeys` | 逻辑不变：写用户字段 + Save；清除后配置为空；随后 `configureProviderClients` 自动用默认 |

### build.sh

- Env：`DEFAULT_CF_API_KEY`（可选覆盖）、`APP_VERSION`（可选）
- 脚本内可设单引号 fallback 默认 key（用户指定可写在脚本中）
- ldflags 两段：`main.appVersion` + `.../configs.DefaultCurseforgeAPIKey`
- 透传 `"$@"` 给 `wails build`
- 不 echo key

示例形状：

```bash
#!/usr/bin/env bash
set -euo pipefail
if [[ -z "${DEFAULT_CF_API_KEY:-}" ]]; then
  DEFAULT_CF_API_KEY='$2a$10$...'  # single-quoted literal; user-supplied
fi
APP_VERSION="${APP_VERSION:-}"
LDFLAGS=(-X "main.appVersion=${APP_VERSION}" -X "github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey=${DEFAULT_CF_API_KEY}")
# join for wails -ldflags
wails build -ldflags "${ldflags_str}" "$@"
```

注意：key 含 `$`，赋值必须用单引号；传给 `-X` 时用双引号包裹整个 `-X path=value`，并避免未引用展开。

## Compatibility

- 未注入默认的 dev 构建：行为与今天一致。
- 已配置用户 key：不受影响。
- MCIM + 空用户 key：仍可构建 CF mirror client（现有 `curseForgeAPIKey != "" || useMCIM`，effective 可能非空，行为更宽松但正确）。
- 契约文档 `provider-api-source.md` 语义从「配置 key 空」改为「**effective** key 空」禁用官方 CF——**本任务实现代码**；spec 文件更新可在 finish 阶段 `trellis-update-spec`，不阻塞实现。

## Trade-offs

| 方案 | 选择 | 原因 |
|------|------|------|
| 变量放 main vs configs | **configs** | downloader/appcore 直接读，无需 Options 透传 |
| 启动时把默认写入 config 字段 | **否** | 避免 Save 把默认持久化进 TOML；清除语义清晰 |
| UI 显示 effective | **是** | 用户知悉 CF 已可用 |
| 默认写在 build.sh | **是（用户要求）** | 简单；用户自理 gitignore/CI |

## Rollback

- 删除 `build.sh`、还原 `configs` / `appcore` 改动即可；无数据迁移。
- 已分发二进制仍含注入 key，无法从代码侧撤回（预期）。

## Risks

- Shell 特殊字符导致注入截断 → 单引号赋值 + 测试 ldflags 构建。
- 漏改某一处仍读裸 `Keys.CurseforgeApiKey` 作出站 → 实现时全局 grep 替换出站路径。
