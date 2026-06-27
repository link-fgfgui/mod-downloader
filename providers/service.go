package providers

import (
	"strings"
	"sync"

	"mod-downloader/database"
	"mod-downloader/logging"
	"mod-downloader/models"
	appstructs "mod-downloader/structs"
)

func SearchMods(req appstructs.SearchModsRequest, emitUpdate func(appstructs.SearchModsUpdate)) {
	req.Query = strings.TrimSpace(req.Query)
	req.Version = strings.TrimSpace(req.Version)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	providerResults := make(map[string][]models.ModProject, len(modProviders))
	var exactResults []models.ModProject
	remaining := len(modProviders)
	if req.Query == "" {
		remaining = len(modProviders)
	} else {
		remaining = len(modProviders) * 2
	}

	emit := func() {
		results := exactResults
		if len(results) == 0 {
			results = mergeProviderSearchResults(providerResults)
		}
		if emitUpdate != nil {
			emitUpdate(appstructs.SearchModsUpdate{
				RequestID: req.RequestID,
				Results:   results,
				Loading:   remaining > 0,
				Append:    req.Offset > 0,
			})
		}
	}

	finish := func(update func()) {
		mu.Lock()
		defer mu.Unlock()
		if update != nil {
			update()
		}
		remaining--
		emit()
	}

	if req.Query != "" {
		for _, provider := range modProviders {
			wg.Add(1)
			go func() {
				defer wg.Done()
				results, err := provider.ExactSearch(req)
				if err != nil {
					logging.Error("search exact mods failed", "provider", provider.Name(), "query", req.Query, "version", req.Version, "modLoader", req.ModLoader, "error", err)
				}
				finish(func() {
					exactResults = mergeSearchResultsByTitle(exactResults, results)
				})
			}()
		}
	}

	for _, provider := range modProviders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := provider.Search(req)
			if err != nil {
				logging.Error("search mods failed", "provider", provider.Name(), "query", req.Query, "version", req.Version, "modLoader", req.ModLoader, "error", err)
			}
			finish(func() {
				if len(exactResults) == 0 {
					providerResults[provider.Name()] = results
				}
			})
		}()
	}
	wg.Wait()
}

func ListMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
	provider, projectIDOrSlug, ok := providerAndProjectFromSearchResult(result)
	if !ok {
		return nil
	}

	req := appstructs.SearchModsRequest{
		Version:   strings.TrimSpace(minecraftVersion),
		ModLoader: strings.ToLower(strings.TrimSpace(modLoader)),
	}
	versions := listProjectVersionsForSearch(provider, projectIDOrSlug, req)
	return filterProjectVersionsForSearch(versions, req)
}

func RefreshMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
	provider, projectIDOrSlug, ok := providerAndProjectFromSearchResult(result)
	if !ok {
		return nil
	}

	filter := normalizeProjectVersionFilter(projectVersionFilter{
		MinecraftVersion: minecraftVersion,
		ModLoader:        modLoader,
	})
	versions, err := provider.ListVersions(projectIDOrSlug, filter)
	if err != nil || len(versions) == 0 {
		logging.Error("refresh matching project versions failed", "provider", provider.Name(), "projectIDOrSlug", projectIDOrSlug, "minecraftVersion", filter.MinecraftVersion, "modLoader", filter.ModLoader, "error", err)
		return nil
	}

	versions = sortModVersions(versions)
	projectID := projectIDFromVersions(versions, projectIDOrSlug)
	if err := saveProjectVersionsSnapshot(provider.Name(), projectIDOrSlug, projectID, versions, filter); err != nil {
		logging.Error("save refreshed project versions snapshot failed", "platform", provider.Name(), "requestedProject", projectIDOrSlug, "projectID", projectID, "versionCount", len(versions), "error", err)
	}
	return filterProjectVersionsForFilter(versions, filter)
}

func ProjectReferenceFromSearchResult(result models.ModProject) string {
	if result.ID != "" {
		return result.ID
	}
	if result.Platform != "" && result.Slug != "" {
		return strings.ToLower(result.Platform) + ":" + result.Slug
	}
	return result.Slug
}

func SplitProjectReference(projectIDOrSlug string) (string, string) {
	return splitProjectReference(projectIDOrSlug)
}

func ProjectVersionSHA1Set(result models.ModProject) map[string]bool {
	set := make(map[string]bool)
	provider, project, ok := providerAndProjectFromSearchResult(result)
	if !ok {
		return set
	}
	addProjectVersionSHA1s(set, provider, project)
	for _, associated := range associatedPlatformProjects(provider, project) {
		addProjectVersionSHA1s(set, associated.provider, associated.projectID)
	}
	return set
}

func providerAndProjectFromSearchResult(result models.ModProject) (modProvider, string, bool) {
	platform, project := splitProjectReference(result.ID)
	if project == "" {
		project = result.Slug
	}
	switch platform {
	case "modrinth":
		if strings.TrimSpace(result.Slug) != "" {
			project = result.Slug
		}
		return modrinthProvider{}, project, project != ""
	case "curseforge":
		if strings.TrimSpace(result.Slug) != "" {
			project = result.Slug
		}
		return curseForgeProvider{}, project, project != ""
	default:
		switch strings.ToLower(result.Platform) {
		case "modrinth":
			if strings.TrimSpace(result.Slug) != "" {
				project = result.Slug
			}
			return modrinthProvider{}, project, project != ""
		case "curseforge":
			if strings.TrimSpace(result.Slug) != "" {
				project = result.Slug
			}
			return curseForgeProvider{}, project, project != ""
		}
	}
	return nil, "", false
}

type associatedPlatformProject struct {
	provider  modProvider
	projectID string
}

func addProjectVersionSHA1s(set map[string]bool, provider modProvider, project string) {
	for _, v := range listProjectVersions(provider, project) {
		if s := strings.ToLower(strings.TrimSpace(v.SHA1)); s != "" {
			set[s] = true
		}
	}
}

func associatedPlatformProjects(provider modProvider, project string) []associatedPlatformProject {
	platform := strings.ToLower(strings.TrimSpace(provider.Name()))
	project = strings.TrimSpace(project)
	if project == "" {
		return nil
	}

	switch platform {
	case "curseforge":
		if association, ok := database.GetAssociationByCurseForge(project); ok && strings.TrimSpace(association.ModrinthProjectID) != "" {
			return []associatedPlatformProject{{provider: modrinthProvider{}, projectID: association.ModrinthProjectID}}
		}
	case "modrinth":
		if association, ok := database.GetAssociationByModrinth(project); ok && strings.TrimSpace(association.CurseForgeProjectID) != "" {
			return []associatedPlatformProject{{provider: curseForgeProvider{}, projectID: association.CurseForgeProjectID}}
		}
	}
	return nil
}
