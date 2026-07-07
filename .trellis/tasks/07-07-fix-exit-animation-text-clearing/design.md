# Design

## Approach

Use leave-time content snapshots for UI surfaces whose visibility and displayed text are driven by the same reactive state. The real state should be allowed to change immediately, but the leaving DOM should render the last non-empty visible payload until its leave lifecycle finishes.

Prefer Vue transition lifecycle hooks (`@after-leave`, Vuetify dialog/model update hooks where available, or small local snapshot refs) over hard-coded timeout delays. This keeps behavior aligned with the active animation mode and avoids duration drift.

## Boundaries

- Frontend only.
- Do not change backend queue, download, favorites, or local mod operation contracts.
- Keep changes near the affected components unless a tiny composable clearly removes repeated snapshot logic.

## Content Snapshot Rules

- Capture the latest visible payload while the surface is open/active.
- When close/leave begins, keep rendering the captured payload.
- On `after-leave`, clear the snapshot if the surface is still closed.
- If the surface reopens before leave completion, prefer the fresh open payload.
- For action bars, snapshot selected count/items for rendering only; actual selection state still clears immediately.

## Candidate Implementation Shape

- Add a small composable in `frontend/src/composables/` if two or more components need the same `visible payload -> closing snapshot -> after-leave cleanup` behavior.
- Use local component refs for one-off cases where the lifecycle is specific to Vuetify dialog behavior.
- For `VirtualList.vue`, expose snapshot-selected items/count to the action slot while the action bar is leaving, without changing the real `selectedIndices`.
- For `App.vue`, snapshot queue state/items while active and render the FAB from that snapshot during leave.
- For dialogs in `Download.vue`, `Favorites.vue`, and `Manage.vue`, render dialog copy from a stable payload and clear payload only after the dialog has fully closed.

## Compatibility

- CSS/Vuetify animation mode: use transition/dialog leave lifecycle instead of fixed durations.
- GSAP mode: use existing transition hooks and ensure snapshots survive until GSAP calls `done`.
- Animations off: leave completion should run effectively immediately and clear snapshots without leaving stale UI data.
