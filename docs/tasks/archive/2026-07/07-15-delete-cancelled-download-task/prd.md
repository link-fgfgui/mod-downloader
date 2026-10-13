# 下载队列支持删除已取消任务

## Goal

允许用户在下载队列中手动移除不再需要的已取消任务，避免只能重试、无法清理单项取消记录。

## Background

- 下载队列将已取消任务保存在 core 的 `retryable` 历史中，并以 `status == "canceled"`、`retryable == true` 展示。
- 当前队列项的重试按钮仅处理左键重试；后端仅提供 `CancelDownload` 和 `RetryDownload`，没有删除单项历史的接口。
- `retryable` 同时保存失败任务，因此删除能力必须校验状态，不能因为任务可重试就允许删除失败记录。

## Requirements

- R1. 仅对状态为 `canceled` 的下载队列项，在重试按钮上提供右键删除操作。
- R2. 左键重试行为保持不变；右键删除不得触发重试或打开浏览器原生上下文菜单。
- R3. 后端按任务 ID 删除且仅删除 `retryable` 历史中状态为 `canceled` 的匹配项；空 ID、不存在的 ID、失败任务、运行中任务和待处理任务均返回失败且不得改变队列。
- R4. 删除成功后立即刷新/广播下载队列状态，使该任务从界面消失，并保持队列汇总、可见性和现有离场快照行为一致。
- R5. 中英文界面为该操作提供可访问名称/提示文本；本任务不删除已下载文件或临时文件。
- R6. 右键删除立即执行，不显示确认对话框。

## Acceptance Criteria

- [x] AC1. 已取消任务的重试按钮左键仍重新入队，右键则移除该任务且不会重新入队。
- [x] AC2. 删除成功后，该 ID 不再出现在 `GetDownloadQueueState()`，前端同步消失；其他队列项保持不变。
- [x] AC3. 后端拒绝删除失败、运行中、待处理、不存在或空 ID，且这些场景无队列状态变更。
- [x] AC4. 失败任务的重试按钮右键不执行删除；浏览器上下文菜单行为仅在可删除的已取消项上被拦截。
- [x] AC5. core 相关单元测试、app 构建/测试、前端 lint 和生产构建通过，Wails 绑定与新增公开方法保持一致。

## Verification Note

- `core/downloader` 与 `core/appcore` 测试通过，core 全量 build/vet 通过。
- core 全量测试仍有与本任务无关的既有失败：`configs/TestNetworkConfigClampsOutOfRangeValues/above_maximum_clamps` 期望并发上限 256，当前实现返回 100；本任务未改动 `core/configs`。

## Out Of Scope

- 批量清空已取消或失败任务。
- 删除失败任务或活动任务。
- 删除磁盘上的目标文件、部分下载文件或缓存。
- 新增独立按钮、下拉菜单或确认对话框。
