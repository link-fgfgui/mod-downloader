# 技术设计

## Boundaries

- `core/downloader`：将 canceled 专用删除操作泛化为队列项删除，锁内支持 failed/canceled retryable 与 pending，拒绝 current。
- `core/appcore`、`app.go`、Wails 绑定：同步通用方法命名并保持薄转发。
- Pinia store：将 `removeCanceled` 泛化为队列项删除动作。
- `App.vue`：重试按钮允许 failed/canceled 右键删除；取消按钮仅允许 pending 右键删除。
- i18n、架构与 `.trellis/spec`：更新通用删除语义及状态矩阵。

## Data Flow And Concurrency

1. UI 根据当前快照状态决定是否拦截右键并调用删除。
2. downloader 在队列互斥锁内重新验证真实状态，优先查找 retryable 的 failed/canceled 项，再查找 pending。
3. pending 若已被 worker 移至 current，则两个集合均不匹配，返回 `false`；删除操作不得调用其 cancel function。
4. 成功删除后清理进度、解锁、发出一次队列快照事件；store 成功后执行 refresh。

## Compatibility

- 本功能延续刚新增且尚未发布的删除 API，将 `RemoveCanceledDownload` 重命名为通用队列删除方法；不保留误导性的重复入口。
- 不修改队列 payload 或状态枚举，重新生成 Wails JS/TypeScript 绑定。

## Rollback

- 无持久化或文件系统变更。回滚通用方法及 UI 右键处理即可恢复 canceled-only 删除。
