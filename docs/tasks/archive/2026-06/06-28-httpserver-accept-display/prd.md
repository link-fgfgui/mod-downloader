# PRD: HTTP Server Accepted Mods Display on Download Page

## Background

Currently when the browser extension POSTs mod payloads to the HTTP server (`POST /`), the server directly calls `downloader.QueueModDownload()` and auto-downloads each item. The user has no opportunity to review or control what gets downloaded from the extension.

## Requirements

Replace the current auto-download behavior with a display-first flow:

1. **Metadata Resolution**: For each accepted payload, resolve full `ModProject` metadata by platform + id/slug (using the corresponding provider's ExactSearch).

2. **Filter by Current Conditions**: Apply the currently selected Minecraft version and mod loader from the active instance (`global.GetSelectedVersion()`) to determine which resolved projects have matching versions. Only show projects that pass the filter.

3. **Display on Download Page**: Emit the filtered results to the frontend via a Wails event. The frontend receives and displays them in the same `SearchResultList` on the Download page, replacing the current search results.

4. **Auto-Pin Version**: If a payload includes a specific version ID (`file` field) AND that version matches the current filter conditions (version + modLoader), automatically pin it via `database.UpsertPinnedMod`.

## Non-Goals

- The HTTP server should no longer call `QueueModDownload` directly.
- No changes to the health check endpoint.
- No changes to the browser extension payload format.

## Acceptance Criteria

- [ ] HTTP server POST handler resolves metadata instead of queueing downloads
- [ ] Only projects with versions matching current version/modLoader filter appear on the download page
- [ ] Items appear in the SearchResultList with correct metadata (title, icon, description, platform)
- [ ] Download button states reflect correct install status (new/installed/update/conflict)
- [ ] When payload includes a versionID that matches current filters, that version is auto-pinned
- [ ] When payload includes a versionID that does NOT match filters, no pin is created
- [ ] HTTP response to extension indicates accepted/skipped status for each payload
- [ ] Existing search functionality is unaffected
