# Complete i18n and language settings

## Goal

Complete the application's Chinese/English localization coverage and let users
choose the UI language from both `mod-downloader.toml` and the Settings page.
New and existing installations default to the operating system language.

## Background

- `frontend/src/plugins/i18n.ts:619` currently hard-codes Chinese as the startup
  locale.
- `core/configs/structs.go:104` has no persisted language preference.
- `frontend/src/views/Download.vue:16`,
  `frontend/src/components/SideBar/VersionChoose.vue:10`, and
  `frontend/src/components/MinecraftTargetFields.vue:31` still contain visible
  English literals outside the translation catalog.
- Native folder/export dialogs in `app.go:242`, `app.go:463`, and `app.go:475`
  also use fixed English labels.
- Existing uncommitted edits to the animation label and Settings button sizing
  must be preserved.

## Requirements

- R1. Add a persisted language preference with exactly three canonical values:
  `system`, `zh`, and `en`.
- R2. Store the preference under `[preferences] language` in
  `mod-downloader.toml`; support `PREFERS_LANGUAGE` through the existing config
  environment binding.
- R3. Treat a missing or empty language preference as `system`. Invalid values
  must normalize safely to `system` at service/UI boundaries.
- R4. Resolve `system` to Chinese for browser/system locales beginning with
  `zh` (case-insensitive), and to English for all other locales.
- R5. Apply the resolved language before the first Vue render so startup does
  not flash the wrong locale.
- R6. Add a Settings language control with System, Chinese, and English options.
  Changing it saves immediately, updates the entire UI immediately, and shows a
  localized success message.
- R7. Expose the language preference through the existing preferences/settings
  Wails contracts and add a save operation that persists it through `appcore`.
- R8. Replace remaining user-facing hard-coded English in Vue templates and
  frontend fallbacks with catalog entries where the text is UI copy rather than
  a product name, platform name, file name, or backend-provided error.
- R9. Localize native folder/export dialog titles and file-filter labels using
  the currently resolved UI locale supplied by the frontend.
- R10. Keep Chinese and English message trees structurally aligned and retain
  English as the vue-i18n fallback locale.

## Acceptance Criteria

- [x] With no `language` key, a `zh-*` system locale renders Chinese and a
  non-Chinese system locale renders English.
- [x] `[preferences] language = "zh"` and `"en"` override the system locale;
  `"system"` restores system-language behavior.
- [x] The Settings page displays the saved choice and switches all visible UI
  text immediately after selection without restarting the app.
- [x] Saving a language choice survives config reload and appears in both
  `GetPreferences` and `GetSettings`.
- [x] Download search, version/folder controls, target labels, version overlay
  fallback, frontend download-error fallbacks, and native dialog labels no
  longer remain fixed English when Chinese is active.
- [x] Config parsing/normalization and service persistence have focused Go tests.
- [x] `npm run lint`, `npm run build`, app Go tests, and core Go tests pass.

## Out Of Scope

- Adding languages other than Simplified Chinese and English.
- Translating provider/mod metadata, filenames, technical identifiers, platform
  names, or error text returned directly by external services.
- Changing the operating system language at runtime; selecting `system` is
  re-resolved when the application starts.

## Notes

- Existing changes in `frontend/src/plugins/i18n.ts` and
  `frontend/src/views/Settings.vue` are user-owned and must not be reverted.
