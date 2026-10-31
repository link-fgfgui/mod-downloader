package appcore

import (
	"errors"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
)

func (s *Service) ListFavoriteLists() []storage.FavoriteList {
	return storage.ListFavoriteLists()
}

func (s *Service) CreateFavoriteList(name string) storage.FavoriteList {
	return s.CreateFavoriteListForScope(name, "", "")
}

func (s *Service) CreateFavoriteListForScope(name, minecraftVersion, modLoader string) storage.FavoriteList {
	list, err := storage.CreateFavoriteListForScope(name, minecraftVersion, modLoader)
	if err != nil {
		logging.Error("create favorite list failed", "name", name, "error", err)
		return storage.FavoriteList{}
	}
	return list
}

func (s *Service) RenameFavoriteList(id, name string) storage.FavoriteList {
	list, ok, err := storage.UpdateFavoriteListName(id, name)
	if err != nil {
		logging.Error("rename favorite list failed", "id", id, "name", name, "error", err)
		return storage.FavoriteList{}
	}
	if !ok {
		return storage.FavoriteList{}
	}
	return list
}

func (s *Service) DeleteFavoriteList(id string) bool {
	deleted, err := storage.DeleteFavoriteList(id)
	if err != nil {
		logging.Error("delete favorite list failed", "id", id, "error", err)
		return false
	}
	return deleted
}

func (s *Service) UpdateFavoriteListMetadata(list storage.FavoriteList) storage.FavoriteList {
	updated, ok, err := storage.UpdateFavoriteListMetadata(list)
	if err != nil {
		logging.Error("update favorite list metadata failed", "id", list.ID, "error", err)
		return storage.FavoriteList{}
	}
	if !ok {
		return storage.FavoriteList{}
	}
	return updated
}

func (s *Service) ReorderFavoriteLists(ids []string) bool {
	if err := storage.ReorderFavoriteLists(ids); err != nil {
		logging.Error("reorder favorite lists failed", "error", err)
		return false
	}
	return true
}

func (s *Service) ListFavoriteMods(listID string) []storage.FavoriteMod {
	return storage.ListFavoriteMods(listID)
}

func (s *Service) ListFavoriteContents(listID string) storage.FavoriteListContents {
	return storage.ListFavoriteContents(listID)
}

func (s *Service) AddFavoriteMod(mod storage.FavoriteMod) storage.FavoriteMod {
	_, existed := favoriteModKeySet(storage.ListFavoriteMods(mod.ListID))[favoriteModIdentityKey(mod)]
	saved, ok, err := storage.UpsertFavoriteMod(mod)
	if err != nil {
		logging.Error("add favorite mod failed", "listID", mod.ListID, "platform", mod.Platform, "modID", mod.ModID, "error", err)
		return storage.FavoriteMod{}
	}
	if !ok {
		return storage.FavoriteMod{}
	}
	if !existed {
		s.incrementUsageStat(storage.UsageStatFavoritesAdded, 1)
	}
	return saved
}

func (s *Service) AddFavoriteModsToLists(req FavoriteBulkAddRequest) FavoriteBulkOperationResult {
	targetListIDs := uniqueNonEmpty(req.TargetListIDs)
	if len(targetListIDs) == 0 || len(req.Mods) == 0 {
		return FavoriteBulkOperationResult{Skipped: len(req.Mods)}
	}

	lists := favoriteListIDSet()
	result := FavoriteBulkOperationResult{}
	for _, targetListID := range targetListIDs {
		if _, ok := lists[targetListID]; !ok {
			result.Skipped += len(req.Mods)
			result.Errors = append(result.Errors, "missing target list: "+targetListID)
			continue
		}

		existing := favoriteModKeySet(storage.ListFavoriteMods(targetListID))
		for _, mod := range req.Mods {
			copied := favoriteModForTargetList(mod, targetListID)
			if copied.Platform == "" || copied.ModID == "" {
				result.Skipped++
				continue
			}
			key := favoriteModIdentityKey(copied)
			_, wasExisting := existing[key]
			if _, ok, err := storage.UpsertFavoriteMod(copied); err != nil {
				logging.Error("bulk add favorite mod failed", "targetListID", targetListID, "platform", copied.Platform, "modID", copied.ModID, "error", err)
				result.Skipped++
				result.Errors = append(result.Errors, err.Error())
			} else if !ok {
				result.Skipped++
			} else if wasExisting {
				result.Updated++
			} else {
				result.Added++
				existing[key] = struct{}{}
			}
		}
	}
	if result.Added > 0 {
		s.incrementUsageStat(storage.UsageStatFavoritesAdded, int64(result.Added))
	}
	return result
}

