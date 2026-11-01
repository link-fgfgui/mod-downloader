// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/logging"
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/providers"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// SearchMods streams search results through the EventSearchModsUpdated event.
func (s *Service) SearchMods(req appstructs.SearchModsRequest) {
	providers.SearchMods(req, func(update appview.SearchModsUpdate) {
		s.emit(EventSearchModsUpdated, update)
	})
}

// SearchModsCollect runs a search and returns only the final page.
func (s *Service) SearchModsCollect(req appstructs.SearchModsRequest) appview.SearchModsUpdate {
	var last appview.SearchModsUpdate
	providers.SearchMods(req, func(update appview.SearchModsUpdate) {
		last = update
	})
	return last
}

// SearchModsTextCollect runs the broad text search only and returns the final
// page.
func (s *Service) SearchModsTextCollect(req appstructs.SearchModsRequest) appview.SearchModsUpdate {
	var last appview.SearchModsUpdate
	providers.SearchModsText(req, func(update appview.SearchModsUpdate) {
		last = update
	})
	return last
}

// ListMatchingProjectVersions returns cached versions matching a scope.
func (s *Service) ListMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
	return providers.ListMatchingProjectVersions(result, minecraftVersion, modLoader)
}

// LookupProject resolves a project by ID or slug.
func (s *Service) LookupProject(platform, idOrSlug, minecraftVersion, modLoader string) (models.ModProject, bool) {
	return providers.LookupProjectByPlatform(platform, idOrSlug, minecraftVersion, modLoader)
}

// LookupProjectBySlug resolves a project by its platform slug.
func (s *Service) LookupProjectBySlug(platform, slug, minecraftVersion, modLoader string) (models.ModProject, bool) {
	return providers.LookupProjectByPlatformSlug(platform, slug, minecraftVersion, modLoader)
}

// LookupProjectByID resolves a project by its platform project ID.
func (s *Service) LookupProjectByID(platform, projectID, minecraftVersion, modLoader string) (models.ModProject, bool) {
	return providers.LookupProjectByPlatformID(platform, projectID, minecraftVersion, modLoader)
}

// GetPinnedModVersion returns the pinned version for a mod in one scope, or
// the zero value when no pin exists.
func (s *Service) GetPinnedModVersion(platform string, modID string, minecraftVersion string, modLoader string) storage.PinnedMod {
	pin, _ := storage.GetPinnedMod(platform, modID, minecraftVersion, modLoader)
	return pin
}

// PinModVersion toggles a pin: pinning the already-pinned version unpins it.
func (s *Service) PinModVersion(req appstructs.ModVersionPinRequest) storage.PinnedMod {
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	modID := strings.ToLower(strings.TrimSpace(req.ModID))
	versionID := strings.TrimSpace(req.VersionID)
	minecraftVersion := strings.TrimSpace(req.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(req.ModLoader))

	if platform == "" || modID == "" || versionID == "" || minecraftVersion == "" || modLoader == "" {
		return storage.PinnedMod{}
	}

	existing, found := storage.GetPinnedMod(platform, modID, minecraftVersion, modLoader)
	if found && strings.EqualFold(existing.VersionID, versionID) {
		_ = storage.DeletePinnedMod(platform, modID, minecraftVersion, modLoader)
		return storage.PinnedMod{}
	}

	pin := storage.PinnedMod{
		Platform:         platform,
		ModID:            modID,
		VersionID:        versionID,
		MinecraftVersion: minecraftVersion,
		ModLoader:        modLoader,
	}
	if err := storage.UpsertPinnedMod(pin); err != nil {
		logging.Error("upsert pinned mod failed", "platform", platform, "modID", modID, "versionID", versionID, "minecraftVersion", minecraftVersion, "modLoader", modLoader, "error", err)
		return storage.PinnedMod{}
	}
	pin, _ = storage.GetPinnedMod(platform, modID, minecraftVersion, modLoader)
	return pin
}

// ListPinnedMods returns every pinned mod.
func (s *Service) ListPinnedMods() []storage.PinnedMod {
	return storage.ListPinnedMods()
}

// UnpinMod removes the pin for a mod in one scope.
func (s *Service) UnpinMod(platform, modID, minecraftVersion, modLoader string) bool {
	platform = strings.ToLower(strings.TrimSpace(platform))
	modID = strings.ToLower(strings.TrimSpace(modID))
	minecraftVersion = strings.TrimSpace(minecraftVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))
	if platform == "" || modID == "" || minecraftVersion == "" || modLoader == "" {
		return false
	}
	if _, found := storage.GetPinnedMod(platform, modID, minecraftVersion, modLoader); !found {
		return false
	}
	if err := storage.DeletePinnedMod(platform, modID, minecraftVersion, modLoader); err != nil {
		logging.Error("unpin mod failed", "platform", platform, "modID", modID, "minecraftVersion", minecraftVersion, "modLoader", modLoader, "error", err)
		return false
	}
	return true
}
