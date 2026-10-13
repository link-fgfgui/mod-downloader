# Configure animation behavior design

## Architecture

The implementation follows the existing settings ownership:

`mod-downloader.toml` -> `core/configs.Preferences` -> `appcore` normalized view/save methods -> `app.go` Wails adapter -> generated frontend bindings -> Pinia settings store -> Settings page and root animation application.

## Config Contract

Add two preference fields:

- `animations_enabled` as an optional boolean in TOML-backed config. A missing value means enabled.
- `animation_duration_multiplier` as a float. Missing, zero, negative, NaN, or otherwise invalid values normalize to `1.0`.

Use config-owned helpers/constants so appcore, tests, and future callers do not duplicate default/range logic.

Recommended range for UI and normalization:

- minimum `0.25`
- maximum `3.0`
- default `1.0`
- UI step `0.25`

This keeps the control useful without allowing extreme values that make route transitions appear broken.

## Backend Contract

Extend `appcore.AppPreferences` and `appcore.SettingsView` with:

- `AnimationEnabled bool`
- `AnimationDurationMultiplier float64`

Add a save request/method for animation settings. The method normalizes input, persists via `configs.Save`, and returns normalized settings. The Wails adapter mirrors the request and view types without Wails dependencies in `core/`.

`GetPreferences` is used on app mount, so it must expose the same normalized values as `GetSettings`.

## Frontend Contract

Create a small animation settings helper in the frontend layer. It owns:

- normalization for defensive frontend use
- setting root CSS variables for `--md-transition-fast`, `--md-transition-normal`, `--md-transition-slow`, and `--md-stagger-delay`
- setting a root data attribute for animation enabled/disabled

Settings store owns drafts and saving. Settings page adds a switch and slider/input control in the existing card layout.

## CSS Contract

Keep existing animation utility classes. Add root-level disabled animation handling similar to the current `prefers-reduced-motion` block, including delay removal so staggered content does not remain delayed.

Multiplier affects existing CSS variables rather than rewriting each animation class.

## Compatibility

Existing config files without the new fields keep current behavior. Saved config files include normalized values after the user saves animation settings or config is otherwise persisted.

Generated Wails bindings must include new API/type fields so TypeScript uses real types instead of local casts.
