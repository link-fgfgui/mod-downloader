# Implementation Plan

## Batch 1: Shared List Foundation

- [x] I24: add coalesced virtual-list remeasurement for mount/activation/resize.
- [x] I25: unify Download and Manage height and scroll ownership.
- [x] I06: suppress text selection only during Shift range selection.
- [x] I12: standardize action slots on icon buttons and tooltips.
- [x] I13: register and forward shortcuts to the one active list with guards.
- [x] I26: defer Manage hover text during scrolling.
- [x] Run frontend lint/build and targeted component tests if present.

## Batch 2: Favorites Structure And Interaction

- [x] I10: remove group UI/store/Wails/service exposure without deleting data.
- [x] I11: constrain the left list and give it internal scrolling.
- [x] I14/I17: stabilize drag semantics, preview, insertion marker, and cleanup.
- [x] I18: avoid full-array replacement and scroll reset after list creation.
- [x] I15: share menu state between context-click and three-dot activation.
- [x] I16: standardize retained switch props.
- [x] Run core/app tests, regenerate Wails bindings, then lint/build frontend.

## Batch 3: Favorite Scope And Pin Reuse

- [x] I01: persist list scope, filter by active tuple, and clear stale selection.
- [x] I07: expose pin as a distinct action in the local version dialog using
  `ModVersionList` and the existing pin service.
- [x] I05: open the same pin flow from Favorites mod icons, including explicit
  referenced-item eligibility.
- [x] Run all layer build/test gates; scope round-trip coverage remains in the final audit.

## Batch 4: Settings And Runtime Configuration

- [x] I02: encode defaults/ranges and tests for downloads/API configuration.
- [x] I21: extend settings DTO, Wails API, frontend store, and controls; apply
  downloader and rate-limiter changes to new work without restart.
- [x] I22: centralize debounced auto-save, pending/error state, rollback,
  snackbar, and API-key sentinel handling; remove save buttons only.
- [x] I20: default dependency cleanup off while preserving explicit values.
- [x] I03: migrate logging disabled-to-enabled across config, env, startup,
  README, and compatibility tests as an isolated change.
- [x] I09: repair MCIM/CurseForge provider request configuration and regress it.
- [x] Regenerate bindings and run core/app/frontend gates.

## Batch 5: Home

- [x] I28: remove current/bottom status UI and obsolete loads/listeners.
- [x] I27: implement and test three-significant-digit compact card formatting.
- [x] I29: emit typed usage changes at mutation points, subscribe only while
  Home is active, keep initial fetch, coalesce refresh, remove refresh button.
- [x] Run event lifecycle tests and all layer gates.

## Batch 6: Independent Page Fixes

- [x] I04: make animation cleanup unconditional, mark GSAP experimental, and
  complete lifecycle-safe element animations.
- [x] I08: add conflict filename contract and place the hint on its own line.
- [x] I19: remove only the redundant cache default-position input.
- [x] I23: remove Unpin filters and make bulk action target the full list.
- [x] Run targeted regressions and all layer gates.

## Batch 7: Completion Audit And External Sync

- [x] Run `gofmt` on changed Go files.
- [x] Run `go build ./...`, `go vet ./...`, and `go test ./...` at repository root.
- [x] Run `go test ./...` in `core/`.
- [x] Run `npm run lint` and `npm run build` in `frontend/`.
- [x] Verify all changed Wails contracts match regenerated bindings.
- [ ] Perform desktop UI smoke checks for virtual lists, dialogs, drag/drop,
  settings auto-save, route animations, and Home activation lifecycle.
- [x] Check every I01-I29 and nested checkbox in `issues.md` only when evidence
  exists, then mark the corresponding exact-list Microsoft To Do task complete.
- [x] Confirm the exact list has no unfinished original task and that only this
  one Trellis task was created for the work.

## Risk And Rollback Points

- Finish and verify each batch before starting dependents; revert only the
  owning batch if a regression appears.
- Do not destructively migrate favorite-group storage.
- Do not apply runtime configuration changes to in-flight downloads.
- Do not complete external To Do items ahead of code verification.
- Keep generated Wails bindings in the same batch as the contract change.

## Final Verification Note

The production frontend build, live Vite endpoint, lifecycle source audit, and
all automated gates passed. Native Wails launch could not run on this host
because `pkg-config` cannot find the required `webkit2gtk-4.0` development
package; `wails build` was attempted and failed only at that system dependency.
