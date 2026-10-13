# Use online metadata and categories for managed mod display

## Goal

The Manage page should display local mods from online metadata once that metadata is available. Today the row name comes from metadata parsed out of the JAR, while the icon is only enriched by trying platform metadata. The desired behavior is stronger: when online metadata is known for a local JAR, Manage should use the online project metadata for the mod's displayed name, icon, and provider-native categories/tags instead of the JAR-parsed display fields.

JAR parsing can still exist for internal identity and safety checks, such as SHA1, declared mod IDs, enabled state, JiJ grouping, update/conflict detection, and replacement logic. It should no longer be the preferred display source after online metadata has been found.

## Confirmed Facts

- `frontend/src/views/Manage.vue` currently renders `group.primary.name || group.primary.id` as the row title and renders `group.primary.iconUrl` when available.
- `structs/minecraft.ModInfo` currently contains JAR-derived fields such as `ID`, `Name`, `Version`, `Description`, `FileName`, `Path`, `SHA1`, `Enabled`, and `JijMods`, plus an optional `IconURL`.
- `appcore.RefreshSelectedVersionMods` and `LocalModsInDir` already enrich scanned local mods with icon URLs by SHA1 through `enrichModIcons`.
- `models.ModProject` already carries online metadata fields including `Platform`, `ProjectID`, `Slug`, `Title`, `IconURL`, `Description`, and `Downloads`.
- `models.ModProject` does not currently carry unified category/tag metadata.
- The local SDKs expose provider-native categories/tags:
  - Modrinth `SearchResult.Categories []string`.
  - Modrinth `Project.Categories []string` and `Project.AdditionalCategories []string`.
  - CurseForge `Mod.Categories []Category`, where `Category` includes `Name` and `Slug`.
- `downloader.downloadJob` already carries both the resolved `models.ModVersion` and the originating `models.ModProject`, so a just-downloaded mod can be enriched from request metadata without waiting for a later SHA1 lookup.
- Local JAR metadata and local mod path indexes are memory-only in `global`; platform metadata is persisted in `database`.

## Requirements

1. When online metadata is available for a local mod, the Manage page row title must use the online project title, not the JAR-parsed `ModInfo.Name`.
2. When online metadata is available, the Manage page icon must use the online project icon URL.
3. Unified online project metadata must include provider-native categories/tags, normalized into a common `models.ModProject.Categories []string` field.
4. Modrinth converters must populate unified categories from search-result categories and from project categories plus additional categories.
5. CurseForge converters must populate unified categories from `Mod.Categories`, preferring category slugs and falling back to names.
6. When online metadata is available, Manage must display provider-native categories/tags such as `library`, `technology`, or `magic`.
7. Category display must handle many tags without breaking the Manage row layout:
   - keep existing JAR-derived subtitle details visible before category chips;
   - show as many category chips as fit in the remaining row space;
   - if any categories exist, show at least one category chip when there is enough row space for one;
   - hide overflow categories from the row and expose hidden categories in hover text;
   - if the preceding subtitle content leaves no room for even one category chip, allow the subtitle/category row to scroll instead of dropping all categories.
8. Download-time local mod records must be enriched from the `ModProject` carried by the download job so newly installed mods do not wait for a later refresh to display online metadata.
9. Refresh/scanned local mod records must use cached or resolvable online metadata by SHA1 so metadata-backed display survives app restart and manual refresh.
10. If online metadata is unavailable, Manage must keep the existing fallback behavior based on JAR-parsed fields.
11. JAR-derived identity fields must remain visible as technical details in the Manage subtitle and remain available for internal logic and existing user actions that depend on declared mod IDs, SHA1s, JiJ data, paths, and enabled state.
12. The implementation must stay in `core/` for metadata merging and local mod data shape. Frontend changes should consume explicit typed fields and should not reconstruct provider categories from unrelated raw fields.

## Acceptance Criteria

- [ ] For a local mod with resolved online metadata, Manage displays the online project title instead of the JAR-declared name.
- [ ] For a local mod with resolved online metadata, Manage displays the online icon URL.
- [ ] `models.ModProject` has a unified category/tag field populated by both Modrinth and CurseForge converters.
- [ ] For a local mod with resolved online metadata, Manage displays provider-native categories/tags from the unified metadata.
- [ ] Manage category chips do not overflow the visible row: excess categories are hidden from the row and available through hover text.
- [ ] When categories exist and there is room for one chip, at least one category chip remains visible.
- [ ] When existing subtitle content leaves no room for one category chip, the subtitle/category row offers horizontal scrolling so categories are still reachable.
- [ ] Immediately after a search-result install completes, the local mod cache contains the online display fields and categories from the install's `ModProject`.
- [ ] After refreshing selected mods, a local JAR whose SHA1 maps to cached platform metadata uses the online title/icon/categories.
- [ ] A local JAR with no online metadata still displays using the current JAR-derived fallback.
- [ ] Existing install status, conflict detection, replacement archiving, hardlink installation, copy IDs, and JiJ grouping behavior keep working.
- [ ] Core tests cover online metadata preference and fallback behavior.
- [ ] `go test ./...` passes inside `core/`; app-level checks run if app-facing generated model surface or frontend behavior is touched.

## Out of Scope

- Physically renaming installed `.jar` files.
- Removing JAR parsing entirely.
- Adding a persistent local-mod database.
- Re-fetching platform metadata on every local scan when cache/hash resolution is absent.
- Redesigning the Manage page beyond consuming and showing the corrected display fields and category chips.

## Decisions

- JAR-derived declared mod IDs/version/file details stay visible in the Manage subtitle when online metadata exists.
- Tags in this task mean provider-native categories/tags from the online metadata, normalized into unified `models.ModProject.Categories`. They are not source labels like platform/project ID.
- Category chips are appended after existing subtitle details. Overflow chips are hidden from the row and surfaced through hover text; if no chip can fit after the subtitle details, the row scrolls horizontally rather than hiding every category.
