package appcore

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

func (s *Service) SearchMods(req appstructs.SearchModsRequest) {
	providers.SearchMods(req, func(update appstructs.SearchModsUpdate) {
		s.emit(EventSearchModsUpdated, update)
	})
}

func (s *Service) SearchModsCollect(req appstructs.SearchModsRequest) appstructs.SearchModsUpdate {
	var last appstructs.SearchModsUpdate
	providers.SearchMods(req, func(update appstructs.SearchModsUpdate) {
		last = update
	})
	return last
}

func (s *Service) SearchModsTextCollect(req appstructs.SearchModsRequest) appstructs.SearchModsUpdate {
	var last appstructs.SearchModsUpdate
	providers.SearchModsText(req, func(update appstructs.SearchModsUpdate) {
		last = update
	})
	return last
}

func (s *Service) ListMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
	return providers.ListMatchingProjectVersions(result, minecraftVersion, modLoader)
}

func (s *Service) LookupProject(platform, idOrSlug, mcVersion, modLoader string) (models.ModProject, bool) {
	return providers.LookupProjectByPlatform(platform, idOrSlug, mcVersion, modLoader)
}

func (s *Service) LookupProjectBySlug(platform, slug, mcVersion, modLoader string) (models.ModProject, bool) {
	return providers.LookupProjectByPlatformSlug(platform, slug, mcVersion, modLoader)
}

func (s *Service) LookupProjectByID(platform, projectID, mcVersion, modLoader string) (models.ModProject, bool) {
	return providers.LookupProjectByPlatformID(platform, projectID, mcVersion, modLoader)
}

func (s *Service) GetPinnedModVersion(platform string, modID string, minecraftVersion string, modLoader string) storage.PinnedMod {
	pin, _ := storage.GetPinnedMod(platform, modID, minecraftVersion, modLoader)
	return pin
}

func (s *Service) PinModVersion(req appstructs.ModVersionPinRequest) storage.PinnedMod {
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	modID := strings.ToLower(strings.TrimSpace(req.ModID))
	versionID := strings.TrimSpace(req.VersionID)
	mcVersion := strings.TrimSpace(req.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(req.ModLoader))

	if platform == "" || modID == "" || versionID == "" || mcVersion == "" || modLoader == "" {
		return storage.PinnedMod{}
	}

	existing, found := storage.GetPinnedMod(platform, modID, mcVersion, modLoader)
	if found && strings.EqualFold(existing.VersionID, versionID) {
		_ = storage.DeletePinnedMod(platform, modID, mcVersion, modLoader)
		return storage.PinnedMod{}
	}

	pin := storage.PinnedMod{
		Platform:         platform,
		ModID:            modID,
		VersionID:        versionID,
		MinecraftVersion: mcVersion,
		ModLoader:        modLoader,
	}
	if err := storage.UpsertPinnedMod(pin); err != nil {
		logging.Error("upsert pinned mod failed", "platform", platform, "modID", modID, "versionID", versionID, "minecraftVersion", mcVersion, "modLoader", modLoader, "error", err)
		return storage.PinnedMod{}
	}
	pin, _ = storage.GetPinnedMod(platform, modID, mcVersion, modLoader)
	return pin
}

func (s *Service) ListPinnedMods() []storage.PinnedMod {
	return storage.ListPinnedMods()
}

func (s *Service) UnpinMod(platform, modID, mcVersion, modLoader string) bool {
	platform = strings.ToLower(strings.TrimSpace(platform))
	modID = strings.ToLower(strings.TrimSpace(modID))
	mcVersion = strings.TrimSpace(mcVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))
	if platform == "" || modID == "" || mcVersion == "" || modLoader == "" {
		return false
	}
	if _, found := storage.GetPinnedMod(platform, modID, mcVersion, modLoader); !found {
		return false
	}
	if err := storage.DeletePinnedMod(platform, modID, mcVersion, modLoader); err != nil {
		logging.Error("unpin mod failed", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader, "error", err)
		return false
	}
	return true
}
