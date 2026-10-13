# Error Handling

## Boundary Rules

- Core packages return ordinary Go `error` values or typed result structs.
  They do not panic for expected invalid input and never import Wails runtime.
- `appcore.Service` logs failures when it can continue with a defined fallback;
  it returns errors when the caller must decide what to do.
- The Wails adapter translates errors according to the existing frontend
  contract: methods that expose `error` return it directly; native-dialog and
  export methods preserve the established `panic("operation failed: " + err.Error())`
  behavior so Wails rejects the call.
- Cancellation is a control outcome, not a generic failure. Preserve
  `context.Canceled` through downloader preflight and queue state transitions.

## Logging

Use structured `logging.Error/Warn` with operation identifiers and safe,
non-secret context. Never log API keys, authorization headers, or credentialed
URLs. Do not log an error and then silently return success unless the fallback
is documented by the method.

## Validation Matrix

| Situation | Core behavior | Adapter behavior |
| --- | --- | --- |
| Invalid request fields | typed skipped/empty result or `error` | return the result/error unchanged |
| Missing local resource | `error` with operation context | return error or show existing UI failure |
| Provider failure with cache fallback | warn, use cache | return cached result |
| Provider failure without fallback | error, return error/failed event | propagate to frontend |
| User cancellation | preserve `context.Canceled`, mark retryable state | expose cancellation state, not success |
| Native dialog canceled | no error; return `{canceled:true}` | frontend treats it as a no-op |

## Good / Bad

```go
// Good: preserve the cause and let the adapter decide presentation.
if _, err := svc.ExportFavoriteListPackwizZipForScope(listID, path, mcVersion, modLoader); err != nil {
    return fmt.Errorf("export favorite list: %w", err)
}
```

```go
// Bad: hide a failed requested operation as an empty success.
if err != nil { return models.ModProject{}, nil }
```

Tests should assert the returned error/result and the observable fallback or
queue state. Prefer sentinel errors or `errors.Is` over matching log text.
