# 管理页面自动刷新 + 通用列表组件抽象

## Goal

将管理页面的 v-table 替换为与下载页面同款的通用虚拟滚动列表组件，并在页面打开时自动刷新模组列表。

## Requirements

### 1. 管理页面自动刷新
- 页面激活时（`onActivated`）自动调用 `refreshSelectedMods()` 刷新模组列表
- 保留手动刷新按钮

### 2. 通用列表组件抽象
- 从 `SearchResultList.vue` 提取通用虚拟滚动列表组件（`VirtualList.vue`）
- 通用能力：虚拟滚动、多选（Ctrl/Shift/Ctrl+A）、选中计数、浮动操作栏（通过 slot 自定义按钮）、滚动方向动画、入场动画
- 业务特定内容通过 slot 注入：每项的渲染内容（`#item`）、操作栏按钮（`#actions`）
- `SearchResultList.vue` 改为使用 `VirtualList.vue` + 下载特定 slot 内容
- `Manage.vue` 使用 `VirtualList.vue` + 管理特定 slot 展示模组信息

### 3. 管理页面列表显示
- 每项显示：模组名称/ID、版本、文件名、启用/禁用状态 chip
- 不需要 load-more / 无限滚动（管理页面数据一次性加载）

## Acceptance Criteria

- [ ] 打开管理页面时自动刷新模组列表
- [ ] 管理页面使用与下载页面相同的通用列表组件渲染
- [ ] `SearchResultList.vue` 重构为基于通用组件，所有功能不变
- [ ] 通用组件支持通过 slot 自定义每项内容和操作栏按钮
- [ ] 无视觉回归（动画、选中高亮、浮动操作栏均正常）
