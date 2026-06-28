# Design: Abstract Platform-Agnostic Unified Provider Structs

## Architecture Overview

### Current State

```
┌─────────────────┐
│  CF/MR SDK      │
└────────┬────────┘
         │ provider-specific conversion
         ▼
┌─────────────────┐     ┌──────────────────┐
│ SearchModResult │ ◄──►│ ModPlatform      │  (overlap)
│ ProjectVersion  │     │ ModPlatformVer   │
└────────┬────────┘     └────────┬─────────┘
         │                       │
         ▼                       ▼
    Frontend DTO           Database Storage
```

**Problems:**
- Duplicate structs: `SearchModResult` vs `ModPlatform`, `ProjectVersionResult` vs `ModPlatformVersion`
- Provider interface returns full objects
- Conversion logic duplicated in cf/mr providers

### Target State

```
┌─────────────────┐
│  CF/MR SDK      │
└────────┬────────┘
         │ SDK → Unified struct
         ▼
┌─────────────────────────────────┐
│  Unified Model (single source)  │
│  - ModProject                    │
│  - ModVersion                    │
│  - ModDependency                 │
└─────────┬───────────────────────┘
          │
          ├─► Frontend (JSON)
          └─► Database (gob)
```

**Benefits:**
- Single data model for all layers
- Provider returns IDs, data fetched on-demand
- No conversion between DTO/storage

## Data Model Design

### Unified Structs

Replace both `structs/search.go` and `database/mods.go` structs with:

```go
// providers/model.go (new file)

// ModProject represents a mod project on any platform (replaces SearchModResult + ModPlatform)
type ModProject struct {
    ID          string `json:"id"`           // "platform:projectID" or just projectID
    Platform    string `json:"platform"`     // "CurseForge" | "Modrinth"
    ProjectID   string `json:"projectId"`    // Numeric ID for CF, slug for MR
    Slug        string `json:"slug"`         // URL slug
    Title       string `json:"title"`        // Display name
    Description string `json:"description"`  // Short description
    IconURL     string `json:"iconUrl"`      // Avatar/logo URL
    Downloads   int64  `json:"downloads"`    // Total download count
    UpdatedAt   int64  `json:"updatedAt"`    // Last fetched timestamp
}

// ModVersion represents a specific version file (replaces ProjectVersionResult + ModPlatformVersion)
type ModVersion struct {
    ID           string           `json:"id"`           // Platform-specific version ID
    Platform     string           `json:"platform"`     // Same as parent project
    ProjectID    string           `json:"projectId"`    // Parent project ID
    VersionID    string           `json:"versionId"`    // Same as ID (for compatibility)
    Name         string           `json:"name"`         // Display name
    VersionNum   string           `json:"version"`      // Version number string
    FileName     string           `json:"fileName"`     // JAR filename
    DownloadURL  string           `json:"downloadUrl"`  // Direct download link
    SHA1         string           `json:"sha1"`         // File hash
    PublishedAt  int64            `json:"publishedAt"`  // Unix timestamp
    Downloads    int64            `json:"downloads"`    // Version-specific downloads
    GameVersions []string         `json:"gameVersions"` // e.g. ["1.20.1", "1.20.2"]
    Loaders      []string         `json:"loaders"`      // e.g. ["fabric", "forge"]
    Dependencies []ModDependency  `json:"dependencies,omitempty"`
}

// ModDependency represents a dependency link (replaces ProjectDependency + ModDependency)
type ModDependency struct {
    ID                  string `json:"id"`                            // Internal ID
    PlatformVersionID   string `json:"platformVersionId,omitempty"`   // Parent version
    DependencyProjectID string `json:"dependencyProjectId"`           // Target project
    DependencyVersionID string `json:"dependencyVersionId,omitempty"` // Target version (optional)
    DependencyType      string `json:"dependencyType,omitempty"`      // "required" | "optional" | etc
    
    // Alias for frontend compatibility
    ProjectID string `json:"projectId,omitempty"` // Alias for DependencyProjectID
    VersionID string `json:"versionId,omitempty"` // Alias for DependencyVersionID
    Type      string `json:"type,omitempty"`      // Alias for DependencyType
}
```

### Frontend Compatibility

Frontend currently expects these fields (from `frontend/wailsjs/go/models.ts`):

