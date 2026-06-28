# Implementation Plan: Abstract Platform-Agnostic Unified Provider Structs

## Overview

This document outlines the step-by-step implementation plan for refactoring the provider layer to use unified platform-agnostic structs. The plan follows a phased approach to minimize risk and maintain backward compatibility during the transition.

## Implementation Phases

### Phase 1: Create New Model Structs ✅ Low Risk

**Goal:** Define unified structs without breaking existing code.

**Files to create:**
- `providers/model.go` - New unified structs

**Tasks:**
1. Create `ModProject` struct with all fields from `SearchModResult` + `ModPlatform`
2. Create `ModVersion` struct with all fields from `ProjectVersionResult` + `ModPlatformVersion`
3. Create `ModDependency` struct with JSON aliases for frontend compatibility
4. Add helper functions:
   - `ProjectKey(platform, projectID string) string` - Generate composite ID
   - `ParseProjectKey(id string) (platform, projectID string)` - Parse composite ID

**Verification:**
```bash
go build ./providers
go test ./providers
```

---

### Phase 2: Update Database Layer ✅ Medium Risk

**Goal:** Make database layer work with new structs while maintaining backward compatibility.

**Files to modify:**
- `database/mods.go`

**Tasks:**

1. **Import new types:**
```go
import "mod-downloader/providers"
```

2. **Update type aliases or embed:**
```go
// Option A: Type alias (simple migration)
type ModProject = providers.ModProject
type ModVersion = providers.ModVersion
type ModDependency = providers.ModDependency

// Option B: Keep separate and add conversion (safer)
// Keep existing ModPlatform/ModPlatformVersion/ModDependency
// Add conversion functions
```

3. **Update cacheState struct:**
```go
type cacheState struct {
    ModProjects          map[platformKey]providers.ModProject      // was ModPlatforms
    ModVersions          map[versionKey]providers.ModVersion       // was PlatformVersions
    ModVersionKeyByID    map[string]versionKey
    PlatformAssociations map[string]PlatformAssociation
    PinnedMods           map[pinnedModKey]PinnedMod
    JarMetadata          map[string][]structs.ModInfo
    PlatformVersionScopes map[versionScopeKey]storedVersionScope
}
```

4. **Add migration logic in `readyDB()`:**
```go
func migrateCache(state *cacheState) {
    // Detect old format (check for type assertion)
    // Convert old structs → new structs
    // Mark as migrated
}
```

5. **Update all getter/setter methods:**
   - `UpsertModPlatform()` → accept `providers.ModProject`
   - `GetModPlatform()` → return `providers.ModProject`
   - `SetPlatformVersions()` → accept `[]providers.ModVersion`
   - `GetPlatformVersions()` → return `[]providers.ModVersion`
   - Keep method signatures compatible

**Verification:**
```bash
go build ./database
go test ./database
# Test migration: backup cache, run migration, verify data
```

---

### Phase 3: Add Provider Lookup Functions ✅ Low Risk

**Goal:** Add functions to fetch full model data by ID.

**Files to modify:**
- `providers/cache.go` (new file)

**Tasks:**

1. **Create cache accessors:**
```go
// GetProjectByID fetches full ModProject from cache by ID
func GetProjectByID(id string) (ModProject, bool) {
    platform, projectID := ParseProjectKey(id)
    return database.GetModPlatform(platform, projectID)
}

// GetProjectsByIDs batch fetch
func GetProjectsByIDs(ids []string) []ModProject {
    // ...
}

// GetVersionByID fetches full ModVersion from cache by ID
func GetVersionByID(platform, projectID, versionID string) (ModVersion, bool) {
    // Query database.GetPlatformVersions, find by versionID
}

// GetVersionsByIDs batch fetch
func GetVersionsByIDs(keys []struct{platform, projectID, versionID string}) []ModVersion {
    // ...
}
```

**Verification:**
```bash
go build ./providers
go test ./providers/cache_test.go
```

---

### Phase 4: Update Provider Interface ✅ High Risk

**Goal:** Change provider methods to return IDs and use store callbacks.

**Files to modify:**
- `providers/modprovider.go`

**Tasks:**

1. **Define new interface alongside old:**
```go
// Keep old interface for now
type modProvider interface {
    Name() string
    ExactSearch(req appstructs.SearchModsRequest) ([]appstructs.SearchModResult, error)
    Search(req appstructs.SearchModsRequest) ([]appstructs.SearchModResult, error)
    ListVersions(projectIDOrSlug string, filter projectVersionFilter) ([]appstructs.ProjectVersionResult, error)
}

// New interface
type modProviderV2 interface {
    Name() string
    ExactSearch(req SearchModsRequest, store func(ModProject)) (projectID string, err error)
    Search(req SearchModsRequest, store func(ModProject)) (projectIDs []string, err error)
    ListVersions(projectIDOrSlug string, filter projectVersionFilter, store func(ModVersion)) (versionIDs []string, err error)
}
```

