# Repository Guidelines

## Project Structure & Module Organization

This is a Wails v2 desktop app for downloading and managing Minecraft mods. Go entry points live in `main.go` and `app.go`. Backend packages are organized by responsibility: `configs/` for TOML/env configuration, `database/` for BuntDB cache access, `downloader/` for download queue logic, `global/` for shared local mod state, `logging/` for shared logging helpers, `minecraft/` for jar and version parsing, `providers/` for mod source abstractions, and `structs/` for shared models. Tests are colocated with Go packages as `*_test.go`.

The frontend lives in `frontend/` and uses Vue, TypeScript, Vite, Pinia, Vuetify, vue-router, and vue-i18n. UI source is under `frontend/src/`; generated Wails bindings are under `frontend/wailsjs/`. Build metadata and platform assets live in `build/`.

## Build, Test, and Development Commands

- `npm ci --prefix frontend`: install frontend dependencies from `package-lock.json`.
- `wails dev`: run the desktop app in development mode from the repository root.
- `wails build`: create a production Wails build.
- `go test ./...`: run all Go unit tests.
- `go test ./minecraft`: run one backend package test suite while iterating.
- `npm run build --prefix frontend`: type-check the Vue app and build frontend assets.
- `npm run preview --prefix frontend`: preview the built frontend outside Wails.

## Coding Style & Naming Conventions

Format Go code with `gofmt`; keep package names short, lowercase, and aligned with directory names. Prefer focused package APIs over cross-package globals, except for existing `global/` patterns. Name Go tests after behavior, for example `TestDetectMinecraftVersion`.

Frontend code uses Vue single-file components with PascalCase component names such as `SearchResultList.vue`. Keep TypeScript modules small, use existing Vuetify patterns, and place reusable UI under `frontend/src/components/`.

## Testing Guidelines

Backend tests use Go's standard `testing` package. Add or update colocated `*_test.go` files for parser, downloader, database, and configuration changes. For frontend changes, run `npm run build --prefix frontend`; there is no dedicated frontend test runner configured.

Avoid committing runtime state. `mod-downloader.toml` and `mods.buntdb` are local data files used by the app during development.

## Commit & Pull Request Guidelines

Recent commits use short, imperative summaries such as `Add Wails build and release workflow` and `Implement download cancellation and jar version detection`. Keep the subject specific and under roughly 72 characters when possible.

Pull requests should describe the user-facing change, list tests run, and mention affected areas such as backend package, provider integration, or frontend view. Include screenshots or screen recordings for visible UI changes, and note any configuration or API-key requirements.
