# Local Mod Online Version Metadata

## Scenario: Enriching Parsed JAR Metadata

### Contracts

- `structs/minecraft.ModInfo.Version` is parsed from the local JAR and must not
  be replaced by provider metadata.
- Provider metadata is stored separately in `OnlineVersion`,
  `OnlineVersionID`, and `OnlineFileName`.
- Cache lookup, asynchronous provider resolution, and install-time local index
  updates must all apply both the matching `models.ModProject` and
  `models.ModVersion` through `modbridge.ApplyPlatformMetadataToModInfo`.
- Local scans calculate raw-file SHA1 for Modrinth plus CurseForge's
  whitespace-normalized MurmurHash2 fingerprint. Missing metadata resolution
  submits each identity only to the provider that understands it.
- Modrinth SHA1 results take precedence when both providers identify the same
  file; CurseForge fills files not matched by Modrinth.
- Successful provider lookups that return no match are cached per provider and
  identity for 24 hours. Modrinth keys by normalized SHA1 and CurseForge keys
  by decimal fingerprint. Concurrent local-metadata resolutions are serialized
  so a waiting refresh observes the first request's result or negative cache.
- Provider errors and unavailable provider clients are not negative matches and
  must not be cached. After the negative-cache TTL expires, the identity is
  eligible for remote resolution again.
- A local mod waiting on remote resolution sets `OnlineMetadataLoading=true` in
  the initial selected-version event. Resolution always clears it and emits the
  updated selected version, including when neither provider finds a match.
- SHA1 and online version ID identify the installed provider version. Empty
  online fields mean the provider match is unavailable; they do not invalidate
  parsed local metadata.

### Tests Required

- Verify provider enrichment preserves the parsed local ID, name, and version.
- Verify cached SHA1 enrichment exposes the online version, version ID, and
  provider filename.
- Verify CurseForge fingerprint calculation ignores spaces, tabs, CR, and LF
  while SHA1 still covers the original bytes.
- Verify CurseForge exact fingerprint matches fetch the project, retain the
  matching file SHA1, and populate the shared storage cache.
- Verify both matched and unmatched resolution clear the loading flag without
  changing the JAR-parsed version.
- Verify two immediate resolutions of an unmatched identity issue one provider
  request, an expired miss retries, and a provider error is retried rather than
  cached.
- Run core and consuming-app tests after changing `ModInfo`, then regenerate
  Wails bindings.

## Scenario: JIJ-Aware Local Dependency Safety

### 1. Scope / Trigger

Use this contract when local mod management decides whether a standalone
library is unused, or when an enable/disable/invert operation needs to project
whether required dependencies remain satisfied.

JIJ metadata is a weak local dependency provider only. It remains excluded
from install status, conflict detection, archive selection, update matching,
and `SetVersionModIDs`; those strong-reference consumers continue to use
`minecraft.PrimaryModIDs`.

### 2. Signatures

```go
func (s *Service) AnalyzeLocalModDisableImpact(
    req structs.LocalModDisableImpactRequest,
) (structs.LocalModDisableImpactResult, error)

func (s *Service) RestoreCachedDependency(
    req structs.RestoreCachedDependencyRequest,
) structs.ModDownloadResult

func storage.FindCachedModVersionsByModID(
    modID, minecraftVersion, modLoader string,
) []storage.CachedModVersionCandidate
```

Public Wails adapter methods forward the two service methods with the same
request/response structs from `core/structs/localmods.go`.

### 3. Contracts

- Group parsed `ModInfo` rows by physical JAR path before dependency analysis.
- An enabled JAR provides its normalized top-level IDs plus normalized
  `JijMods[].ID` values. A disabled or projected-disabled JAR provides neither.
- JIJ dependency matching is case-insensitive by mod ID only. Do not compare
  versions because `JijModInfo` intentionally stores only ID and name.
- Only dependency types `required` and the parser's empty required default
  participate.
- Disable/invert preflight reports only dependencies that were satisfied before
  the operation and become missing afterward. Pre-existing missing dependencies
  are not attributed to an unrelated operation.
- A dependent projected disabled in the same batch is not an affected mod.
- Candidate discovery scans the existing metadata cache only. It must not call
  provider search/refresh or remote JAR mod-ID backfill.
- Cache candidates must match mod ID, Minecraft version, and loader, and must
  have an exact project, version ID, and download URL.
- Known CurseForge/Modrinth associations collapse into one logical candidate.
  One logical candidate restores directly; multiple unrelated candidates are
  returned in stable order for explicit user selection.
- `RestoreCachedDependency` re-runs cache validation against the selected
  instance before queueing. The frontend candidate is not trusted as authority.

### 4. Validation & Error Matrix

| Condition | Behavior |
|---|---|
| No selected instance during analysis | Return `no selected version` error |
| Action is enable/delete/unknown | Return an empty impact result |
| Dependency was already missing | Do not report a new impact |
| Compatible mod ID is absent from cache | Return no restore candidates; do not use network |
| Cached version lacks a download URL | Exclude it from candidates |
| Restore request is empty or no instance is selected | Return skipped result with `invalid cached dependency` |
| Restore candidate no longer matches cache | Return skipped result with `cached dependency unavailable` |
| Valid candidate | Queue through the existing download pipeline and return its real result |

### 5. Good/Base/Bad Cases

- Good: `main` requires `architectury`; an enabled host JAR embeds
  `architectury`; the standalone library can be a cleanup candidate under the
  existing evidence rules.
- Good: disabling that host warns when it is the last provider, and offers only
  compatible cached projects declaring `architectury`.
- Base: another enabled direct or JIJ provider remains, so no warning is shown.
- Base: cache miss leaves restore unavailable while cancel/continue remain
  available.
- Bad: add JIJ IDs to `PrimaryModIDs`, `SetVersionModIDs`, install status, or
  conflict/archive logic.
- Bad: search providers by JIJ name or mod ID after a cache miss.
- Bad: accept raw platform/project/version fields from the frontend without
  revalidating them against cache.

### 6. Tests Required

- Unused scan: enabled JIJ replacement makes the standalone library removable;
  disabling the JIJ host protects the standalone provider.
- Projection: single disable, invert, another provider, dependent disabled in
  the same batch, and pre-existing missing dependency.
- Cache: normalized ID lookup, MC/loader filtering, newest compatible version,
  cache reopen, miss, missing URL, known association deduplication, and
  unrelated multiple candidates.
- Restore: stale candidate never reaches the queue boundary; valid candidate
  queues with the selected instance tuple and exact cached version.
- Regenerate Wails bindings, then run core/app Go test-build-vet plus frontend
  lint/build.

### 7. Wrong vs Correct

Wrong:

```go
// Expands weak JIJ metadata into the strong installed-ID set.
ids := append(minecraft.PrimaryModIDs(mods), mods[0].JijMods[0].ID)
```

Correct:

```go
// Strong consumers stay unchanged; local dependency analysis owns JIJ use.
ids := minecraft.PrimaryModIDs(mods)
impact, err := service.AnalyzeLocalModDisableImpact(req)
```
