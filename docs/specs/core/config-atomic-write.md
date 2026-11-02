# 配置文件原子写入与临时文件清理

## 结论

`core/configs.Save` 采用 **同目录临时文件 + 编码/fsync + rename** 替换 `mod-downloader.toml`，
并在每次保存前清扫同模式的陈旧临时文件。

## 现象与根因

进程在保存过程中被强杀（SIGKILL、断电、容器回收）时，Go 的 `defer` 不会执行，
`.mod-downloader.toml.<random>.tmp` 会永久残留。反复被杀会持续堆积临时文件，
污染工作目录（`mod-downloader.toml` 常与用户实例目录同层）。

## 约束

**Don't**

- 不要直接 `os.OpenFile(path, O_TRUNC)` 写真实配置：编码失败会截断用户已有配置。
- 不要在 `CreateTemp` 前假设目录里没有残留：崩溃/强杀路径不执行 defer。
- 不要用固定临时文件名：并发保存会互相覆盖（虽然 core 单实例，但测试与工具链会并发跑）。

**Instead**

- 临时文件名保持 `.<configBase>.*.tmp` 模式（`os.CreateTemp` 保证随机后缀）。
- 保存前用 `filepath.Glob` 清扫同模式残留；**前提是 core 进程级单实例**
  （见 [`single-instance.md`](./single-instance.md)），否则清扫可能删掉另一个进程正在写的文件。
- 写失败路径必须 `Close` + `Remove`；成功路径 `Rename` 后再返回。

## 验证

- `TestSaveReplacesConfigAtomicallyAndLeavesNoTempFiles`：连续两次保存后目录无 `*.tmp`。
- `TestSaveSweepsStaleTempFilesFromKilledWrites`：预置残留 temp 文件，保存后它被清除且新配置生效。
- `TestSaveFailurePreservesOriginalConfig`：编码/写入失败时原配置内容不变。

位置：`core/configs/config.go`（`Save`）、`core/configs/config_test.go`。
