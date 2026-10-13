# Implementation Plan

## Checklist

- [x] Load frontend/app specs with `trellis-before-dev` before editing.
- [x] Inspect Vuetify dialog/overlay lifecycle support in this codebase and choose the least invasive snapshot hook per component.
- [x] Implement stable leave snapshots for `VirtualList.vue` floating action bar.
- [x] Implement stable leave snapshots for `App.vue` download queue FAB/menu content.
- [x] Implement stable close snapshots for `Download.vue` replacement confirmation dialog.
- [x] Implement stable close snapshots for `Favorites.vue` delete dialog.
- [x] Implement stable close snapshots for `Manage.vue` delete dialog.
- [x] Search for other transition/dialog surfaces with the same pattern and either fix them or record why they are unaffected.
- [x] Run frontend validation: `npm run build` from `frontend/`.
- [x] Run a focused manual or code-level verification of the affected leave lifecycles.

## Verification Results

- `npm run build` from `frontend/` passed.
- `npm run lint` from `frontend/` passed.
- Additional transition/dialog audit found no other surfaces that both animate out and clear visible variable text during close. Static dialogs/overlays and dialogs whose payload is not cleared on close were left unchanged.

## Risk Areas

- Snapshot state can become stale if it is not cleared on after-leave or overwritten on reopen.
- `VirtualList.vue` selection snapshots must not affect actual selection behavior, keyboard handling, or emitted selected indices.
- Download queue snapshot must not keep the FAB mounted forever after the queue is inactive.

## Rollback Point

All changes are frontend-only. If snapshot abstraction proves too broad, revert to local snapshot refs in the affected components while preserving the PRD acceptance criteria.