**SearchModResult → ModProject mapping:**
- `id`, `platform`, `title`, `icon`, `iconUrl`, `description`, `downloads`, `slug` ✅ Direct match
- `icon`: Remove (was hardcoded "mdi-package-variant" or "mdi-leaf", should be frontend's responsibility)

**ProjectVersionResult → ModVersion mapping:**
- All fields match except:
  - `version` → `versionNum` (rename in Go, keep JSON tag as `version`)
  - `versionId` → Add as alias to `id`

**ProjectDependency → ModDependency mapping:**
- Add JSON aliases: `projectId`→`dependencyProjectId`, `versionId`→`dependencyVersionID`, `type`→`dependencyType`

### Storage Integration

**database/mods.go changes:**
1. Remove `ModPlatform`, `ModPlatformVersion`, `ModDependency` structs
2. Import `providers.ModProject`, `providers.ModVersion`, `providers.ModDependency`
3. Keep cache keys (`platformKey`, `versionKey`) unchanged
4. Add migration: read old gob → convert to new structs → write back

**Cache structure:**
```go
type cacheState struct {
    ModProjects      map[platformKey]ModProject      // was ModPlatforms
    ModVersions      map[versionKey]ModVersion       // was PlatformVersions
    ModVersionKeyByID map[string]versionKey          // unchanged
    PlatformAssociations map[string]PlatformAssociation // unchanged
    PinnedMods       map[pinnedModKey]PinnedMod      // unchanged
    JarMetadata      map[string][]structs.ModInfo    // unchanged
    PlatformVersionScopes map[versionScopeKey]storedVersionScope // unchanged
}
```

## Provider Interface Design

### New Interface

```go
// providers/modprovider.go

type modProvider interface {
    Name() string
    
    // Search returns project IDs only, stores full data in cache
    Search(req SearchRequest, store func(ModProject)) (projectIDs []string, err error)
    
    // ExactSearch returns single project ID if found
    ExactSearch(req SearchRequest, store func(ModProject)) (projectID string, err error)
    
    // ListVersions returns version IDs only, stores full data in cache
    ListVersions(projectIDOrSlug string, filter VersionFilter, store func(ModVersion)) (versionIDs []string, err error)
}

type SearchRequest struct {
    Query            string
    MinecraftVersion string
    ModLoader        string
    Offset           int
    Limit            int
}

type VersionFilter struct {
    MinecraftVersion string
    ModLoader        string
}
```

### Provider Implementation Pattern

Each provider:
1. Calls SDK
2. Converts SDK type → `ModProject` or `ModVersion`
3. Calls `store()` callback to save to cache
4. Returns ID list only

**Example (CurseForge):**
```go
func (p curseForgeProvider) Search(req SearchRequest, store func(ModProject)) ([]string, error) {
    response, err := client.SearchMod(...)
    if err != nil {
        return nil, err
    }
    
    ids := make([]string, 0, len(response.Data))
    for _, mod := range response.Data {
        project := p.sdkToProject(mod)  // Convert
        store(project)                   // Store
        ids = append(ids, project.ID)    // Collect ID
    }
    return ids, nil
}
```

### Service Layer Changes

**providers/service.go:**
- `SearchMods()`: Get IDs from providers, fetch `ModProject` by ID, emit to frontend
- `ListMatchingProjectVersions()`: Get version IDs, fetch `ModVersion` by ID, return list
- Add: `GetProjectByID(id string) (ModProject, bool)`
- Add: `GetVersionByID(id string) (ModVersion, bool)`

## Migration Strategy

### Phase 1: Add New Structs (backward compatible)
1. Create `providers/model.go` with new structs
2. Keep old structs in place
3. Add conversion helpers: `oldToNew()`, `newToOld()`

### Phase 2: Update Database Layer
1. Add support for new structs in `database/mods.go`
2. Add migration: detect old format → convert → save new format
3. Update all `Get`/`Set` methods to use new types

### Phase 3: Update Provider Interface
1. Add new `modProviderV2` interface alongside old
2. Implement new interface in cf/mr providers
3. Update `modProviders` slice to use V2

### Phase 4: Update Callers
1. `providers/service.go`: Use new interface, return new types
2. `app.go`: Update method signatures
3. `downloader/download.go`: Use new types
4. Run `wails generate module` to update frontend types

### Phase 5: Cleanup
1. Remove old structs from `structs/search.go`
2. Remove old provider interface
3. Remove conversion helpers

## Testing Strategy

1. **Unit tests**: Each provider's SDK→model conversion
2. **Integration tests**: Search + ListVersions roundtrip
3. **Migration test**: Old gob file → new format
4. **Frontend test**: Verify `wails generate` output matches expectations

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Frontend breaks due to type mismatch | High | Use JSON aliases for compatibility; test with `wails generate` |
| Database migration fails | High | Backup old cache, detect version, fallback to re-fetch |
| Performance regression from extra lookups | Medium | Measure before/after; cache hit rate should be high |
| Breaking change in public API | Medium | Phase migration; keep old types during transition |

## Open Questions

1. **Icon field**: Remove from model (frontend decides based on platform)? → YES
2. **ID format**: Keep "platform:projectID" or split? → Keep composite for uniqueness
3. **Null fields**: Use pointers vs zero values? → Zero values (simpler JSON)
4. **Version ID duplication**: `ID` vs `VersionID` field? → Keep both for compatibility

## Success Metrics

- [ ] All existing tests pass
- [ ] `wails generate module` produces valid TypeScript
- [ ] No duplicate struct definitions across packages
- [ ] Provider interface methods < 3 lines (just ID passing)