func (s *Service) CopyFavoriteListToList(req FavoriteListCopyRequest) FavoriteBulkOperationResult {
	sourceListID := strings.TrimSpace(req.SourceListID)
	targetListID := strings.TrimSpace(req.TargetListID)
	if sourceListID == "" || targetListID == "" {
		return FavoriteBulkOperationResult{}
	}
	if sourceListID == targetListID {
		contents := storage.ListFavoriteContents(sourceListID)
		return FavoriteBulkOperationResult{
			Skipped: len(contents.Mods),
			Errors:  []string{"source and target lists are the same"},
		}
	}
	contents := storage.ListFavoriteContents(sourceListID)
	mods := make([]storage.FavoriteMod, 0, len(contents.Mods))
	for _, entry := range contents.Mods {
		mods = append(mods, entry.FavoriteMod)
	}
	return s.AddFavoriteModsToLists(FavoriteBulkAddRequest{
		TargetListIDs: []string{targetListID},
		Mods:          mods,
	})
}

func (s *Service) PreviewFavoriteListCrossVersionCopy(req FavoriteCrossVersionCopyRequest) FavoriteCrossVersionCopyPreview {
	sourceListID := strings.TrimSpace(req.SourceListID)
	minecraftVersion := strings.TrimSpace(req.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(req.ModLoader))
	preview := FavoriteCrossVersionCopyPreview{
		SourceListID:     sourceListID,
		MinecraftVersion: minecraftVersion,
		ModLoader:        modLoader,
	}
	if sourceListID == "" {
		preview.Errors = append(preview.Errors, "source list is required")
		return preview
	}
	if minecraftVersion == "" || modLoader == "" {
		preview.Errors = append(preview.Errors, "minecraft version and modloader are required")
		return preview
	}
	sourceList, ok := storage.GetFavoriteList(sourceListID)
	if !ok {
		preview.Errors = append(preview.Errors, "source list not found")
		return preview
	}
	preview.TargetListName = sourceList.Name
	if sourceList.MinecraftVersion == minecraftVersion && sourceList.ModLoader == modLoader {
		preview.Errors = append(preview.Errors, "target scope must differ from source scope")
		return preview
	}
	if _, found := storage.GetFavoriteListByNameAndScope(sourceList.Name, minecraftVersion, modLoader); found {
		preview.NameConflict = true
		preview.Errors = append(preview.Errors, "a favorite list with the same name already exists in the target scope")
		return preview
	}

	contents := storage.ListFavoriteContents(sourceListID)
	for _, entry := range contents.Mods {
		project, ok := s.favoriteCrossVersionCopyProject(entry.FavoriteMod, minecraftVersion, modLoader)
		if !ok {
			preview.Conflicts = append(preview.Conflicts, FavoriteCrossVersionCopyConflict{
				Source: entry,
				Reason: "project not found",
			})
			continue
		}
		versions := providers.ListMatchingProjectVersions(project, minecraftVersion, modLoader)
		if len(versions) == 0 {
			preview.Conflicts = append(preview.Conflicts, FavoriteCrossVersionCopyConflict{
				Source: entry,
				Reason: "matching version not found",
			})
			continue
		}
		target := favoriteModForTargetList(entry.FavoriteMod, "")
		target.MinecraftVersion = minecraftVersion
		target.ModLoader = modLoader
		target.VersionID = favoriteCrossVersionCopyVersionID(versions[0])
		if strings.TrimSpace(target.ModID) == "" {
			target.ModID = strings.TrimSpace(project.ProjectID)
		}
		if strings.TrimSpace(target.Platform) == "" {
			target.Platform = strings.TrimSpace(project.Platform)
		}
		if strings.TrimSpace(target.Slug) == "" {
			target.Slug = strings.TrimSpace(project.Slug)
		}
		preview.Matched = append(preview.Matched, FavoriteCrossVersionCopyMatch{
			Source:  entry,
			Target:  target,
			Project: project,
			Version: versions[0],
		})
	}
	return preview
}

