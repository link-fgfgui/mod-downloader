// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package view

import "github.com/link-fgfgui/mod-downloader/core/storage"

// FavoriteOverview is the scope-filtered favorites projection. Both sections
// are ordered by core (pinned DESC, sort_order ASC, created_at ASC, name ASC),
// so adapters render them without re-sorting.
type FavoriteOverview struct {
	Pinned    []storage.FavoriteList `json:"pinned"`
	Ungrouped []storage.FavoriteList `json:"ungrouped"`
}
