// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package host owns the UI-independent runtime: the core service, the HTTP
// server and the typed event fan-out. GUI and TUI adapters only configure it
// and register handlers; neither reimplements lifecycle or event plumbing.
//
// Start acquires the process-wide instance guard, starts the core service and
// then the extension HTTP server. The guard binds the extension port, so a
// failed acquire returns instance.ErrAlreadyRunning and must abort startup
// before a window or shared storage is created.
//
// On and OnEvent register typed handlers. Payload assertion happens once inside
// host and a mismatched payload is logged instead of panicking. Adapters must
// subscribe through these functions and must not assert event payload types
// themselves.
//
// StartupStatus reports typed readiness. Stage uses the structs.StartupStage*
// constants, and ExtensionServerError is a non-fatal notice that must be shown
// separately from a startup failure.
//
// Shutdown stops the HTTP server, shuts down the core and releases the guard.
package host
