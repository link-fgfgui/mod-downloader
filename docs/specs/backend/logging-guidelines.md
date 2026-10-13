# Logging Guidelines

> Structured application logging, output selection, and shutdown contracts.

## Scenario: stderr And File Logging Policy

### 1. Scope / Trigger

Use this contract whenever application logs, logging configuration, startup,
or shutdown behavior changes. Desktop builds may have no usable stderr, while
CLI/dev builds normally do. Logging must remain observable without forcing a
file on every process, and users must be able to disable it completely.

### 2. Signatures

Configuration:

```toml
[logging]
enabled = true
force_file = false
```

```go
type LoggingConfig struct {
    Enabled   *bool `toml:"enabled,omitempty" json:"enabled,omitempty" env:"-"`
    Disabled  *bool `toml:"disabled,omitempty" json:"-" env:"-"`
    ForceFile bool `toml:"force_file" json:"forceFile" env:"FORCE_FILE"`
}

func (c LoggingConfig) EnabledValue() bool
func (c LoggingConfig) Canonical() LoggingConfig

type logging.Options struct {
    Disabled  bool
    ForceFile bool
    FilePath  string
}

func configs.LoadLogging() (configs.LoggingConfig, error)
func logging.Configure(options logging.Options) error
func logging.Close()
```

Canonical environment keys are `LOGGING_ENABLED` and `LOGGING_FORCE_FILE`.
Legacy `LOGGING_DISABLED` remains read-compatible. The default file is
`mod-downloader.log` in the process working directory (or appcore
`Runtime.WorkDir` when explicitly supplied).

### 3. Contracts

- `enabled=false` has highest priority: all levels go to `io.Discard`, no log
  file is created, and `force_file` is ignored.
- When `enabled` is absent, legacy `disabled` is inverted once. If both are
  present, positive `enabled` wins. Missing values default to enabled.
- `cleanenv` cannot decode pointer booleans. `LOGGING_ENABLED` and legacy
  `LOGGING_DISABLED` are parsed explicitly with `strconv.ParseBool` after the
  rest of the environment; do not change them to plain booleans or presence
  information is lost.
- `configs.Save` canonicalizes logging and emits `enabled` only. It never
  rewrites the legacy `disabled` key.
- Default mode writes Debug and above to stderr when `os.Stderr.Stat()`
  succeeds. It does not create a file.
- If stderr is unavailable, default mode appends to `mod-downloader.log`.
- `force_file=true` always appends to the file and also writes stderr when
  stderr is usable.
- `Service.Startup` calls the silent `configs.LoadLogging` bootstrap before the
  regular config loader. Therefore valid `enabled`/`force_file` settings apply
  to the first config-loader message, not only after startup completes.
- The regular full config remains the source of truth and reconfigures logging
  after config overrides are applied.
- File mode uses `O_CREATE|O_WRONLY|O_APPEND` with mode `0644` and never
  truncates an existing log.
- Logger replacement and writes share a read/write mutex so reconfiguration or
  close cannot close a file during an active write.
- `Service.Shutdown` and `Service.Close` stop provider background tasks, close
  storage, then close the active log file so every shutdown message is retained.
- Missing `[logging]` keys use current defaults. Keep the existing legacy
  `disabled` reader until compatibility is intentionally retired.

### 4. Validation & Error Matrix

- `enabled=false`, any other values -> discard all logs; no file.
- Both positive and legacy keys -> positive `enabled` wins.
- Invalid `LOGGING_ENABLED` or `LOGGING_DISABLED` boolean -> config load error.
- `force_file=true`, usable stderr -> stderr plus file.
- `force_file=false`, unusable stderr -> file only.
- `force_file=false`, usable stderr -> stderr only.
- Empty `FilePath` when file output is required -> use
  `logging.DefaultFileName`.
- File open failure -> `Configure` returns the OS error and preserves the
  previous logger; appcore reports the failure through that previous logger and
  continues startup.
- Logging bootstrap parse failure -> continue to the regular config load; its
  normal error handling decides startup configuration.
- Repeated configure -> atomically install the new logger and close the old
  file descriptor.

### 5. Good/Base/Bad Cases

- Good: packaged GUI has no stderr, so startup creates and appends
  `mod-downloader.log` automatically.
- Good: user sets `enabled=false` and `force_file=true`; disabled logging wins and even config-load messages
  are suppressed.
- Good: an old file contains `disabled=true`; it loads as disabled and the next
  save writes `enabled=false` without `disabled`.
- Good: CLI runs with stderr redirected to a valid file; stderr remains usable,
  so no additional application log file is created unless forced.
- Base: terminal/dev startup logs Debug and above to stderr only.
- Bad: configure file logging only after `configs.Load`; disabled mode leaks
  bootstrap messages before being applied.
- Bad: close the old log file before preventing concurrent writers; an active
  log call can then write to a closed descriptor.

### 6. Tests Required

- Logging unit test: disabled suppresses stderr and does not create a file even
  when force-file is true.
- Logging unit test: forced mode writes the same structured record to stderr
  and file.
- Logging unit test: unavailable stderr falls back to file; usable stderr
  default creates no file.
- Config tests decode canonical and legacy TOML/environment values, assert
  positive-key precedence, and assert saves contain `enabled` but not
  `disabled`.
- Appcore integration test loads `force_file=true` from TOML and asserts the
  config loader's first message plus a later marker are in the file.
- Appcore integration test loads disabled+forced and asserts no file exists.
- Run logging/config race tests and repeated appcore bootstrap integration race
  tests, plus full root/core test, vet, and build checks.

### 7. Wrong vs Correct

Wrong:

```go
cfg, _ := configs.Load()      // already emitted startup logs
logging.Configure(cfg.Logging)
```

Correct:

```go
bootstrap, _ := configs.LoadLogging()
configureLogging(bootstrap.EnabledValue())
cfg, _ := configs.Load()
configureLogging(cfg.Logging.EnabledValue())
```

## Log Levels

- Debug: cache lookups, config read/write starts, and diagnostic state useful
  during troubleshooting.
- Info: completed lifecycle events and successful user-visible operations.
- Warn: recoverable failures with a defined fallback, including stale-cache
  use and optional persistence failure.
- Error: failed requested operations or startup subsystems without a lower
  severity fallback.

## Structured Fields

Use `log/slog` key/value fields through `logging.Debug/Info/Warn/Error`. Include
stable identifiers and counts that explain the operation, such as `platform`,
`projectID`, `path`, `versionCount`, and `error`. Do not encode fields into a
preformatted message when a structured field is available.

## Sensitive Data

Never log API keys, authorization headers, tokens, passwords, full request
headers, or config structs containing secrets. Boolean presence flags and
explicitly masked values are allowed. Paths are allowed when they are required
to diagnose local file operations; do not log file contents.
