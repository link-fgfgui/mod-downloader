# Cross-Layer Thinking Guide

Use this guide when a change crosses Vue, Wails, `appcore`, providers,
downloader, storage, or Minecraft parsing. Most regressions in this project
occur at those boundaries.

## Before Editing

1. Trace the complete path: `view/store -> generated binding -> App ->
   Service -> lower layer -> event/UI`.
2. Identify the owner of each field and transformation. Shared provider data
   belongs in `core/models`; requests/responses belong in `core/structs`;
   persistence belongs in `storage`.
3. Define input, output, normalization, errors, cancellation, and event names
   at every boundary before changing code.
4. Search all consumers of changed fields, constants, event kinds, and JSON keys.
5. Verify a representative value survives the full write/read or
   request/event round trip without losing identity, null/empty distinctions,
   ordering, or normalization.

## Project Boundary Rules

- `app.go` owns Wails runtime calls, native dialogs, frontend method names, and
  event-name translation. `core` never imports Wails runtime.
- `appcore.Event` is adapter-neutral; Wails maps it to runtime events.
- `models` is the sole source of truth for `ModProject`, `ModVersion`, and
  dependency types. Do not add aliases or provider re-exports.
- `providers` and `minecraft` must not depend on each other. `modbridge` is
  their convergence point for local/install-state interpretation.
- Blocking download preflight belongs inside the queue context so
  `CancelDownload` can stop it before dependencies or files are enqueued.
- Native dialog cancellation is a successful no-op result, not an error.

## Contract Checklist

- [ ] Frontend entry point, Wails method, service method, and lower-layer owner identified
- [ ] Every changed field has one canonical type/decoder
- [ ] Error and cancellation behavior defined at each boundary
- [ ] Event payload and lifecycle behavior traced in both directions
- [ ] Derived state retains the source identity (`id`, `version`, queue ID, or
      composite key) instead of inventing an unrelated cursor
- [ ] Tests cover empty, invalid, canceled, and fallback paths where applicable
- [ ] No lower layer imports an adapter-only package

## Common Failure Patterns

### Scattered payload parsing

If two consumers cast the same untyped payload, add one typed decoder or
projection at the owning boundary. UI code should render the projection, not
redefine the contract.

### Duplicate state ownership

Do not let both a store and `appcore` invent independent queue, selection, or
cache state. Pick one source of truth, then emit a typed update.

### Unchecked external mode decisions

Distinguish not-found from transient provider errors. A timeout must not
silently select a fallback source or make a cache look authoritative. Reset
prefetched/cache state when the provider or source changes, and keep shortcut
paths subject to the same error handling as the normal path.

When parsing remote metadata, consume the complete response or use a streaming
decoder; never treat a fixed-size prefix as complete JSON. When reconstructing
a composite ID, assert every component and its position, for example
`platform:projectID` rather than a partially rebuilt key.

### Weak JIJ identity

`ModInfo.JijMods` is display metadata. Use `minecraft.PrimaryModIDs(mods)` for
install identity, conflict detection, archive selection, and version storage.
Never expand JIJ IDs into the local index or `SetVersionModIDs`. Archive
candidates come from `LocalModPathsForModIDs(primaryModIDs, instanceID)`;
partial top-level overlap remains a real conflict and must not be filtered out.

## When To Add Flow Documentation

Add a focused contract file when a workflow spans three or more layers, has a
non-trivial payload, or has caused a regression. Link it from the owning
package index and keep examples tied to real files.
