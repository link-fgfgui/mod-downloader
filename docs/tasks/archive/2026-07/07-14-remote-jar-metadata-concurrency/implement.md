# Implementation Plan

1. Add a dynamically configurable remote mod-ID parse gate in `core/modbridge`.
2. Submit each drained search backfill batch concurrently while retaining one
   completion event after every item finishes.
3. Acquire the gate only around the cache-owning remote JAR parse and guarantee
   release on success and failure.
4. Raise the remote JAR metadata deadline to 45 seconds.
5. Configure the gate from normalized concurrent-download settings during
   `Service.Startup` and `Service.SaveNetworkSettings`.
6. Add focused tests proving batch parallelism, the configured cap, waiter
   release, and preserved
   same-key deduplication behavior.
7. Run formatting, focused tests, the core test suite, race-sensitive package
   tests, build, and vet.

## Validation

```bash
cd core && gofmt -w modbridge/modbridge.go modbridge/modbridge_test.go appcore/service.go
cd core && go test ./modbridge ./appcore
cd core && go test -race ./modbridge
cd core && go test ./...
cd core && go build ./... && go vet ./...
```

## Review Gates

- Confirm cache hits and duplicate URL/loader callers do not acquire slots.
- Confirm runtime limit reductions do not oversubscribe new work.
- Confirm no app/frontend contract or generated binding changed.
