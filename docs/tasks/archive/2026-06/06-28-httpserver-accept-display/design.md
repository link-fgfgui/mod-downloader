# Design: HTTP Server Accepted Mods Display

## Data Flow

```
Extension POST → httpserver.handleRoot()
  → parse payloads (unchanged)
  → get current filter state from global.GetSelectedVersion()
  → for each payload (concurrent):
      → providers.LookupProjectByPlatform(platform, idOrSlug)
      → providers.ListMatchingProjectVersions(project, version, modLoader)
      → if versions non-empty → include in results
      → if payload.File matches a version → auto-pin via database.UpsertPinnedMod
  → emit "extension-mods-accepted" event with filtered ModProject[]
  → return HTTP response to extension
  
Frontend receives "extension-mods-accepted":
  → set searchResults to received projects
  → refreshDownloadStates()
```

## Backend Changes

### 1. `providers/service.go` — New lookup function

```go
func LookupProjectByPlatform(platform, idOrSlug string) (models.ModProject, bool)
```

Resolves a single ModProject by platform name and project ID or slug. Uses the platform-specific provider's `ExactSearch` internally.

Helper:
```go
func providerByPlatform(platform string) modProvider
```

### 2. `httpserver/server.go` — Revised handleRoot

Remove `downloader.QueueModDownload` call. Replace with:
1. Read current filter from `global.GetSelectedVersion()` → minecraftVersion, modLoader
2. Concurrently resolve each payload's metadata via `providers.LookupProjectByPlatform`
3. For each resolved project, call `providers.ListMatchingProjectVersions` to check filter match
4. If payload has `File` (versionID) and that version is in matching list, auto-pin
5. Emit `extensionModsAcceptedEvent` with all matching projects
6. Return JSON response to extension with per-item status

### 3. `structs/search.go` — New event struct (if needed)

Reuse `SearchModsUpdate` for the event payload since it already has the right shape (results + loading flag).

### 4. Auto-pin logic

When `payload.File` is non-empty:
- Find the version in matching versions list by ID
- If found, call `database.UpsertPinnedMod()` directly (not the toggle-semantic PinModVersion)
- The modID for pinning is extracted from the ModProject (platform-specific project ID, not the composite key)

## Frontend Changes

### 5. `stores/downloadSearch.ts` — Listen for new event

In `start()`, register listener for `extension-mods-accepted`:
- Set `searchResults` to received projects
- Clear `isSearching` / `isLoadingMore`
- Call `refreshDownloadStates()`

In `stop()`, unregister the listener.

### 6. No changes to `Download.vue` or `SearchResultList.vue`

The SearchResultList already renders whatever is in `searchResults`. The new event just provides a different source of results.

## Edge Cases

- **No selected version**: If `global.GetSelectedVersion()` has no minecraftVersion or modLoader, skip filtering — show all resolved projects but don't auto-pin.
- **Provider lookup fails**: Skip that payload item, mark as skipped in HTTP response.
- **Duplicate projects**: If extension sends duplicate slugs/IDs, deduplicate by composite project key.
- **Download page not active**: Events still fire; if the user navigates to the download page later, the results won't be there (ephemeral). This is acceptable — the extension use case assumes the app is open.
