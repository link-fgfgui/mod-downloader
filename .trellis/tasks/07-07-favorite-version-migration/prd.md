# Favorite version migration

## Goal

Let users migrate a favorite list to a different Minecraft version and modloader while previewing missing target versions before any write.

## Requirements

- Preview migration from a source list to a target list/scope.
- Resolve source list contents including live references.
- For every favorite mod, find a matching target-scope project version.
- Report resolvable mods and missing/conflicting mods.
- Applying without `ignoreConflicts` must abort if conflicts exist.
- Applying with `ignoreConflicts` writes only resolvable target-scope rows and reports skipped conflicts.

## Acceptance Criteria

- [ ] Preview returns matched and missing rows without writing favorite data.
- [ ] Apply without ignore conflicts writes nothing when any conflict exists.
- [ ] Apply with ignore conflicts writes matched rows and skips missing rows.
- [ ] Migrated rows use target `minecraftVersion`, target `modLoader`, and matched `versionId`.
- [ ] Source favorite rows remain unchanged.
- [ ] Tests cover all-match, partial-conflict, all-conflict, missing project lookup, and referenced-list source contents.

## Dependencies

- Requires `07-07-favorite-sql-storage-references`.
- Should run after `07-07-favorite-bulk-copy-operations` if it reuses bulk write result types.
