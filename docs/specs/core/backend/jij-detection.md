# Configurable Jar-in-Jar (JIJ) Detection

> Toggle recursive nested-JAR scanning to keep large-modpack scans fast.

## Contract

- Config: `preferences.enable_jij_detection` (`*bool`, default **true**;
  `configs.DefaultEnableJijDetection`, accessor
  `Preferences.EnableJijDetectionValue()`). A pointer is required so an unset
  value means "on" while an explicit `false` disables it.
- Settings surface: `SettingsView.EnableJijDetection` /
  `SaveJijDetectionSettingsRequest` (`appcore` + app adapter), persisted by
  `SaveJijDetectionSettings`.
- Runtime switch: `minecraft.SetJijDetectionEnabled(bool)` stores a
  process-wide `atomic.Bool`. `Startup` seeds it from config, and
  `SaveJijDetectionSettings` updates it live.

## Why a package-wide flag

The parser (`core/minecraft`) is called from several packages (local scan,
downloader, modbridge) that do not thread config through. A package-level
atomic is the least invasive way to honor an app-level setting; the value is
read on every parse, so a toggle takes effect on the next scan.

## Cache scope must include the flag

`ParseModJarWithSHA1` / `ParseLocalModJarWithSHA1` key the in-memory JAR cache
by `sha1 + scope`. Because disabling JIJ changes the parsed result for the same
file, the scope string must encode the flag (`...:jij=0|1`). Otherwise a file
parsed with JIJ on would be served from cache after the user turns JIJ off.

## Short-circuit points

When disabled, skip both the nested-jar loop in `parseZipReader` and the
`META-INF/jarjar/metadata.json` read in `parseDeclaredMetadata`. The top-level
mod list is unaffected — only `JijMods` is dropped.

## Sample the flag once per parse

Compute the flag once at the entry point (`ParseModJarWithSHA1` /
`ParseLocalModJarWithSHA1`) and thread it through `modJarParseContext`.

### Don't: re-read the atomic while parsing

**Problem**: the cache scope is built from `JijDetectionEnabled()` at the start,
but gating also calls it inside `parseZipReader` / `parseDeclaredMetadata` /
recursive nested parses.

**Why it's bad**: the settings switch updates the atomic live while a background
scan is running. A scan that starts with JIJ on and finishes after the user turns
it off can write a JIJ-less result into the `:jij=1` cache slot — the toggle
appears to do nothing until the cache expires.

**Instead**: `jijEnabled := JijDetectionEnabled()` once, build the scope from
that value, and pass it into the parse context; every nested check reads
`ctx.jijEnabled`.

## Unused-dependency scan fails closed

`appcore.Service.ScanUnusedDependencies` returns an empty result when JIJ
detection is off. Without nested-jar metadata the provider index cannot see IDs
supplied by embedded jars, so a jar that is the only provider of a required ID
could be reported as unused. Fail closed rather than offer a destructive
suggestion; the Manage sort option is disabled in the same state.
