# Implementation Plan

## Checklist

1. Load pre-development specs with `trellis-before-dev` before editing code.
2. Add `Categories []string` to `core/models.ModProject`.
3. Populate `ModProject.Categories` in Modrinth converters from `SearchResult.Categories`, `Project.Categories`, and `Project.AdditionalCategories`.
4. Populate `ModProject.Categories` in CurseForge converters from `Mod.Categories`, preferring category slugs and falling back to names.
5. Add optional online display/category fields to `core/structs/minecraft.ModInfo`.
6. Add a focused helper to apply `models.ModProject` to `ModInfo` online display fields and categories without mutating JAR-derived identity fields.
7. Wire download-time enrichment in `core/downloader/upsertDownloadedMod` before `global.UpsertLocalMod`.
8. Ensure the download path preserves enough platform metadata for later SHA1-based enrichment on refresh.
9. Replace or broaden `appcore.enrichModIcons` so it enriches online name/icon/categories, not only icon URL.
10. Update the async Modrinth SHA1 resolution path to apply the same online display/category helper.
11. Update `frontend/src/views/Manage.vue` to display `onlineName || name || id`, keep the existing JAR-derived subtitle details, and render provider category chips after those details.
12. Add Manage category overflow behavior:
   - one-line chip strip inside the existing virtual-list row height;
   - show as many chips as fit;
   - hide overflow chips and expose hidden names in hover text;
   - keep at least one chip visible when categories exist and one can fit;
   - use horizontal scrolling when subtitle details leave no room for a single chip.
13. Add or update tests for:
   - provider converters populating unified categories,
   - online metadata preferred over JAR name in enriched `ModInfo`,
   - icon and categories populated from `ModProject`,
   - fallback to JAR fields when metadata is missing,
   - JAR-derived IDs/SHA1/JiJ fields preserved for internal behavior.
14. Run verification:
   - `go test ./...` from `core/`
   - app-level `go test ./...` if app-facing code changes
   - frontend type/lint check if existing project scripts support it
   - responsive visual check of Manage rows with many categories and long subtitle content.

## Risk Points

- Do not remove JAR parsing; only stop preferring JAR display fields when online metadata exists.
- Do not overwrite `ModInfo.ID`, `Name`, `Version`, `SHA1`, `Path`, or `JijMods` if existing logic depends on them.
- Do not make `minecraft` import `providers` or `database`.
- Do not physically rename installed `.jar` files.
- If persistent database structs change, bump `database.cacheVersion`.

## Review Gate Before Start

Confirm category chip presentation: recommended default is display up to a small capped number of provider category strings as compact chips, preserving the existing JAR-derived subtitle.

The overflow behavior is decided: preserve existing subtitle details, show fitting category chips, hide overflow in hover text, and fall back to horizontal scrolling if no category chip fits after the subtitle content.
