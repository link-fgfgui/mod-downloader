# 技术设计

## Boundaries

- `core/downloader`：新增按 ID 移除已取消重试历史的并发安全操作，并发出队列更新事件。
- `core/appcore` 与 `app.go`：逐层转发该操作，保持 Wails runtime 只存在于 app 适配层。
- `frontend/wailsjs`：由 Wails 生成新增公开方法绑定。
- `frontend/src/stores/downloadQueue.ts`：封装删除调用，成功后刷新状态。
- `frontend/src/App.vue`：仅在已取消项的重试按钮上处理 `contextmenu`，保留左键重试。

## Contract And Data Flow

1. 用户在 `status == "canceled"` 的重试按钮上触发右键操作；删除立即执行，不打开确认对话框。
2. Vue 阻止该次上下文菜单和事件冒泡，调用 store 删除动作。
3. store 调用 Wails `RemoveCanceledDownload(id)`（最终命名在实现中与现有动词风格保持一致）。
4. appcore 转发至 downloader；downloader 持锁查找 `retryable`，仅当状态严格为 `canceled` 时移除。
5. 成功时发出 `download-queue-updated` 并返回 `true`；前端随后刷新，确保调用方即使未收到事件也获得最新状态。

## Compatibility And Safety

- 不修改现有请求/响应结构，仅新增公开方法。
- 不复用 `RetryDownload` 的移除逻辑，因为重试会重建 ID 并入队，删除必须是独立语义。
- 状态校验同时放在 UI 可见性和后端边界；后端是最终授权点。
- 删除仅影响内存中的重试历史，不触碰文件系统。

## Rollback

- 该改动无持久化迁移。回滚新增方法、绑定、store 动作和事件处理即可恢复原行为。