func (s *Service) ApplyFavoriteListCrossVersionCopy(req FavoriteCrossVersionCopyRequest) FavoriteCrossVersionCopyApplyResult {
	preview := s.PreviewFavoriteListCrossVersionCopy(req)
	result := FavoriteCrossVersionCopyApplyResult{Preview: preview}
	if len(preview.Errors) > 0 {
		result.Result.Errors = append(result.Result.Errors, preview.Errors...)
		return result
	}
	if len(preview.Conflicts) > 0 && !req.IgnoreConflicts {
		result.Result.Skipped = len(preview.Matched) + len(preview.Conflicts)
		result.Result.Errors = append(result.Result.Errors, "copy conflicts must be ignored or resolved before applying")
		return result
	}
	if len(preview.Matched) == 0 {
		result.Result.Skipped = len(preview.Conflicts)
		result.Result.Errors = append(result.Result.Errors, "no matching mods to copy")
		return result
	}
	mods := make([]storage.FavoriteMod, 0, len(preview.Matched))
	for _, match := range preview.Matched {
		mods = append(mods, match.Target)
	}
	targetList, saved, err := storage.CreateFavoriteListWithModsForScope(
		preview.TargetListName,
		preview.MinecraftVersion,
		preview.ModLoader,
		mods,
	)
	if err != nil {
		if errors.Is(err, storage.ErrFavoriteListNameConflict) {
			result.Preview.NameConflict = true
			result.Result.Errors = append(result.Result.Errors, "a favorite list with the same name already exists in the target scope")
		} else {
			logging.Error("cross-version favorite copy failed", "sourceListID", preview.SourceListID, "minecraftVersion", preview.MinecraftVersion, "modLoader", preview.ModLoader, "error", err)
			result.Result.Errors = append(result.Result.Errors, err.Error())
		}
		result.Result.Skipped = len(preview.Matched) + len(preview.Conflicts)
		return result
	}
	result.TargetList = targetList
	result.Result.Added = len(saved)
	result.Result.Skipped = len(preview.Conflicts)
	result.Applied = result.Result.Added > 0
	if result.Applied {
		s.incrementUsageStat(storage.UsageStatFavoritesAdded, int64(result.Result.Added))
	}
	return result
}

func (s *Service) favoriteCrossVersionCopyProject(mod storage.FavoriteMod, minecraftVersion, modLoader string) (models.ModProject, bool) {
	platform := strings.TrimSpace(mod.Platform)
	modID := strings.TrimSpace(mod.ModID)
	slug := strings.TrimSpace(mod.Slug)
	if platform == "" {
		return models.ModProject{}, false
	}
	if modID != "" {
		if project, ok := storage.GetModPlatform(platform, modID); ok {
			return favoriteCrossVersionCopyProjectWithID(project), true
		}
		if project, ok := storage.GetModPlatformBySlug(platform, modID); ok {
			return favoriteCrossVersionCopyProjectWithID(project), true
		}
	}
	if slug != "" {
		if project, ok := storage.GetModPlatformBySlug(platform, slug); ok {
			return favoriteCrossVersionCopyProjectWithID(project), true
		}
		if project, ok := storage.GetModPlatform(platform, slug); ok {
			return favoriteCrossVersionCopyProjectWithID(project), true
		}
	}
	if modID != "" {
		if project, ok := favoriteCrossVersionCopyLookupProjectByID(platform, modID, minecraftVersion, modLoader); ok {
			return favoriteCrossVersionCopyProjectWithID(project), true
		}
	}
	if slug != "" {
		if project, ok := favoriteCrossVersionCopyLookupProjectBySlug(platform, slug, minecraftVersion, modLoader); ok {
			return favoriteCrossVersionCopyProjectWithID(project), true
		}
	}
	return models.ModProject{}, false
}

