# Frontend Specifications Index

This directory registers GUI-specific (React) display, interaction, and
animation contracts. Business rules that still hold after replacing the UI
framework belong in `docs/specs/core/` and in `core/`, not here.

| Topic | File | Summary |
| --- | --- | --- |
| Radix dialog exit animation | [`radix-dialog-exit-animation.md`](./radix-dialog-exit-animation.md) | Keep the final exit frame to avoid a full-dialog flash at close |
| Manage view preference | [`manage-view-mode-preference.md`](./manage-view-mode-preference.md) | List/Tree is a GUI-local preference persisted in localStorage across routes and reloads |