2. **Update CurseForge provider:**
   - Add `ExactSearchV2`, `SearchV2`, `ListVersionsV2` methods
   - Convert `cfSchema.Mod` → `ModProject` in one place
   - Convert `cfSchema.File` → `ModVersion` in one place
   - Call `store()` callback after conversion
   - Return ID lists only

3. **Update Modrinth provider:**
   - Same pattern as CurseForge
   - Convert `*modrinth.Project` / `*modrinth.SearchResult` → `ModProject`
   - Convert `*modrinth.Version` → `ModVersion`

4. **Update provider registry:**
```go
var modProvidersV2 = []modProviderV2{
    curseForgeProviderV2{},
    modrinthProviderV2{},
}
```

**Verification:**
```bash
go build ./providers
go test ./providers -run TestCurseForgeProvider
go test ./providers -run TestModrinthProvider
```

---

### Phase 5: Update Service Layer ✅ High Risk

**Goal:** Make `service.go` use new provider interface and return new types.

**Files to modify:**
- `providers/service.go`

**Tasks:**

1. **Update `SearchMods()`:**
```go
func SearchMods(req appstructs.SearchModsRequest, emitUpdate func(appstructs.SearchModsUpdate)) {
    // ...
    for _, provider := range modProvidersV2 {
        go func() {
            projectIDs, err := provider.Search(req, func(project ModProject) {
                // Store in database
                _ = database.UpsertModProject(project)
            })
            // Fetch full projects by IDs
            projects := GetProjectsByIDs(projectIDs)
            // Convert to old SearchModResult temporarily (for compatibility)
            results := projectsToSearchResults(projects)
            // Emit
        }()
    }
}
```

2. **Update `ListMatchingProjectVersions()`:**
```go
func ListMatchingProjectVersions(result appstructs.SearchModResult, mcVersion string, modLoader string) []appstructs.ProjectVersionResult {
    // Get provider
    provider, projectID, ok := providerAndProjectFromSearchResult(result)
    // Call ListVersions
    versionIDs, err := provider.ListVersions(projectID, filter, storeCallback)
    // Fetch full versions by IDs
    versions := GetVersionsByIDs(versionIDs)
    // Convert to ProjectVersionResult (temporarily)
    return versionsToProjectVersionResults(versions)
}
```

3. **Add conversion helpers (temporary):**
```go
func projectToSearchResult(p ModProject) appstructs.SearchModResult {
    return appstructs.SearchModResult{
        ID: p.ID,
        Platform: p.Platform,
        Title: p.Title,
        // No Icon field (removed)
        IconURL: p.IconURL,
        Description: p.Description,
        Downloads: p.Downloads,
        Slug: p.Slug,
    }
}
```

**Verification:**
```bash
go build ./providers
go test ./providers -run TestSearchMods
go test ./providers -run TestListMatchingProjectVersions
```

---

### Phase 6: Update App Layer ✅ Medium Risk

**Goal:** Update `app.go` to use new types in method signatures.

**Files to modify:**
- `app.go`

**Tasks:**

1. **Update method signatures:**
```go
// Change return type
func (a *App) ListMatchingProjectVersions(result providers.ModProject, mcVersion string, modLoader string) []providers.ModVersion {
    return providers.ListMatchingProjectVersions(result, mcVersion, modLoader)
}

// SearchMods can keep SearchModsUpdate (it uses SearchModResult internally)
// We'll update SearchModResult in next phase
```

2. **Update internal usage:**
   - Change `appstructs.SearchModResult` → `providers.ModProject` where appropriate
   - Change `appstructs.ProjectVersionResult` → `providers.ModVersion` where appropriate

**Verification:**
```bash
go build .
go test ./app_test.go
```

---

### Phase 7: Update Downloader ✅ High Risk

**Goal:** Update downloader to use new types.

**Files to modify:**
- `downloader/download.go`

**Tasks:**

1. **Update `downloadJob` struct:**
```go
type downloadJob struct {
    ID               string
    Version          providers.ModVersion        // was appstructs.ProjectVersionResult
    Result           providers.ModProject        // was appstructs.SearchModResult
    TargetDir        string
    InstanceID       string
    MinecraftVersion string
    ModLoader        string
    cancel           context.CancelFunc
}
```

2. **Update all functions using old types:**
   - `queueModDownload()` - Change parameter types
   - `downloadVersionForRequest()` - Return `ModVersion`
   - `downloadVersionsForRequest()` - Return `[]ModVersion`
   - `projectVersionSHA1Set()` - Accept `[]ModVersion`
   - `localModButtonStatus()` - Use new types internally
   - All helper functions

3. **Update struct conversion in queue state:**
```go
func downloadQueueItemFromJob(job downloadJob, status string, cancelable bool) appstructs.DownloadQueueItem {
    title := strings.TrimSpace(job.Result.Title)  // ModProject has Title
    // ...
}
```

