# App Backend Guidelines

> Package-scoped entry point for the Wails app shell in `mod-downloader`.

The app package owns Wails runtime integration, frontend bindings, and adapter
code in `app.go`. Reusable domain logic belongs in the `core/` submodule.

## Required References

- [Shared Backend Guidelines](../../backend/index.md)
- [Directory Structure](../../backend/directory-structure.md)
- [Quality Guidelines](../../backend/quality-guidelines.md)

## Pre-Development Checklist

- Check whether the change belongs in `app.go` / frontend bindings or in the
  `core/` submodule.
- Keep Wails runtime imports in the app adapter layer only.
- Preserve `replace github.com/link-fgfgui/mod-downloader-core => ./core`.

## Quality Check

- Run `go test ./...` from the app repo after app adapter changes.
- Run `go build ./...` when Go signatures or imports change.
- If public Wails API signatures change, regenerate frontend bindings.
