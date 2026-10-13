# Design: Export single favorite list as packwiz ZIP

## Architecture

Add a small packwiz exporter in the Go core/app layer and expose it through Wails.

Proposed boundaries:

- `core/appcore`: orchestration API, favorite list lookup, version resolution, and export result/error shape.
- New focused package or file under `core/appcore` or `core/packwiz`: pure generation of packwiz TOML file contents and ZIP assembly from resolved entries.
- `app.go`: Wails method that opens a save-file dialog and delegates to appcore.
- `frontend/src/stores/favorites.ts`: export action state and Wails call wrapper.
- `frontend/src/views/Favorites.vue`: export button in the selected-list header.

Keep packwiz file generation testable without Wails runtime. Wails should only choose the destination path and translate errors for the frontend.

## Data Flow

1. User selects a favorite list on the Favorites page and clicks Export.
2. Frontend calls a Wails method such as `ExportFavoriteListPackwizZip(listId)`.
3. Wails opens a save-file dialog defaulting to a safe version of the favorite list name plus `.zip`.
4. Appcore loads the favorite list and favorite items.
5. Appcore determines the export scope:
   - Favorite item `minecraftVersion` values must collapse to one non-empty Minecraft version.
   - Favorite item `modLoader` values must collapse to one non-empty mod loader.
   - Mixed or missing scope values fail export with a user-visible error.
   - Favorite item `versionId` remains the exact-version override for that item.
6. For each favorite item, appcore builds a `ModDownloadRequest` and calls `modbridge.ResolveVersion`.
7. Appcore validates each resolved version has `FileName`, `DownloadURL`, and `SHA1`.
8. Packwiz generator creates:
   - per-mod metafile TOML bytes
   - `index.toml` bytes referencing metafiles with SHA-256
   - `pack.toml` bytes referencing `index.toml` with SHA-256
   - final ZIP bytes or writes directly to a temp file
9. Appcore writes to a temp file next to the destination and renames it into place only after successful ZIP generation.
10. Frontend shows success, cancellation, or error.

## Packwiz Contract

Generate packwiz format `packwiz:1.1.0`.

`pack.toml`:

- `name`: favorite list name
- `author`: omitted for MVP unless a safe app/user value is already available
- `version`: omitted or defaulted to `1.0.0`; prefer omitted unless tests show tools expect it
- `pack-format`: `packwiz:1.1.0`
- `[index]`: `file = "index.toml"`, `hash-format = "sha256"`, `hash = <sha256(index.toml)>`
- `[versions]`: `minecraft = <favorite item scope minecraft version>`; loader key only if a concrete loader version is available

`index.toml`:

- `hash-format = "sha256"`
- one `[[files]]` per generated metafile
- `file = "mods/<safe>.pw.toml"`
- `hash = <sha256(metafile)>`
- `metafile = true`

`mods/*.pw.toml`:

- `name`: favorite title, slug, or mod ID
- `filename`: resolved jar filename
- `side`: `both`
- `[download]`: direct URL, `sha1`, resolved SHA1
- `[update.modrinth]`: string project ID plus version ID when platform is Modrinth
- `[update.curseforge]`: numeric project ID plus numeric file ID when platform is CurseForge

## Compatibility and Errors

- Do not include jar files; packwiz metafiles reference external downloads.
- Do not write partial packs. Write via temp file and rename on success.
- Reject empty lists, mixed/missing favorite item scope, unresolved versions, missing download URLs, missing SHA1 values, and invalid destination paths with explicit errors.
- If multiple favorite items resolve to the same jar filename, keep both metafiles only if they refer to distinct favorite entries; metafile paths must remain unique.
- If a CurseForge ID cannot be parsed as a number, omit the update block only if the download metadata is otherwise valid; log a warning. Modrinth update metadata remains string based.

## Trade-offs

- Inferring pack scope from favorite items keeps this task independent from the separate TODO of displaying favorites by selected Minecraft version + modloader tuple, but mixed favorite lists must fail instead of exporting a partial or ambiguous pack.
- Adding loader-version support would make `pack.toml` more complete. The current `VersionInfo` exposes loader name but not loader version, while raw version manifests contain patch versions. This can be added if the user wants strict packwiz installer setup in MVP.

## Validation Strategy

- Unit-test TOML generation and SHA hashing with deterministic fixture entries.
- Unit-test ZIP assembly by opening the generated ZIP and checking file paths/content.
- Unit-test resolution validation failures with fake entries or isolated generator inputs.
- Run Go tests for touched packages and frontend type/lint/build checks after implementation.
