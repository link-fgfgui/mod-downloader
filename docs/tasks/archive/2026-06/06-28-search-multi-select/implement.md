# Implementation Plan

## 1. Add selection state to SearchResultList.vue

- Add `selectedIndices` reactive Set, `lastClickedIndex` ref
- Add `isSelected(index)` / `toggleSelect(index, event)` / `selectAll()` / `clearSelection()` helpers
- Wire Shift+Click range logic in `toggleSelect`

## 2. Wire click/keyboard handlers

- Add `@click` handler on `v-list-item` body → `toggleSelect(item.index, $event)`
- Add `@click.stop` on existing download button to prevent selection
- Add `@keydown` listener on scroll container for Ctrl+A and Escape
- Add `tabindex="0"` on scroll container for keyboard focus

## 3. Visual selection styling

- Bind dynamic `bg-color` on `v-list-item`: selected → custom class, unselected → `"surface"`
- Add `.search-result-selected` scoped CSS with primary color at low opacity
- Add transition on background-color for smooth toggle

## 4. Floating action bar

- Add `<Transition name="fab-bar">` wrapper with `v-if="selectedIndices.size > 0"`
- Bar contains: selected count chip, Download All btn, Unpin btn, Copy Names btn, Deselect btn
- Position: absolute bottom of scroll container parent (need a wrapper div with `position: relative`)
- Add slide-up/slide-down CSS transition

## 5. Emit batch events

- Add `batch-download` and `batch-unpin` emits
- Download All: `emit('batch-download', selectedResults)`
- Unpin: `emit('batch-unpin', selectedResults)`
- Copy Names: `navigator.clipboard.writeText(titles.join('\n'))`
- Deselect: `clearSelection()`

## 6. Parent (Download.vue) handlers

- Add `@batch-download` handler: loop and call `installMod()` per selected mod
- Add `@batch-unpin` handler: loop and call unpin per selected mod

## 7. Clear selection on new search

- Watch `props.results` reference change (not just length) — if results array identity changes and it's not a load-more append, clear selection

## 8. i18n keys

- Add keys under `download.selection.*` for action labels and count text

## Validation

- `npm run build` must pass
- Manual test: Ctrl+A, Ctrl+Click, Shift+Click, Escape
- Floating bar visibility, animation, all 4 actions
