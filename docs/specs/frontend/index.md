# Frontend 规范索引

本目录登记 GUI（React）专属的展示、交互与动效契约。凡是换 UI 框架后仍然成立的业务规则，
应放在 `docs/specs/core/` 并落在 `core/`，不要写在这里。

| 主题 | 文件 | 摘要 |
| --- | --- | --- |
| Radix 弹窗退场动画 | [`radix-dialog-exit-animation.md`](./radix-dialog-exit-animation.md) | 退出动画必须保持最后一帧，避免关闭末尾闪现完整弹窗 |
| Manage 视图偏好 | [`manage-view-mode-preference.md`](./manage-view-mode-preference.md) | List/Tree 是 GUI 本地偏好，使用 localStorage 跨路由与刷新保留 |
