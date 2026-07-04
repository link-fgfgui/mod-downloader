# 区分 mods.toml 声明 modid 与 jij mod 的优先级

## Goal

在 Forge / NeoForge JAR 解析中，明确 `mods.toml` 里声明的 `[[mods]].modId` 与 nested jar / jar-in-jar(JIJ) 解析出的 modID 的优先级规则，避免下载状态、冲突判断、替换归档和本地展示把宿主包声明的 modID 与内嵌 modID 等同看待。

## Confirmed Facts

- `ParseModZipReader` 当前会先收集当前 JAR 的声明元数据，再递归收集 nested jar / jarjar 的 mod 元数据，最后按 modID 去重并保留首次出现项，见 `minecraft/modparser.go:340`, `minecraft/modparser.go:359`, `minecraft/modparser.go:363`, `minecraft/modparser.go:498`。
- Forge / NeoForge 的声明 modID 来自 `mods.toml` / `neoforge.mods.toml` 的 `[[mods]].modId`，见 `minecraft/modparser.go:95`, `minecraft/modparser.go:113`, `minecraft/modparser.go:123`。
- 现有测试已经证明 Forge 解析会同时返回顶层 `mods.toml` 里的多个 `modId` 与 jar-in-jar 子包 `modId`，且会忽略 `[[dependencies.*]]` 中的 `modId`，见 `minecraft/modparser_test.go:50`。
- 远端版本安装状态与冲突判断会把一个版本解析出的全部 modID 作为同一组参与匹配，见 `modbridge/modbridge.go:152`, `modbridge/modbridge.go:183`, `modbridge/modbridge.go:520`。
- 下载执行阶段会在真正写入新文件前，先查找命中的本地模组并归档可替换文件，见 `downloader/download.go:537`, `downloader/download.go:542`。
- 当前产品模型是“有冲突先提示、执行时再替换”：列表状态用 `conflict` 表达跨项目命中，真正点击安装时才归档旧文件并安装新文件，见 `modbridge/modbridge.go:202`, `modbridge/modbridge.go:208`, `downloader/download.go:542`。
- JIJ 替换规则当前也是按“新包覆盖的 modID 集合是否完整覆盖旧包 modID 集合”判断是否允许归档旧文件，见 `modbridge/modbridge.go:535`。
- 本地扫描后，每个解析出的 `ModInfo` 都会单独写入本地索引；同一个 JAR 可以对应多个 `LocalModFile` 记录，共享同一个 SHA1 与路径，见 `global/localmods.go:37`。
- 管理页当前按文件名/路径分组，首条记录作为 primary，其余记录作为 `jij` 展示，因此解析顺序会影响哪个 modID 被当作主显示项，见 `frontend/src/views/Manage.vue:98`。
- 用户当前决策：管理页暂时保留现有显示逻辑，不额外区分“顶层多 mod 声明”和真正的 JIJ；tooltip 不承担更复杂的语义区分。

## User Scenario

- `jei` 是标准 jar，`mods.toml` 声明 `jei`。
- `tmrv` 是标准 jar，但它的顶层 `mods.toml` 同时声明 `tmrv` 与 `jei` 两个 `[[mods]]` 块。
- 该场景下，安装 `jei` 时应先把 `tmrv` 识别为冲突对象；用户确认安装后，再按现有替换流程归档 `tmrv` 并安装 `jei`。
- JIJ / nested jar 只是弱引用，不应因为命中其内嵌 modID 就触发同等级冲突或替换。

## Requirements

- 为 Forge / NeoForge JAR 明确定义两类 modID 语义：
  - 顶层 `mods.toml` / `neoforge.mods.toml` `[[mods]].modId`：强引用
  - nested jar / jar-in-jar 解析出的 modID：弱引用
- 强引用 modID 必须在至少以下场景中参与同等级匹配：
  - 远端版本 modID 解析与缓存
  - 安装按钮状态判断（new / installed / update / conflict）
  - 替换归档判断
  - 本地管理页展示
- 弱引用 modID 不能与强引用 modID 等价参与安装冲突或替换命中；它们只用于表达宿主包包含了哪些内嵌模组。
- 当一个顶层 JAR 在 `mods.toml` 中声明多个 `[[mods]].modId` 时，这些 modID 都属于同一个物理文件的强引用集合。
- 列表状态与安装执行的产品模型保持不变：冲突先显示 `conflict`，实际替换只在用户执行安装时发生。
- 保持 `[[dependencies.*]]` 的 `modId` 不参与当前 modID 集合。
- 保持现有 Fabric nested jar 递归能力与 Forge / NeoForge jarjar 递归能力。
- 变更后应能通过测试清楚表达：哪些 modID 属于强引用集合，哪些仅属于 JIJ 弱引用集合。
- 管理页本次不新增 UI 语义区分，继续沿用当前分组与 tooltip 展示。

## Acceptance Criteria

- [ ] 对一个 Forge / NeoForge JAR，同时包含顶层 `mods.toml` 声明 modID 与 jar-in-jar 子模组时，测试能明确断言二者分别落入强/弱两类语义。
- [ ] `tmrv` 这类在顶层 `mods.toml` 同时声明 `tmrv` 与 `jei` 的标准 jar，会在安装 `jei` 时先被识别为 `conflict`，用户执行安装后再按替换规则归档。
- [ ] 安装状态判断不会因为仅命中 JIJ 子模组 modID，就把宿主包错误标成 `installed` / `update` / `conflict`。
- [ ] 替换归档逻辑遵守同一套强/弱规则，不会因为次级 JIJ modID 命中而误归档不该替换的现有文件。
- [ ] 管理页本次无需新增展示区分，继续保持当前按文件分组与 tooltip 展示的行为。
- [ ] 现有“忽略 dependencies modId”行为继续保留。

## Out Of Scope

- 新增 `mods.toml` `[[dependencies.*]]` 或 `fabric.mod.json depends` 的依赖解析。
- 改动 Fabric 的基础 modID 解析语义，除非为了统一 UI 展示需要共享非行为性结构。
- 新增新的 provider API 字段或改变外部平台返回数据结构。
