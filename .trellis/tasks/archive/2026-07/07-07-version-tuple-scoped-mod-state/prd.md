# Version tuple scoped mod state

## Goal

Move Minecraft version / modloader selection out of the Download page and into the sidebar, make the sidebar version selector the source of truth for auto-selecting its MC version and modloader, and ensure download/local-management pin and favorite behavior is scoped by the active `(minecraftVersion, modLoader)` tuple.

The user value is predictable state separation: switching from one loader/version tuple to another should not leak pins, favorites, download button state, or local management assumptions across tuples.

## Confirmed Facts

- `frontend/src/views/Download.vue` currently owns separate `selectedVersion` and `selectedModLoader` selects.
- `frontend/src/components/SideBar/VersionChoose.vue` already owns launcher instance selection and can derive `minecraftVersion` / `modLoader` from `minecraftStore.selectedVersion`.
- Download search, version matching, pinning, and queueing consume `downloadSearch.selectedVersion` and `downloadSearch.selectedModLoader`.
- Backend pin persistence is already keyed by `platform/modID/minecraftVersion/modLoader` through `database.PinnedMod` and `GetPinnedMod` / `UpsertPinnedMod`.
- Backend favorite persistence is already keyed by `listID/platform/modID/minecraftVersion/modLoader` through `database.FavoriteMod`.
- Local mod management is already instance-oriented through `minecraftStore.selectedVersion`, with a no-instance state in `Manage.vue`.

## Requirements

- R1: Remove the MC version and modloader controls from the Download page.
- R2: Add MC version and modloader controls to the sidebar version area.
- R3: Selecting a launcher version/instance in the sidebar must automatically populate the sidebar MC version and modloader fields from that instance.
- R4: Manually changing the sidebar MC version or modloader must clear the selected launcher version/instance so tuple-scoped actions are no longer tied to the previous instance.
- R5: Download search, version overlay, download button states, queueing, pinning, batch unpin, and add-to-favorites from the Download page must use the sidebar tuple.
- R6: Favorite drafts created from the local Manage page must use the active sidebar tuple consistently, falling back to the selected local instance metadata when present.
- R7: Pinned mod and favorite UI behavior must treat different `(minecraftVersion, modLoader)` tuples as independent scopes.
- R8: If no launcher version/instance is selected, local mod management must keep showing the existing no-instance/empty behavior instead of operating on stale mods.
- R9: If the active tuple lacks either MC version or modloader, download actions must not queue downloads. It is acceptable for the normal disabled button-state path to enforce this rather than adding duplicate UI guards everywhere.

## Acceptance Criteria

- [ ] Download page no longer renders MC version or modloader selects.
- [ ] Sidebar version area renders launcher version/instance, MC version, and modloader controls.
- [ ] Choosing a launcher version/instance sets MC version and modloader from that instance.
- [ ] Editing MC version or modloader clears the selected launcher version/instance and clears local mods tied to the previous instance.
- [ ] Download search requests use the sidebar MC version and modloader.
- [ ] Download state, install, pin, batch-unpin, and favorite draft calls include the same active tuple.
- [ ] Favorite additions from Manage use the same tuple key behavior as Download and do not collapse entries from different loader/version tuples.
- [ ] Pinned mod and favorite list displays/removals continue to identify entries by the full tuple.
- [ ] Frontend build/type check passes.

## Notes

- Keep `prd.md` focused on requirements, constraints, and acceptance criteria.
- Lightweight tasks can remain PRD-only.
- For complex tasks, add `design.md` for technical design and `implement.md` for execution planning before `task.py start`.
