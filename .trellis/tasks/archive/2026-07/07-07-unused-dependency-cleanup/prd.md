# Unused dependency cleanup in mod manager

## Goal

Improve Local Mod Management so users can safely identify and clean unused library/dependency mods after deleting mods, and can also run the same scan manually. The feature must avoid network-only decisions: local JAR dependency declarations are the base signal, and online metadata is an additional confidence signal for identifying library-style mods.

## Background

- `frontend/src/views/Manage.vue` already supports grouped local mod rows and batch enable, disable, invert, and delete actions through `ApplyLocalModBatchOperation`.
- Deletion currently asks for confirmation, mutates selected files, refreshes the selected instance, and shows only generic success/failure feedback.
- `core/appcore/service.go` owns local mod file validation and mutation. `app.go` only exposes Wails adapter methods.
- `core/minecraft/modparser.go` parses local JAR identity, enabled state, SHA1, and JiJ metadata. It explicitly does not yet parse JAR-embedded dependency declarations.
- `core/appcore.RefreshSelectedVersionMods` already enriches local mods from cached platform metadata by SHA1 and asynchronously resolves Modrinth hash metadata through `providers.ResolveProjectsByHashes`.
- `structs/minecraft.ModInfo` already exposes online display fields and normalized provider categories/tags such as `library`.
- Settings persistence already flows through `core/configs`, `core/appcore.SettingsView`, `app.go`, `frontend/src/stores/settings.ts`, and `frontend/src/views/Settings.vue`.
- Related archived tasks established the local mod bulk operation contract, online metadata display fields, and category/tag display behavior.

## Requirements

1. Add local JAR dependency parsing for supported loaders:
   - Fabric: parse dependency declarations from `fabric.mod.json` where they represent required runtime dependencies.
   - Forge/NeoForge: parse required dependency declarations from `mods.toml` / `neoforge.mods.toml`.
   - Ignore Minecraft, Java, loader/platform pseudo-dependencies, empty IDs, unresolved placeholders, and optional/recommended/incompatible relationships for cleanup purposes.
2. Preserve current local mod identity behavior: parsed dependencies must not turn JiJ entries into install identities, and must not expand `JijMods` into strong local mod rows.
3. Add a core scan API that analyzes the currently selected instance and returns unused dependency candidates without deleting anything.
4. A candidate is unused only when all of these are true:
   - the candidate is present as a local top-level mod file in the selected instance;
   - it looks like a library/dependency mod through online metadata categories/tags containing `library` or equivalent local evidence when online metadata is absent;
   - no remaining enabled top-level local mod requires any declared mod ID from that candidate after excluding selected/deleted files;
   - the scan can explain why it was selected and what evidence was used.
5. Use online metadata when available to improve classification:
   - cached SHA1 metadata and async Modrinth hash resolution may provide categories/tags;
   - a `library` category/tag is a positive classification signal;
   - online metadata absence must not block the pure local dependency graph scan.
6. After a successful delete action in Manage, if automatic cleanup prompts are enabled, run the unused-dependency scan against the refreshed selected instance with the deleted paths excluded from dependent analysis.
7. After delete-triggered analysis completes, notify the user:
   - no candidates found;
   - candidates found and ready for review;
   - scan failed, while preserving the successful delete result.
8. Add an independent Manage page action to scan unused dependencies on demand.
9. Provide a review-and-confirm UI before deleting suggested dependencies. The app must never silently delete candidates.
10. Add a Settings toggle to disable the automatic post-delete scan/prompt. Manual scanning remains available even when the automatic prompt is disabled.
11. Localized Chinese and English text must cover settings, scan button, scan progress, empty result, candidate review, delete confirmation, and errors.
12. The implementation must keep reusable analysis in `core/` and Wails runtime-specific behavior in `app.go` / frontend only.
13. Generated Wails bindings must be updated if new public App methods or structs are exposed.

## Acceptance Criteria

- [ ] Local JAR parsing captures required dependency IDs for Fabric, Forge, and NeoForge fixtures.
- [ ] Dependency parsing ignores optional/recommended/incompatible dependencies and Minecraft/Java/loader pseudo-dependencies.
- [ ] A selected-instance scan returns an empty result when every library-like local mod is still required by at least one enabled top-level mod.
- [ ] A selected-instance scan returns an unused library-like local mod when no remaining enabled top-level mod requires it.
- [ ] A selected-instance scan does not flag ordinary content mods that lack library/dependency evidence, even if no other mod depends on them.
- [ ] A local mod with online category/tag `library` can be classified as a library/dependency candidate.
- [ ] Manual scan button in Manage runs the scan, shows loading state, and presents localized results.
- [ ] Delete-triggered scan runs only after the delete operation succeeds and only when the Settings toggle is enabled.
- [ ] Delete-triggered scan failure does not report the delete as failed; it reports cleanup scan failure separately.
- [ ] Candidate review UI lists candidate name, file path, evidence/reason, and lets the user confirm or cancel cleanup.
- [ ] Confirming cleanup deletes only the reviewed candidate paths through the existing local mod batch delete validation path.
- [ ] Cancelling cleanup leaves all candidate files unchanged.
- [ ] Settings page exposes and persists the automatic unused-dependency cleanup prompt toggle.
- [ ] `go test ./...` passes from the app repo after Wails adapter changes.
- [ ] `go test ./...` passes from `core/` after core changes.
- [ ] Frontend build passes after bindings and UI changes.

## Out of Scope

- Automatically deleting unused dependencies without explicit user confirmation.
- Replacing the existing grouped Manage list design.
- Building a persistent local-mod database for dependency analysis.
- Relying on network-only metadata to decide that a mod is unused.
- CurseForge hash lookup by fingerprint beyond existing cached metadata paths.
- Solving all semantic version constraint compatibility; this task only needs dependency presence/absence by mod ID for cleanup safety.

## Task Structure

Use a single complex task. The deliverables share one cross-layer contract: parser output feeds scan results, scan results feed both manual and delete-triggered Manage UI, and the Settings toggle gates only the automatic prompt. Splitting into child tasks would make the contract harder to review than the implementation.

## Decisions

- Disabled mods do not count as dependents during unused-dependency scans. Only enabled top-level mods keep a dependency alive because disabled mods are not part of the active instance.
