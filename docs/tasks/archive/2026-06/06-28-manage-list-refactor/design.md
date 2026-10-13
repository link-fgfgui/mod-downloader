# 技术设计：通用列表组件抽象

## 组件拆分

### `VirtualList.vue`（新建）

从 `SearchResultList.vue` 提取的通用虚拟滚动列表组件。

**Props:**
- `items: Array` — 列表数据
- `itemHeight: number` — 每项高度（默认 88）
- `itemKey: (item, index) => string` — key 生成函数
- `hasMore: boolean` — 是否有更多数据（控制 load-more 哨兵）
- `loadingMore: boolean` — 正在加载更多
- `selectable: boolean` — 是否启用多选（默认 true）
- `noMoreText: string` — 无更多数据时的提示文本

**Slots:**
- `#item="{ item, index, selected }"` — 每项内容渲染
- `#actions="{ selectedItems, selectedIndices, clearSelection }"` — 浮动操作栏自定义按钮
- `#footer` — 列表底部自定义内容（可选，覆盖默认 load-more）

**Emits:**
- `load-more` — 触发加载更多
- `item-click(index, event)` — 项被点击（选择逻辑内部处理，同时暴露事件）

**内部管理:**
- 多选状态（Ctrl/Shift/Ctrl+A/Escape）
- IntersectionObserver 加载更多
- 滚动方向检测 + CSS 变量
- 入场动画 delay 计算

### `SearchResultList.vue`（重构）

改为包裹 `VirtualList.vue`，仅负责：
- 传入 `#item` slot：图标、标题、描述、平台 chip、下载按钮
- 传入 `#actions` slot：批量下载、批量 unpin、复制名称、取消选中按钮
- 传入 download-specific props（states, downloadingKeys）
- 将 item-click 转发为 install emit（如果需要）

### `Manage.vue`（改造）

- 移除 `v-table`，使用 `VirtualList.vue`
- `#item` slot：模组名/ID、版本、文件名、启用/禁用 chip
- 不传 `hasMore`（一次性数据，默认 false）
- 添加 `onActivated` 钩子调用 `refreshSelectedMods()`

## 数据流

```
VirtualList (通用)
  ├── 内部: 选择状态、滚动、Observer
  ├── slot #item → 业务渲染
  └── slot #actions → 业务操作栏

SearchResultList (下载页)
  └── VirtualList
        ├── #item: 图标 + 标题 + 平台chip + 下载按钮
        └── #actions: 批量下载/unpin/复制/取消

Manage.vue (管理页)
  └── VirtualList
        └── #item: 模组名 + 版本 + 文件名 + 状态chip
```

## 样式处理

- 通用样式（wrapper、scroll、浮动操作栏、选中高亮、动画）留在 `VirtualList.vue`
- 业务样式（下载按钮 transition、icon-fade）留在 `SearchResultList.vue`
- 管理页面特定样式留在 `Manage.vue`
