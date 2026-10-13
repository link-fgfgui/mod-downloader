# Language Preference

## Scenario: Persisted UI Language

### 1. Scope / Trigger

Use this contract when adding a locale, changing language startup behavior,
editing user-facing translation keys, or changing a Wails method that opens a
localized native dialog.

The feature crosses `core/configs`, `core/appcore`, `app.go`, generated Wails
bindings, Pinia, and vue-i18n. All layers must keep the same canonical values.

### 2. Signatures

```go
// core/configs
type Language string
const (
	LanguageSystem Language = "system"
	LanguageChinese Language = "zh"
	LanguageEnglish Language = "en"
)

type Preferences struct {
	Language Language `toml:"language" json:"language" env:"LANGUAGE" env-default:"system"`
}

// core/appcore
func (s *Service) GetPreferences() AppPreferences
func (s *Service) GetSettings() SettingsView
func (s *Service) SaveLanguage(language string) string

// Wails adapter
func (a *App) SaveLanguage(language string) string
func (a *App) ChooseMinecraftDir(locale string) string
func (a *App) ChooseCacheDir(locale string) SettingsView
func (a *App) ExportFavoriteListPackwizZip(listID, minecraftVersion, modLoader, locale string) ExportFavoritePackwizResult
```

```ts
type LanguagePreference = "system" | "zh" | "en";
type AppLocale = "zh" | "en";

function normalizeLanguagePreference(value: unknown): LanguagePreference;
function resolveLanguagePreference(preference: unknown, systemLocales?: readonly string[]): AppLocale;
function applyLanguagePreference(preference: unknown): AppLocale;
function currentLocale(): AppLocale;
```

### 3. Contracts

- TOML path: `[preferences] language = "system" | "zh" | "en"`.
- Environment override: `PREFERS_LANGUAGE`.
- Missing, empty, or invalid values normalize to `system` at service/frontend
  boundaries. Persisted output is canonical.
- `system` selects `zh` only when the primary browser locale begins with `zh`
  case-insensitively; every other primary locale selects `en`.
- `GetPreferences` and `GetSettings` both return the saved preference, not the
  resolved locale.
- Vue must apply the resolved locale before `app.mount()`.
- Settings saves the preference immediately, then applies it without restart.
- Native dialog methods receive the current resolved locale because Wails v2
  runtime environment metadata does not expose the OS locale.
- Chinese and English message trees must have identical key paths. English is
  the vue-i18n fallback locale.

### 4. Validation & Error Matrix

- Missing `language` -> saved preference reports `system`; browser locale is
  resolved on startup.
- `zh`, `zh-CN`, or `zh-Hans` input -> canonical `zh`.
- `en` input -> canonical `en`.
- Unknown config/service input -> `system`; no panic.
- `system` with primary `zh-TW` -> resolved `zh`.
- `system` with primary `en-US` and secondary `zh-CN` -> resolved `en`.
- Unknown native-dialog locale -> English dialog labels.
- `GetPreferences` unavailable during frontend bootstrap -> resolve `system`
  locally and continue mounting.

### 5. Good/Base/Bad Cases

- Good: save `system`, restart under `zh-TW`, mount directly in Chinese, and
  pass `zh` to `ChooseCacheDir`.
- Good: add the same new message key to both `messages.zh` and `messages.en`.
- Base: an older config without `language` starts with the system-derived locale
  and writes `language = "system"` on the next config save.
- Bad: store the resolved `zh`/`en` value when the user selected `system`; this
  prevents future startup from following OS changes.
- Bad: let each component inspect `navigator.language`; resolution must stay in
  the shared i18n helper.
- Bad: hard-code English native dialog titles in `app.go`.

### 6. Tests Required

- `core/configs`: parse canonical/compatible values, invalid normalization,
  default `system`, TOML decode, and `PREFERS_LANGUAGE`.
- `core/appcore`: `SaveLanguage` updates preferences/settings and survives
  `configs.Load()`.
- App adapter: Wails preferences/settings mapping and Chinese/English dialog
  text selection.
- Frontend: lint and type-check; verify `zh`/`en` message-key equality and
  primary-locale resolution cases.
- Regenerate Wails bindings whenever language fields or dialog method
  signatures change, then run the frontend production build.

### 7. Wrong vs Correct

Wrong:

```ts
const i18n = createI18n({ locale: "zh" });
const locale = navigator.languages.some((value) => value.startsWith("zh")) ? "zh" : "en";
```

This hard-codes the initial UI and incorrectly lets a secondary locale override
the primary system language.

Correct:

```ts
const preference = await GetPreferences();
applyLanguagePreference(preference.language);
app.mount("#app");
```

`resolveLanguagePreference` owns primary-locale detection, and the persisted
preference remains `system` when that is what the user selected.
