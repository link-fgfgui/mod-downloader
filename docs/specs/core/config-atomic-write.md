# Atomic Config Writes and Temp-File Cleanup

## Decision

`core/configs.Save` replaces `mod-downloader.toml` using a **same-directory temp
file + encode/fsync + rename**, and sweeps stale temp files of the same pattern
before every save.

## Symptom and root cause

If the process is killed during a save (SIGKILL, power loss, container
reclamation), Go `defer` blocks do not run and
`.mod-downloader.toml.<random>.tmp` is left behind permanently. Repeated kills
accumulate temp files and pollute the working directory (which often sits next to
the user's instance directory).

## Constraints

**Don't**

- Do not write the real config with `os.OpenFile(path, O_TRUNC)`: an encoding
  failure would truncate the user's existing config.
- Do not assume the directory has no leftovers before `CreateTemp`: crash/kill
  paths do not run `defer`.
- Do not use a fixed temp file name: concurrent saves would overwrite each other
  (core is single-instance, but tests and tooling run concurrently).

**Instead**

- Keep the temp-file pattern `.<configBase>.*.tmp` (`os.CreateTemp` guarantees a
  random suffix).
- Sweep leftovers of the same pattern with `filepath.Glob` before saving. **This
  relies on the process-wide single instance** (see
  [`single-instance.md`](./single-instance.md)); otherwise the sweep could delete
  a file another process is writing.
- On the failure path, `Close` and `Remove`; on the success path, `Rename` before
  returning.

## Verification

- `TestSaveReplacesConfigAtomicallyAndLeavesNoTempFiles`: two consecutive saves
  leave no `*.tmp` files.
- `TestSaveSweepsStaleTempFilesFromKilledWrites`: a pre-seeded stale temp file is
  removed and the new config takes effect.
- `TestSaveFailurePreservesOriginalConfig`: the original content is unchanged
  when encoding/writing fails.

Location: `core/configs/config.go` (`Save`), `core/configs/config_test.go`.
