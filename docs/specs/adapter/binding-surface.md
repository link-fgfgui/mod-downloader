# App Binding Surface

## Decision

`App` exposes only business methods the frontend actually consumes.
`app_contract_test.go` locks the current **68** business methods. Adding or
removing a method requires updating the contract test and regenerating
`frontend/bindings/`.

## Removed dead bindings

These methods had zero frontend calls (the TUI calls `appcore.Service` directly),
so they were removed from `App`:

- `GetAppVersion` (version comes from the window/config; the frontend does not
  consume it)
- `GetPreferences` (the frontend uses `GetSettings`)
- `ListFavoriteMods` / `ListFavoriteContents` (the frontend uses
  `ListFavoriteContentsForScope`)
- `RemoveFavoriteListReference` / `ListFavoriteListRefs` (the interaction is not
  exposed in the frontend)

## Constraints

**Don't**

- Do not add core service methods that the frontend does not yet use just
  because they exist: every binding method expands the contract test and
  generated-code maintenance surface.
- Do not hand-write parallel TypeScript interfaces; consume only the generated
  types in `frontend/bindings/`.
- After removing an `App` method, run `wails3 generate bindings -clean=true -ts`
  and update the method count and list in `app_contract_test.go`.
