# Design: Multi-Select + Floating Action Bar

## Selection State

Selection lives entirely in `SearchResultList.vue` as a `Set<number>` of selected indices. No store changes needed — the component owns selection state and emits batch events to the parent.

### State variables

- `selectedIndices: Set<number>` — reactive set of selected item indices
- `lastClickedIndex: number | null` — anchor for Shift+Click range select

### Keyboard/Mouse handlers

- `@click` on `v-list-item` body → toggle selection (check `e.ctrlKey`, `e.shiftKey`)
- `@keydown.ctrl.a` on scroll container → select all, `preventDefault`
- `@keydown.escape` on scroll container → clear selection
- Download button click must `@click.stop` to avoid triggering selection

### Visual feedback

- Selected items: `bg-color` override to `primary` with low opacity (Vuetify `bg-color="primary"` + scoped CSS for transparency)
- No checkboxes — clean highlight only

## Floating Action Bar

A `<div>` positioned `absolute` at bottom of the scroll container wrapper. Uses `<Transition name="slide-up">` for enter/leave animation.

### Actions emit to parent

| Action | Emit | Payload |
|---|---|---|
| Download All | `batch-download` | `ModProject[]` |
| Unpin | `batch-unpin` | `ModProject[]` |
| Copy Names | (handled internally) | clipboard write |
| Deselect All | (handled internally) | clear `selectedIndices` |

### Parent (Download.vue) handling

- `batch-download`: iterate selected mods and call `downloadStore.installMod()` for each
- `batch-unpin`: call backend unpin API for each selected mod's pinned version

## Data flow

```
SearchResultList (owns selection state)
  ├── click/keyboard → update selectedIndices
  ├── floating bar actions → emit batch-download / batch-unpin
  └── copy names → navigator.clipboard.writeText()

Download.vue (parent)
  ├── @batch-download → loop installMod()
  └── @batch-unpin → loop unpinMod()
```

## Compatibility

- `v-virtual-scroll` recycles DOM nodes. Selection by index (not DOM ref) is safe.
- `item-height="88"` remains unchanged.
- Existing `@click` on avatar (show-versions) uses `.stop` already — no conflict.
- Download button needs `@click.stop` to prevent selection toggle.
