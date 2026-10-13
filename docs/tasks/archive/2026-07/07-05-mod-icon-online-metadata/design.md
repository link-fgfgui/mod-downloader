# Design: Use Online Metadata and Categories for Managed Mod Display

## Boundaries

This is a core metadata-display change with a small Manage-page consumption change.

- `minecraft` continues to parse local JARs and must not import `providers` or `database`.
- `providers` and `database` continue to own online platform metadata.
- `modbridge` remains the convergence point for SHA1-to-platform metadata lookup.
- `downloader` may apply online metadata to the local mod cache at install completion because the download job already has `models.ModProject`.
- `appcore` enriches scanned local mods before returning them to Wails/CLI callers.
- `frontend/src/views/Manage.vue` should choose explicit online display fields when present and fallback to JAR fields otherwise.

## Data Shape

Add unified provider-native categories to `models.ModProject`:

- `Categories []string json:"categories,omitempty"`: provider-native category/tag slugs or names, deduplicated and stable for display.

Populate it in provider converters:

- Modrinth search results: `SearchResult.Categories`.
- Modrinth project details: `Project.Categories` plus `Project.AdditionalCategories`.
- CurseForge mods: `Mod.Categories`, preferring each category's `Slug` and falling back to `Name`.

Add explicit optional online display fields to `structs/minecraft.ModInfo`:

- `OnlineName string json:"onlineName,omitempty"`: display name from `ModProject.Title`.
- `OnlinePlatform string json:"onlinePlatform,omitempty"`: source platform.
- `OnlineProjectID string json:"onlineProjectId,omitempty"`: provider project ID.
- `OnlineSlug string json:"onlineSlug,omitempty"`: provider slug when available.
- `Categories []string json:"categories,omitempty"`: provider-native categories copied from `ModProject.Categories`.

Do not overwrite existing JAR-derived fields. `ID`, `Name`, `Version`, `Description`, `FileName`, `Path`, `SHA1`, `Enabled`, and `JijMods` remain the local technical record. The display rule becomes:

```text
displayName = onlineName || name || id
displayIcon = iconUrl || fallback icon
displayCategories = categories when present
```

This keeps internal behavior stable while making the Manage display prefer online metadata and provider-native categories.

## Metadata Application

Create a focused helper that applies a `models.ModProject` to `ModInfo` display metadata:

1. Trim and normalize platform/project fields.
2. If `ModProject.Title` is present, set `OnlineName`.
3. If `ModProject.IconURL` is present, set `IconURL`.
4. Set `OnlinePlatform`, `OnlineProjectID`, and `OnlineSlug` from the project.
5. Copy provider-native categories from `ModProject.Categories`.
6. Leave JAR-derived fields unchanged.

The helper should be accessible from code that already bridges online and local metadata. Avoid making `minecraft` depend on online packages.

## Data Flow

Download path:

1. `QueueModDownload` receives a search result with `models.ModProject`.
2. `downloadModJob` installs or hardlinks the JAR.
3. `upsertDownloadedMod` parses the JAR for local identity fields.
4. Before `global.UpsertLocalMod`, apply the download job's `ModProject` so newly installed local records already carry online display metadata.
5. Persist or preserve platform metadata where existing database APIs allow it, so future refreshes can recover the same metadata by SHA1.

Refresh/scan path:

1. `minecraft.ScanModsDir` returns JAR-derived `ModInfo`.
2. `appcore` enriches those records by SHA1 through `modbridge.PlatformMetadataForSHA1`.
3. If metadata is found, apply all online display fields, not only `IconURL`.
4. Keep the existing asynchronous Modrinth hash resolution behavior for misses, but update it to apply online name/categories as well as icon.

Frontend path:

1. `Manage.vue` uses `onlineName || name || id` for row title.
2. Existing copy ID/name helpers keep using declared mod IDs, and the row subtitle keeps showing JAR-derived technical details.
3. It renders category chips from `categories` after the existing subtitle details.
4. The category chip strip must be layout-aware:
   - keep chips on one horizontal line in the Manage row;
   - show as many chips as fit after the subtitle details;
   - if categories exist and at least one chip can fit, keep the first fitting chip visible;
   - hide overflow chips from the row and expose the hidden category names through hover text;
   - if subtitle details consume the available row width so no chip fits, allow horizontal scrolling for the subtitle/category row instead of dropping all category chips.

The implementation can satisfy this with measured chip visibility or an equivalent CSS/layout approach, but it must avoid text overlap and row-height churn in the virtual list.

## Compatibility

Adding optional JSON fields to `ModProject` and `ModInfo` is backward-compatible for Wails and CLI consumers. No public Wails method signature should change.

Because persisted platform metadata stores `models.ModProject`, adding `Categories` to that nested model is expected to be compatible with the existing cache shape. If `cacheState` itself changes, bump `database.cacheVersion`.

## Trade-Offs

This design does not remove JAR parsing. That parsing still protects install status, replacement, conflict detection, JiJ grouping, and local file indexing. The change is to stop using JAR-parsed display fields when online metadata exists and to add provider category chips without destabilizing the fixed-height Manage list rows.
