# Favorite bulk copy operations

## Goal

Expose backend operations for adding selected favorite mods, copying a whole favorite list, and adding live references between lists.

## Requirements

- Add appcore methods for copying selected favorite mods to one or more target lists.
- Add appcore methods for copying all resolved source-list mods to another list as concrete rows.
- Add appcore methods for adding/removing/listing live favorite-list references.
- Preserve duplicate upsert semantics: duplicate favorite keys update metadata instead of creating duplicate rows.
- Keep Wails adapter methods thin delegations.

## Acceptance Criteria

- [x] Selected favorite mods can be copied to one or more target lists.
- [x] Copying a whole list writes concrete mod rows to the target list.
- [x] Adding a list reference creates a live relationship without duplicating source rows.
- [x] Removing a list reference does not remove source or target list mods.
- [x] Existing single-mod `AddFavoriteMod` and `RemoveFavoriteMod` behavior remains compatible.
- [x] Core service tests cover selected-mod copy, whole-list copy, reference add/remove, duplicate handling, and missing-list behavior.

## Dependencies

- Requires `07-07-favorite-sql-storage-references`.
