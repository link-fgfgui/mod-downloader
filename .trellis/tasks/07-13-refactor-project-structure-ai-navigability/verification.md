# Verification Evidence

## Automated checks

- `go test ./...` (repository root): passed.
- `cd core && go test ./...`: passed.
- `go vet ./...` (repository root and `core/`): passed.
- `cd frontend && npm run build`: passed; Vite emitted only the existing large
  chunk advisory.
- `cd frontend && npm run lint`: passed.
- `git diff --check`: passed.

## Mimo probe

Prompt used:

> 随机选择一个工作流并给出最终梳理。先读取 ARCHITECTURE.md、
> core/appcore/README.md、frontend/README.md 和 core/models/README.md。
> 只引用实际存在的文件和函数；若第一次猜错文件名，立即用目录列表校正后再继续。
> 输出 5 个以上文件路径、关键函数、数据流和一个明确的不确定点；不要输出工具过程。

The final stdio result selected the search workflow and cited real files and
functions: `core/appcore/service.go` (`Service.SearchMods`,
`ListMatchingProjectVersions`, `PinModVersion`, `LookupProject`),
`core/models/models.go` (`ModProject`, `ModVersion`, `ProjectKey`), and the
frontend path `Download.vue` → `downloadSearch.ts` → `App.SearchMods`. It
described the provider callback/event flow and named downloader cancellation
as an explicit remaining uncertainty. An earlier probe guessed a nonexistent
model filename; the corrected maps caused the final run to read
`core/models/models.go` successfully.

## Observed documentation bug fixed

The root README advertised `./cmd/mod-downloader-cli`, but that directory is
absent from this checkout. The development section now points to the sibling
CLI repository instead of an unrunable command.
