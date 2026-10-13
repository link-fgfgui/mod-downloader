# Design

## Boundaries

The root module remains the Wails application shell. `core` remains the
reusable backend submodule and is not copied into the root module. Frontend
bindings remain generated under `frontend/wailsjs`.

## Structure changes

1. Add `ARCHITECTURE.md` at the repository root as the canonical navigation
   map. It will document dependency direction, workflow entry points, event
   flow, and generated-file rules.
2. Split `app.go` into responsibility-oriented files in the same `main`
   package: lifecycle/runtime setup, event bridge, settings/preferences,
   download API, and favorites/local-mod API. Keep `app.go` as the small App
   type and constructor/lifecycle entry point where practical; do not rename
   exported Wails methods.
3. Add `core/appcore/README.md` and package docs that group service methods by
   workflow, while keeping the service API stable. Extract only cohesive type
   declarations or helpers when doing so improves discoverability without a
   broad behavior rewrite.
4. Add lightweight frontend navigation documentation linking each route and
   store to its Wails calls.

## Compatibility

No public method, JSON field, event name, import path, or persistence schema is
changed. File moves within a Go package do not affect callers. Generated
bindings are regenerated only if a signature unexpectedly changes.

## Bug policy

Run focused tests and inspect any failing path encountered during the split.
Only fix bugs that are directly evidenced by tests, static analysis, or a
reproducible runtime trace; document the cause and regression coverage.
