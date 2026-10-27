# Motion Guidelines

Governs all motion in the Wails/Vue/Vuetify frontend. Read this before touching
`animations.css`, `useAnimationSettings.ts`, `style.css` (app shell), route/FAB
transitions, or any dialog / overlay / menu / expand transition. For leave-time
snapshot behavior that composes with these transitions, see
[UI State Lifecycle](./ui-state-lifecycle.md).

## Fixed Implementation — No Modes

There is exactly one motion implementation. Do not reintroduce `off` /
`vuetify` / `gsap` mode switches, `data-md-animations`, or a second engine.

| Surface | Owner |
| --- | --- |
| Route page transition (root only) | GSAP (`useAnimationSettings.ts`) |
| Download queue FAB | GSAP (`useAnimationSettings.ts`) |
| Select / combobox / autocomplete menus | CSS `md-menu-pop` via Vuetify defaults |
| Dialog / overlay / context menu | Vuetify native transitions |
| Expand / collapse | Vuetify `v-expand-transition` |
| Hover / press feedback | CSS utilities in `animations.css` |
| Reduced motion | CSS media query + `gsapDuration()` returning 0 |

GSAP owns the route root and the FAB. The route root animates as one surface;
there is no per-child stagger. Select-family menus use a lightweight CSS
transition so picking an option visibly collapses instead of snapping shut.

## Fixed App Shell and Scroll Ownership

`v-main` and `.route-stage` are fixed to the viewport with `overflow: hidden`.
Every page root is `height: 100%`; Home / Settings / Unpin scroll on the page
root, Download / Manage / Favorites scroll inside their lists. Never size a page
with `calc(100vh - Npx)` and never let `body` become the scroll container: that
makes the scrollbar appear/disappear between routes and shifts content sideways.

WebKitGTK 2.41 ignores `scrollbar-gutter`. Every vertical scroll container must
reserve space for the overlay scrollbar with `padding-inline-end: 16px`
(`VirtualList` uses `pe-4`). Do not globally restyle `::-webkit-scrollbar`: it
forces classic scrollbars to take width and reintroduces layout jumps.

## Single Source of Timing

`applyAnimationSettings()` is the only place that writes `--md-transition-*` on
`documentElement`. It is driven by the settings store and once from `App.vue` on
mount.

- Tokens: `fast 120ms`, `normal 180ms`, `slow 240ms`.
- The duration multiplier scales all three tokens.
- GSAP reads `--md-transition-normal` through `gsapDuration()`; CSS uses the
  tokens directly.
- Component styles must not hardcode durations such as `0.2s` or `0.4s`.

## Route Transition Contract

- The route `<transition>` uses `css: false`, `appear: true`, `mode="out-in"`,
  and the GSAP hooks.
- Only the page root animates. Enter: `opacity 0→1`, `y 8→0`, 180ms `expo.out`.
  Leave: `opacity 1→0`, `y 0→-4`, 100ms `power1.in`. No child stagger and no
  `.v-card` / `h1` / list-row entrance.
- `mode="out-in"` prevents two pages from cross-fading their text. The fixed
  stage keeps the height stable, so `.route-leave-active` must **not** use
  `position: absolute`; it only disables pointer events and marks `will-change`.
- Every enter/leave path must call Vue's `done` exactly once.
  `finishGsapAnimation()` is the only cleanup point, and `armFinishGuard()`
  guarantees a `done` within `duration + 400ms` even if rAF is throttled or a
  timeline is killed.
- Rapid navigation must never leave a page transparent or leave the router
  waiting.

## Menu Transitions

- `plugins/vuetify.ts` sets `VSelect.transition = "md-menu-pop"` and
  `VCombobox/VAutocomplete.menuProps.transition = "md-menu-pop"`.
- Any explicit `:menu-props` object must include the same transition, otherwise
  it replaces the defaults.
- `VMenu` keeps Vuetify's anchored `VDialogTransition` for context menus and
  the queue panel; do not override it globally.
- `md-menu-pop` animates `opacity` and `scale(0.97)` with the anchor
  `transform-origin` Vuetify provides. Do not add directional `translateY`,
  because menus can open above or below the activator.

## No Layout-Property Animations

Do not transition `flex`, `width`, `height`, or `max-height` in app CSS. Those
trigger layout on every frame and jank the virtual lists. Use
`transform`/`opacity`, or `v-expand-transition` (which measures height once) for
expand/collapse. Keep its duration aligned with the tokens:

```css
.expand-transition-enter-active,
.expand-transition-leave-active {
  transition-duration: var(--md-transition-normal);
  transition-timing-function: var(--md-ease-out);
}
```

## Dialog and Overlay Rules

- `v-dialog` uses Vuetify's default `VDialogTransition` (origin-aware). Do not
  override it with `scale-fade`.
- `v-overlay` may use the shared `scale-fade`; its inner content must not
  animate separately.
- List ↔ empty-state swaps use the shared `md-fade-slide` (`mode="out-in"`).
- Do not hand-write a `max-height: 0 → auto` expand transition; CSS cannot
  interpolate it.

## Stable Hover Hit Areas

Hover feedback on list rows must not move the hovered element itself. Use a
shadow (`.md-hover-lift`) or another effect that keeps its pointer hit area
fixed. Button scale feedback (`.md-hover-scale` / `.md-btn-press`) is separate
and may continue to use `transform`.

## Verification

- Run `npm run build` and `npm run lint` from `frontend/`.
- Navigate across every route and reload directly on each route: content must
  be visible in every case.
- Click through routes rapidly: no page may stay transparent, no scrollbar may
  jump, and `done` must not hang.
- Switch Manage list/tree rapidly: the primary indicator must track the active
  button and the content must not go blank.
- Open and pick from a select/combobox: the menu must visibly collapse.
- Drag virtual-list scrollbars and check the right edge: no row action button
  may sit under the overlay scrollbar.
- Check Download idle / results / no-results states, dialogs, overlays, the
  sidebar version panel, the queue FAB, 0.5x / 1x / 2x speed multipliers, and OS
  reduced-motion mode.
