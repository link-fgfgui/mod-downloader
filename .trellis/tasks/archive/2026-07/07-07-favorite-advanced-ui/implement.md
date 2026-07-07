# Favorite advanced UI implementation

## Steps

1. Start after backend Wails bindings are regenerated.
2. Update generated type imports and favorites store actions.
3. Update Favorites page rail for groups, pinning, icons, and ordering.
4. Add dialogs for bulk copy/reference, group/icon editing, and migration.
5. Preserve Download/Manage add-to-favorite callers.
6. Run frontend build and manually inspect desktop/mobile layout through dev server.

## Validation

- `npm run build --prefix frontend`
- `go test ./...`
- Manual Favorites page smoke test in local dev server.