**Verification:**
```bash
go build ./downloader
go test ./downloader -run TestQueueModDownload
go test ./downloader -run TestDownloadStates
```

---

### Phase 8: Remove Old Structs ✅ Breaking Change

**Goal:** Delete `structs/search.go` and old struct definitions.

**Files to modify:**
- `structs/search.go` - DELETE `SearchModResult`, `ProjectVersionResult`, `ProjectDependency`
- Keep other structs (SearchModsRequest, SearchModsUpdate, ModDownloadRequest, etc.)

**Files to modify:**
- Update remaining imports from `appstructs.SearchModResult` → `providers.ModProject`

**Tasks:**

1. **Remove old struct definitions:**
```go
// DELETE from structs/search.go:
// - SearchModResult
// - ProjectVersionResult  
// - ProjectDependency
```

2. **Update SearchModsUpdate:**
```go
type SearchModsUpdate struct {
    RequestID string              `json:"requestId"`
    Results   []providers.ModProject `json:"results"`  // was []SearchModResult
    Loading   bool                `json:"loading"`
    Append    bool                `json:"append"`
}
```

3. **Update ModDownloadRequest:**
```go
type ModDownloadRequest struct {
    ProjectID        string               `json:"projectId"`
    Result           providers.ModProject `json:"result"`  // was SearchModResult
    MinecraftVersion string               `json:"minecraftVersion"`
    ModLoader        string               `json:"modLoader"`
}
```

4. **Update DownloadStatesRequest:**
```go
type DownloadStatesRequest struct {
    Results          []providers.ModProject `json:"results"`  // was []SearchModResult
    MinecraftVersion string                 `json:"minecraftVersion"`
    ModLoader        string                 `json:"modLoader"`
}
```

**Verification:**
```bash
go build ./...
go test ./...
# Should have ZERO references to old struct names
grep -r "SearchModResult\|ProjectVersionResult" --include="*.go" .
```

---

### Phase 9: Update Frontend Types ✅ Low Risk

**Goal:** Regenerate TypeScript types from new Go structs.

**Files affected:**
- `frontend/wailsjs/go/models.ts` (auto-generated)

**Tasks:**

1. **Generate new TypeScript types:**
```bash
wails generate module
```

2. **Verify frontend still compiles:**
```bash
cd frontend
npm run build
```

3. **Check type compatibility:**
   - `ModProject` should have all fields `SearchModResult` had (except `icon`)
   - `ModVersion` should have all fields `ProjectVersionResult` had
   - `ModDependency` should have JSON aliases working

4. **Update frontend if needed:**
   - Remove hardcoded icon logic (should be in Vue component, not from backend)
   - Update any direct field access

**Verification:**
```bash
cd frontend
npm run type-check
npm run build
```

---

### Phase 10: Integration Testing ✅ Final Verification

**Goal:** End-to-end testing of the entire flow.

**Tasks:**

1. **Manual testing:**
   - Start the app: `wails dev`
   - Test search functionality
   - Test version list overlay
   - Test download functionality
   - Test pin version functionality
   - Test download queue

2. **Automated testing:**
```bash
go test ./...
cd frontend && npm run test
```

3. **Performance testing:**
   - Measure search response time before/after
   - Measure memory usage
   - Check cache hit rate

4. **Migration testing:**
   - Backup production cache
   - Run app with new code
   - Verify old cache migrates correctly
   - Test fallback behavior if migration fails

**Success Criteria:**
- [ ] All unit tests pass
- [ ] All integration tests pass
- [ ] Frontend builds without errors
- [ ] No runtime errors in manual testing
- [ ] Performance metrics within acceptable range
- [ ] Database migration succeeds

---

## Rollback Plan

If critical issues are discovered:

1. **Immediate:** Revert to previous commit
2. **Database:** Restore cache backup
3. **Frontend:** Regenerate old types: `wails generate module`
4. **Investigation:** Identify root cause before re-attempting

## Estimated Timeline

| Phase | Estimated Time | Risk Level |
|-------|---------------|------------|
| Phase 1: New Structs | 1 hour | Low |
| Phase 2: Database Layer | 2 hours | Medium |
| Phase 3: Lookup Functions | 1 hour | Low |
| Phase 4: Provider Interface | 3 hours | High |
| Phase 5: Service Layer | 2 hours | High |
| Phase 6: App Layer | 1 hour | Medium |
| Phase 7: Downloader | 2 hours | High |
| Phase 8: Remove Old Structs | 1 hour | Breaking |
| Phase 9: Frontend Types | 1 hour | Low |
| Phase 10: Integration Testing | 2 hours | Final |
| **Total** | **16 hours** | |

## Dependencies

- Go 1.21+
- Wails v2
- Node.js (for frontend build)
- All existing dependencies

## Notes

- Each phase should be committed separately for easy rollback
- Run tests after each phase before proceeding
- Keep old and new code side-by-side during transition (Phases 4-7)
- Only remove old code in Phase 8 after everything else works
