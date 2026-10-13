# 收藏夹跨版本复制修复实施计划

1. 在 `core/storage/userdb.go` 实现 schema v3 事务升级，重建收藏夹相关表并加入名称/作用域约束；更新 schema 升级测试。
2. 在 storage 层增加目标作用域重名查询和“创建收藏夹并批量写入收藏项”的原子接口，复用规范化和行扫描逻辑，覆盖成功、重名与 rollback 测试。
3. 将 appcore Migration 请求、预览、匹配、冲突和应用结果重命名为 CrossVersionCopy，移除 `targetListId` 输入，加入源列表验证、同作用域拒绝、目标名称冲突和原子 apply。
4. 重写迁移服务测试，验证全匹配、冲突策略、全冲突不建空列表、重名、预览后竞态、引用内容、源不变和目标作用域可达。
5. 更新 `app.go` Wails 方法名称并重新生成 `frontend/wailsjs` bindings，确认不存在旧 Migration 公共方法或类型。
6. 更新 `frontend/src/stores/favorites.ts` 的类型和 actions：传递新请求、刷新列表、保留全局作用域不变，并返回目标列表/计数。
7. 更新 `Favorites.vue` 对话框状态和生命周期：显示自动目标信息、重名/错误、loading，严格控制执行按钮；成功后关闭并提示，不切换 Minecraft store。
8. 更新中英文 i18n 文案，搜索并清除该功能所有用户可见的“迁移/Migration”命名。
9. 运行格式化与聚焦测试：
   - `gofmt` 处理变更的 Go 文件。
   - `cd core && go test ./storage ./appcore`。
10. 运行完整质量门：
   - `cd core && go test ./... && go build ./... && go vet ./...`。
   - `cd frontend && npm run lint && npm run build`。
   - 仓库根目录运行 `go test ./... && go build ./... && go vet ./...`。
11. 检查 `git diff`、Wails 生成文件、schema 破坏性升级范围和成功后不切换全局状态的完整数据流；发现契约漂移则回到对应步骤修正。

## Risk And Rollback Points

- schema v3 会删除所有本地收藏夹数据；只允许删除四张收藏夹相关表，必须用测试证明 pins/stats 保留。
- 唯一约束会影响普通创建/重命名路径；这些路径必须同步处理重名结果。
- Provider 解析不能放在长事务中；事务仅包围最终重名检查、目标创建和批量写入。
- Wails 方法和类型被重命名后必须生成 bindings，禁止手工只改一侧。
- 若原子 storage 接口失败，目标收藏夹和收藏项都必须不可见；不得用应用层补偿删除模拟事务。
