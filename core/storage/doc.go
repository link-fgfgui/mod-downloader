// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package storage owns the gob+zstd metadata cache and the SQLite user
// database (favorites, pins and usage stats).
//
// # Lifecycle
//
// Open and OpenAt publish the metadata cache and user database through atomic
// pointers. Readers never take the lifecycle lock. OpenAt opens the new stores
// before publishing them together; the previous stores are retired rather than
// closed so an in-flight reader cannot observe a closed database. Close swaps
// the pointers to nil and closes every current and retired store after saving
// the metadata caches.
//
// A cache-directory switch must go through OpenAt. The retiring cache is saved
// before it is retired, so switching A to B to A does not reload stale data
// from disk. The appcore service rolls the live store back when persisting the
// new cache directory fails.
//
// Close is safe to call more than once. Callers must stop background storage
// users before invoking it.
//
// # Results and errors
//
// Lookups return a value and a bool that reports whether the row exists; list
// reads return newly allocated slices. Mutations return a bool or an error so
// callers can distinguish a missing target from a failing store. Read results
// are copies; callers must not expect later store updates to mutate them.
package storage
