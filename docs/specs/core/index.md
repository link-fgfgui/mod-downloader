# Core 规范索引

本目录登记 GUI 与 TUI 共享的 `core/` 领域规范。凡是换掉 React/tview 后仍然成立的规则，
必须只有一份 Go 实现并住在这里；前端专属的展示与交互契约见 `docs/specs/frontend/`。

| 主题 | 文件 | 摘要 |
| --- | --- | --- |
| 全局单实例 | [`single-instance.md`](./single-instance.md) | 启动时绑定 `127.0.0.1:18801`，端口即进程锁，绑定失败必须在建窗前中止 |
| 双前端共享决策投影 | [`ui-decision-projection.md`](./ui-decision-projection.md) | `LocalModGroup` 决策字段、依赖森林、搜索/版本比较、disabled 保留与批量操作唯一入口 |
| 配置原子写入 | [`config-atomic-write.md`](./config-atomic-write.md) | 临时文件 + fsync + rename；保存前清扫强杀残留的 `.*.tmp`（依赖单实例前提） |
