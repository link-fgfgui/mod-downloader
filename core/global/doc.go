// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package global owns the process-wide Minecraft directory, version cache and
// provider client singletons shared by every adapter.
//
// Directory, version and mod-cache getters return snapshots. Mutating a
// returned slice or a nested slice does not affect the cache, so callers may
// retain the value without holding a lock. SetSelectedVersion is copy-on-write
// and replaces cache entries instead of mutating a list another goroutine may
// be iterating.
//
// The selected version is scoped to the current Minecraft directory. Reading it
// after the directory changes returns the zero value until a new selection is
// made. GetVersionsForDir returns true only when the cached version list belongs
// to the requested directory.
//
// Global state is process-wide by design because the core is single-instance;
// adapters must not keep a second copy.
package global
