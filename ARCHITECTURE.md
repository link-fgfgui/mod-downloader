# Architecture Map

This repository is a Wails desktop shell around the reusable `core/` Go
module. Start here when tracing a workflow.

## Dependency Direction

```text
Vue views/stores/components
        │ generated frontend/wailsjs bindings
        ▼
root App (Wails adapter: app.go and app_*.go)
        │ typed requests, native dialogs, runtime events
        ▼
core/appcore.Service (workflow orchestration, no Wails imports)
        ├── providers   Modrinth / CurseForge API and metadata
        ├── downloader  queue, dependency preflight, file transfer
        ├── modbridge   provider ↔ local-mod/install-state bridge
        ├── minecraft   launcher discovery and JAR metadata parsing
        ├── storage     SQLite user data and platform/cache snapshots
        ├── configs     TOML and environment configuration
        └── global      process clients and in-memory indexes
```

The root module owns process startup (`main.go`), Wails event translation,
native file dialogs, and frontend-facing method names. `core` owns all domain
logic and may be reused by the sibling CLI project. Generated files under
`frontend/wailsjs/` are outputs, not hand-authored API definitions.

## Workflow Entry Points

| Workflow | Frontend entry | Wails adapter | Core boundary | Main lower layers |
| --- | --- | --- | --- | --- |
| Search | `views/Download.vue`, `stores/downloadSearch.ts` | `App.SearchMods` | `Service.SearchMods` | `providers`, `storage` |
| Download/install and queue actions | `views/Download.vue`, `App.vue`, `stores/downloadQueue.ts` | `App.QueueModDownload`, `CancelDownload`, `RetryDownload`, `RemoveCanceledDownload` | matching `Service` methods | `downloader`, `modbridge`, `storage` |
| Local mods | `views/Manage.vue`, `stores/minecraft.ts` | `App.RefreshSelectedVersionMods` / `ApplyLocalModBatchOperation` | `Service.RefreshSelectedVersionMods` / local-mod methods | `minecraft`, `global`, `storage` |
| Settings | `views/Settings.vue`, `stores/settings.ts` | `App.GetSettings` and `Save*` methods | `Service.GetSettings` and `Save*` methods | `configs`, `downloader`, `providers` |
| Favorites/pins | `views/Favorites.vue`, `views/Unpin.vue` | corresponding `App` methods | `Service` favorite/pin methods | `storage`, `providers` |

## Event Flow

`appcore.Event` is adapter-neutral. `App.emitCoreEvent` maps its `EventKind` to
the kebab-case Wails event names consumed by Vue. HTTP bridge events follow the
same pattern through `App.emitHTTPServerEvent`. Do not put Wails runtime calls
in `core/`.

## Reading Rules

1. Identify the frontend route/store and its Wails method.
2. Read the matching `core/appcore.Service` method, then follow the named
   lower-layer package; do not scan every package first.
3. Treat `models` as canonical cross-layer data. Requests/responses belong in
   `core/structs`; persistence belongs in `core/storage`.
4. For downloads, read queue state and cancellation contracts in
   `core/downloader` before following provider or file-transfer details.

See `core/appcore/README.md` and `frontend/README.md` for package-local maps.
