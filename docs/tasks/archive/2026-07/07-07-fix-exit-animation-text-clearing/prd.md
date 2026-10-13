# 修复退场动画期间可变文字过早清空

## Goal

When a UI element is leaving the screen, any text still visible during its exit animation should remain the same text the user saw before dismissal. Closing, confirming, deselecting, or backend state updates must not blank or change visible copy before the element has finished leaving.

This improves perceived polish and avoids confusing intermediate states such as empty dialog bodies, zero-count action bars, or queue panels whose labels disappear while they are still animating out.

## Confirmed Facts

- Vue/Vuetify transitions are used for route/FAB, overlay, dialog, and floating action bar motion.
- The download queue FAB leaves with a transition keyed by `downloadQueueStore.queue.active`; its badge, summary, item titles, metadata, and reasons are read live from the queue store while the FAB can still be visible during leave animation. Evidence: `frontend/src/App.vue:22`, `frontend/src/App.vue:31`, `frontend/src/App.vue:53`, `frontend/src/App.vue:67`.
- The virtual list floating action bar leaves when `selectedIndices.size` becomes `0`; the visible selected count and action slot data also read from `selectedIndices` / `selectedItemsList`. Evidence: `frontend/src/components/VirtualList.vue:20`, `frontend/src/components/VirtualList.vue:23`, `frontend/src/components/VirtualList.vue:26`.
- The favorites delete dialog body interpolates `deleteDialog.list?.name || ""`, and `deleteList` sets `deleteDialog.show = false` and immediately clears `deleteDialog.list = null`. Evidence: `frontend/src/views/Favorites.vue:149`, `frontend/src/views/Favorites.vue:152`, `frontend/src/views/Favorites.vue:216`.
- The download replacement confirmation dialog body/title depend on `confirmDialog.status`; confirming closes the dialog and immediately clears `status`, `result`, and `key`. Evidence: `frontend/src/views/Download.vue:174`, `frontend/src/views/Download.vue:180`, `frontend/src/stores/downloadSearch.ts:190`.
- The manage delete dialog uses `pendingDeleteCount` in its body, and successful deletion closes the dialog while immediately clearing the pending groups/count. Evidence: `frontend/src/views/Manage.vue:156`, `frontend/src/views/Manage.vue:160`, `frontend/src/views/Manage.vue:384`.

## Requirements

- R1. Any component or page element that remains visible during an exit animation must render a stable snapshot of its variable text/content until the leave transition completes.
- R2. Fix the known affected areas: download queue FAB/menu content, virtual-list floating selection action bar, favorites delete dialog, download replacement confirmation dialog, and manage delete dialog.
- R3. Closing or confirming dialogs must still clear underlying state after the close is complete, so stale data is not shown the next time the dialog opens.
- R4. Deselecting or clearing list selections must still update the real selection state immediately for behavior and keyboard handling, while the leaving action bar visually keeps its pre-clear count/action context.
- R5. The fix must work with the existing animation modes: Vuetify/CSS transitions, GSAP mode, and animations off/reduced-duration settings.
- R6. The implementation should prefer a reusable frontend pattern for leave-time content snapshots over scattered timeout constants.
- R7. Audit additional transition/dialog/action surfaces discovered during implementation and fix any that match the same visible-text-cleared-during-leave pattern.

## Acceptance Criteria

- [x] Closing the favorites delete dialog after deleting a list never shows an empty list name during the dialog leave animation.
- [x] Confirming the download replacement/conflict dialog never changes or blanks the title/body during the dialog leave animation.
- [x] Confirming a manage batch delete never changes the delete count to `0` during the dialog leave animation.
- [x] Clearing a virtual-list selection keeps the floating action bar's visible count and action labels stable until the action bar has fully left.
- [x] When the download queue becomes inactive, the leaving FAB/menu content does not blank, drop to zero, or lose item titles/reasons while still visible.
- [x] Reopening any affected dialog/action surface starts from the current fresh state, not a stale leave snapshot from the previous close.
- [x] `npm run build` succeeds from `frontend/`.

## Out of Scope

- Redesigning animation timing or visual style.
- Changing backend queue/download behavior.
- Adding new product flows beyond preserving visible text/content during exit.
