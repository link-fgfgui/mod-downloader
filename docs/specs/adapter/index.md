# Adapter 规范索引

本目录登记 Wails 适配层（仓库根 `main.go` / `app.go` / `app_events.go`）的契约。
core 领域规则见 `docs/specs/core/`。

| 主题 | 文件 | 摘要 |
| --- | --- | --- |
| App 绑定面 | [`binding-surface.md`](./binding-surface.md) | 只暴露前端实际消费的方法；`app_contract_test.go` 锁定 68 个业务方法 |
