# UI State Lifecycle

## Pattern: Leave-Time Content Snapshots

**Problem**: Vue/Vuetify leave transitions keep DOM visible briefly after the
reactive state that controls visibility has already changed. If the same state
also owns visible text, closing a dialog or clearing a selection can blank or
change text while the element is still animating out.

**Rule**: Any UI surface that remains visible during a leave animation must
render stable visible content until the leave lifecycle completes.

Use this pattern when all of these are true:

- The surface uses `v-if`, `v-model`, `Transition`, `v-dialog`, `v-overlay`, or
  a similar animated leave lifecycle.
- Text or item content is derived from state that is cleared or replaced on
  close, confirm, deselect, or backend event updates.
- Users can see the surface during the leave animation.

### Correct

```vue
<v-dialog v-model="dialog.show" @after-leave="clearClosedDialog">
  <v-card-text>{{ dialog.item?.name || "" }}</v-card-text>
</v-dialog>
```

```ts
async function confirmDelete() {
  if (!dialog.item) return;
  await store.delete(dialog.item.id);
  dialog.show = false;
}

function clearClosedDialog() {
  if (dialog.show) return;
  dialog.item = null;
}
```

### Wrong

```ts
async function confirmDelete() {
  if (!dialog.item) return;
  await store.delete(dialog.item.id);
  dialog.show = false;
  dialog.item = null;
}
```

This clears `dialog.item` before the leave animation has finished, so the
visible dialog body can render empty text while fading or scaling out.

## Action Bars and Selection State

For selection action bars, actual selection state should still clear
immediately so keyboard and command behavior remains correct. Snapshot only the
rendered count/items used by the leaving action bar.

```ts
function clearSelection() {
  snapshotVisibleSelection();
  selectedIndices.clear();
}

function clearActionBarSnapshot() {
  if (selectedIndices.size === 0) {
    selectionSnapshot.value = null;
  }
}
```

Do not keep the real selection alive just to preserve exit text; that changes
behavior outside the visual leave lifecycle.

## Queue or Backend-Driven Surfaces

When a backend event can replace active state with an inactive or empty payload,
snapshot the last active payload and render from that snapshot during leave.
Clear the snapshot in `after-leave` if the surface is still inactive.

```ts
watch(queueState, (queue) => {
  if (queue.active) {
    visibleQueueSnapshot.value = cloneQueue(queue);
  }
}, { deep: true, immediate: true });

function clearQueueSnapshot() {
  if (!queueState.value.active) {
    visibleQueueSnapshot.value = null;
  }
}
```

Avoid hard-coded timeout delays. Prefer Vue/Vuetify transition lifecycle hooks
such as `@after-leave`, or the existing GSAP `done`/`onAfterLeave` transition
path when GSAP owns the motion.

## Menus Inside Dialogs

Vuetify positions a select or combobox menu against the viewport, not against
the dialog's remaining content area. A downward-opening menu can therefore
cover `v-card-actions` while remaining above the dialog in the overlay stack.
The visible Cancel/Add buttons then receive no click because a menu item is the
actual hit target.

Keep nested menu state explicit, close it from the dialog's close path, and
place or size the menu so its content cannot cover the action row.

```vue
<!-- Correct: the nested overlay stays clear of the dialog actions. -->
<v-combobox
  v-model="selectedChoice"
  v-model:menu="listMenuOpen"
  :menu-props="{ location: 'top', maxHeight: 200, offset: 4 }"
/>
<v-btn @click="closeDialog">Cancel</v-btn>
```

```ts
function closeDialog() {
  listMenuOpen.value = false;
  dialogOpen.value = false;
}
```

Do not accept a visual screenshot alone as verification. With the nested menu
open, assert that `document.elementFromPoint()` at the center of each dialog
action resolves to that action, then click Cancel and assert that both the menu
and dialog overlays become inactive. Repeat at desktop and narrow viewports.

## Virtual Lists and Route Animation

Virtualized rows are recycled while scrolling. Their wrapper height is part of
the scrollbar calculation and must remain equal to the `item-height` passed to
`v-virtual-scroll`, including the row's bottom spacing. Apply fixed
`height`/`min-height`/`max-height`, `contain: layout size style`, and a stable
scrollbar gutter to the virtual wrapper items.

Shared virtual-list components must preserve the scroll container's `scrollTop`
across KeepAlive activation and re-enter Vuetify's native scroll handler after
the wrapper becomes visible or changes size. Calling `calculateVisibleItems`
alone is insufficient because it calculates from Vuetify's cached scroll
offset instead of reading the current DOM offset.

Do not attach CSS or GSAP entrance animations to virtualized rows. Recycling an
animated row during a scrollbar drag causes flashes, unnecessary main-thread
work, and a changing scroll range. Hover and direct-interaction transitions are
allowed because they do not run as rows enter the viewport.

Route-level GSAP animation must obey these contracts:

- Exclude `.v-list-item` from global page-content targets.
- Animate the route root with opacity only. Never translate or scale a root
  containing scrollable content, because that also moves or scales scrollbars.
- Select visible content targets before both hiding and animating them. Do not
  hide inactive dialogs or other zero-size targets that the matching enter
  animation will not restore.
- Animation-off mode must not apply Vue route transition classes. CSS animation
  utilities must use their final visible state instead of a `0.01ms` first
  keyframe.

Correct:

```ts
const targets = getVisiblePageContentTargets(routeRoot);
gsap.set(routeRoot, { opacity: 0 });
gsap.set(targets, { opacity: 0, y: 24 });
```

Wrong:

```ts
const targets = routeRoot.querySelectorAll(".v-list-item, .v-card");
gsap.set(routeRoot, { opacity: 0, y: 20, scale: 0.985 });
gsap.set(targets, { opacity: 0 }); // Also hides inactive dialog content.
```

Verification must cover animation modes `off`, `vuetify`, and `gsap`; rapid
scrollbar dragging; route leave/re-entry; and opening content that was hidden
during the initial route enter. Always run frontend lint and production build.
