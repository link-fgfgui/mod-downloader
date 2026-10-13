# 实施计划

- [x] 将 downloader 删除方法泛化为 failed/canceled retryable 与 pending，并重写测试覆盖成功、状态拒绝、事件和 pending 不留历史。
- [x] 更新 appcore、App 公开方法并重新生成 Wails 绑定。
- [x] 泛化 store 删除动作和文案，分别接入重试按钮与取消按钮的状态受限右键处理。
- [x] 更新 `ARCHITECTURE.md` 和下载队列操作 code-spec 的签名、契约、矩阵、案例与测试要求。
- [x] 运行 gofmt、core 相关测试/build/vet、app build/test、前端 lint/build 和 diff 检查。

## Risks And Rollback Points

- pending 与 worker 提升存在竞态，后端锁内状态是最终判定，不能只信 UI 快照。
- 不得复用 `CancelDownload` 删除 pending，否则会创建 canceled retryable 历史。
- 公共 Wails 方法重命名后必须由生成器同步绑定，不能只改前端导入。
