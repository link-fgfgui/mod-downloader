# 收藏夹跨版本复制修复设计

## Problem Boundary

现有实现把目标作用域记录写回源收藏夹，而收藏夹和收藏项都按作用域过滤，造成数据库写入成功但 UI 永久不可达。修复后的核心不变量是：一个收藏夹拥有一个规范化作用域，其直接收藏项必须属于同一作用域；跨版本操作创建新的目标作用域收藏夹，不改变源收藏夹。

## Public Contract

未发布版本不保留旧“Migration” API。将 Wails、appcore 和前端 store 契约统一重命名为跨版本复制：

```go
type FavoriteCrossVersionCopyRequest struct {
    SourceListID     string `json:"sourceListId"`
    MinecraftVersion string `json:"minecraftVersion"`
    ModLoader        string `json:"modLoader"`
    IgnoreConflicts  bool   `json:"ignoreConflicts,omitempty"`
}

type FavoriteCrossVersionCopyPreview struct {
    SourceListID     string                              `json:"sourceListId"`
    TargetListName   string                              `json:"targetListName"`
    MinecraftVersion string                              `json:"minecraftVersion"`
    ModLoader        string                              `json:"modLoader"`
    NameConflict     bool                                `json:"nameConflict"`
    Matched          []FavoriteCrossVersionCopyMatch     `json:"matched"`
    Conflicts        []FavoriteCrossVersionCopyConflict  `json:"conflicts"`
    Errors           []string                            `json:"errors,omitempty"`
}

type FavoriteCrossVersionCopyApplyResult struct {
    Applied   bool                            `json:"applied"`
    TargetList storage.FavoriteList           `json:"targetList"`
    Preview   FavoriteCrossVersionCopyPreview `json:"preview"`
    Result    FavoriteBulkOperationResult     `json:"result"`
}
```

`App` 暴露 `PreviewFavoriteListCrossVersionCopy` 和 `ApplyFavoriteListCrossVersionCopy`，并重新生成 Wails bindings。请求不再接受调用方提供的 `targetListId`，避免再次把源收藏夹误当目标。

## Preview Flow

1. 读取并验证源收藏夹，目标名称固定为源名称。
2. 规范化目标 Minecraft 版本和 Mod Loader，拒绝空值及与源作用域相同的目标。
3. 按与数据库唯一约束相同的规则检查目标作用域重名；重名时设置 `nameConflict` 和明确错误，不进行应用。
4. 读取源收藏夹内容（包括现有引用），按现有 provider 匹配策略解析目标版本。
5. 返回 matched、conflicts、errors 和完整目标身份；预览不写数据库。

应用会重新执行预览，防止远端匹配或目标名称状态在预览后变化。

## Atomic Apply

Provider 查询全部在事务前完成。仅当预览无错误、没有重名、冲突策略允许且至少有一个 matched 项时，调用 storage 的原子写接口：

1. 开启 SQLite 事务。
2. 再次检查目标作用域重名。
3. 创建与源同名、绑定目标作用域的新收藏夹。
4. 将所有 matched 收藏项写入新收藏夹。
5. 任一写入失败则 rollback；全部成功才 commit。

新目标收藏夹不存在既有记录，因此成功结果只计 `added`；冲突计入 `skipped`。全冲突或零 matched 不创建空收藏夹，也不返回成功。

## Schema V3

`core/storage/userdb.go` 新增 schema v3。检测不到 v3 标记时，在事务中删除并重建 `favorite_list_refs`、`favorite_mods`、`favorite_lists`、`favorite_groups`，明确弃用所有旧收藏夹数据；`pinned_mods`、`usage_stats` 等表不受影响。

v3 加入以下约束：

- 同一规范化 `(name, minecraft_version, mod_loader)` 只能存在一个收藏夹，为预览后的竞态提供数据库最终保护。
- 收藏项写入时，其 `minecraft_version` 和 `mod_loader` 必须与父收藏夹一致，防止再次产生不可达记录。
- 作用域字段继续使用 trim 后的 Minecraft 版本、lowercase Mod Loader 和空字符串而非 NULL。

普通收藏夹创建与重命名也必须正确处理同作用域重名，不得因唯一约束向前端制造未处理异常。

## Frontend Behavior

- 菜单、按钮、弹窗、预览、成功和失败文案统一使用“跨版本复制”/“Copy Across Versions”。
- 弹窗不提供目标收藏夹选择；展示将自动创建的目标名称和目标作用域。
- 目标字段变化后清空旧预览。预览返回 `nameConflict` 或 errors 时展示原因并禁用执行。
- 应用期间提供 loading/防重复提交状态。
- 成功后关闭操作弹窗并显示包含目标收藏夹名称及 added/skipped 的通知；不改变 `minecraftStore` 全局选择。
- 关闭动画期间保留稳定弹窗内容，并在 `after-leave` 后清理本次状态。

## Compatibility And Rollback

- 应用未发布，因此删除旧 Migration API 和弃用旧收藏夹数据可接受，不提供兼容别名。
- schema v3 是破坏性升级，回滚到旧二进制不保证识别新约束；源代码回滚时需同时删除本地开发 SQLite 文件。
- 平台元数据缓存不变，不提升 gob cache version。

## Test Strategy

- Storage：v2 到 v3 清空收藏夹相关数据但保留 pins/stats；同作用域重名；作用域不一致写入；原子创建写入及 rollback。
- Appcore：全匹配、部分冲突、全冲突、目标重名、预览后竞态、源保持不变、引用内容、目标列表作用域与结果。
- Frontend：仓库没有测试 runner，不为本任务引入新框架；使用 lint、Vue/TypeScript 类型检查、生产构建和 store/dialog 完整状态流审查验证绑定与模板。
- App adapter：生成 bindings 后运行 Go test/build 和前端 build。
