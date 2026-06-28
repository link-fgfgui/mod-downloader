# 实现计划

## 步骤

### 1. 创建 `VirtualList.vue` 通用组件
- [ ] 新建 `frontend/src/components/VirtualList.vue`
- [ ] 从 `SearchResultList.vue` 提取：虚拟滚动、多选逻辑、IntersectionObserver、滚动方向检测、浮动操作栏、入场动画
- [ ] 定义 props: items, itemHeight, itemKey, hasMore, loadingMore, selectable, noMoreText
- [ ] 定义 slots: #item, #actions, #footer
- [ ] 定义 emits: load-more, item-click
- [ ] 迁移相关通用样式

### 2. 重构 `SearchResultList.vue`
- [ ] 改为使用 `VirtualList.vue` 作为内部实现
- [ ] 通过 `#item` slot 渲染下载特定内容（图标、标题、平台 chip、下载按钮）
- [ ] 通过 `#actions` slot 渲染批量操作按钮
- [ ] 保持所有现有 props/emits 接口不变
- [ ] 保留业务特定样式

### 3. 改造 `Manage.vue`
- [ ] 添加 `onActivated` 调用 `minecraftStore.refreshSelectedMods()`
- [ ] 移除 `v-table`，使用 `VirtualList.vue`
- [ ] 通过 `#item` slot 渲染模组信息（名称/ID、版本、文件名、状态 chip）
- [ ] 调整样式适配新组件

### 4. 验证
- [ ] 下载页面功能不变（搜索、滚动加载、多选、批量操作）
- [ ] 管理页面打开时自动刷新
- [ ] 管理页面列表显示正确
- [ ] 动画和交互正常
