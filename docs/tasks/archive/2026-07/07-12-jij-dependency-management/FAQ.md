# FAQ: JIJ Dependency Recovery

## Why can a standalone dependency be considered unused when a mod embeds it?

An enabled JIJ entry provides the same normalized mod ID to the loader. If an
enabled JAR embeds `architectury` and no remaining dependency specifically
needs a separate provider, the standalone Architectury JAR can be treated as a
redundant library candidate. JIJ and standalone versions are not compared in
this feature.

## Why warn when disabling the JIJ host?

After the standalone dependency is removed, the host JAR may be the last
provider of that mod ID. Disabling it can leave another enabled mod with an
unsatisfied required dependency. The warning is calculated before files are
renamed and the user may cancel or continue.

## How does the restore button find the prerequisite?

It uses the missing dependency's normalized mod ID to query remote version
metadata already present in `mod-metadata.tmp`. Cached versions already carry
their parsed mod IDs, platform, project ID, Minecraft versions, and loaders.
Only candidates compatible with the selected instance are eligible.

## Does pressing restore search the internet?

No. Candidate discovery is cache-only and manually triggered from the warning.
It does not run provider search, refresh metadata, or parse a remote JAR to fill
missing mod IDs. Queueing a cache hit uses the normal download pipeline, which
of course downloads the selected file.

## What happens when the mod ID is not in cache?

No restore action is offered for that dependency. The user can cancel the
disable operation or continue knowing that the prerequisite will be missing.
There is no guessed name search or automatic repair.

## Does the app remember which standalone JAR was deleted?

No. This task adds no deletion-history table or dedicated recovery record. The
existing metadata cache is the only recovery source.

## What if cached metadata is stale?

The backend validates the cached candidate again when the user presses restore
and passes it through the existing download queue. A stale or incompatible
candidate reports a failure or skipped result instead of claiming success.

## What if multiple cached projects declare the same mod ID?

Known CurseForge/Modrinth equivalents are grouped as one logical candidate. If
one logical candidate remains, the restore button queues it directly. If
multiple unrelated projects remain, the button opens a chooser listing their
platform and project so the user selects the intended source. The application
does not guess between unrelated projects.
