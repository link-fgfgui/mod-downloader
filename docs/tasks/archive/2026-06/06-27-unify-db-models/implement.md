# Implementation Plan: Unify database layer with models types

## Execution Order

### Step 1: Upgrade cacheVersion and add migration logic
- `database/database.go`: bump `cacheVersion` from 1 to 2
- In `loadCacheState`: if loaded state's `Version < cacheVersion`, return `newCacheState()` (discard old cache)
- This ensures old gob files with `ModPlatform`/`ModPlatformVersion` types don't cause deserialization errors

### Step 2: Replace database record types with models types
- `database/mods.go`: remove `ModPlatform`, `ModPlatformVersion`, `database.ModDependency`, `ModJarMetadata` (ModJarMetadata is unused externally)
- Add `import "mod-downloader/models"`
- All function signatures change per design.md API table
- Internal helpers (`copyVersion`, `copyDependencies`, `savePlatformVersion`, `normalizeDependencies`, etc.) operate on models types
- `savePlatformVersion`: use a separate internal ID field approach — store the internal UUID in a side map or use `ModVersion.ID` for the internal UUID and `ModVersion.VersionID` for the platform version ID (this matches the existing ModVersion struct where both fields exist)
- `database/database.go`: update `cacheState` map value types, update `copyVersion`/`copyDependencies`

### Step 3: Update database tests
- `database/mods_test.go`: change `ModPlatform{...}` → `models.ModProject{...}`, `ModPlatformVersion{...}` → `models.ModVersion{...}`, `ModDependency{...}` → `models.ModDependency{...}`
- Verify: `go test ./database/...`

### Step 4: Remove providers conversion layer
- Delete `providers/bridge.go`
- `providers/cache.go`: remove `modPlatformToModProject`, `modProjectToModPlatform`, `modPlatformVersionToModVersion`, `modVersionToModPlatformVersion`; simplify `GetProjectByID`/`StoreProject`/`GetVersionByID`/`GetVersionsByProject`/`StoreVersion`/`StoreVersions` to pass models types through directly
- `providers/modprovider.go`: remove `projectVersionResultsToDB`, `dbProjectVersionsToResults`, `projectDependenciesToDB`, `dbDependenciesToResults`; update `saveProjectVersionsSnapshot` and `getProjectSnapshotPlatform` to use `models.ModProject` / `models.ModVersion`; update callers that read from database

### Step 5: Verify full build and tests
- `go build ./...`
- `go test ./...`

## Validation Commands

```bash
go build ./...
go test ./...
go vet ./...
```

## Rollback

All changes are in-tree and uncommitted until step 5 passes. `git checkout -- .` to revert.
