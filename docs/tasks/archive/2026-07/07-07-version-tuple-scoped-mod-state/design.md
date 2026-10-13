# Design

## Boundaries

- Keep backend persistence unchanged. `PinnedMod` and `FavoriteMod` already include `minecraftVersion` and `modLoader` in their key contracts.
- Move active download tuple state into `minecraftStore` so sidebar controls and pages consume one source of truth.
- Keep `downloadSearch` focused on search results, download states, overlays, and queue operations. It should read the active tuple from setters/watchers rather than owning the UI controls.

## Data Flow

1. Sidebar `VersionChoose.vue` lists launcher versions from `minecraftStore.versions`, release MC versions from `minecraftStore.releaseVersions`, and known modloaders.
2. Selecting a launcher version calls `minecraftStore.selectVersion(versionKey)`.
3. `minecraftStore.applySelectedVersion` stores the selected instance and derives `selectedMinecraftVersion` / `selectedModLoader` from instance metadata.
4. Editing sidebar MC version or modloader calls a manual setter on `minecraftStore` that updates the tuple and clears `selectedVersion`.
5. `Download.vue` watches the `minecraftStore` tuple and calls `downloadStore.setTargetTuple(mcVersion, modLoader)`.
6. `downloadSearch` uses that tuple for search requests, state requests, installs, pin/unpin operations, and favorite drafts.
7. `Manage.vue` continues to read local mods from `minecraftStore.selectedVersion`; when manual tuple edits clear that value, its no-instance guard prevents stale local operations.

## Compatibility

- Existing Wails method signatures stay unchanged, so generated bindings should not be regenerated.
- Existing SQLite rows remain valid because the persistent tuple key already exists.
- Existing favorite lists and pinned mod records remain visible and removable by their full tuple.

## Trade-Offs

- The sidebar becomes the single tuple-control location, which avoids duplicated selectors but means Download page watchers must synchronize from `minecraftStore`.
- The selected launcher version is cleared on manual tuple edits. This is stricter than keeping the previous instance while only changing search filters, but it prevents local-management actions from implying the wrong instance.
- Download buttons can rely on backend download state disabled/skipped behavior when tuple fields are empty, but the frontend store should avoid refresh calls that would falsely show actionable states for incomplete tuples.
