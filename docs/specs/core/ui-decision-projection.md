# 双前端共享决策投影

## 结论

GUI 与 TUI 必须表现一致的本地 mod 规则由 core 计算，适配器只渲染字段：

- `LocalModGroup.RowKey`：物理 jar 的稳定行键（`id|sha1|path|fileName` 中非空部分）。
- `LocalModGroup.CanReplaceVersion`：是否存在在线平台 + 项目 ID。
- `LocalModGroup.OnlineProject`：由本地 mod 的在线元数据构造的 `models.ModProject`。
- `LocalModGroup.Incompatible` / `SearchText` / `Categories` / `Loaders`：原有投影。

适配器不得再自行拼接行键或在线项目字段；需要“是否已选实例”等 UI 状态时，
在 core 字段之上追加判断，而不是重新实现字段规则。

## 依赖森林

`core/modgraph` 是 Manage 依赖树的唯一构建实现：provider 匹配、JIJ 子节点、缺失叶子与
环收缩都在 core。`LocalModView.Graph` 以“扁平节点 + key 引用”返回，避免递归 children
产生循环 JSON；`LocalModView.Dependents` 以 group `RowKey` 索引反向依赖。GUI 只负责
key → 节点水合和拼音搜索增强；TUI 当前仍用 `BuildLocalModGroups` 渲染扁平列表，
需要树视图时直接消费同一投影，不得重新实现图构建。

## 搜索表达式

`core/searchexpr` 是本地 mod 搜索语法的唯一 Go 实现：

```text
expression := branch ( '|' branch )*
branch     := term ( '&' term )*
```

`&` 比 `|` 绑定更紧；空查询与纯操作符查询匹配全部。TUI 的 Manage 过滤直接调用
`searchexpr.Matches`。GUI 仍使用等价的 TypeScript 解析器做逐键过滤（避免每帧 RPC），
两端共享同一组测试用例；修改语法时必须同时更新
`core/searchexpr/expression_test.go` 与 `frontend/src/utils/searchExpression.test.ts`。

## 版本比较

`core/minecraft` 提供 `CompareVersions` / `IsVersionDowngrade` /
`HasMajorVersionChange`。TUI 在替换版本列表中用它们标记“降级或跨大版本”风险；
GUI 的 TypeScript 版本比较必须与 Go 测试保持同一组边界用例。

## 安装时保留 disabled 状态

`core/downloader` 在替换旧 jar 时检查被替换文件是否全部为 `.jar.disabled`，
新文件写入后自动恢复 `.disabled` 后缀。适配器**不得**在安装完成后自行 rename：

- CLI `upgrade` 不再执行 `KeepDisabled` 后处理；
- GUI/TUI 的版本替换自动获得同一行为。

## 批量本地 mod 操作

`ApplyLocalModBatchOperation` 是 enable/disable/delete 的唯一入口。一次性 CLI 命令
通过 `LocalModBatchOperationRequest.TargetDir` 指定显式 mods 目录；GUI/TUI 留空，
使用当前选中实例。CLI 不得再直接 `os.Remove` 或 `os.Rename` 操作 jar。

## 禁止

- 不要在 `app.go`、`tviewui`、`frontend/src` 重新实现上述字段/规则。
- 不要为 CLI 单独保留一套 `.jar.disabled` 后缀判断：使用
  `minecraft.IsDisabledModName` / `DisableModPath` / `EnableModPath`。
