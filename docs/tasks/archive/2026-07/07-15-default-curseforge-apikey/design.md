# Design: 编译期默认 CurseForge API Key

## Architecture

```
.github/workflows/build.yml / wails -ldflags
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
| GitHub Actions workflow | 从 `secrets.DEFAULT_CF_API_KEY` 取值并注入 ldflags |
| app `main` / `version.go` | 不新增 key 变量（与 version 分离，避免 shell 多包混乱） |
| Git / workflow | 只跟踪 secret 引用，不保存真实 key |

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

### GitHub Actions build

- Secret：`secrets.DEFAULT_CF_API_KEY`
- `APP_VERSION`：tag 构建使用 tag 名，其它构建使用短 commit SHA
- ldflags 两段：`main.appVersion` + `.../configs.DefaultCurseforgeAPIKey`
- workflow 不硬编码或输出真实 key

等价的本地 smoke 命令：

```bash
export APP_VERSION=dev
export DEFAULT_CF_API_KEY='...'
wails build -ldflags "-X main.appVersion=${APP_VERSION} -X github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey=${DEFAULT_CF_API_KEY}"
```

注意：本地 key 含 `$` 时赋值必须用单引号；CI 值必须通过 Secret
注入并在 linker 参数中正确引用。

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
| 默认 key 存放 | **GitHub Actions Secret** | workflow 可构建发布包且仓库不保存真实值 |

## Rollback

- 移除 workflow 的默认 key ldflag、还原 `configs` / `appcore` 改动即可；无数据迁移。
- 已分发二进制仍含注入 key，无法从代码侧撤回（预期）。

## Risks

- Shell 特殊字符导致注入截断 → 单引号赋值 + 测试 ldflags 构建。
- 漏改某一处仍读裸 `Keys.CurseforgeApiKey` 作出站 → 实现时全局 grep 替换出站路径。
