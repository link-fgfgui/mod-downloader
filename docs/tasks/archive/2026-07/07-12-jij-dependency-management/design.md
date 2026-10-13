# Technical Design

## Boundaries

- `core/appcore` owns local dependency graph construction, projected-state
  analysis, and the public preflight service API.
- `core/storage` owns cache-only reverse lookup from normalized mod ID to
  cached remote versions. The lookup must not call providers or mutate cache.
- `core/modbridge` converts a resolved cached project/version into the existing
  download request shape and applies current selected-instance rules.
- `app.go` only forwards typed Wails calls.
- `frontend/src/views/Manage.vue` owns warning-dialog state and the explicit
  continue/cancel/restore interactions.

## Dependency Graph

Group parsed `ModInfo` rows by physical path because one JAR may declare
multiple top-level mods. Each JAR group contains:

- path and enabled state;
- directly provided normalized top-level mod IDs;
- normalized JIJ-provided mod IDs;
- normalized required dependency IDs;
- display metadata for affected-mod messages.

For a projected state, provider counts are built from enabled JAR groups. A
required dependency is satisfied when its normalized ID has at least one
enabled direct or JIJ provider. Versions are intentionally absent from this
contract because `JijModInfo` does not retain them and the requested behavior
accepts either equal or different versions.

The unused-dependency scanner and disable preflight must call shared graph
helpers rather than maintaining separate provider semantics.

## Disable Flow

```text
Manage action
  -> AnalyzeLocalModDisableImpact(paths, action)
  -> project final enabled/disabled state
  -> return missing dependency impacts
  -> no impacts: apply existing operation
  -> impacts: show warning
       -> cancel: no file operation
       -> continue: apply original operation unchanged
       -> restore: queue cache-resolved prerequisite, then keep dialog state
```

The analysis result is advisory and does not authorize a file change. The
existing `ApplyLocalModBatchOperation` remains the mutation boundary. This
keeps cancellation safe and avoids adding a force flag to file operations.

Each impact contains the missing mod ID, affected enabled dependent JARs, the
selected provider JARs being disabled, and an optional cached restore
candidate.

For invert, projection follows the actual planned per-path result. Dependents
that are also projected disabled are excluded from warnings.

## Cache Reverse Lookup

Add a read-only storage query that scans cached `PlatformVersions` and returns
versions whose normalized `ModIDs` contain the requested ID and whose
`GameVersions` and `Loaders` match the selected instance. Returned values are
copies and sorted deterministically; map iteration order must never affect UI
or downloads.

The query is cache-only:

- no `providers.SearchMods`;
- no provider refresh;
- no remote JAR mod-ID backfill;
- no separate persistence or cache schema/version change.

Cached project metadata is joined by platform/project ID when available. A
usable restore candidate requires platform, project ID, version ID, and a
downloadable compatible cached version. The existing download queue performs
its normal final validation.

Cross-platform associations identify known equivalent CurseForge and Modrinth
projects and collapse them into one logical candidate. They cannot prove that
two arbitrary projects declaring the same mod ID are equivalent. When multiple
unrelated logical candidates remain, return all of them in stable order for
explicit user selection.

## Contracts

Introduce core structs for:

- disable-impact analysis request: paths and action;
- affected local mod summary: path, file name, mod IDs, display name;
- cached restore candidate: mod ID, platform, project ID, version ID, title;
- disable-impact result: ordered impact list.

Expose Wails methods for analysis and for explicitly queueing one resolved
cache candidate. Do not let the frontend construct provider requests from raw
fields without backend validation.

## Compatibility And Failure Behavior

- Existing local operations remain unchanged when analysis returns no impact.
- Cache-not-open, cache miss, incomplete cached version, or incompatible scope
  produce no restore candidate; they do not block the disable warning.
- A stale candidate may fail normal queue validation. Report that failure in
  the dialog/snackbar and do not claim installation completed.
- No migration is needed because no persistent schema changes are introduced.

## UI

Use one warning dialog listing each missing dependency and affected mods. The
dialog provides cancel and continue commands. A restore icon/button is shown
per missing dependency only when a usable cached candidate exists. One logical
candidate queues directly; multiple logical candidates open a compact chooser
showing platform and project title. The chosen candidate has a loading state
while queueing and a queued/failed result afterward.

Batch and row-toggle entry points share the same preflight function so behavior
does not diverge.

## Validation

- Unit-test graph semantics separately from file operations.
- Unit-test cache lookup using persisted/reopened metadata cache fixtures.
- Test service projection for disable and invert batches.
- Verify generated Wails bindings, frontend localization/build/lint, core and
  app Go tests/build/vet.
