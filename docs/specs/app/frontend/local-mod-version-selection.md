# Local Mod Version Selection

## Scenario: Replacing An Installed Mod With A Selected Version

### Contracts

- The Manage list displays the local JAR-parsed `version` as its primary value.
  Provider `onlineVersion` and `onlineFileName` belong in the hover text.
- Clicking a matched local version opens the shared `ModVersionList` component.
  Download uses the same component for pin actions; Manage uses it for explicit
  replacement actions.
- Installed rows are highlighted by `onlineVersionId` or SHA1. The installed
  row cannot enqueue a redundant replacement.
- A Manage replacement calls `QueueModDownload` with the selected provider
  `versionId`, selected Minecraft version, and selected loader.
- Keep the selected group snapshot until the dialog leave transition finishes.
- After the replacement queue becomes inactive, refresh selected-instance mods
  so the Manage list reflects the installed file and provider version.

### Validation

- Regenerate Wails bindings after online metadata fields change.
- Run frontend lint and build after changing the shared list or either consumer.
- Verify both pin and replacement modes, including installed highlighting,
  loading, empty results, close/reopen, skipped queue requests, and errors.
