# Keep disabled mods visible after toggle

## Goal

Keep a local Mod visible in the Manage page immediately after it is disabled or enabled, without requiring a manual refresh.

## Background

The Manage page applies the backend `VersionInfo` response directly to the Pinia snapshot after `ApplyLocalModBatchOperation`. The incremental refresh path currently re-scans only targets ending in `.jar`; disabling a Mod renames it to `.jar.disabled`, so the changed Mod is omitted from that response until a full scan.

## Requirements

- Incremental local-Mod refresh must rescan both enabled `.jar` targets and disabled `.jar.disabled` targets.
- The returned `VersionInfo.Mods` snapshot must retain a disabled Mod after a disable operation, with its path and `Enabled=false` reflecting the renamed file.
- Existing enable, invert, delete, error recovery, and full-refresh behavior must remain unchanged.
- Keep the fix within the existing app service and local-Mod scanning contracts; no frontend or Wails API changes are required.

## Acceptance Criteria

- [x] Disabling a visible Mod leaves it in the Manage page when the enabled-state filter is `all`, marked disabled and using the `.jar.disabled` path, without refreshing the page.
- [x] Enabling a disabled Mod leaves it visible immediately with `Enabled=true` and the `.jar` path.
- [x] Existing local-Mod batch-operation tests pass, including invert and delete behavior.
- [x] A regression test proves the disable operation response includes the disabled Mod before any full refresh.
- [x] Frontend build/lint and relevant Go tests pass.
