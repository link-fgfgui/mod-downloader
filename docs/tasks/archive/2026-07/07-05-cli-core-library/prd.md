# Extract core library and add CLI

## Goal

Add a command-line version of `mod-downloader` by separating UI-independent Go logic into a reusable library/service layer, then wiring both the existing Wails UI and a new CLI entrypoint through that layer.

The CLI should make the app usable without the desktop UI for common Minecraft mod discovery, local instance inspection, and install workflows.

## Background

- The project is currently documented as a desktop Minecraft mod downloader and local mod manager built with Go, Wails, Vue, Pinia, and Vuetify. It searches Modrinth and CurseForge, filters by Minecraft version and loader, installs jars into an instance `mods` directory, resolves required dependencies, supports pinned versions, scans local mods, parses jar metadata, and caches platform/jar metadata. Evidence: `README.md:3`, `README.md:9`.
- Configuration is read from `mod-downloader.toml` in the working directory and can also come from environment variables. Evidence: `README.md:32`, `configs/config.go`.
- The Wails `App` currently owns startup/shutdown orchestration: config load, global Minecraft directory setup, database open, provider client initialization, Minecraft release version fetch, and local HTTP server startup. Evidence: `app.go:51`.
- Several Wails-exposed methods are thin wrappers around backend packages, but they still emit Wails runtime events directly. Examples: search emits `search-mods-updated` in `App.SearchMods`; download-state backfill emits `download-states-updated` in `App.GetDownloadStates`. Evidence: `app.go:87`, `app.go:271`.
- Minecraft directory selection, version discovery, selected-version mutation, mod refresh, icon enrichment, and hardlink indexing currently live in `app.go`, not in a reusable service package. Evidence: `app.go:292`, `app.go:344`, `app.go:363`, `app.go:408`.
- Download queue logic lives in `downloader`, but it imports Wails runtime and uses `context.Context` partly as an event carrier. Evidence: `downloader/download.go:21`, `downloader/download.go:48`.
- `modbridge` is already documented as the convergence point between local jar analysis and platform API analysis, with dependency direction `downloader -> modbridge -> {providers, database, global, minecraft}`. This boundary should be preserved. Evidence: `modbridge/modbridge.go:1`.
- `go.mod` already includes `github.com/urfave/cli/v2` indirectly through Wails tooling, but the project does not currently expose a CLI binary. Evidence: `go.mod:51`, `main.go`.

## Requirements

- R1: Introduce a UI-independent Go service/library boundary for app lifecycle and core workflows.
  The service must cover initialization, shutdown, configuration access/update, provider client setup, database lifecycle, Minecraft directory/version discovery, selected instance handling, local mod refresh, search, version lookup, pin/unpin, download queue operations, and download state computation.

- R2: Keep Wails-specific code out of the new core service.
  Wails runtime calls, directory dialogs, and frontend event names must remain in Wails adapter code. The core service may expose callbacks, typed events, channels, or explicit return values, but it must not import `github.com/wailsapp/wails/v2/pkg/runtime`.

- R3: Preserve current desktop behavior while rerouting Wails methods through the core service.
  Existing frontend-facing method names and response shapes should remain compatible unless a design artifact explicitly documents a necessary binding change.

- R4: Add a CLI entrypoint as a separate binary target rather than replacing the Wails binary.
  The desktop entrypoint should continue to build through Wails. The CLI should be buildable and runnable through standard Go tooling.

- R5: CLI commands must be non-interactive by default.
  Commands should accept flags for Minecraft directory, Minecraft version/instance, mod loader, provider/platform where needed, query/project identifiers, version identifiers, and API keys/config overrides where needed. Interactive prompts are out of scope for the first version unless explicitly approved later.

- R6: The CLI must reuse the same models and backend behavior as the UI.
  Do not introduce aliases, re-export packages, or parallel converter paths. Shared data types continue to come from `models` and existing request/response structs unless the design chooses a narrower CLI output DTO for serialization.

- R7: CLI output must be script-friendly.
  Human-readable output may be the default, but JSON output must be available for commands that return structured data such as search results, versions, installed mods, queue status, and download results.

- R8: Tests must cover the new service boundary and CLI command behavior without relying on Wails.
  Existing package tests must remain green.

- R9: The first CLI version must include the MVP command surface: `config`, `versions`, `search`, `install`, and `mods`.
  This scope is intended to cover a complete UI-free workflow: inspect effective configuration, discover instance/version keys, search available mods, install a selected mod, and inspect installed local mods.

## Acceptance Criteria

- [x] A new core service/library package exists and can be tested without importing Wails runtime.
- [x] Existing Wails `App` methods delegate business behavior to the core service while keeping UI-only concerns in `app.go` or a Wails-specific adapter.
- [x] `go list` or an equivalent dependency check shows the core service and CLI packages do not import Wails runtime.
- [x] A separate CLI binary target exists and builds with standard Go tooling.
- [x] The CLI can initialize config/provider/database state using the same config semantics documented in `README.md`.
- [x] The CLI can discover supported Minecraft instances from a configured or flag-provided Minecraft root.
- [x] The CLI can search mods using the existing provider logic.
- [x] The CLI can install/download a selected mod version into a target instance using the same dependency resolution, pinning, conflict/update, local parsing, and hardlink behavior as the UI path.
- [x] The CLI can inspect local installed mods for an instance.
- [x] The CLI exposes `config`, `versions`, `search`, `install`, and `mods` commands.
- [x] Structured JSON output is available for data-returning CLI commands.
- [x] The desktop UI still passes its existing backend tests and generated binding expectations after rerouting.
- [x] Verification commands pass: `go build ./...`, `go vet ./...`, and `go test ./...`.

## Out Of Scope

- Publishing installers, Homebrew/Scoop packages, or release automation.
- A terminal UI or interactive wizard.
- Replacing the Wails desktop app.
- Adding support for new mod providers or new mod loaders.
- Changing persistent cache format unless the implementation proves it is required and documents migration/compatibility.

## Decisions

- D1: The MVP CLI command surface is `config`, `versions`, `search`, `install`, and `mods`.
