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

## Scenario: Incremental Local Mod Refresh

### 1. Scope / Trigger

Use this contract for `fsnotify` events and local enable/disable operations
that update the selected instance without rescanning unrelated JAR files.

### 2. Signatures

```go
func minecraft.ScanModFile(path, instanceID, minecraftVersion, modLoader, pathRoot string) []structs.ModInfo
func (s *appcore.Service) RefreshSelectedVersionMods() structs.VersionInfo
```

### 3. Contracts

- A changed path is removed from the local index before its replacement is
  inserted.
- Both `.jar` and `.jar.disabled` paths are parsed so incremental snapshots
  retain the Mod and accurately expose its enabled state; disabled paths are
  never treated as active by the parser.
- Watcher events are debounced and emitted through the existing
  `selected-version-changed` snapshot event.
- Watchers are bound to the selected instance and stopped on instance change
  or service shutdown.
- Selecting an instance performs its initial full scan and binds its watcher in
  the same service operation. Changing the Minecraft directory stops the old
  watcher before invalidating version state.
- The Manage view may perform one initial full scan for an uninitialized
  directory/instance pair. Re-activating the view must reuse the selected
  snapshot; only manual refresh and explicit recovery paths remain full scans.

### 4. Validation & Error Matrix

- Missing or non-JAR path -> no parsed records; caller may remove stale path.
- Watcher initialization failure -> log warning and retain manual full refresh
  as recovery path.
- Selected instance changed before debounce fires -> ignore stale event batch.

### 5. Good/Base/Bad Cases

- Good: one changed JAR is removed and rescanned while unrelated `ModInfo`
  records remain unchanged.
- Base: manual refresh still performs a complete directory scan.
- Bad: every filesystem event clears all local mods and rehashes all JARs.

### 6. Tests Required

- Assert a changed single file appears in `GetSelectedVersion().Mods`.
- Assert a deleted path disappears while an unchanged path remains.
- Assert watcher events from a previous instance do not update the new one.
- Assert selecting an instance binds its `mods` directory watcher.
- Run the frontend build and lint after changing Manage initialization logic.

### 7. Wrong vs Correct

```go
// Wrong: refresh the complete directory for every one-file event.
s.RefreshSelectedVersionMods()

// Correct: remove and scan only the event path, then publish one snapshot.
global.RemoveLocalModByPath(path)
minecraft.ScanModFile(path, instanceID, version, loader, root)
```
