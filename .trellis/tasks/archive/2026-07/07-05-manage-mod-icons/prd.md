# 管理页面模组显示图标

## Goal

管理页面的模组列表当前使用静态占位图标（`mdi-package-variant`）。改为尽可能显示实际的模组图标（iconUrl），通过 SHA1 反查在线平台元数据获取。

## Requirements

1. **本地缓存优先**：扫描模组时，对每个 mod 的 SHA1 调用 `modbridge.PlatformMetadataForSHA1`，从本地数据库缓存反查 `ModProject.IconURL`。
2. **在线 API 兜底**：本地缓存未命中的 SHA1，批量调用 Modrinth SDK 的 `VersionFiles.GetFromHashes`（SHA1 算法），获取 version → project → iconUrl。结果写入数据库缓存供后续复用。
3. **降级处理**：API 也未命中的模组，显示空图标（现有的 `mdi-package-variant` 占位）。
4. **不阻塞加载**：首次扫描时快速返回已有缓存的 icon，API 查询异步/延迟执行后通知前端刷新。或整体在 refresh 流程中同步完成（取决于设计，API 请求通常 <500ms 可接受同步）。
5. **前端条件渲染**：`ModInfo` 增加 `iconUrl` 字段；`Manage.vue` 有 iconUrl 时用 `<v-img>`，否则用现有 `<v-icon>` 占位。

## Constraints

- 复用已有代码，不引入新的 HTTP 客户端或第三方库。
- Modrinth SDK `GetFromHashes` 已存在于 `go-modrinth` v0.6.0。
- CurseForge 使用 Murmur2 fingerprint（非 SHA1），本期不走 CF 反查路径，仅在本地缓存命中时生效。
- 批量查询一次最多处理实例中所有 mod 的 SHA1（通常 <200 个），单次 POST 请求。

## Acceptance Criteria

- [ ] 管理页面中，本地缓存有对应平台数据的模组显示实际图标
- [ ] 本地缓存未命中时，通过 Modrinth API 批量查询并显示图标
- [ ] API 也未命中的模组正常显示占位图标，不报错
- [ ] 获取到的平台元数据写入数据库缓存，下次刷新无需重复请求
- [ ] 前端模组列表渲染 iconUrl 时使用 `<v-img>`，无 iconUrl 时回退 `<v-icon>`
