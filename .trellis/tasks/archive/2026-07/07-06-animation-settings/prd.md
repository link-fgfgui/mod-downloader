# Configure animation behavior

## Goal

Allow users to control app animation behavior from both `mod-downloader.toml` and the Settings page.

The feature should preserve current behavior by default: animations remain enabled and use the current timing unless the user changes the new preference.

## Confirmed Facts

- Configuration is owned by `core/configs` and persisted to `mod-downloader.toml` via `configs.Save`.
- User preferences currently live in `configs.Preferences` and include `theme` and `minecraft_dir`.
- Core settings snapshots are exposed through `appcore.GetPreferences`, `appcore.GetSettings`, and Wails adapter types in `app.go`.
- The Settings page already uses a Pinia store, Wails settings APIs, and i18n strings in `frontend/src/views/Settings.vue`, `frontend/src/stores/settings.ts`, and `frontend/src/plugins/i18n.ts`.
- Animation timing is centralized in `frontend/src/styles/animations.css` with CSS variables such as `--md-transition-fast`, `--md-transition-normal`, and `--md-transition-slow`.
- The app already respects system reduced-motion preferences through CSS.

## Requirements

- Add a persisted animation enabled flag under preferences.
- Add a persisted animation duration multiplier under preferences.
- Defaults:
  - animations enabled: `true`
  - duration multiplier: `1.0`
- Use `animation_duration_multiplier` semantics: lower values make animations shorter/faster; higher values make animations longer/slower.
- Clamp or normalize invalid duration multipliers so missing, zero, negative, NaN, or out-of-range values do not break the UI.
- Expose animation settings through app startup preferences and Settings page snapshots.
- Add a Settings page control to enable/disable animations.
- Add a Settings page control to adjust duration multiplier.
- Apply changes immediately after saving from the Settings page.
- Keep the existing system `prefers-reduced-motion` behavior.
- Do not send API key plaintexts to the frontend while adding settings fields.
- Regenerate or update Wails frontend bindings when public Wails API/types change.

## Acceptance Criteria

- [ ] `mod-downloader.toml` can contain preference fields for animation enabled/disabled and duration multiplier.
- [ ] Loading a missing animation config preserves current behavior: animations enabled at `1.0x`.
- [ ] Invalid multiplier values normalize to a safe value and are persisted as a safe value after saving.
- [ ] `GetPreferences` and `GetSettings` include normalized animation settings.
- [ ] Settings page shows current animation enabled state and multiplier.
- [ ] Saving from the Settings page persists both animation settings and applies them without requiring restart.
- [ ] Disabling animations makes app animations/transitions effectively instant while keeping the UI usable.
- [ ] Duration multiplier changes affect the existing CSS animation tokens.
- [ ] Go tests cover config normalization and Wails/core settings save behavior.
- [ ] Frontend type-check/build passes with the updated bindings.

## Out of Scope

- Per-animation-category controls.
- Runtime animation presets beyond the boolean and numeric multiplier.
- Changing theme behavior or API key behavior.
