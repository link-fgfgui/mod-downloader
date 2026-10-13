# 内置默认 CurseForge API Key（GitHub Actions Secret + ldflags）

## Goal

打包后的应用自带可用的默认 CurseForge API Key，用户无需手动配置即可访问官方 CurseForge 源。密钥保存在 GitHub Actions Secret 中，由 release workflow 在构建时通过 ldflags 注入二进制，不写入 Git 跟踪的 Go 源码或 workflow。

## Background

- 运行时配置：`mod-downloader.toml` 的 `[keys] curseforge_api_key`、环境变量 `KEYS_CF_API_KEY`、UI `SaveApiKeys`。
- 当前无编译期内置 key：配置为空且未开 MCIM 时 `SetCurseForgeClient(nil)`（见 `core/appcore/service.go` `configureProviderClients`）。
- 编译期注入先例：`version.go` 的 `var appVersion` + `-ldflags "-X main.appVersion=..."`（见 `.trellis/spec/app/backend/build-version.md`、CI `wails build`）。
- core 模块路径：`github.com/link-fgfgui/mod-downloader-core`（`replace => ./core`）。
- 生产构建入口为 `.github/workflows/build.yml`；真实 key 由
  `secrets.DEFAULT_CF_API_KEY` 提供，仓库只跟踪 secret 引用。
- 分支：`api-key`。

## Decisions

| 决策 | 选择 |
|------|------|
| UI「清除」key | **回退到内置默认**（配置字段清空，effective key = 编译期默认） |
| 默认 key 存放 | GitHub Actions Secret `DEFAULT_CF_API_KEY` |
| CI / `.gitignore` | 更新 CI workflow；不需要跟踪本地 key 文件 |

## Requirements

1. **编译期默认 key 变量**
   - 在 `core/configs` 增加 package-level `var DefaultCurseforgeAPIKey string`（源码默认空）。
   - 通过 `-ldflags -X github.com/link-fgfgui/mod-downloader-core/configs.DefaultCurseforgeAPIKey=<value>` 注入。
   - 提供 `EffectiveCurseforgeAPIKey(configured string) string`：非空配置优先，否则 trim 后的默认值。

2. **运行时使用 effective key**
   - Provider 初始化、下载队列、可选依赖安装等**出站**使用 `EffectiveCurseforgeAPIKey(config.Keys.CurseforgeApiKey)`，不得只读裸配置字段。
   - `GetSettings` 的 `hasCurseforgeKey` / `curseforgeKeyMask` 基于 **effective** key（有内置默认时显示已设置）。
   - 用户保存/清除只改配置中的用户 key；**不得**把默认 key 写回 `mod-downloader.toml`。
   - 清除（空字符串）后：配置为空，effective 回退默认；官方 CF 在默认非空时仍可用。

3. **GitHub Actions release build**
   - 在 `.github/workflows/build.yml` 的 Wails build step 中读取
     `secrets.DEFAULT_CF_API_KEY`。
   - 组装一个 ldflags 字符串，同时注入 `main.appVersion` 和
     `configs.DefaultCurseforgeAPIKey`。
   - workflow 只保存 secret 引用，不打印或硬编码真实 key。
   - secret 未设置时构建仍成功，默认变量保持空。
   - 对传入 linker 的 key 值加引号，避免 `$` 等字符破坏参数解析。

4. **兼容**
   - 用户配置 / `KEYS_CF_API_KEY` / CLI `ConfigOverrides` 非空时始终覆盖默认。
   - MCIM 无 key 仍可用（现有契约保留）。
   - 不修改 provider 搜索/下载业务逻辑本身。

## Acceptance Criteria

- [ ] AC1：源码中 `DefaultCurseforgeAPIKey` 默认为空；仅 ldflags 或测试可赋值。
- [ ] AC2：配置 curseforge key 为空且编译期默认非空时，官方 CurseForge 客户端仍会初始化。
- [ ] AC3：配置 curseforge key 非空时，出站与 UI mask 使用用户 key，不用默认。
- [ ] AC4：`SaveApiKeys` 清除后配置字段为空，且不会把默认 key 写入 TOML；effective 回退默认。
- [ ] AC5：`GetSettings` 在仅有默认 key 时 `hasCurseforgeKey=true` 且 mask 对应默认 key。
- [ ] AC6：GitHub Actions build 使用 `secrets.DEFAULT_CF_API_KEY` 完成带双 ldflags 的构建，且 workflow 和日志不泄露 key。
- [ ] AC7：相关单元测试覆盖 effective 回退、用户优先、清除不写回默认。
- [ ] AC8：Go 源码、workflow 与其它提交内容中无硬编码真实 key。

## Out of Scope

- 新增本地构建 wrapper 或把真实 key 写入仓库文件。
- Modrinth 内置 key。
- 改动 CurseForge/Modrinth API 业务逻辑。
- 强制 `wails dev` 注入默认 key（本地 dev 仍可通过配置文件或 env）。

## Technical Notes

- 参考 `version.go` + `build-version.md` 的 ldflags 模式，但变量放在 **core/configs** 以便下载/provider 统一使用，无需经 app 层透传。
- 现有 provider 契约「空 key 禁用官方 CF」需改为「**effective** 空 key 禁用官方 CF」；MCIM 例外不变。
