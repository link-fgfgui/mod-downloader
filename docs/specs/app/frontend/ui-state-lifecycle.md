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

### Don't: Rely on page `height: 100%` without the shell stylesheet

**Problem**: Page roots and virtual lists use `height: 100%`, but the fixed
application shell (`src/style.css`: `html/body/#app`, `.v-application__wrap`,
`.v-main`, `.route-stage`) is not imported by `frontend/src/main.ts`.

**Why it's bad**: `height: 100%` resolves against an auto-height parent, so a
long virtual list grows the page instead of scrolling inside it. The floating
selection action bar is anchored with `position: absolute; bottom: 16px` inside
the list wrapper, so it lands thousands of pixels below the viewport and looks
like it never rendered. Short lists hide the bug.

**Instead**: Keep `import "./style.css"` in `frontend/src/main.ts` and verify a
long list (60+ rows) keeps the action bar inside the window, including at small
viewport heights.

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

When a `v-combobox` accepts a user-defined value (for example, creating a
favorite list from the add-to-favorites dialog), keep its model as the entered
string and resolve an existing item's ID only when submitting. Combining
`return-object` with free-form input can leave the submit path without the
custom value in Vuetify 4.

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

Do not attach page-level entrance animations to virtualized rows. Recycling an
animated row during a scrollbar drag causes flashes, unnecessary main-thread
work, and a changing scroll range. Hover and direct-interaction transitions are
allowed because they do not run as rows enter the viewport.

Route-level GSAP animation must obey these contracts:

- Only the page root animates. There are no page-content targets: no `.v-card`,
  `h1`, `.v-list-item`, or `.v-data-table__tr` entrance, because virtualized rows
  recycle and must never animate.
- Enter is `opacity 0→1` plus `y 8→0`; leave is `opacity 1→0` plus `y 0→-4`.
  Never scale the root.
- Use `mode="out-in"`. The outgoing page stays in flow, so it must not be
  absolutely positioned. The fixed `.route-stage` (`overflow: hidden`) keeps the
  height stable.
- Every enter/leave path calls `done` exactly once through
  `finishGsapAnimation()`. `armFinishGuard()` fires a fallback `done` within
  `duration + 400ms`, so a throttled rAF or a killed timeline cannot hang the
  router.
- The page root owns scrolling (`height: 100%`) and the shell never scrolls.
  Scroll containers reserve `padding-inline-end: 16px` for WebKitGTK overlay
  scrollbars; `scrollbar-gutter` is unsupported there.

Correct:

```ts
gsap.set(pageRoot, { opacity: 0, y: 8 });
gsap.to(pageRoot, { opacity: 1, y: 0, duration: gsapDuration(0.18), ease: "expo.out" });
```

Wrong:

```ts
const targets = pageRoot.querySelectorAll(".v-card, .v-list-item");
gsap.set(targets, { opacity: 0, y: 12 }); // Per-element stagger; recycles rows.
pageRoot.classList.add("route-leave-active"); // Must not become absolute.
```

Verification must cover rapid navigation, the Manage list/tree switch, picking
from a select/combobox, dragging virtual-list scrollbars, route leave/re-entry,
and opening content that was hidden during the initial route enter. Always run
frontend lint and production build.
