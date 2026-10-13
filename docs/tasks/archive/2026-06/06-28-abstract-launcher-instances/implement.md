# Implementation Plan

## Checklist

- [x] Add a launcher layout abstraction in the `minecraft` package.
- [x] Move standard `.minecraft` version directory scanning into the shared abstraction or expose it through a callback without duplicating manifest parsing.
- [x] Move Prism instance aggregation out of `app.go` into the `minecraft` package abstraction.
- [x] Update `app.loadVersionsFromDisk` to call the shared abstraction.
- [x] Keep `minecraft.VersionDirPath` as the launcher-agnostic path resolver, internally delegating to the selected layout as needed.
- [x] Update or add tests for standard and Prism loading through the abstraction.
- [x] Run formatting and validation.

## Validation Commands

```bash
gofmt -w app.go minecraft/*.go
go test ./...
go vet ./...
go build ./...
```

## Risk Points

- Prism composite IDs are used as local-mod instance IDs; changing the string format would break local mod lookup.
- `VersionInfo.Name` is used as display name and lookup key; Prism must keep using the instance name.
- Prism instances without `.minecraft` must still resolve to the instance root.
- `loadVersionsFromDisk` starts asynchronous hardlink scanning; path resolution must remain correct there.

## Review Gate

Before starting implementation, confirm the plan preserves current behavior and keeps new-launcher support out of scope for this task.
