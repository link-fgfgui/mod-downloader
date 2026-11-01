# Manage 视图偏好（List / Tree）

## 结论

Manage 页面右上角的列表 / 依赖树视图切换是 **GUI 展示偏好**，不属于 `core` 业务状态，
也不与 TUI 共享。它使用 `localStorage` 持久化，跨路由卸载与刷新都会保留。

- 存储键：`mod-downloader.manage.view-mode`
- 允许值：`"list"` / `"tree"`
- 默认值：`"list"`
- 实现：`frontend/src/hooks/useManageViewMode.ts`
- 消费：`frontend/src/pages/Manage.tsx`

## 约束

**Don't**

- 不要把它放进 `useSettingsStore` 的后端设置字段或 `core/structs`：它不写回 core，
  也不要求 GUI/TUI 行为一致。
- 不要只保留在 `ManagePage` 的 `useState` 中：路由切换会卸载页面并重置。
- 不要在读取到未知值时抛错或阻塞页面；未知值必须回退到 `list`。

**Instead**

- 初始化时读取 `localStorage`，写入失败（隐私模式、存储被禁用）时保留内存状态即可。
- 修改视图后同步写入存储，保证立即切换路由再回来仍然生效。
- 单测覆盖默认值、跨挂载保留、非法值回退：
  `frontend/src/hooks/useManageViewMode.test.ts`。
