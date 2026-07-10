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
