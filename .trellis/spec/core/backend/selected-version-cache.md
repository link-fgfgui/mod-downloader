# Selected Version Cache

## Scenario: Refreshing The Selected Version Snapshot

### 1. Scope / Trigger

Use this contract whenever a core workflow rescans local mods or enriches the
currently selected Minecraft instance. The refreshed snapshot must remain the
same across the immediate return value, later getters, and async event updates.

### 2. Signatures

```go
func global.SetSelectedVersion(version minecraft.VersionInfo)
func global.GetSelectedVersion() minecraft.VersionInfo
func global.GetVersionsForDir(dir string) ([]minecraft.VersionInfo, bool)
func (s *appcore.Service) RefreshSelectedVersionMods() minecraft.VersionInfo
```

### 3. Contracts

- `SetSelectedVersion` selects by `ID` when present, otherwise by `Name`.
- When the key already belongs to the active version cache, it must replace
  that entry in both the ordered `versions` slice and every applicable
  `versionMap` alias.
- `GetSelectedVersion`, `GetVersionsForDir`, and emitted
  `selected-version-changed` payloads must expose the same refreshed `Mods`.
- A key not present in the active cache is not inserted implicitly.

### 4. Validation & Error Matrix

- Empty ID and name -> selected lookup remains empty.
- Cache belongs to another Minecraft directory -> selected lookup is empty.
- Selected key is removed by a version-list reload -> selection is cleared.
- Known selected key with rescanned mods -> cached slice and lookup map are
  both updated.

### 5. Good/Base/Bad Cases

- Good: scan a newly added JAR, call `SetSelectedVersion(refreshed)`, then
  `GetSelectedVersion()` returns the new mod.
- Base: selecting an unchanged cached instance refreshes aliases and preserves
  list order.
- Bad: update only `selectedVersionKey`; later getters then return the stale
  object still stored in `versionMap`.

### 6. Tests Required

- `global`: set a cached version with a changed `Mods` slice and assert both
  `GetSelectedVersion()` and `GetVersionsForDir()` return it.
- `appcore`: add a JAR after the initial scan, refresh, then assert a subsequent
  `GetSelectedVersion()` still contains that JAR.
- Run `go test ./...`, `go build ./...`, and `go vet ./...` from `core/`.

### 7. Wrong vs Correct

```go
// Wrong: changes only which stale map entry is selected.
selectedVersionKey = versionKey(version)

// Correct: select the key and replace the known cached snapshot.
selectedVersionKey = versionKey(version)
versions[index] = version
versionMap[version.ID] = version
versionMap[version.Name] = version
```
