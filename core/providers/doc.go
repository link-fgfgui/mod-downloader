// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package providers integrates the Modrinth and CurseForge APIs behind a single
// provider interface and persists their metadata to the cache.
//
// LookupProjectByPlatformWithError returns ErrProjectNotFound when no project
// matches and ErrUnsupportedPlatform for an unknown platform. The bool-returning
// lookup functions collapse both failures into a missing result.
//
// SearchMods emits partial results as providers finish and sets
// SearchModsUpdate.Error only on the final page, when one or more providers
// failed. SearchModsText does the same for the broad text search without the
// extra exact-title probes.
//
// Provider clients and background refresh tasks are process-wide. appcore
// configures them at startup; this package must not read adapter configuration
// or hold UI state.
package providers
