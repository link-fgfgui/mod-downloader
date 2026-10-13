# 下载队列支持删除失败和等待中任务

## Goal

扩展下载队列单项删除能力，使用户也能立即清理失败和等待中的任务，而不必先重试或先取消再删除。

## Background

- 已取消项当前可通过右键重试按钮调用 `RemoveCanceledDownload` 删除。
- 失败项与已取消项同处 `retryable` 历史并显示重试按钮，但当前后端状态校验拒绝删除失败项。
- 等待中项位于 `pending` 队列并仅显示取消按钮；现有 `CancelDownload` 会把它转存为 canceled 历史，因此不能用于“直接删除”。

## Requirements

- R1. 已取消和失败任务均可从重试按钮的右键操作立即删除；左键重试保持不变。
- R2. 等待中任务可从取消按钮的右键操作立即删除；左键取消仍生成可重试的 canceled 历史。
- R3. 后端按任务 ID 删除 `retryable` 中状态为 `canceled` 或 `failed` 的项，或删除 `pending` 中的匹配项；删除 pending 不得创建 canceled 历史。
- R4. 空 ID、不存在 ID 和运行中任务均返回失败且不改变队列；若 pending 已并发转为 running，删除也必须失败，不能取消运行任务。
- R5. 删除成功仅移除匹配项并清理对应进度，立即广播/刷新队列状态，不删除磁盘文件或缓存。
- R6. 所有右键删除均立即执行、不显示确认框，并只在允许删除的状态阻止原生上下文菜单。
- R7. 将状态范围已扩大的方法与文案从 canceled 专用命名统一为通用队列项删除语义，更新 Wails 绑定和跨层规范。

## Acceptance Criteria

- [x] AC1. 失败项右键重试按钮后从队列消失且不会重新入队；左键仍可重试。
- [x] AC2. 等待中项右键取消按钮后直接消失且不会产生 canceled 项；左键仍按现有逻辑取消并保留重试记录。
- [x] AC3. 已取消项继续支持右键删除，其他队列项和汇总状态保持正确。
- [x] AC4. 运行中、空 ID 和不存在 ID 不可删除，不发出队列变更事件。
- [x] AC5. 可删除状态成功时恰好发出一次队列状态事件，前端绑定、store、可访问文案和原生菜单拦截与状态一致。
- [x] AC6. core 相关测试、core build/vet、app build/test、前端 lint 和生产构建通过。

## Out Of Scope

- 删除运行中任务或将“删除”隐式转换为“取消”。
- 批量清空队列。
- 删除下载文件、部分文件或缓存。
- 新增独立删除按钮、菜单或确认对话框。

## Verification Note

- `core/downloader` 与 `core/appcore` 测试、core 全量 build/vet、app build/vet/test、前端 lint/build 均通过。
- core 全量测试仍有与本任务无关的既有失败：`configs/TestNetworkConfigClampsOutOfRangeValues/above_maximum_clamps` 期望 256，当前实现返回 100；本任务未改动 `core/configs`。
