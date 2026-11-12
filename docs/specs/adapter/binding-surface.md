# App Binding Surface

## Decision

`App` exposes only business methods the frontend actually consumes.
`app_contract_test.go` locks the current **68** business methods and their full
signature hash. Adding or removing a method, or changing a parameter/result
type, requires updating the contract test and regenerating
`frontend/bindings/`.

## Error-returning bindings

The App layer uses the core `...WithError` variants so a failed database or
provider operation is not silently converted into an empty result. Legacy core
methods remain for the TUI; see
`go doc github.com/link-fgfgui/mod-downloader/core/appcore` for the public Go
contract and [`../frontend/error-handling.md`](../frontend/error-handling.md)
for the frontend display rules.

- Favorites: `CreateFavoriteListForScope`, `RenameFavoriteList`,
  `DeleteFavoriteList`, `UpdateFavoriteListMetadata`, `ReorderFavoriteLists`,
  `AddFavoriteMod`, `AddFavoriteListReference`, `RemoveFavoriteMod`.
- Pins: `GetPinnedModVersion`, `PinModVersion`, `UnpinMod`.
- Lookup: `LookupProjectBySlug`.
- Settings: `SaveTheme` and `SaveLanguage` return the persistence error
  directly; the view-returning `Save*` bindings carry `SettingsView.SaveError`.

`ChooseCacheDir` returns a `CacheDirSelectionResult` with explicit `canceled`
and `error` fields, plus the latest settings snapshot. A cancel is not an error,
and a failed cache switch must not look like a cancel.

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

## Renamed bindings

The old names collided with unscoped core methods and made the selected-instance
scope invisible at the binding layer:

| Old | New |
| --- | --- |
| `CreateFavoriteList(name)` | `CreateFavoriteListForScope(name, minecraftVersion, modLoader)` |
| `ExportFavoriteListPackwizZip(listID, mcVersion, modLoader, locale)` | `ExportFavoriteListPackwizZipForScope(listID, mcVersion, modLoader, locale)` |

## Constraints

**Don't**

- Do not add core service methods that the frontend does not yet use just
  because they exist: every binding method expands the contract test and
  generated-code maintenance surface.
- Do not hand-write parallel TypeScript interfaces; consume only the generated
  types in `frontend/bindings/`.
- Do not update only `expectedAppMethods`; run
  `wails3 generate bindings -f '-tags server,production' -clean=true -ts -i`
  and refresh the signature hash in the same change.
- Do not panic from a binding when a typed result can carry the failure.
