# Implementation Plan: 区分 mods.toml 声明 modid 与 jij mod 的优先级

## Checklist

### Step 1 — `structs/minecraft/modinfo.go`: 加 `IsJij` 字段

- [ ] 在 `ModInfo` struct 末尾加：`IsJij bool \`json:"isJij,omitempty"\``
- 验证：`go build ./...` 无编译错误

---

### Step 2 — `minecraft/modparser.go`: 标记 JIJ mods

- [ ] `parseNestedJar` 函数返回前，对所有返回的 `ModInfo` 设 `IsJij = true`：
  ```go
  result := ctx.parseZipReader(zr, sourceName+" > "+nestedPath, depth)
  for i := range result {
      result[i].IsJij = true
  }
  return result
  ```
- [ ] 新增导出 helper `PrimaryModIDs(mods []structs.ModInfo) []string`：
  ```go
  func PrimaryModIDs(mods []structs.ModInfo) []string {
      out := make([]string, 0, len(mods))
      for _, m := range mods {
          if !m.IsJij {
              if id := strings.TrimSpace(m.ID); id != "" {
                  out = append(out, strings.ToLower(id))
              }
          }
      }
      return deduplicateStringSlice(out)  // 用现有 dedup 工具或 inline
  }
  ```
  > 注：`declaredModID` 已做 TrimSpace + 空值过滤，`PrimaryModIDs` 只需额外检查 `IsJij`
- 验证：`go build ./minecraft/...`

---

### Step 3 — `minecraft/modparser.go` 本地扫描写入：只写强引用

调用位置：`minecraft/modparser.go:311`（`ScanModJarsInDirectory` 或类似函数）

- [ ] 找到循环写入 `UpsertLocalMod` 的位置，在写入前加 guard：
  ```go
  for i := range mods {
      if mods[i].IsJij {
          continue  // 弱引用不写入本地索引
      }
      mods[i].FileName = baseName
      mods[i].Path = relPath
      mods[i].SHA1 = hash
      mods[i].Enabled = enabled
      global.UpsertLocalMod(mods[i], instanceID, minecraftVersion, modLoader)
  }
  ```
- 验证：`go build ./minecraft/...`

---

### Step 4 — `downloader/download.go` `upsertDownloadedMod`：只写强引用

调用位置：`downloader/download.go:632`

- [ ] 同 Step 3，`for i := range mods` 循环内加 `if mods[i].IsJij { continue }`
- 验证：`go build ./downloader/...`

---

### Step 5 — `downloader/download.go` 安装流程：使用 `PrimaryModIDs`

涉及三处 modID 提取 + 使用：

**5a. `downloadModWithLocalParse`（约 line 520）**
- [ ] 替换 `modIDs := make([]string, ...)` + 手动遍历为：`modIDs := minecraft.PrimaryModIDs(mods)`
- [ ] 对应 `PersistVersionModIDs`、`LocalModPathsForModIDs`、`FilterFullyCoveredPaths` 传入 `modIDs` 不变（已是强引用）

**5b. `tryHardlinkInstall`（约 line 301）**
- [ ] 同理替换 `parsedModIDs` 提取为 `parsedModIDs := minecraft.PrimaryModIDs(parsedMods)`
- [ ] 对应 `PersistVersionModIDs`、`LocalModPathsForModIDs`、`FilterFullyCoveredPaths`（`archiveSupersededModJars(parsedExisting)` 改为 `archiveSupersededModJars(modbridge.FilterFullyCoveredPaths(parsedModIDs, parsedExisting))`，验证 5b 是否已经调用 FilterFullyCoveredPaths）

  > 检查：`tryHardlinkInstall` 中 `archiveSupersededModJars(parsedExisting)` 没有经过 FilterFullyCoveredPaths，需要确认并修正（line ~317）

- 验证：`go build ./downloader/...`

---

### Step 6 — `modbridge/modbridge.go` `VersionModIDs`：远端解析使用 PrimaryModIDs

调用位置：`modbridge/modbridge.go:299-312`（远端 JAR parse 后构建 modIDs 的循环）

- [ ] 替换手动遍历为 `modIDs := minecraft.PrimaryModIDs(mods)`
- [ ] 删除原 `modIDs = deduplicateStrings(modIDs)`（`PrimaryModIDs` 内已做 dedup）
- 验证：`go build ./modbridge/...`

---

### Step 7 — 测试：明确断言强/弱引用分类

`minecraft/modparser_test.go`

- [ ] 扩展 `TestParseForgeJarJarRecursivelyAndIgnoresDependencyModIDs`（或新增 `TestForgeModIDStrengthClassification`）：
  - 构造一个顶层 JAR：`mods.toml` 声明 `topforge` + `jei`（两个 `[[mods]]` 块），内嵌一个 nested jar 声明 `childmod`
  - 调用 `ParseModZipReader`，断言：
    - `topforge` 和 `jei` 的 `IsJij == false`
    - `childmod` 的 `IsJij == true`
  - 调用 `PrimaryModIDs`，断言结果为 `["topforge", "jei"]`，不含 `childmod`
- [ ] 运行：`go test ./minecraft/... -run TestForge`

---

### Step 8 — 完整测试

- [ ] `go test ./...`（不允许新增失败用例）

---

## Validation Commands

```bash
go build ./...
go test ./minecraft/... -run TestForge -v
go test ./minecraft/... -v
go test ./modbridge/... -v
go test ./downloader/... -v
go test ./...
```

## Rollback Points

- Step 1-2 是纯加法（新字段 + 新函数），不影响现有行为，可随时 revert
- Step 3-6 是行为变更，若出现问题可逐步还原，先恢复 downloader，再 modbridge

## Review Gates

- 每个 Step 后跑对应包的 `go build` 确认编译通过
- Step 7 测试要能清楚展示两类 modID 的分类结果
- Step 8 全量测试绿

## Notes

- `deduplicateStringSlice` 或 `deduplicateStrings`：先确认 `modparser.go` 中现有 dedup 函数名称，保持一致
- `tryHardlinkInstall` 第 ~317 行 `archiveSupersededModJars(parsedExisting)` 是否已通过 `FilterFullyCoveredPaths` 需在 Step 5b 实现前核实
