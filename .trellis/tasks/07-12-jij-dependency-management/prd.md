# Integrate JIJ into dependency management

## Goal

Treat enabled jar-in-jar (JIJ) mods as dependency providers in local mod
management, and warn before disabling a JAR when doing so would leave another
enabled mod without a required dependency. When the missing dependency was an
independent JAR previously removed because a JIJ copy made it redundant, let
the user restore that independent dependency from the warning.

## Background

- Parsed local mod metadata already exposes top-level required dependencies as
  `ModInfo.Dependencies` and nested JIJ mod IDs as `ModInfo.JijMods`.
- `JijModInfo` currently contains only `id` and `name`; nested versions are not
  retained. This feature therefore treats a matching normalized mod ID as a
  provider regardless of version, matching the requested behavior for same or
  different versions.
- The unused-dependency scanner currently counts requirements from enabled
  top-level mods but does not count JIJ-provided mod IDs.
- Local mod enable/disable/invert actions currently apply file renames without
  a dependency-impact preflight.
- The metadata cache stores remote versions with normalized parsed `ModIDs`,
  platform, project ID, compatible Minecraft versions, and loaders. It does
  not yet expose a reverse lookup from mod ID to cached project/version.

## Requirements

### R1. Shared dependency satisfaction semantics

- Build one backend-owned dependency satisfaction model used by both unused
  dependency scanning and disable-impact analysis.
- Enabled top-level mod IDs and JIJ mod IDs exposed by enabled JARs are
  dependency providers.
- Disabled JARs, and JARs projected to become disabled by the pending action,
  do not provide top-level or JIJ mod IDs.
- Only required dependencies participate. Optional, incompatible, and other
  non-required dependency types do not block an action.
- Matching is case-insensitive after trimming whitespace and ignores versions.

### R2. JIJ-aware unused dependency scanning

- An enabled standalone dependency JAR may be returned as unused when every
  required mod ID it provides remains available from an enabled JIJ provider.
- The behavior applies both to manual scans and the post-delete scan using
  `excludedPaths`.
- A standalone dependency must remain protected when at least one enabled
  dependent would lose its last provider.
- Existing evidence-based candidate filtering remains in force: ordinary mods
  must not become cleanup candidates merely because nothing requires them.

### R3. Disable-impact preflight

- Before a local operation that would change one or more selected enabled JARs
  to disabled, analyze the projected final state for missing required
  dependencies of all remaining enabled mods.
- Cover single-item disable, batch disable, and the disabling side of invert.
- Return structured impact data identifying each missing mod ID, affected
  dependent mods, and the selected JAR or JARs whose removal as providers
  caused the issue.
- Show a confirmation dialog before applying the operation when impacts exist.
- The user can cancel without changing files or explicitly continue with the
  disabling operation.
- Enabling-only operations do not show this warning.
- Deletion behavior is not broadened by this task; it keeps its existing
  confirmation and unused-dependency cleanup flow.

### R4. Restore a dependency from cached metadata

- For each missing dependency mod ID in the disable warning, query only the
  existing metadata cache for compatible remote versions declaring that mod
  ID.
- Offer a manual restore action only when the cache lookup resolves a usable
  project/version candidate. A cache miss does not trigger provider search,
  remote JAR parsing, or automatic repair.
- When one compatible cached project remains after deduplicating known
  cross-platform equivalents, the restore action queues it directly.
- When multiple unrelated compatible cached projects declare the same mod ID,
  the restore action asks the user to choose the platform and project before
  queueing. The application must not choose an unrelated project implicitly.
- Restoration uses the existing download queue and selected instance's
  Minecraft version and mod loader compatibility rules.
- The warning clearly distinguishes a queued restoration from a completed
  install.
- The feature does not persist a separate history of deleted dependencies.

### R5. Cross-layer contract and localization

- Shared request/result structs live in the core package and are exposed
  through the Wails adapter and generated frontend bindings.
- User-facing warning, affected-mod details, continue/cancel actions, recovery
  action, success, and failure states are localized in Chinese and English.
- Backend logic remains independent of Wails UI/runtime types.

## Acceptance Criteria

- [ ] Given enabled `architectury-api.jar` providing `architectury` and another
  enabled JAR whose `jijMods` contains `architectury`, unused-dependency scan
  can return the standalone Architectury JAR when it meets existing library
  evidence rules.
- [ ] The same result holds whether the standalone and nested versions are
  equal or different because matching is by normalized mod ID only.
- [ ] If an enabled dependent still needs a mod ID and no other enabled
  top-level or JIJ provider remains, the standalone provider is not reported as
  unused.
- [ ] Disabling a JAR whose JIJ entry is the last provider for another enabled
  mod produces a warning before any file rename occurs.
- [ ] Canceling that warning leaves every selected file unchanged; continuing
  applies the originally requested single, batch, or invert operation.
- [ ] Batch analysis uses the projected final state, so providers and
  dependents selected together do not create false warnings for dependents
  that will also be disabled.
- [ ] A missing dependency with a compatible cached project/version declaring
  the same normalized mod ID can be queued from the warning without navigating
  to the download page.
- [ ] A cache miss performs no network lookup and offers no misleading active
  restore action.
- [ ] A single compatible cached project can be queued directly; multiple
  unrelated cached projects open a deterministic platform/project chooser and
  only the selected candidate is queued.
- [ ] Core unit tests cover provider matching, disabled providers, multiple
  providers, projected batch state, cancellation-safe preflight data, and
  cache lookup hits, misses, compatibility filtering, and ambiguity handling.
- [ ] Frontend build/lint and app/core Go build, vet, and tests pass after Wails
  bindings are regenerated.

## Out Of Scope

- Comparing or enforcing JIJ dependency versions or version ranges.
- Treating JIJ entries as top-level installed projects for update, conflict,
  favorite, or general download-button state.
- Automatically disabling dependent mods.
- Automatically downloading a replacement before the user chooses an action.
- Persisting deletion history or a dedicated mod-ID-to-project recovery table.
- Online provider search or remote JAR parsing on a restore cache miss.
- General dependency repair outside the local disable warning.
