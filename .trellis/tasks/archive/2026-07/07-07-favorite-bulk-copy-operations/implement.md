# Favorite bulk copy operations implementation

## Steps

1. Start only after SQL storage/reference child is complete.
2. Add appcore request/result types.
3. Add database helper return values for inserted/updated/skipped where useful.
4. Implement selected-mod bulk add service method.
5. Implement whole-list concrete copy service method.
6. Implement reference service methods.
7. Add Wails adapter methods and regenerate bindings.
8. Add database/appcore tests.

## Validation

- `cd core && go test ./database ./appcore`
- `go test ./...`
- `wails generate module`
- `npm run build --prefix frontend`
