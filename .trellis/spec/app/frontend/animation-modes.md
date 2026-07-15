# Animation Modes

Governs the three animation modes (`off` / `vuetify` / `gsap`) and how motion is
applied across the app. Read this before touching `animations.css`,
`useAnimationSettings.ts`, the settings animation flow, or route/FAB transitions.
For leave-time snapshot behavior that composes with these modes, see
[UI State Lifecycle](./ui-state-lifecycle.md).

## Single Application Entry

**Rule**: `applyAnimationSettings()` is the only place that writes the active
mode, the `documentElement.dataset.mdAnimations` attribute, and the
`--md-transition-*` / `--md-stagger-delay` CSS variables. It must be driven from
exactly one owner — the settings store — so mode *and* duration multiplier always
apply together.

- `stores/settings.ts` calls it in `load()` and after `saveAnimationSettings()`.
- `App.vue` calls `applyAnimationSettings(preferences)` once on mount.
- Views must not add their own `watch` to re-apply settings.

### Wrong

```ts
// Settings.vue — watches mode only, so a multiplier-only change never applies
watch(() => settingsStore.view?.animationMode, () => {
  if (settingsStore.view) applyAnimationSettings(settingsStore.view);
});
```

A multiplier-only edit leaves `animationMode` unchanged, the watch never fires,
and the new duration is persisted but not applied until reload.

### Correct

```ts
// stores/settings.ts — apply the persisted result, covering mode AND multiplier
this.view = await SaveAnimationSettings(req);
applyAnimationSettings(this.view);
```

## Mode Ownership — No Overlap

Each mode owns exactly one motion layer per surface. Never let one mode drive a
layer that another mode also drives on the same element.

| Layer | `off` | `vuetify` | `gsap` |
| --- | :---: | :---: | :---: |
| CSS entrance utilities (`.md-page`, `.md-stagger`, `.md-animate-*`) | none | owns | none |
| Vuetify micro-transitions (dialog/menu/tab/ripple) | none | default | preserved |
| Route transition | none | CSS `slide-fade` | GSAP hooks |
| FAB transition | none | CSS `md-fab-*` | GSAP hooks |
| Hover / press (`md-hover-*`, `md-btn-press`) | none | on | on |

**Rule**: `gsap` mode must not use a global `* { transition-duration: 0s }`
sledgehammer. Silence only the CSS entrance utilities; leave Vuetify's own
micro-transitions live so `gsap` mode is not less animated than `vuetify`.

Only `off` mode applies the global stop (`transition/animation: 0s`, plus
`scroll-behavior: auto`). `prefers-reduced-motion` is independent of all three
modes and overrides them.

## Stable Hover Hit Areas

**Rule**: Hover feedback on list rows must not move the hovered element itself.
Use a shadow or another effect that keeps its pointer hit area fixed. Translating
a row upward moves its bottom edge away from a pointer resting there, repeatedly
toggling `:hover`. Button-scale feedback remains separate and may continue to
use `transform`.

```css
/* Wrong: the hover state moves the boundary that activates it. */
.md-hover-lift:hover {
  transform: translateY(-3px);
}

/* Correct: elevation changes without moving the row's hit area. */
.md-hover-lift:hover {
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
}
```

## Hidden State Comes Only From `animation ... both`

**Rule**: An entrance utility's hidden initial state must come solely from the
`both` fill of its own `animation` shorthand. Never give it a standalone
`opacity: 0` (or `transform`) base rule.

This makes "animation disabled = element visible" an automatic consequence:
removing the animation (`animation: none`) reverts the element to its natural
`opacity: 1`. No per-mode `opacity`/`transform` reset is needed, and no JS-set
readiness attribute is needed to un-hide content.

### Wrong

```css
.md-stagger > * {
  opacity: 0;                                   /* standalone hidden base */
  animation: md-fade-in-up var(--md-transition-slow) var(--md-ease-out) forwards;
}
/* Then gsap mode only removes the animation, leaving opacity: 0 stuck, and
 * needs a JS-set [data-md-gsap-ready] ancestor to restore visibility. */
:root[data-md-animations="gsap"] [data-md-gsap-ready] .md-stagger > * {
  opacity: 1 !important;
}
```

On first paint or a reload directly onto a stagger route, no route-enter hook
fires, `[data-md-gsap-ready]` is never set, and the content is stuck hidden.

### Correct

```css
.md-stagger > * {
  animation: md-fade-in-up var(--md-transition-slow) var(--md-ease-out) both;
}
:root[data-md-animations="off"] .md-stagger > *,
:root[data-md-animations="gsap"] .md-stagger > * {
  animation: none !important;   /* → reverts to opacity: 1, visible */
}
```

## First-Paint Coverage and Single Keep-Alive

**Rule**: The route `<transition>` must set `appear: true` in the animated
branches so the initial load and reloads on any route trigger the enter hook.
GSAP-owned entrances rely on that hook; without `appear` they are skipped.

**Rule**: Use one `<keep-alive>` inside one adaptive `<transition>`, with props
swapped per mode. Do not branch the template into separate keep-alive instances
for `off` vs animated modes — switching modes at runtime would destroy the cache
and re-create the current view.

### Correct

```vue
<router-view v-slot="{ Component, route }">
  <transition v-bind="routeTransitionProps">
    <keep-alive>
      <component :is="Component" :key="route.path" />
    </keep-alive>
  </transition>
</router-view>
```

## GSAP Cleanup on Mode Switch

`applyAnimationSettings()` calls `cleanupActiveGsapAnimations()` whenever the
mode changes: it kills tracked tweens, clears inline `opacity`/`transform`, and
invokes any pending Vue transition `done` callbacks. This prevents stale inline
styles and stuck leave transitions when switching away from `gsap`.

## Verification

Cover all three modes (`off`, `vuetify`, `gsap`) for every change:

- Route navigation, plus initial load / reload directly on `/download` and
  `/manage` (the stagger routes) — content must be visible in every mode.
- Duration-multiplier-only edit in Settings — applies live without reload.
- Toggling modes at runtime — no stuck-hidden content, no keep-alive state loss.
- FAB show/hide in each mode, and the leave-time snapshot cleanup.

Always run `npm run build` and `npm run lint` from `frontend/`.
