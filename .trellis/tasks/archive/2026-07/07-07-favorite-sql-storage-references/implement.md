# Favorite SQL storage and references implementation

## Steps

1. Load backend and database specs.
2. Add new database structs and normalization helpers.
3. Implement schema migration version 2 with idempotent column/table/index creation.
4. Add group CRUD/reorder functions and tests.
5. Add list metadata update/reorder functions and tests.
6. Add reference CRUD with recursive CTE cycle detection and tests.
7. Add SQL-backed list content resolution with dedupe and tests.
8. Run `cd core && go test ./database`.

## Validation

- `cd core && go test ./database`
- `cd core && go test ./...`

## Notes

- Keep all new public database APIs additive.
- Existing tests in `core/database/mods_test.go` must continue to pass without changing their intent.
