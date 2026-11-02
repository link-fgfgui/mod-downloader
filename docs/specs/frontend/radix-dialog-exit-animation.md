# Radix Dialog Exit Animation

## Symptom

Closing any Dialog / AlertDialog / DropdownMenu / Tooltip plays the exit animation
to its last frame, then briefly flashes a complete, visible dialog before the node
is removed from the DOM.

## Root cause

- `tw-animate-css`'s `animate-out` defaults to `animation-fill-mode: none`, so the
  element snaps back to its base style (`opacity: 1`, `scale: 1`) when the
  animation ends.
- Radix Presence waits for `animationend` before unmounting. In its handler it
  temporarily sets `style.animationFillMode = "forwards"`, then clears the inline
  value with `setTimeout(0)`.
- Under WebKitGTK / React 18, that inline reset can happen before React commits
  the unmount. The animation has ended and `fill-mode` is cleared, so the browser
  paints one frame of the full dialog: the "last-frame flash".

## Constraints

**Don't**

- Do not rely only on Radix's inline `animationFillMode` fallback.
- Do not omit `data-[state=closed]:fill-mode-forwards` on any Radix Presence
  element using `data-[state=closed]:animate-out`.
- Do not animate only the content layer and skip the overlay; mismatched
  durations make the scrim disappear first.

**Instead**

- Every component class using `data-[state=closed]:animate-out` must also include
  `data-[state=closed]:fill-mode-forwards`, so the stylesheet holds the final
  `opacity: 0` / scaled frame until Radix actually unmounts the node.
- Dialog / AlertDialog overlay and content use `duration-md` together, so the
  scrim and content exit at the same user animation multiplier.
- Add `data-[state=closed]:pointer-events-none` during exit so the animating
  dialog cannot receive clicks.

Current coverage:

- `frontend/src/components/ui/dialog.tsx`
- `frontend/src/components/ui/alert-dialog.tsx`
- `frontend/src/components/ui/dropdown-menu.tsx`
- `frontend/src/components/ui/tooltip.tsx`

## Verification

Close dialogs in the headless server with Playwright and observe `animationend`:

1. After Radix handles it, computed style must be `animation-fill-mode: forwards`
   and `opacity: 0`.
2. Manually clear inline `animationFillMode` (simulating Radix's `setTimeout`
   reset); computed style must still be `forwards` / `opacity: 0`.
3. No sampled frame before unmount may show `opacity: 1`.
