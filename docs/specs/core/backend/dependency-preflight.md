# Dependency Pre-flight And Reverse-Dependency Risk

> Classify a mod's required dependencies before install, and warn when a
> version change breaks an existing local dependency chain.

## Pre-flight check (install)

`appcore.Service.CheckDependencyFulfillment` must answer the same question
`QueueModDownload` answers a moment later, so it reuses the downloader's
resolution instead of re-implementing it:

- **Target version**: `modbridge.ResolveVersion` (explicit `VersionID`, then the
  pinned version, then the preferred release type). Never re-derive the ordering:
  a pinned mod would otherwise be analyzed at the wrong version. The preferred
  release type defaults to `latest` (no channel filter, so the newest matching
  version wins); `release`/`beta`/`alpha` restrict the channel and fall back to
  all versions when no version of that channel exists.
- **Dependency status**: `modbridge.InstallStatusPrecise` on the request built by
  `modbridge.DependencyDownloadRequest`. `new` → `missing`, `installed` →
  `fulfilled`, anything else (`update`/`conflict`) → `conflict`. The downloader
  only queues `new`, so the statuses line up.
- **Required-only filter**: `modbridge.IsDependencyType(dep, "required")`, the
  same predicate as `missingRequiredDependencyJobs`. An empty dependency type is
  not required for install purposes.

`InstallStatusPrecise` resolves mod IDs and may therefore read cached metadata or
parse a remote jar; it still never installs anything. The frontend keeps the
download key busy for the whole pre-flight so repeated clicks cannot start two
checks, and the queue remains the source of truth if the call fails.

`localModsByOnlineProject` is display-only (names/versions, including disabled
mods). Correctness never depends on it, because it uses platform project IDs
while `InstallStatusPrecise` uses the loader mod IDs.

### Required-only classification is unit-tested

`CheckDependencyFulfillment` resolves the target and then delegates to
`buildPreflightEntries`, which takes the status classifier as a parameter. This
keeps the required-only filter, status mapping, display fallback, and ordering
testable without provider/network state. `preflightResultFromEntries` owns the
counts. Tests must use the shared `DependencyPreflight*` constants rather than
literal status strings.

The frontend runs the same check in three places: a single install from search,
a batch install (entries from all targets are aggregated into one dialog), and
a Manage version replacement (passing the selected `versionId`). A failed
pre-flight always falls back to the normal queue path.

## Reverse-dependency downgrade risk

`ModVersionList` accepts `installedVersion`, `dependencyBreakRisk`,
`dependencyBreakMods`, and `dependencyBreakDetails`. The host view (Manage)
computes the dependents with `findReverseDependents` from the dependency graph
composable, so the required-only and pseudo-dependency filters match the tree.
The declared version fields are compared with `utils/versionCompare.ts`
(`isVersionDowngrade` or `hasMajorVersionChange`). A risky change shows a
"Breaks dependency chain" chip, a tooltip containing each dependent mod's
declared constraint, and a second confirmation before queueing the replacement.

### Don't: compare file names against versions

`compareVersions("sodium-fabric-0.5.8.jar", "0.5.11")` returns `1`, because the
first segment is text and wins the string comparison. Use
`models.ModVersion.version` for the comparison and reserve `fileName`/`name`
for display; otherwise no downgrade is ever flagged.

## Version comparison caveat

`compareVersions` is numeric-segment based, not full SemVer. It is only used to
answer "is this strictly older"; do not reuse it for range/constraint checks.
