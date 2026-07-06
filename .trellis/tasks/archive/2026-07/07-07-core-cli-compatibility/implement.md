# Implementation Plan

## Checklist

- [x] Record current app and CLI git/submodule status.
- [x] Check out core `56f8e8b` inside `../mod-downloader-cli/core`.
- [x] Run `go test ./...` in `../mod-downloader-cli` and inspect failures.
- [x] For each failure, inspect the corresponding core API change and choose
  the correct side to edit.
- [x] Apply focused source fixes. No Go source fixes were required.
- [x] Run gofmt on modified Go files. No Go files were modified.
- [x] Run validation:
  - `go test ./...` from `core/`
  - `go build ./...` from `../mod-downloader-cli`
  - `go test ./...` from `../mod-downloader-cli`
  - Wails-runtime dependency checks for core service packages and CLI packages
- [x] Summarize conflicts, side choices, changed files, and validation results.

## Risky Areas

- Submodule pointer changes in `../mod-downloader-cli`.
- Core service exported methods and structs consumed directly by CLI.
- CLI tests that assert exact JSON output for core-owned structs.

## Review Gate

Before implementation starts, confirm the PRD, design, and implementation plan
exist and the task is moved to `in_progress`.
