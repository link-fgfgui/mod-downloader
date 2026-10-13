# Limit remote JAR metadata concurrency

## Goal

Bound remote JAR metadata parsing so search-result mod ID backfills cannot
create unbounded concurrent HTTP Range traffic, while allowing the existing
download concurrency setting to control the amount of parallel work.

## Background

- Remote mod IDs are resolved by parsing JAR metadata over HTTP Range requests
  in `core/modbridge/modbridge.go`.
- A single `GetDownloadStates` backfill slice is currently processed serially,
  but overlapping state refreshes and other precise-status callers can parse
  different JARs concurrently without a global bound.
- `downloads.concurrent_downloads` is already normalized to the range 1-16,
  defaults to 1, and is applied both during service startup and after network
  settings are saved.

## Requirements

- Increase the complete remote JAR metadata parsing deadline from 30 seconds
  to 45 seconds per JAR.
- Limit the number of distinct remote JAR metadata parses running at once to
  the normalized `downloads.concurrent_downloads` value.
- Allow one search-result backfill batch to parse different uncached JARs in
  parallel up to that global limit instead of retaining the old serial loop.
- Apply the configured limit during service startup and immediately after
  `SaveNetworkSettings`; already-running parses may finish, while newly
  starting parses must honor the current limit.
- Enforce the limit at the shared remote parsing boundary so search backfills,
  install-time precise checks, and other `VersionModIDs` callers use the same
  global capacity.
- Preserve existing memory/DB cache reads, same-URL-and-loader request
  coalescing, per-version in-flight deduplication, failure TTL behavior, and
  provider API rate limiting.
- A missing or invalid concurrent-download value must retain the existing
  normalized default of one.

## Acceptance Criteria

- [x] A remote JAR metadata parse uses a 45-second overall deadline.
- [x] With `concurrent_downloads = N`, no more than N distinct uncached remote
      JAR parses enter the HTTP parsing function concurrently.
- [x] A single backfill batch can start multiple distinct JAR parses when
      `concurrent_downloads` is greater than one.
- [x] Waiting parses begin when capacity becomes available and do not consume
      capacity when they return from memory, DB, or same-URL cache.
- [x] Startup and saved network-setting paths both configure the remote JAR
      parse limit from the normalized concurrent-download value.
- [x] Lowering the limit does not cancel in-flight parses; it prevents new
      parses from starting until active work falls below the new limit.
- [x] Focused concurrency/deduplication tests and the core test suite pass.

## Out Of Scope

- Adding a separate user-facing setting for metadata parsing.
- Applying the file chunk concurrency setting to ZIP metadata Range reads.
- Canceling or resizing JAR parses that have already started.
