# Implementation Plan

1. Add a dynamic range helper and segment-pool derivation in `core/downloader/filetransfer/stdlib.go`, retaining the existing worker and adaptive APIs.
2. Update ranged memory/temp paths to use the derived ranges and ensure resume validation remains correct.
3. Add focused arithmetic tests for balanced ranges and edge cases; adjust backend expectations that depend on fixed 4 MiB boundaries.
4. Run `gofmt` and targeted file-transfer tests.
5. Run `go test -race ./downloader/...`, then `go test ./...`, `go build ./...`, and `go vet ./...` from `core/`.

Rollback point: revert the range helper and caller changes; no config or persisted-data migration is introduced.
