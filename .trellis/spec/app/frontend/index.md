# App Frontend Guidelines

> Package-scoped frontend guidelines for the Wails Vue application in `frontend/`.

The app frontend owns Vue views, Pinia stores, Vuetify components, and Wails
frontend bindings. Backend contracts and reusable domain behavior belong in the
app adapter or `core/` specs.

## Required References

- [UI State Lifecycle](./ui-state-lifecycle.md)
- [Shared Thinking Guides](../../guides/index.md)

## Pre-Development Checklist

- Read [UI State Lifecycle](./ui-state-lifecycle.md) before changing dialogs,
  overlays, transitions, action bars, or other leaving UI surfaces.
- Keep UI-only state in Vue components or Pinia stores; do not change Wails
  backend contracts for visual-only behavior.
- Before adding a shared composable, search `frontend/src/composables/` and
  existing components for an equivalent pattern.

## Quality Check

- Run `npm run build` from `frontend/` after frontend TypeScript or template
  changes.
- Run `npm run lint` from `frontend/` after frontend changes.
- For animation/lifecycle fixes, verify both open/reopen behavior and the
  closing/leave lifecycle so stale snapshot data cannot leak into the next open.
