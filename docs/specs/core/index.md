# Core Specifications Index

This directory registers the `core/` domain rules shared by the GUI and TUI.
Any rule that still holds after replacing React/tview must have exactly one Go
implementation and live here. Frontend-only display and interaction contracts
belong in `docs/specs/frontend/`.

| Topic | File | Summary |
| --- | --- | --- |
| Global single instance | [`single-instance.md`](./single-instance.md) | Bind `127.0.0.1:18801` at startup; the port is the lock; abort before creating a window if the bind fails |
| Shared decision projections | [`ui-decision-projection.md`](./ui-decision-projection.md) | `LocalModGroup` decision fields, dependency forest, search/version comparison, disabled-state preservation, single entry point for batch operations |
| Atomic config writes | [`config-atomic-write.md`](./config-atomic-write.md) | Temp file + fsync + rename; sweep stale `.*.tmp` files before saving (relies on single instance) |
