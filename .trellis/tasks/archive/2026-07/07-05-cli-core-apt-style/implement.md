# Implementation Plan

## Steps

1. Add env-only config loading and runtime option types.
2. Add database cache path APIs and tests for explicit/default paths.
3. Extend download request structs with target fields.
4. Add install target resolution in `modbridge` and update downloader calls.
5. Preserve target fields in dependency download requests.
6. Add direct local mods scan service method.
7. Update focused tests for config, database, target resolution, dependencies,
   and GUI compatibility.
8. Run `go test ./...` from `core/`; run build/vet if exported API changes need
   broader validation.

## Validation Commands

```bash
cd core && go test ./...
cd core && go build ./... && go vet ./...
```

## Risk Points

- Global selected-instance state is shared and must not be required for direct
  CLI target installs.
- Local mod paths may be relative for GUI instances but should be absolute or
  target-relative consistently for direct CLI scans.
- Cache path changes must keep repeated `database.Open()` calls idempotent.
