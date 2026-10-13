# Favorite version migration design

## Request/Response

```go
type FavoriteMigrationRequest struct {
    SourceListID      string `json:"sourceListId"`
    TargetListID      string `json:"targetListId"`
    MinecraftVersion  string `json:"minecraftVersion"`
    ModLoader         string `json:"modLoader"`
    IgnoreConflicts   bool   `json:"ignoreConflicts,omitempty"`
}

type FavoriteMigrationPreview struct {
    SourceListID string `json:"sourceListId"`
    TargetListID string `json:"targetListId"`
    Matched      []FavoriteMigrationMatch `json:"matched"`
    Conflicts    []FavoriteMigrationConflict `json:"conflicts"`
}
```

## Resolution Strategy

For each resolved favorite mod:

1. Build project lookup candidates from `platform`, `modId`, and `slug`.
2. Prefer `LookupProjectByID`; fall back to `LookupProjectBySlug`.
3. Call `ListMatchingProjectVersions(project, targetMinecraftVersion, targetModLoader)`.
4. Pick the first sorted matching version, matching existing provider sort semantics.
5. Produce a target `database.FavoriteMod` preserving display metadata but replacing scope and `versionId`.

## Writes

- Preview never writes.
- Apply re-runs preview to avoid stale writes.
- If conflicts exist and `IgnoreConflicts=false`, return the preview plus `Applied=false`.
- If ignoring conflicts, upsert only matched rows.

## Provider Risk

Provider lookups may use network calls. The service should return conflict rows with reasons rather than panic. Frontend will show loading and failure states in the UI child task.
