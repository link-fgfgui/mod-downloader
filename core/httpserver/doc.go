// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package httpserver exposes a thin HTTP transport for scan results produced by
// the mod-downloader browser extension.
//
// # Endpoints
//
//   - POST /v1/extension/projects is the canonical project-submission endpoint.
//   - POST / is the legacy path kept for older extension builds; it uses the
//     same handler and validation.
//   - GET /health returns {"status":"ok","apiVersion":1}.
//
// Incompatible request or response changes require a new path and a new
// APIVersion. Additive response fields are allowed within version 1.
//
// # Request contract
//
// The body is a JSON array of ExtensionProjectPayload values, each shaped as
// {"p":"mr"|"cf","id":string|null,"slug":string,"file":string}. The method must
// be POST (OPTIONS is answered for CORS preflight) and Content-Type must start
// with application/json; otherwise the server returns 415 Unsupported Media
// Type. An empty Origin is allowed for native tools; extension origins must use
// the chrome-extension:// or moz-extension:// scheme, and any other origin gets
// 403 Forbidden. Bodies larger than 4 MiB and arrays with more than 256 entries
// get 413 Request Entity Too Large. Malformed JSON gets 400 Bad Request, and a
// missing Options.Accept handler gets 503 Service Unavailable. Unknown paths get
// 404 Not Found. Extension origins receive matching CORS headers.
//
// # Response contract
//
// A successful response returns one ModDownloadResult per input payload, in
// input order, and sets X-Mod-Downloader-API-Version: 1. Accepted reports that
// metadata resolved and the project matches the selected instance. Pinned
// reports that an explicit version was auto-pinned. Skipped with Reason reports
// a rejected or unavailable project. Error reports a provider or storage
// failure while partial results remain usable. Queued is always false:
// acceptance emits projects for the frontend to queue.
//
// The transport only decodes, validates and forwards. Metadata resolution,
// filtering and auto-pinning belong to appcore.Service.AcceptExtensionProjects,
// which bounds project-level concurrency.
package httpserver
