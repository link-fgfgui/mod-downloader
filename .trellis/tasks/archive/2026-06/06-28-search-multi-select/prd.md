# SearchResultList Multi-Select with Floating Action Bar

## Goal

Add file-explorer-style multi-select to SearchResultList with a floating action bar for batch operations.

## Requirements

### Multi-Select Interactions

| Interaction | Behavior |
|---|---|
| Click item body | Toggle single selection (select/deselect) |
| Ctrl+Click | Toggle individual item without affecting other selections |
| Shift+Click | Range select from last-clicked item to current item |
| Ctrl+A | Select all visible results |
| Escape | Clear all selections |

- Selection state is visual (highlight + checkbox), independent of existing download button.
- Selection clears on new search; load-more appends do NOT clear.
- Existing per-item download button remains functional and independent.

### Floating Action Bar

Appears when >= 1 item selected. Fixed at bottom of scroll area, centered.

| Action | Icon | Behavior |
|---|---|---|
| Download All | `mdi-download-multiple` | Queue download for all selected mods |
| Unpin | `mdi-pin-off` | Unpin selected mods (enabled only if any selected is pinned) |
| Copy Names | `mdi-content-copy` | Copy titles to clipboard, one per line |
| Deselect All | `mdi-selection-off` | Clear selection |

**Bar UI:** shows selected count, slide-up/down animation, semi-transparent surface bg with elevation.

### Non-Goals

- Drag-select / marquee selection
- Paste action
- Drag-and-drop reorder
- Persist selection across page navigation

## Acceptance Criteria

- [ ] Ctrl+A selects all search results; Escape clears
- [ ] Ctrl+Click toggles a single item's selected state
- [ ] Shift+Click selects contiguous range from last-clicked to current
- [ ] Click on item body (not download button) toggles selection
- [ ] Floating bar appears with animation when selection count > 0
- [ ] "Download All" queues downloads for all selected items
- [ ] "Copy Names" copies titles to clipboard
- [ ] "Deselect All" clears selection and hides bar
- [ ] Selection clears on new search but NOT on load-more
- [ ] Existing per-item download button works unchanged
