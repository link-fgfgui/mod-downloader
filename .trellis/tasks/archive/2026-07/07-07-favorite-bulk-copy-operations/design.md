# Favorite bulk copy operations design

## API Shape

Prefer request/response structs over long positional argument lists for new APIs:

```go
type FavoriteBulkAddRequest struct {
    TargetListIDs []string `json:"targetListIds"`
    Mods          []database.FavoriteMod `json:"mods"`
}

type FavoriteListCopyRequest struct {
    SourceListID string `json:"sourceListId"`
    TargetListID string `json:"targetListId"`
}

type FavoriteBulkOperationResult struct {
    Added   int      `json:"added"`
    Updated int      `json:"updated"`
    Skipped int      `json:"skipped"`
    Errors  []string `json:"errors,omitempty"`
}
```

Reference APIs can use direct IDs:

- `AddFavoriteListReference(parentListID, childListID string) database.FavoriteListRef`
- `RemoveFavoriteListReference(parentListID, childListID string) bool`

## Behavior

- Bulk add validates every target list once.
- Invalid target lists are skipped and reported.
- Source and target can be the same only for idempotent selected-mod add; whole-list copy to self should be skipped.
- Whole-list copy reads resolved source contents from database so referenced mods can be copied concretely.
- Reference add delegates cycle checks to database.

## Boundaries

- `core/database` owns persistence and duplicate detection.
- `core/appcore` owns request normalization and loggable failures.
- `app.go` only delegates to `service()`.
- Frontend bindings are regenerated after Wails method additions.
