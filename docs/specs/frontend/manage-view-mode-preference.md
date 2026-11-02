# Manage View Preference (List / Tree)

## Decision

The List / dependency-tree toggle in the top-right of the Manage page is a **GUI
display preference**. It is not core business state and is not shared with the
TUI. It is persisted in `localStorage`, surviving route unmounts and reloads.

- Storage key: `mod-downloader.manage.view-mode`
- Allowed values: `"list"` / `"tree"`
- Default: `"list"`
- Implementation: `frontend/src/hooks/useManageViewMode.ts`
- Consumer: `frontend/src/pages/Manage.tsx`

## Constraints

**Don't**

- Do not put it into `useSettingsStore` backend fields or `core/structs`: it does
  not write back to core and does not require GUI/TUI parity.
- Do not keep it only in `ManagePage`'s `useState`: route changes unmount the page
  and reset it.
- Do not throw or block the page on an unknown stored value; fall back to `list`.

**Instead**

- Read `localStorage` during initialization; if writing fails (private mode,
  storage disabled), keep the in-memory state.
- Write to storage immediately after a view change so switching routes and
  returning still works.
- Unit tests cover the default, persistence across mounts, and invalid-value
  fallback: `frontend/src/hooks/useManageViewMode.test.ts`.
