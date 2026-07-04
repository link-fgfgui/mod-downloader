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

	mu.Lock()
	var finalResults []models.ModProject
	if len(exactResults) > 0 {
		finalResults = exactResults
	} else {
		finalResults = mergeProviderSearchResults(providerResults)
	}
	mu.Unlock()
	go cacheSearchResults(finalResults)
}

func ResolveProjectsByHashes(hashes []string) map[string]models.ModProject {
	if len(hashes) == 0 {
		return nil
	}
	resolved, err := modrinthProvider{}.resolveProjectsByHashes(hashes)
	if err != nil {
		logging.Error("resolve projects by hashes failed", "hashCount", len(hashes), "error", err)
		return nil
	}
	// Cache projects to the database.
	for _, project := range resolved.projects {
		if strings.TrimSpace(project.Title) == "" {
			continue
		}
		if err := database.UpsertModPlatform(project); err != nil {
			logging.Error("cache resolved project failed", "platform", project.Platform, "projectID", project.ProjectID, "error", err)
		}
	}
	// Cache versions so that GetVersionBySHA1 finds them on next sync pass.
	versionsByProject := make(map[string][]models.ModVersion)
	for _, v := range resolved.versions {
		if v.ProjectID == "" {
			continue
		}
		versionsByProject[v.ProjectID] = append(versionsByProject[v.ProjectID], v)
	}
	for projectID, versions := range versionsByProject {
		if err := database.SetPlatformVersions("Modrinth", projectID, versions); err != nil {
			logging.Error("cache resolved versions failed", "projectID", projectID, "error", err)
		}
	}
	return resolved.projects
}

func cacheSearchResults(results []models.ModProject) {
	for _, r := range results {
		if strings.TrimSpace(r.Title) == "" {
			continue
		}
		if err := database.UpsertModPlatform(r); err != nil {
			logging.Error("cache search result failed", "platform", r.Platform, "projectID", r.ProjectID, "error", err)
		}
	}
}

func ListMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
	provider, projectIDOrSlug, ok := providerAndProjectFromSearchResult(result)
	if !ok {
		return nil
	}

	go refreshProjectMetadataIfStale(provider, provider.Name(), projectIDOrSlug)
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

func LookupProjectByPlatform(platform, idOrSlug, mcVersion, modLoader string) (models.ModProject, bool) {
	provider := providerByPlatform(platform)
	if provider == nil {
		return models.ModProject{}, false
	}
	results, err := provider.ExactSearch(appstructs.SearchModsRequest{
		Query:     idOrSlug,
		Version:   strings.TrimSpace(mcVersion),
		ModLoader: strings.ToLower(strings.TrimSpace(modLoader)),
	})
	if err != nil || len(results) == 0 {
		return models.ModProject{}, false
	}
	return results[0], true
}

func providerByPlatform(platform string) modProvider {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "curseforge":
		return curseForgeProvider{}
	case "modrinth":
		return modrinthProvider{}
	}
	return nil
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
	go refreshProjectMetadataIfStale(provider, provider.Name(), project)
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
