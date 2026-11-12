// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"errors"
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/logging"
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/providers"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// SearchMods streams search results through the EventSearchModsUpdated event.
// Partial results are emitted as providers finish; the final update sets
// SearchModsUpdate.Error when one or more providers failed.
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

// LookupProjectBySlugWithError preserves provider failures so adapters can
// distinguish a missing project from a network or provider error. It returns
// providers.ErrProjectNotFound when nothing matches and
// providers.ErrUnsupportedPlatform for an unknown platform.
func (s *Service) LookupProjectBySlugWithError(platform, slug, minecraftVersion, modLoader string) (models.ModProject, error) {
	return providers.LookupProjectByPlatformSlugWithError(platform, slug, minecraftVersion, modLoader)
}

// LookupProjectByID resolves a project by its platform project ID.
func (s *Service) LookupProjectByID(platform, projectID, minecraftVersion, modLoader string) (models.ModProject, bool) {
	return providers.LookupProjectByPlatformID(platform, projectID, minecraftVersion, modLoader)
}

// LookupProjectByIDWithError preserves provider failures. It returns
// providers.ErrProjectNotFound when nothing matches and
// providers.ErrUnsupportedPlatform for an unknown platform.
func (s *Service) LookupProjectByIDWithError(platform, projectID, minecraftVersion, modLoader string) (models.ModProject, error) {
	return providers.LookupProjectByPlatformIDWithError(platform, projectID, minecraftVersion, modLoader)
}

// ErrPinnedModNotFound reports a missing pin for adapters that need to
// distinguish it from a storage failure.
var ErrPinnedModNotFound = errors.New("pinned mod not found")

// GetPinnedModVersion returns the pinned version for a mod in one scope, or
// the zero value when no pin exists.
func (s *Service) GetPinnedModVersion(platform string, modID string, minecraftVersion string, modLoader string) storage.PinnedMod {
	pin, _ := s.GetPinnedModVersionWithError(platform, modID, minecraftVersion, modLoader)
	return pin
}

// GetPinnedModVersionWithError is GetPinnedModVersion with storage errors
// preserved. A missing pin is a normal empty result, not an error.
func (s *Service) GetPinnedModVersionWithError(platform string, modID string, minecraftVersion string, modLoader string) (storage.PinnedMod, error) {
	pin, found, err := storage.GetPinnedModWithError(platform, modID, minecraftVersion, modLoader)
	if err != nil {
		return storage.PinnedMod{}, err
	}
	if !found {
		return storage.PinnedMod{}, nil
	}
	return pin, nil
}

// PinModVersion toggles a pin: pinning the already-pinned version unpins it.
func (s *Service) PinModVersion(req appstructs.ModVersionPinRequest) storage.PinnedMod {
	pin, _ := s.PinModVersionWithError(req)
	return pin
}

// PinModVersionWithError is PinModVersion with storage errors preserved. It
// returns ErrPinnedModNotFound when a required request field is empty. Pinning
// the already-pinned version unpins it and returns the zero value with a nil
// error.
func (s *Service) PinModVersionWithError(req appstructs.ModVersionPinRequest) (storage.PinnedMod, error) {
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	modID := strings.ToLower(strings.TrimSpace(req.ModID))
	versionID := strings.TrimSpace(req.VersionID)
	minecraftVersion := strings.TrimSpace(req.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(req.ModLoader))

	if platform == "" || modID == "" || versionID == "" || minecraftVersion == "" || modLoader == "" {
		return storage.PinnedMod{}, ErrPinnedModNotFound
	}

	existing, found, err := storage.GetPinnedModWithError(platform, modID, minecraftVersion, modLoader)
	if err != nil {
		return storage.PinnedMod{}, err
	}
	if found && strings.EqualFold(existing.VersionID, versionID) {
		if err := storage.DeletePinnedMod(platform, modID, minecraftVersion, modLoader); err != nil {
			return storage.PinnedMod{}, err
		}
		return storage.PinnedMod{}, nil
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
		return storage.PinnedMod{}, err
	}
	saved, _, err := storage.GetPinnedModWithError(platform, modID, minecraftVersion, modLoader)
	if err != nil {
		return storage.PinnedMod{}, err
	}
	return saved, nil
}

// ListPinnedMods returns every pinned mod.
func (s *Service) ListPinnedMods() []storage.PinnedMod {
	return storage.ListPinnedMods()
}

// UnpinMod removes the pin for a mod in one scope.
func (s *Service) UnpinMod(platform, modID, minecraftVersion, modLoader string) bool {
	ok, _ := s.UnpinModWithError(platform, modID, minecraftVersion, modLoader)
	return ok
}

// UnpinModWithError is UnpinMod with storage errors preserved. A missing pin is
// an idempotent no-op: it returns false with a nil error.
func (s *Service) UnpinModWithError(platform, modID, minecraftVersion, modLoader string) (bool, error) {
	platform = strings.ToLower(strings.TrimSpace(platform))
	modID = strings.ToLower(strings.TrimSpace(modID))
	minecraftVersion = strings.TrimSpace(minecraftVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))
	if platform == "" || modID == "" || minecraftVersion == "" || modLoader == "" {
		return false, nil
	}
	if _, found, err := storage.GetPinnedModWithError(platform, modID, minecraftVersion, modLoader); err != nil {
		return false, err
	} else if !found {
		return false, nil
	}
	if err := storage.DeletePinnedMod(platform, modID, minecraftVersion, modLoader); err != nil {
		logging.Error("unpin mod failed", "platform", platform, "modID", modID, "minecraftVersion", minecraftVersion, "modLoader", modLoader, "error", err)
		return false, err
	}
	return true, nil
}
