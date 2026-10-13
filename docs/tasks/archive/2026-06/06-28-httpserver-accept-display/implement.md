# Implementation Plan

## Checklist

### Backend

- [ ] 1. Add `LookupProjectByPlatform` and `providerByPlatform` to `providers/service.go`
- [ ] 2. Add `VersionMatchesFilter(version ModVersion, mcVersion, modLoader string) bool` to `providers/service.go` (export existing `versionMatchesSearchRequest` logic, or add a new thin wrapper)
- [ ] 3. Rewrite `httpserver/server.go` `handleRoot()`:
  - Remove `downloader.QueueModDownload` import/usage
  - Read current filter from `global.GetSelectedVersion()`
  - Concurrent metadata resolution per payload
  - Filter by matching versions
  - Auto-pin matching explicit versions
  - Emit `extension-mods-accepted` event via `runtime.EventsEmit`
  - Return response to extension
- [ ] 4. Add import for `providers`, `database`, `global` in httpserver; remove `downloader` import
- [ ] 5. Update `httpserver/server_test.go` if it exists (adjust for new behavior)

### Frontend

- [ ] 6. Add `extensionModsAcceptedEvent` constant and listener in `stores/downloadSearch.ts`
  - In `start()`: register listener that sets `searchResults`, calls `refreshDownloadStates()`
  - In `stop()`: unregister listener
- [ ] 7. Store the unregister function in state (`stopListeningExtensionModsAccepted`)

### Validation

- [ ] 8. Manual test: POST sample payload to HTTP server, verify mods appear on download page
- [ ] 9. Manual test: POST payload with versionID, verify auto-pin
- [ ] 10. Manual test: existing search still works normally
- [ ] 11. Build check: `cd frontend && npm run build` passes
- [ ] 12. Go build: `go build ./...` passes

## Notes

- The `extension-mods-accepted` event reuses the same data shape as `search-mods-updated` — an object with `results: ModProject[]` — so the existing listener logic can be adapted.
- Auto-pin uses `database.UpsertPinnedMod` directly (not `PinModVersion` which has toggle semantics).
- httpserver already imports `runtime` via wails for events.
