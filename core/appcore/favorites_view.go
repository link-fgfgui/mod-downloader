// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/storage"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// BuildFavoriteOverview filters the list snapshot down to the active scope and
// splits it into pinned/ungrouped sections. Ordering matches the storage query
// (pinned DESC, sort_order ASC, created_at ASC, name ASC) so adapters never
// need their own comparator.
func BuildFavoriteOverview(lists []storage.FavoriteList, minecraftVersion, modLoader string) appview.FavoriteOverview {
	overview := appview.FavoriteOverview{
		Pinned:    []storage.FavoriteList{},
		Ungrouped: []storage.FavoriteList{},
	}
	for _, list := range lists {
		if !FavoriteListMatchesScope(list, minecraftVersion, modLoader) {
			continue
		}
		if list.Pinned {
			overview.Pinned = append(overview.Pinned, list)
		} else {
			overview.Ungrouped = append(overview.Ungrouped, list)
		}
	}
	sortFavoriteLists(overview.Pinned)
	sortFavoriteLists(overview.Ungrouped)
	return overview
}

// FavoriteListMatchesScope reports whether a list belongs to the active scope.
// A list with no scope of its own matches every scope; otherwise the tuple must
// match exactly.
func FavoriteListMatchesScope(list storage.FavoriteList, minecraftVersion, modLoader string) bool {
	listVersion := normalizeFavoriteMinecraftVersion(list.MinecraftVersion)
	listLoader := normalizeFavoriteKeyPart(list.ModLoader)
	if listVersion == "" && listLoader == "" {
		return true
	}
	scopeVersion := normalizeFavoriteMinecraftVersion(minecraftVersion)
	scopeLoader := normalizeFavoriteKeyPart(modLoader)
	return hasFavoriteScope(scopeVersion, scopeLoader) && listVersion == scopeVersion && listLoader == scopeLoader
}

// FilterFavoriteContents keeps only the entries that belong to the active
// scope. Without a complete scope the result is empty, matching the previous
// adapter-side filter.
func FilterFavoriteContents(contents storage.FavoriteListContents, minecraftVersion, modLoader string) storage.FavoriteListContents {
	scopeVersion := normalizeFavoriteMinecraftVersion(minecraftVersion)
	scopeLoader := normalizeFavoriteKeyPart(modLoader)
	filtered := make([]storage.FavoriteModEntry, 0, len(contents.Mods))
	if hasFavoriteScope(scopeVersion, scopeLoader) {
		for _, entry := range contents.Mods {
			if normalizeFavoriteMinecraftVersion(entry.MinecraftVersion) != scopeVersion {
				continue
			}
			if normalizeFavoriteKeyPart(entry.ModLoader) != scopeLoader {
				continue
			}
			filtered = append(filtered, entry)
		}
	}
	contents.Mods = filtered
	return contents
}

// ListFavoriteOverview is the adapter entry point: fetch the current snapshot
// and project it onto the active scope.
func (s *Service) ListFavoriteOverview(minecraftVersion, modLoader string) appview.FavoriteOverview {
	return BuildFavoriteOverview(s.ListFavoriteLists(), minecraftVersion, modLoader)
}

// ListFavoriteContentsForScope fetches a list's contents and drops entries from
// other scopes.
func (s *Service) ListFavoriteContentsForScope(listID, minecraftVersion, modLoader string) storage.FavoriteListContents {
	return FilterFavoriteContents(s.ListFavoriteContents(listID), minecraftVersion, modLoader)
}

func sortFavoriteLists(lists []storage.FavoriteList) {
	sort.SliceStable(lists, func(i, j int) bool {
		a, b := lists[i], lists[j]
		if a.Pinned != b.Pinned {
			return a.Pinned
		}
		if a.SortOrder != b.SortOrder {
			return a.SortOrder < b.SortOrder
		}
		if a.CreatedAt != b.CreatedAt {
			return a.CreatedAt < b.CreatedAt
		}
		return a.Name < b.Name
	})
}

func normalizeFavoriteKeyPart(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeFavoriteMinecraftVersion(value string) string {
	return strings.TrimSpace(value)
}

func hasFavoriteScope(minecraftVersion, modLoader string) bool {
	return minecraftVersion != "" && modLoader != ""
}
