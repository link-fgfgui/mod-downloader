# App Frontend Guidelines

> Package-scoped frontend guidelines for the Wails Vue application in `frontend/`.

The app frontend owns Vue views, Pinia stores, Vuetify components, and Wails
frontend bindings. Backend contracts and reusable domain behavior belong in the
app adapter or `core/` specs.

## Required References

- [UI State Lifecycle](./ui-state-lifecycle.md)
- [Animation Modes](./animation-modes.md)
- [Local Mod Version Selection](./local-mod-version-selection.md)
- [Download Completion Sound](./download-completion-sound.md)
- [Download Result Snapshots](./download-result-snapshots.md)
- [Download Queue Actions](./download-queue-actions.md)
- [Language Preference](./language-preference.md)
- [Shared Thinking Guides](../../guides/index.md)

## Pre-Development Checklist

- Read [UI State Lifecycle](./ui-state-lifecycle.md) before changing dialogs,
  overlays, transitions, action bars, or other leaving UI surfaces.
- Read [Animation Modes](./animation-modes.md) before changing `animations.css`,
  `useAnimationSettings.ts`, the settings animation flow, or route/FAB
  transitions. Keep one application entry, non-overlapping mode ownership, and
  `animation ... both` as the only source of an entrance's hidden state.
- Keep UI-only state in Vue components or Pinia stores; do not change Wails
  backend contracts for visual-only behavior.
- Before adding a shared composable, search `frontend/src/composables/` and
  existing components for an equivalent pattern.
- Reuse `ModVersionList` for provider version choices and keep local parsed
  versions separate from online metadata; see
  [Local Mod Version Selection](./local-mod-version-selection.md).
- Preserve the success-event plus queue-drain plus unfocused gate for audible
  download notifications; see [Download Completion Sound](./download-completion-sound.md).
- Keep cancel, retry, and canceled-history removal semantics aligned across
  Vue, Wails, appcore, and downloader; see [Download Queue Actions](./download-queue-actions.md).
- Keep persisted language values, startup resolution, Settings behavior, and
  localized native-dialog locale arguments aligned; see
  [Language Preference](./language-preference.md).

## Quality Check

- Run `npm run build` from `frontend/` after frontend TypeScript or template
  changes.
- Run `npm run lint` from `frontend/` after frontend changes.
- For animation/lifecycle fixes, verify both open/reopen behavior and the
  closing/leave lifecycle so stale snapshot data cannot leak into the next open.
