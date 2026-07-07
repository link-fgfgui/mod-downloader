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
