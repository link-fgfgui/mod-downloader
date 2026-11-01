# App 绑定面

## 结论

`App` 只暴露前端实际消费的业务方法。`app_contract_test.go` 锁定当前 **67** 个业务方法；
新增/删除方法必须同步更新契约测试，并重新生成 `frontend/bindings/`。

## 已删除的死绑定

以下方法前端 0 调用，TUI 直接调用 `appcore.Service`，因此从 `App` 移除：

- `GetAppVersion`（版本由窗口/配置读取，前端不消费）
- `GetPreferences`（前端使用 `GetSettings`）
- `ListFavoriteMods` / `ListFavoriteContents`（前端使用 `ListFavoriteContentsForScope`）
- `RemoveFavoriteListReference` / `ListFavoriteListRefs`（前端未暴露该交互）

## 约束

**Don't**

- 不要把 core 已有、但前端暂未使用的服务方法顺手加到 `App`：绑定面每增加一个方法，
  都会扩大契约测试与生成代码的维护面。
- 不要手写平行 TypeScript interface；只消费 `frontend/bindings/` 生成类型。
- 删除 `App` 方法后必须运行 `wails3 generate bindings -clean=true -ts`，并更新
  `app_contract_test.go` 的方法数与清单。
