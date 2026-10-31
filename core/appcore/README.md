# `appcore` Service Map

`appcore` is the adapter-neutral orchestration layer. It owns the service
object, lifecycle/configuration, and workflows exposed to both Wails and the
CLI. It must not import Wails runtime packages.

## Where To Start

- `contracts.go`: event taxonomy, service options, and typed request/response
  contracts shared by adapters.
- `../models/models.go`: canonical `ModProject`/`ModVersion` definitions used
  by the service; this is the only model file in the package.
- `service_runtime.go`: lifecycle, configuration application, provider clients,
  mods watcher, and shutdown ordering.
- `service_search.go`, `service_favorites.go`, `service_settings.go`,
  `service_downloads.go`, `service_localmods.go`, `service_metadata.go`:
  workflow methods grouped by domain, with method names kept stable.
- `service_extension.go`: browser-extension payload resolution, filtering,
  auto-pinning, and the `extensionModsAccepted` event.
- `dependency_impact.go`, `unused_dependencies.go`: local dependency analysis.
- `packwiz_export.go`: packwiz export orchestration; native dialogs stay in the
  root Wails adapter.
- `user_agent.go`: versioned outbound identity.

Use function names as the stable anchors rather than line numbers.

## Contracts

- Return typed `models` and `structs` values; do not add aliases or re-exports.
- Emit `appcore.Event` for UI progress; adapters decide how to display it.
  `App.emitCoreEvent` is the only Wails event-name mapping; transport adapters
  such as `httpserver` must not own event names.
- Event payloads use the same `core/structs` types as RPC results. The frontend
  mirrors event-only payloads once in `frontend/src/utils/events.ts`;
  there is no second PascalCase contract.
- Keep persistence and provider access behind their existing packages.
- Keep cancellation checks inside the downloader queue lifecycle. `Startup`
  opens the queue, `Shutdown` cancels and drains it before storage closes.
