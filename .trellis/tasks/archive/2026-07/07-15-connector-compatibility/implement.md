# Connector compatibility implementation plan

## Checklist

- [x] Add transient Connector state to `VersionInfo` and normalized loader ownership to `ModInfo`; update generated-model expectations and focused struct consumers.
- [x] Scope the local JAR metadata cache by SHA1 plus parse mode/preferred loader, preserving defensive copies and existing requested-loader behavior.
- [x] Add all-loader local parsing with stable loader annotation/merging, wire `ScanModsDir` and `ScanModFile` to it, and cover Fabric/Forge/NeoForge/multi-loader/cache-isolation cases.
- [x] Add appcore helpers that initialize the actual loader, detect only enabled top-level `connector`, reconcile state after full/incremental refresh, and reset virtual state on explicit selection/version reload.
- [x] Add `ToggleConnectorLoader` to appcore and `App`, update selected cache/event behavior, and test round trips plus unavailable/disabled/removed Connector paths.
- [x] Regenerate Wails bindings and extend `minecraftStore` to initialize selected mods once and expose a guarded non-persistent toggle action.
- [x] Add the Connector switch button and localized destination labels to `VersionChoose.vue`.
- [x] Extend `VirtualList` with an item-selectability predicate that excludes non-data rows from click, range, and select-all state.
- [x] Partition Manage groups by active loader, render the default-collapsed incompatible tail in the same virtual list, preserve existing actions, reset folding on tuple changes, and add localized labels/counts.
- [x] Run the full quality gate, inspect the final diff including the core submodule pointer, and update the relevant Trellis spec with the new parsing/selected-version contract.

## Validation Results

- `wails generate module` passed.
- `go test -race ./appcore ./minecraft ./global` passed in `core/`.
- Focused `core` tests for `appcore`, `minecraft`, `global`, `modbridge`, and `downloader` passed.
- `go build ./...` and `go vet ./...` passed in both `core/` and the root app.
- Root `go test ./...` passed.
- `npm run lint` and `npm run build` passed in `frontend/`.
- `git diff --check` passed for root and `core/`.
- Full `core/go test ./...` runs every package but retains one unrelated baseline failure: `configs.TestNetworkConfigClampsOutOfRangeValues/above_maximum_clamps` uses input `ConcurrentDownloads: 100` while `MaxConcurrentDownloads` is `256`, so the test receives `100` instead of its expected `256`. This task does not modify `core/configs`.

## Validation Commands

```bash
(cd core && gofmt -w <changed-go-files>)
(cd core && go test ./minecraft ./global ./appcore ./modbridge ./downloader)
(cd core && go test ./... && go build ./... && go vet ./...)
wails generate module
npm run lint --prefix frontend
npm run build --prefix frontend
gofmt -w <changed-root-go-files>
go test ./...
go build ./...
go vet ./...
git diff --check
git status --short
```

## Review Gates

- Confirm a requested-loader remote/download parse still returns only that loader's metadata.
- Confirm a local multi-loader JAR produces one logical mod ID with all declared loaders.
- Confirm toggling changes provider/download tuple behavior but not instance ID, name, Minecraft version, or target directory.
- Confirm reselect/reload and disabled/removed Connector restore the actual loader without persistence.
- Confirm Ctrl+A/range selection never includes the incompatible-section control row.
- Confirm search and enabled filters apply equally to compatible and folded incompatible groups.

## Risk And Rollback Points

- `core/minecraft/modparser.go` and `core/global/jarcache.go` affect conflict/install identity; land parser/cache tests before appcore wiring and revert these files together if requested-loader behavior changes.
- `core/global/global.go` must keep selected-version ordered cache and aliases synchronized after toggles; retain the existing selected-version cache tests.
- `frontend/src/components/VirtualList.vue` is shared by Download and Manage; keep the new predicate opt-in with a default that preserves all existing lists.
- `frontend/wailsjs/` is generated output; regenerate rather than hand-maintain parallel API definitions.
- No storage/config migration is allowed, so any unexpected persisted Connector field is a stop condition.
