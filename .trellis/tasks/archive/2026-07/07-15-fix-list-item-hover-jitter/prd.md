# Fix generic list item hover jitter

## Goal

Prevent shared list rows from repeatedly entering and leaving their hover state
when the pointer rests near the row's bottom edge, while preserving clear hover
feedback.

## Background

The shared `.md-hover-lift` interaction moves its target upward by 3px on
hover (`frontend/src/styles/animations.css:241`). Moving the hovered element
also moves its hit area away from a pointer positioned at the bottom edge,
which can immediately cancel and re-trigger the hover state. The utility is
currently used by search results, managed-mod rows, and favorite-mod rows.

## Requirements

- The hover state of `.md-hover-lift` must not change the list row's vertical
  position or pointer hit area.
- Shared list rows must retain a visible hover affordance through the existing
  shadow transition.
- The fix must cover all current `.md-hover-lift` consumers without duplicating
  per-view CSS.
- Button hover scaling and the `off` / `vuetify` / `gsap` animation-mode
  ownership rules must remain unchanged.

## Acceptance Criteria

- [x] Holding the pointer at the bottom edge of a search, managed-mod, or
  favorite-mod row does not cause the row to jump between hovered and
  non-hovered positions.
- [x] Hovering those rows still applies the existing elevated shadow feedback.
- [x] `.md-hover-scale` button interactions are unchanged.
- [x] `npm run build` and `npm run lint` pass in `frontend/`.

## Out Of Scope

- Redesigning list-row layout, spacing, selection, or virtualization.
- Changing page entrances, route transitions, or button interaction effects.
