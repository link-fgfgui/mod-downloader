# Core Specifications Index

This directory registers the core domain rules shared by the GUI and TUI. Any
rule that still holds after replacing React/tview must have exactly one Go
implementation and a behavioral specification here. Frontend-only display and
interaction contracts belong in the repository-root `docs/specs/frontend/`.

The public Go API is documented in this module with package and symbol doc
comments (`go doc github.com/link-fgfgui/mod-downloader/core/...`). This
directory records behavioral decisions, constraints and anti-patterns that are
not obvious from the API signature alone; it must not duplicate the godoc
surface.

## Stability

The exported core API is frozen for the current GUI/TUI line. Keep existing
signatures source-compatible: add a `...WithError` variant or a new field
instead of changing a legacy return. Any public API or event change must still
build and test the TUI repository. The App binding surface, event wire names and
extension HTTP `APIVersion` are additionally locked mechanically by
`app_contract_test.go`, `core/appcore/contracts_test.go` and the
`core/httpserver` tests.

| Topic | File | Summary |
| --- | --- | --- |
| Global single instance | [`single-instance.md`](./single-instance.md) | Bind `127.0.0.1:18801` at startup; the port is the lock; abort before creating a window if the bind fails |
| Shared decision projections | [`ui-decision-projection.md`](./ui-decision-projection.md) | `LocalModGroup` decision fields, dependency forest, search/version comparison, disabled-state preservation, single entry point for batch operations |
| Atomic config writes | [`config-atomic-write.md`](./config-atomic-write.md) | Temp file + fsync + rename; sweep stale `.*.tmp` files before saving (relies on single instance) |
