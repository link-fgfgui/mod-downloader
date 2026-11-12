# Frontend Specifications Index

This directory registers GUI-specific (React) display, interaction, and
animation contracts. Business rules that still hold after replacing the UI
framework belong in `core/docs/specs/` and in `core/`, not here.

| Topic | File | Summary |
| --- | --- | --- |
| Radix dialog exit animation | [`radix-dialog-exit-animation.md`](./radix-dialog-exit-animation.md) | Keep the final exit frame to avoid a full-dialog flash at close |
| Manage view preference | [`manage-view-mode-preference.md`](./manage-view-mode-preference.md) | List/Tree is a GUI-local preference persisted in localStorage across routes and reloads |
| Render error boundary | [`error-boundary.md`](./error-boundary.md) | Root boundary keeps a localized reload screen instead of a blank WebView after an uncaught render/lifecycle error |
| Error handling and empty results | [`error-handling.md`](./error-handling.md) | Keep "empty" and "failed" distinct: missing pins, idempotent deletes, partial search results |
