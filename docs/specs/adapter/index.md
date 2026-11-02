# Adapter Specifications Index

This directory registers contracts for the Wails adapter layer (`main.go`,
`app.go`, `app_events.go` at the repository root). Core domain rules live in
`docs/specs/core/`.

| Topic | File | Summary |
| --- | --- | --- |
| App binding surface | [`binding-surface.md`](./binding-surface.md) | Expose only methods the frontend actually consumes; `app_contract_test.go` locks 68 business methods |
