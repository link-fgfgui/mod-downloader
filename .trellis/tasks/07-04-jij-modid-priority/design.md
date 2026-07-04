# Design: 区分 mods.toml 声明 modid 与 jij mod 的优先级

## Problem Summary

当前 `ParseModZipReader` 把顶层 `mods.toml` 声明的 modID 与 nested jar/JIJ 递归解析出的 modID 混合在同一个 `[]structs.ModInfo` 列表中，调用方无法区分两者语义。这导致：

1. 安装状态判断：仅命中 JIJ 子模组 modID 就把宿主包标为 conflict/installed
2. 替换归档：`FilterFullyCoveredPaths` 的覆盖集用了全部 modID（包含 JIJ），判断结果不准确
3. 版本 modID 缓存：持久化全部 modID，查询时同等对待强/弱引用

## Design Decisions

### 1. 携带层级信息的解析结果结构

在 `structs/minecraft/modinfo.go` 中为 `ModInfo` 增加一个 `IsJij bool` 字段：

```go
type ModInfo struct {
    // ... existing fields ...
    IsJij bool `json:"isJij,omitempty"` // true = nested jar / JIJ; false = top-level declaration
}
```

**替代方案分析**
- 方案 A：返回两个切片 `(primary []ModInfo, jij []ModInfo)` — 破坏现有所有调用方接口，改动范围大。
- 方案 B（选中）：在现有 `ModInfo` 中加 `IsJij bool` — 接口签名不变，调用方按需过滤，改动集中。
- 方案 C：返回新结构体 `ParsedJar { Primary, Jij []ModInfo }` — 比方案 B 更清晰但同样破坏接口；相比改动量不值得，且调用方需要组合两个切片场景（如 UI 展示）同样要处理。

选择方案 B：只加一个字段，调用方接口零破坏，过滤逻辑在调用侧用 helper 函数集中处理。

### 2. 解析层标记来源

`parseZipReader` 中，顶层 `depth == 0` 调用 `parseDeclaredMetadata` 的 mods → `IsJij = false`；
`parseNestedJar` 返回的 mods → 递归调用前设 `IsJij = true`（或在递归返回后标记）。

具体：`parseNestedJar` 调用 `ctx.parseZipReader(zr, ...)` 返回后，统一将所有结果的 `IsJij` 置为 `true`（无论递归深度，嵌套 jar 内部的顶层声明对宿主来说仍是弱引用）。

### 3. 强引用 modID 提取 helper

在 `minecraft/modparser.go` 中新增：

```go
// PrimaryModIDs 返回 mods 中强引用（顶层 mods.toml 声明）的 modID 集合。
func PrimaryModIDs(mods []structs.ModInfo) []string
```

调用方：
- `downloader/download.go`：`ParseModJarWithSHA1` 返回后，用 `PrimaryModIDs` 替换全量 modID，用于：
  - `PersistVersionModIDs`（版本缓存只存强引用 modID）
  - `LocalModPathsForModIDs`（冲突检测只用强引用）
  - `FilterFullyCoveredPaths`（替换归档只用强引用）
- `modbridge/modbridge.go` 中 `VersionModIDs` 走远端 JAR 解析路径同样需过滤

### 4. `FilterFullyCoveredPaths` 覆盖集

`FilterFullyCoveredPaths(newModIDs, existing)` 已经是按 `LocalModIDsBySHA1(existing.FileSHA1)` 查询本地 modID 集合，判断是否完全覆盖。  
**本次变更不改 `FilterFullyCoveredPaths` 内部逻辑**——它比较的是两套"本地全量 modID"（存在 `LocalModFile` 里的），这批数据的正确性取决于写入时是否只存强引用。  
通过在 `UpsertLocalMod` 调用前过滤（只写强引用 ModInfo），`LocalModIDsBySHA1` 自然返回强引用集合。

### 5. 本地索引写入（`global/localmods.go`）

当前 `UpsertLocalMod(info structs.ModInfo, ...)` 每条 `ModInfo` 单独写入，调用方负责过滤。  
**不改 `UpsertLocalMod` 本身** — 调用方（local mod scanner）改为只传 `IsJij == false` 的 ModInfo。

扫描入口位于哪里？需确认（见 implement.md 研究步骤）。

### 6. `LocalModPathsInInstanceByModID` 与全局索引

本地索引写入只含强引用 ModInfo 后，`LocalModPathsInInstanceByModID` 的查询结果自然只命中强引用 modID，无需改函数本身。

### 7. 版本 modID 持久化（DB 层）

`SetVersionModIDs` / `GetVersionModIDs` 存储的是解析出的 modID 列表，只要写入时传强引用 modID，读取侧无需修改。

### 8. 管理页 / 前端

PRD 已确认不改展示逻辑，跳过。

## Data Flow After Change

```
ParseModZipReader(r, name, loader)
  └─ parseDeclaredMetadata → mods with IsJij=false
  └─ parseNestedJar → all returned mods marked IsJij=true
  └─ uniqueModsByID(all mods)   ← 去重，保留 IsJij 字段

PrimaryModIDs(mods) → filter IsJij==false → strongIDs []string

# downloader/download.go 安装流程
modIDs = PrimaryModIDs(ParseModJarWithSHA1(...))
PersistVersionModIDs(version.ID, modIDs)   ← 只写强引用
existing = LocalModPathsForModIDs(modIDs)  ← 只查强引用
archiveSupersededModJars(FilterFullyCoveredPaths(modIDs, existing))

# 本地扫描
for _, info := range mods {
    if !info.IsJij {
        UpsertLocalMod(info, ...)
    }
}
```

## Compatibility & Rollout

- `ModInfo.IsJij` 加了 `omitempty`，序列化时 `false` 值不出现，前端 / 外部接口不受影响。
- 不改 DB schema；`SetVersionModIDs` 存 `[]string`，内容变"只含强引用"。已缓存的旧数据（可能含 JIJ modID）在下次安装/刷新时按版本 ID 被覆盖，不需要 migration。
- 现有 Fabric 路径：Fabric `fabric.mod.json` 中的 `jars` 字段解析出的嵌套 jar 同样通过 `parseNestedJar` 走，会被标 `IsJij=true`，行为一致。

## Files Changed

| 文件 | 变更内容 |
|------|----------|
| `structs/minecraft/modinfo.go` | 加 `IsJij bool` 字段 |
| `minecraft/modparser.go` | `parseNestedJar` 返回结果标 `IsJij=true`；新增 `PrimaryModIDs` helper |
| `downloader/download.go` | 使用 `PrimaryModIDs` 过滤，影响持久化、冲突检测、归档 |
| `modbridge/modbridge.go` | `VersionModIDs` 远端解析路径使用 `PrimaryModIDs` |
| 本地扫描入口（待确认） | 写入 `UpsertLocalMod` 前过滤 `IsJij` |
| `minecraft/modparser_test.go` | 新增测试断言强/弱引用分类 |
