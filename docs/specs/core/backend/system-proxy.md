# System Proxy

## Scenario: Routing Outbound HTTP Through The System Proxy Environment

### 1. Scope / Trigger

Use this contract whenever creating an outbound HTTP client or transport in
core. Provider APIs, Minecraft metadata, remote JAR reads, file transfer, and
discard fetches must share the same proxy behavior.

### 2. Signatures

```go
func networking.NewTransport() *http.Transport
func networking.NewClient(timeout time.Duration) *http.Client
```

Supported environment variables are `HTTP_PROXY`, `HTTPS_PROXY`, and
`NO_PROXY`, including the lowercase forms recognized by
`http.ProxyFromEnvironment`.

### 3. Contracts

- `networking.NewTransport` clones `http.DefaultTransport` so Go connection
  pooling, TLS, HTTP/2, and timeout defaults are preserved.
- The clone explicitly sets `Proxy: http.ProxyFromEnvironment`.
- Each caller receives an independent transport; callers may customize it
  without mutating `http.DefaultTransport` or another client.
- Provider rate limiting wraps the system-proxy transport rather than replacing
  it.
- The standard-library file-transfer backend uses `networking.NewClient` when
  the caller does not inject a client.
- This project does not modify desktop proxy settings or import a platform
  proxy-management library. "System proxy" means the proxy environment of the
  application process.

### 4. Validation & Error Matrix

- No proxy variables -> direct connection.
- Matching HTTP/HTTPS proxy -> route through the resolved proxy URL.
- Host matched by `NO_PROXY` -> direct connection.
- Invalid proxy environment value -> return the resolver error from the HTTP
  request.
- Caller-injected file-transfer client -> use it unchanged; tests and embedded
  consumers retain control.

### 5. Good/Base/Bad Cases

- Good: provider and file requests use clients backed by
  `networking.NewTransport`, while the provider limiter remains an outer
  `RoundTripper`.
- Base: no environment proxy is set and behavior is equivalent to the default
  Go transport.
- Bad: construct `&http.Transport{}` locally; its nil `Proxy` silently bypasses
  the configured proxy.
- Bad: mutate `http.DefaultTransport.(*http.Transport).Proxy`; this changes
  process-global behavior and creates test/client coupling.

### 6. Tests Required

- Inject a proxy resolver and assert `NewTransport().Proxy` invokes it.
- Start an `httptest` proxy, resolve an unreachable origin through it, and
  assert `NewClient` receives the proxy response without origin DNS access.
- Search production code for raw `http.Transport` construction and verify any
  custom `http.Client` wraps `networking.NewTransport` directly or indirectly.
- Run core and app test/vet/build plus downloader race tests.

### 7. Wrong vs Correct

Wrong:

```go
client := &http.Client{Transport: &http.Transport{}}
```

Correct:

```go
client := networking.NewClient(20 * time.Second)
limited := rateLimitedTransport{
    limiter: limiter,
    base: networking.NewTransport(),
}
```
