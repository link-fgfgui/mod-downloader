# 统一默认网络请求 User-Agent

## Goal

确保所有应用发出的 HTTP 请求都携带统一的应用 User-Agent，避免未显式设置请求时退回 Go 默认的 `Go-http-client/1.1`。

## Requirements

- 默认请求使用 `mod-downloader/dev`，调用方提供版本时使用 `mod-downloader/<version>`。
- 覆盖 Minecraft 版本清单、HTTP Range Reader 和 filetransfer 默认客户端。
- 保留调用方显式传入的 `User-Agent`，不被默认注入逻辑覆盖。
- 继续使用现有代理配置和标准库 HTTP 客户端，不引入第三方网络库。

## Acceptance Criteria

- [x] 默认网络客户端注入应用 UA。
- [x] Minecraft 相关请求不再使用 Go 默认 UA。
- [x] filetransfer 默认客户端使用应用 UA。
- [x] 显式 UA 可覆盖默认值。
- [x] `core` 中 `go test ./...` 通过。

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
