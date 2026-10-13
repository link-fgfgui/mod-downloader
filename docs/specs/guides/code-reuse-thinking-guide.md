# Code Reuse Thinking Guide

Use this guide before adding a helper, constant, converter, component, or
payload reader. Duplication in this project has caused divergent provider
converters, stale UI contracts, and inconsistent state transitions.

## Search First

```bash
rg "similarName|distinctiveField" core frontend/src app.go
rg "EventKind|status|action|jsonField" core frontend/src
```

Answer:

- Does an existing service/store/component already own this behavior?
- Is the value a canonical model, request/response, or adapter-only type?
- Would a new helper create a second normalization or validation path?
- If copying code, should the shared owner be extended instead?

## Project Rules

- Same logic used three or more times should have one owner.
- `models` owns provider-neutral types; do not create aliases or re-exports.
- A converter should have one canonical target and one active implementation.
  Delete obsolete converters in the same change that wires the replacement.
- Reuse an existing component or composable when behavior and lifecycle match;
  add a typed prop/variant instead of creating an almost-identical component.
- If two consumers read the same untyped event/payload field, add a shared
  decoder, type guard, or projection before adding another reader.
- Constants and configuration defaults have one definition; search all call
  sites before changing them.
- Reducers or `switch` statements should own action/status transitions rather
  than scattering partial updates across consumers. Search every switch when
  adding a new event kind, queue status, loader, or provider; a default branch
  must not silently map a new value to an old one.

## Parallel Outputs

When two mechanisms must describe the same API, avoid hand-maintained parallel
lists. Public `App` methods and generated `frontend/wailsjs` bindings are one
such pair: regenerate bindings after signature changes and build the frontend
to catch drift. If two paths cannot share an implementation, add a comparison
test that proves their outputs stay equivalent.

## Do Not Abstract Prematurely

Keep one-off trivial code local. Add an abstraction when duplication is
observable, the logic is complex enough to drift, or multiple layers need the
same contract. The abstraction must reduce the number of places that can be
wrong.

## Completion Checklist

- [ ] Searched for an existing equivalent before adding code
- [ ] Confirmed the canonical type/owner
- [ ] Removed or wired obsolete duplicate paths
- [ ] Searched all consumers after changing a shared field or constant
- [ ] Regenerated or compared any derived/parallel output
- [ ] Added focused tests for shared normalization or state transitions
