// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package configs loads, normalizes and persists the TOML/environment
// configuration shared by every adapter.
//
// Load merges environment overrides on top of mod-downloader.toml.
// LoadLogging and LoadHardwareAcceleration are bootstrap-only reads used before
// logging or the webview exists.
//
// Save writes atomically: it encodes to a same-directory temporary file, fsyncs
// it and renames it over the real configuration. A failed encode or write leaves
// the existing file untouched. Stale temp files left by a killed process are
// swept before saving; this relies on the process-wide single-instance guard
// because another live writer would own a temp file.
//
// Save also normalizes the logging aliases before encoding. The caller owns the
// in-memory snapshot; this package never mutates shared configuration state.
package configs
