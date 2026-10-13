# 实施计划

- [x] 在 `core/downloader/download.go` 实现只删除 canceled retryable 项的操作，并补充成功与拒绝场景测试。
- [x] 在 `core/appcore/service.go` 和 `app.go` 增加薄转发方法。
- [x] 运行 Wails 绑定生成，确认 `frontend/wailsjs/go/main/App.js` 与 `App.d.ts` 更新。
- [x] 在下载队列 store 增加删除动作，在 `App.vue` 将已取消项重试按钮的右键绑定到该动作，并增加中英文文案。
- [x] 运行 `gofmt`、core 相关测试、app build/vet/test、前端 lint 与 build。
- [x] 对照 PRD 检查左键重试不变、非 canceled 状态不可删除、成功后状态同步。

## Risk And Rollback Points

- 风险集中在同一 `retryable` slice 的并发修改和 Wails 生成绑定；所有 slice 操作必须在现有队列锁内完成。
- 若绑定生成失败，不手工猜测类型签名；先修复生成环境或明确记录无法验证项。
- 若前端刷新与事件重复导致可见状态异常，保留现有 store 的“成功后 refresh”模式，与 cancel/retry 行为一致。
