# Export single favorite as packwiz zip

## Goal

Add a user-facing export action for one selected favorite list that writes a ZIP containing a packwiz-compatible modpack. The exported pack should let users move a curated favorite list into packwiz tooling without downloading mod jars into the ZIP.

The feature is scoped to a single favorite list at a time, not bulk export of all favorite lists.

## Background

- Favorite items already store the fields needed to resolve online versions: `platform`, `modId`, optional `versionId`, `minecraftVersion`, and `modLoader` in `core/database/mods.go:38`.
- The appcore service exposes favorite list CRUD and item listing in `core/appcore/service.go:292` and `core/appcore/service.go:326`.
- Wails currently exposes favorite APIs through `app.go:139`, `app.go:155`, and related methods, but no export API exists.
- The Favorites page has a selected-list header and refresh action at `frontend/src/views/Favorites.vue:45`, which is the natural placement for an export action.
- The frontend favorites store imports only favorite CRUD/list APIs today at `frontend/src/stores/favorites.ts:1`.
- Version resolution can reuse `modbridge.ResolveVersion`, which honors an explicit version ID first, then pins, then best matching provider version (`core/modbridge/modbridge.go:34`).
- Resolved `models.ModVersion` includes file name, download URL, SHA1, project ID, platform, and dependency data (`core/models/models.go:24`).
- Provider version lookups and refresh paths already filter by Minecraft version and mod loader (`core/providers/service.go:200` and `core/providers/service.go:215`).
- Packwiz reference files are available locally:
  - `/home/link/Downloads/packwiz-spec-1.zip`
  - `/home/link/Downloads/packwiz-example-pack-1.zip`
  - `/home/link/Downloads/packwiz-website-main.zip`
- Packwiz example output uses `pack.toml`, `index.toml`, and per-mod metafiles under `mods/*.pw.toml`.
- Packwiz schemas require `pack.toml` to include `pack-format`, `name`, `index`, and `versions.minecraft`; each mod metafile requires `name`, `filename`, and `[download]` with `url`, `hash`, and `hash-format`.

## Requirements

1. Add an export action for the currently selected favorite list on the Favorites page.
2. The export action must ask the user for a `.zip` destination and cancel cleanly if no path is chosen.
3. The backend must generate a ZIP with this root structure:
   - `pack.toml`
   - `index.toml`
   - `mods/<safe-mod-name>.pw.toml` for every exported favorite item
4. `pack.toml` must use `pack-format = "packwiz:1.1.0"` and include an `[index]` section whose hash is the SHA-256 hash of the generated `index.toml`.
5. `index.toml` must use `hash-format = "sha256"` and list every generated mod metafile with `metafile = true` and the SHA-256 hash of that metafile.
6. Each mod metafile must include:
   - `name`
   - `filename`
   - `side = "both"` unless a future field provides a better side value
   - `[download]` with `url`, `hash-format = "sha1"`, and `hash`
   - `[update.modrinth]` for Modrinth items with `mod-id` and `version`
   - `[update.curseforge]` for CurseForge items with numeric `project-id` and `file-id`
7. Export must resolve every favorite item to a concrete provider version. If a favorite item has `versionId`, that exact version should be used when available; otherwise resolve the best matching version for the export's Minecraft version and mod loader.
8. Export scope must be derived from the favorite items' stored `minecraftVersion` and `modLoader` fields. The exporter must require all exported items to share one non-empty Minecraft version and one non-empty mod loader.
9. If a favorite list contains mixed or missing Minecraft/mod loader scope values, export must fail with a clear frontend-visible error. This task must not implement the separate TODO of filtering/displaying favorite lists by the selected Minecraft version + modloader tuple.
10. Export must fail with a clear frontend-visible error when any selected-list item cannot be resolved to a downloadable version with filename, URL, and SHA1. Do not silently create a partial pack.
11. The feature must handle duplicated or unsafe filenames by generating unique, packwiz-safe metafile names while preserving the actual jar filename in each mod metafile.
12. The frontend must expose busy/success/error states and avoid triggering concurrent exports for the same list.
13. Tests must cover packwiz TOML/ZIP generation, scope validation, resolution failures, filename/path sanitization, and the Wails-facing export path at the appropriate layer.

## Acceptance Criteria

- [ ] From a non-empty favorite list, a user can click Export, choose a ZIP destination, and get a `.zip` file on disk.
- [ ] The ZIP contains valid `pack.toml`, `index.toml`, and `mods/*.pw.toml` files; it does not contain mod jar binaries.
- [ ] `pack.toml` references `index.toml` with a correct SHA-256 hash.
- [ ] `index.toml` references every generated metafile with correct SHA-256 hashes and `metafile = true`.
- [ ] Each generated mod metafile has a valid download URL, SHA1, filename, side, and provider update metadata where the source platform supports it.
- [ ] Export succeeds only when the favorite list's items share a single Minecraft version and mod loader scope.
- [ ] If one or more favorites cannot be resolved, no partial ZIP is left behind and the user sees an actionable error.
- [ ] Mixed-scope favorite lists fail clearly without adding tuple-based favorite filtering/display behavior in this task.
- [ ] Empty favorite lists cannot export a meaningless pack; the user sees a clear disabled state or error.
- [ ] Existing favorite CRUD and download flows continue to work.
- [ ] Automated tests pass for backend generation/resolution behavior and existing relevant frontend checks.

## Out of Scope

- Exporting all favorite lists at once.
- Downloading or embedding jar files in the packwiz ZIP.
- Editing optional mod state, client/server side, loader version, author, or pack version from the UI in this task.
- Exporting CurseForge `.zip` or Modrinth `.mrpack` formats.
- Filtering or partitioning favorites by selected Minecraft version + modloader tuple.
- Adding new persistent favorite-list metadata unless required by the implementation review.
