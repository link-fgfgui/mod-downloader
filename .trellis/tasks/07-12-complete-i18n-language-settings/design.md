# Design: Complete i18n and language settings

## Boundaries

The persisted preference belongs to `core/configs.Preferences`. `appcore`
normalizes and saves it, `app.go` exposes it through Wails, and the frontend
owns system-locale detection plus vue-i18n activation.

## Contract

- Config value: `system | zh | en`.
- Wails fields: `language` on `AppPreferences` and `SettingsView`.
- Wails command: `SaveLanguage(language string) string`, returning the canonical
  saved preference.
- Frontend resolved locale: `zh | en`.

`configs.Language` follows the existing `Theme` pattern: text marshal/unmarshal,
parse, normalize, and a default of `system`. This keeps TOML, environment, JSON,
service, and UI validation in one owner.

## Startup Data Flow

1. Vue bootstrap asks `GetPreferences()` before mounting.
2. The frontend normalizes the saved preference.
3. `system` is resolved from `navigator.languages` / `navigator.language`;
   `zh-*` selects `zh`, otherwise `en`.
4. The resolved locale is assigned to vue-i18n, then the app mounts.

When the user changes the Settings selector, the store calls `SaveLanguage`,
updates its snapshot, and applies the resolved locale immediately.

## Native Dialogs

Wails does not expose the OS locale. Frontend callers therefore pass the current
resolved locale to native dialog methods. The Go adapter selects a small set of
Chinese/English dialog strings and defaults unknown locale values to English.
This does not place translation logic in `core` and keeps native runtime code in
`app.go`.

## Message Coverage

The existing single catalog remains the source of truth. New keys cover the
language selector, search controls, sidebar target controls, version overlay
fallbacks, and internal frontend fallback messages. Product names and domain
terms such as Mod Downloader, Minecraft, Fabric, Forge, Modrinth, and
CurseForge remain unchanged.

## Compatibility And Rollback

Existing config files omit `language`, which normalizes to `system`; no migration
rewrite is required until the next settings save. Reverting the feature leaves
an unknown TOML key for older binaries, so rollback should also remove the key
from the config file if an older cleanenv version rejects unknown fields.
