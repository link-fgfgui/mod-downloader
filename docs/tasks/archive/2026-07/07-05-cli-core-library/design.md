# Design: Extract core library and add CLI

## Architecture

Create a UI-independent service package that owns application lifecycle and domain workflows. The existing Wails `App` becomes an adapter that translates frontend calls and Wails events to service calls. A new CLI command package becomes another adapter over the same service.

Proposed shape:

- `core` or `appcore`: reusable service package with `Service`, `Options`, lifecycle methods, and workflow methods.
- `cmd/mod-downloader-cli` or equivalent: CLI binary entrypoint.
- Existing root `main.go`: remains the Wails binary entrypoint.
- Existing domain packages stay in place: `providers`, `downloader`, `modbridge`, `minecraft`, `database`, `configs`, `global`, `models`, and `structs`.

## CLI MVP

The first CLI version includes these commands:

- `config`: show effective configuration and optionally update supported preferences/API keys.
- `versions`: list supported Minecraft instances discovered from the configured or flag-provided root.
- `search`: search provider metadata using the existing provider layer.
- `install`: install a selected project/version into a target instance using the same backend install path as the UI.
- `mods`: list installed local mods for a target instance.

## Boundaries

- Core service must not import Wails runtime.
- Wails adapter owns directory dialogs and frontend event emission.
- CLI adapter owns argument parsing, stdout/stderr formatting, exit codes, and JSON rendering.
- `modbridge` remains the bridge between local jar analysis and platform metadata; do not move provider/minecraft convergence into CLI or Wails code.
- Shared model types remain in `models`; request/response structs stay canonical unless a command needs a narrow serialized output type.

## Data Flow

1. Adapter constructs service options from config, flags, or Wails startup context.
2. Service initializes config, database, provider clients, Minecraft release metadata, and optional local HTTP server behavior if still required by the desktop UI.
3. Adapter calls service workflow methods.
4. Service returns typed values and/or emits service-level events through callbacks.
5. Adapter maps service events to Wails runtime events or CLI progress/output.

## Compatibility

- Existing Wails method names should remain stable to avoid frontend churn.
- If Go API signatures change, regenerate Wails bindings and document the binding change in implementation notes.
- Config file and environment variable behavior should match current `configs.Load` semantics unless explicitly changed.

## Trade-Offs

- Keeping global package state initially lowers refactor risk, but the service should centralize access so future state injection is possible.
- Adding a service event callback is less invasive than replacing the download queue with a full event bus; it still removes Wails imports from reusable logic.
- The first task can deliver the MVP CLI because the command surface is limited to `config`, `versions`, `search`, `install`, and `mods`; additional commands should be follow-up work.

## Risks

- Download queue behavior currently couples event emission and queue state; changing it can regress UI loading states.
- Version selection currently relies on `global` selected-version state; CLI commands need deterministic flag-based selection or explicit service selection before install.
- The desktop HTTP server may be UI-only; moving lifecycle too aggressively could start unnecessary services in CLI mode.