func favoriteCrossVersionCopyProjectWithID(project models.ModProject) models.ModProject {
	if strings.TrimSpace(project.ID) == "" {
		project.ID = models.ProjectKey(project.Platform, project.ProjectID)
	}
	return project
}

func favoriteCrossVersionCopyVersionID(version models.ModVersion) string {
	if id := strings.TrimSpace(version.VersionID); id != "" {
		return id
	}
	return strings.TrimSpace(version.ID)
}

func (s *Service) AddFavoriteListReference(parentListID, childListID string) storage.FavoriteListRef {
	ref, ok, err := storage.CreateFavoriteListRef(parentListID, childListID)
	if err != nil {
		logging.Error("add favorite list reference failed", "parentListID", parentListID, "childListID", childListID, "error", err)
		return storage.FavoriteListRef{}
	}
	if !ok {
		return storage.FavoriteListRef{}
	}
	return ref
}

func (s *Service) RemoveFavoriteListReference(parentListID, childListID string) bool {
	deleted, err := storage.DeleteFavoriteListRef(parentListID, childListID)
	if err != nil {
		logging.Error("remove favorite list reference failed", "parentListID", parentListID, "childListID", childListID, "error", err)
		return false
	}
	return deleted
}

func (s *Service) ListFavoriteListRefs(parentListID string) []storage.FavoriteListRef {
	return storage.ListFavoriteListRefs(parentListID)
}

func (s *Service) RemoveFavoriteMod(listID, platform, modID, mcVersion, modLoader string) bool {
	deleted, err := storage.DeleteFavoriteMod(listID, platform, modID, mcVersion, modLoader)
	if err != nil {
		logging.Error("remove favorite mod failed", "listID", listID, "platform", platform, "modID", modID, "error", err)
		return false
	}
	return deleted
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func favoriteListIDSet() map[string]struct{} {
	lists := storage.ListFavoriteLists()
	out := make(map[string]struct{}, len(lists))
	for _, list := range lists {
		if id := strings.TrimSpace(list.ID); id != "" {
			out[id] = struct{}{}
		}
	}
	return out
}

func favoriteModKeySet(mods []storage.FavoriteMod) map[string]struct{} {
	out := make(map[string]struct{}, len(mods))
	for _, mod := range mods {
		if key := favoriteModIdentityKey(mod); key != "" {
			out[key] = struct{}{}
		}
	}
	return out
}

func favoriteModForTargetList(mod storage.FavoriteMod, targetListID string) storage.FavoriteMod {
	mod.ID = ""
	mod.ListID = strings.TrimSpace(targetListID)
	mod.CreatedAt = 0
	mod.UpdatedAt = 0
	return mod
}

func favoriteModIdentityKey(mod storage.FavoriteMod) string {
	platform := strings.ToLower(strings.TrimSpace(mod.Platform))
	modID := strings.ToLower(strings.TrimSpace(mod.ModID))
	mcVersion := strings.TrimSpace(mod.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(mod.ModLoader))
	if platform == "" || modID == "" {
		return ""
	}
	return platform + "|" + modID + "|" + mcVersion + "|" + modLoader
}
