// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package appcore is the adapter-neutral business facade shared by the Wails
// GUI and the tview TUI. It owns provider access, the download queue, storage,
// settings normalization and the event taxonomy.
//
// A process runs at most one active Service: it manages process-wide provider
// clients, logging, storage and the download queue. New constructs a service;
// Startup must run before any other method. Shutdown persists configuration and
// drains background work, while Close releases resources without saving.
// Startup failures are returned as *StartupError with a Stage from the
// core/structs.StartupStage* constants.
//
// # Error contract
//
// Methods ending in WithError preserve storage, provider and config
// persistence failures. The original methods remain for source compatibility
// with the TUI and collapse failures into a zero value; new adapter code
// should call the WithError variant.
//
//   - A successful read returns its value and a nil error.
//   - A read that finds nothing returns the zero value and a nil error; an
//     empty result is normal, not a failure.
//   - Create, rename, metadata update and reference creation return
//     ErrFavoriteListNotFound when the target list does not exist.
//   - Delete, unpin and remove are idempotent: a missing target returns false
//     and a nil error.
//   - A closed or failing store returns the underlying storage error.
//   - Provider lookups return providers.ErrProjectNotFound when nothing
//     matches and providers.ErrUnsupportedPlatform for an unknown platform.
//
// # Configuration and concurrency
//
// Service methods may be called from multiple goroutines. Config returns an
// immutable snapshot; callers may read it after the call and mutations do not
// affect the service. Writes clone the current snapshot, persist it under a
// mutex, and publish the new value only after the save succeeds. A failed save
// leaves the live snapshot unchanged and is reported through
// SettingsView.SaveError, or directly by SaveThemeWithError and
// SaveLanguageWithError. Runtime effects run only after a successful persist.
//
// Storage and the global caches publish snapshots atomically. See package
// storage and package global for their lifecycle rules.
//
// # Events
//
// EventKind.Name returns the stable kebab-case wire name. Adapters subscribe
// through host.On or host.OnEvent and must not declare a second event enum.
// Event payload types are part of the contract: search and
// extension-accepted events carry view.SearchModsUpdate, download queue updates
// carry view.DownloadQueueState, download failures carry
// structs.DownloadFailedEvent, usage statistics carry storage.UsageStats and
// selection updates carry minecraft/structs.VersionInfo.
// EventMinecraftDirChanged carries the simplified directory string;
// EventDownloadStatesUpdated and EventDownloadCompleted carry a nil payload and
// ask adapters to re-read state. SearchModsUpdate.Error is set only on the
// final page and may accompany partial results from healthy providers.
package appcore
