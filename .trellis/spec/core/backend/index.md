# Core Backend Guidelines

> Package-scoped entry point for the `core/` git submodule.

The core package is the local checkout of
`github.com/link-fgfgui/mod-downloader-core`. It owns app-independent service,
provider, download, storage, Minecraft parsing, and HTTP bridge logic.

## Required References

- [Shared Backend Guidelines](../../backend/index.md)
- [Directory Structure](../../backend/directory-structure.md)
- [Quality Guidelines](../../backend/quality-guidelines.md)
- [Storage Guidelines](../../backend/storage-guidelines.md)
- [Selected Version Cache](./selected-version-cache.md)
- [Outbound User-Agent](./outbound-user-agent.md)
- [File Transfer](./file-transfer.md)
- [Network Runtime Configuration](./network-runtime.md)
- [Provider API Source](./provider-api-source.md)
- [System Proxy](./system-proxy.md)
- [Local Mod Online Version Metadata](./local-mod-online-version.md)
- [Language Preference](../../app/frontend/language-preference.md)

## Pre-Development Checklist

- Make core changes inside `core/`, not by recreating packages in the app repo.
- Keep `appcore`, `httpserver`, and lower layers free of Wails runtime imports.
- Keep shared types in `models`; do not add aliases or re-export packages.
- Keep refreshed selected-version snapshots synchronized across the ordered
  cache and key lookup map; see [Selected Version Cache](./selected-version-cache.md).
- Propagate the versioned application identity through provider and download
  requests; see [Outbound User-Agent](./outbound-user-agent.md).
- Keep file downloads in the project-owned standard-library backend; see
  [File Transfer](./file-transfer.md).
- Apply file concurrency, queue concurrency, and provider API rate limits
  through [Network Runtime Configuration](./network-runtime.md).
- Keep the official/MCIM API and file source switch synchronized through
  [Provider API Source](./provider-api-source.md).
- Route every outbound HTTP client through [System Proxy](./system-proxy.md).
- Preserve parsed JAR versions when adding provider version metadata; see
  [Local Mod Online Version Metadata](./local-mod-online-version.md).
- Keep config parsing and appcore language fields aligned with the app/frontend
  contract; see [Language Preference](../../app/frontend/language-preference.md).

## Quality Check

- Run `go test ./...` from `core/` after core changes.
- Run `go build ./... && go vet ./...` from `core/` when changing exports or dependencies.
- Re-run app or CLI tests when a core change affects their adapters.
