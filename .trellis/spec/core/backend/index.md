# Core Backend Guidelines

> Package-scoped entry point for the `core/` git submodule.

The core package is the local checkout of
`github.com/link-fgfgui/mod-downloader-core`. It owns app-independent service,
provider, download, database, Minecraft parsing, and HTTP bridge logic.

## Required References

- [Shared Backend Guidelines](../../backend/index.md)
- [Directory Structure](../../backend/directory-structure.md)
- [Quality Guidelines](../../backend/quality-guidelines.md)
- [Database Guidelines](../../backend/database-guidelines.md)

## Pre-Development Checklist

- Make core changes inside `core/`, not by recreating packages in the app repo.
- Keep `appcore`, `httpserver`, and lower layers free of Wails runtime imports.
- Keep shared types in `models`; do not add aliases or re-export packages.

## Quality Check

- Run `go test ./...` from `core/` after core changes.
- Run `go build ./... && go vet ./...` from `core/` when changing exports or dependencies.
- Re-run app or CLI tests when a core change affects their adapters.
