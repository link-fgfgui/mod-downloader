# 技术设计

## 边界

文件系统监听使用 Go `fsnotify`，监听和增量索引属于 `core/appcore`/`core/minecraft`；Wails `app.go` 继续只负责事件适配；前端 Store 订阅“所选版本变化”事件并应用快照。监听器由 Service 持有，服务启动/关闭和选中实例变化负责生命周期。

## 数据流

1. `SelectVersion` 或 `RefreshSelectedVersionMods` 确定实例目录后，Service 同步 watcher 目标。
2. watcher 将事件按绝对路径去重并 debounce。
3. 增量协调器读取事件后的文件状态：
   - 有效 `.jar`：仅当 size/mtime 或内容指纹变化时扫描该文件，并替换该路径的本地索引记录。
   - `.jar.disabled` 与重命名：更新启用状态/路径，不解析无效扩展名。
   - 删除：移除该路径的所有索引记录。
4. 协调器构造当前选中版本快照，应用已有在线元数据 enrichment，并发出 `selected-version-changed`。
5. 无法可靠判断事件（目录级事件、watcher error、批量变化、跨目录 rename）时调用现有全量刷新作为兜底。

## 一致性与性能

- 全局本地索引按实例和路径提供删除/替换能力；增量扫描不清空其他路径。
- 对同一文件的 burst 事件进行 debounce，避免写入过程中解析半成品。
- 启用/禁用后直接把 rename 前后的路径映射到增量处理，避免重新计算未变化 JAR 的 hash。
- 手动刷新仍执行全量扫描并重建索引。

## 兼容与回滚

若平台 watcher 不可用或初始化失败，功能降级为现有手动/操作后刷新流程；保留全量刷新入口作为回滚路径。新增事件不改变现有事件名称和前端数据结构。
