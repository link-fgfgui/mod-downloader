package providers

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

const localMetadataMissTTL = 24 * time.Hour

var localMetadataResolveMu sync.Mutex

var providerBackground = struct {
	lifecycle sync.Mutex
	mu        sync.Mutex
	accepting bool
	starts    int
	wg        sync.WaitGroup
}{accepting: true}

func SearchMods(req appstructs.SearchModsRequest, emitUpdate func(appstructs.SearchModsUpdate)) {
	req = normalizeSearchModsRequest(req)

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

func SearchModsText(req appstructs.SearchModsRequest, emitUpdate func(appstructs.SearchModsUpdate)) {
	req = normalizeSearchModsRequest(req)

	var wg sync.WaitGroup
	var mu sync.Mutex
	providerResults := make(map[string][]models.ModProject, len(modProviders))
	remaining := len(modProviders)

	emit := func() {
		if emitUpdate != nil {
			emitUpdate(appstructs.SearchModsUpdate{
				RequestID: req.RequestID,
				Results:   mergeProviderSearchResults(providerResults),
				Loading:   remaining > 0,
				Append:    req.Offset > 0,
			})
		}
	}

	finish := func(providerName string, results []models.ModProject) {
		mu.Lock()
		defer mu.Unlock()
		providerResults[providerName] = results
		remaining--
		emit()
	}

	for _, provider := range modProviders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results, err := provider.Search(req)
			if err != nil {
				logging.Error("search text mods failed", "provider", provider.Name(), "query", req.Query, "version", req.Version, "modLoader", req.ModLoader, "error", err)
			}
			finish(provider.Name(), results)
		}()
	}
	wg.Wait()

	mu.Lock()
	finalResults := mergeProviderSearchResults(providerResults)
	mu.Unlock()
	go cacheSearchResults(finalResults)
}

func normalizeSearchModsRequest(req appstructs.SearchModsRequest) appstructs.SearchModsRequest {
	req.Query = strings.TrimSpace(req.Query)
	req.Version = strings.TrimSpace(req.Version)
	req.ModLoader = strings.ToLower(strings.TrimSpace(req.ModLoader))
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	return req
}

func ResolveMetadataByHashes(hashes []string) (map[string]models.ModProject, map[string]models.ModVersion) {
	files := make(map[string]uint32, len(hashes))
	for _, hash := range hashes {
		if hash = strings.ToLower(strings.TrimSpace(hash)); hash != "" {
			files[hash] = 0
		}
	}
	return ResolveMetadataByLocalFiles(files)
}

// ResolveMetadataByLocalFiles resolves local SHA1 values through Modrinth and
// CurseForge fingerprints through CurseForge. Results are keyed by SHA1.
func ResolveMetadataByLocalFiles(files map[string]uint32) (map[string]models.ModProject, map[string]models.ModVersion) {
	localMetadataResolveMu.Lock()
	defer localMetadataResolveMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-localMetadataMissTTL).Unix()
	hashes := make([]string, 0, len(files))
	fingerprintSet := make(map[uint32]struct{})
	for hash, fingerprint := range files {
		if hash = strings.ToLower(strings.TrimSpace(hash)); hash != "" && !hasRecentLocalMetadataMiss("modrinth", hash, cutoff) {
			hashes = append(hashes, hash)
		}
		if fingerprint != 0 && !hasRecentLocalMetadataMiss("curseforge", strconv.FormatUint(uint64(fingerprint), 10), cutoff) {
			fingerprintSet[fingerprint] = struct{}{}
		}
	}
	if len(hashes) == 0 && len(fingerprintSet) == 0 {
		return nil, nil
	}
	sort.Strings(hashes)
	projects := make(map[string]models.ModProject)
	versions := make(map[string]models.ModVersion)
	if len(hashes) > 0 && global.GetModrinthClient() != nil {
		resolved, err := modrinthProvider{}.resolveProjectsByHashes(hashes)
		if err != nil {
			logging.Error("resolve Modrinth projects by hashes failed", "hashCount", len(hashes), "error", err)
		} else {
			for hash, project := range resolved.projects {
				projects[strings.ToLower(hash)] = project
			}
			for hash, version := range resolved.versions {
				versions[strings.ToLower(hash)] = version
			}
			recordLocalMetadataMisses("modrinth", hashes, func(identity string) bool {
				_, ok := resolved.projects[identity]
				return ok
			}, now.Unix())
		}
	}
	fingerprints := make([]uint32, 0, len(fingerprintSet))
	for fingerprint := range fingerprintSet {
		fingerprints = append(fingerprints, fingerprint)
	}
	sort.Slice(fingerprints, func(i, j int) bool { return fingerprints[i] < fingerprints[j] })
	if len(fingerprints) > 0 && global.GetCurseForgeClient() != nil {
		curseForgeResolved, err := curseForgeProvider{}.resolveProjectsByFingerprints(fingerprints)
		if err != nil {
			logging.Error("resolve CurseForge projects by fingerprints failed", "fingerprintCount", len(fingerprints), "error", err)
		} else {
			identities := make([]string, 0, len(fingerprints))
			for _, fingerprint := range fingerprints {
				identities = append(identities, strconv.FormatUint(uint64(fingerprint), 10))
			}
			recordLocalMetadataMisses("curseforge", identities, func(identity string) bool {
				fingerprint, parseErr := strconv.ParseUint(identity, 10, 32)
				if parseErr != nil {
					return false
				}
				_, ok := curseForgeResolved.projects[uint32(fingerprint)]
				return ok
			}, now.Unix())
			for hash, fingerprint := range files {
				hash = strings.ToLower(strings.TrimSpace(hash))
				if hash == "" || fingerprint == 0 {
					continue
				}
				if _, exists := projects[hash]; exists {
					continue
				}
				if project, ok := curseForgeResolved.projects[fingerprint]; ok {
					projects[hash] = project
					version := curseForgeResolved.versions[fingerprint]
					if strings.TrimSpace(version.SHA1) == "" {
						version.SHA1 = hash
					}
					versions[hash] = version
				}
			}
		}
	}
	cacheResolvedMetadata(projects, versions)
	return projects, versions
}

func hasRecentLocalMetadataMiss(provider, identity string, cutoff int64) bool {
	checkedAt, ok := storage.GetLocalMetadataMissCheckedAt(provider, identity)
	return ok && checkedAt >= cutoff
}

func recordLocalMetadataMisses(provider string, identities []string, resolved func(string) bool, checkedAt int64) {
	misses := make([]string, 0, len(identities))
	for _, identity := range identities {
		if !resolved(identity) {
			misses = append(misses, identity)
		}
	}
	if err := storage.SetLocalMetadataMisses(provider, misses, checkedAt); err != nil {
		logging.Error("cache local metadata misses failed", "provider", provider, "identityCount", len(misses), "error", err)
	}
}

func cacheResolvedMetadata(projects map[string]models.ModProject, versions map[string]models.ModVersion) {
	for _, project := range projects {
		if strings.TrimSpace(project.Title) == "" {
			continue
		}
		if err := storage.UpsertModPlatform(project); err != nil {
			logging.Error("cache resolved project failed", "platform", project.Platform, "projectID", project.ProjectID, "error", err)
		}
	}
	// Cache versions so that GetVersionBySHA1 finds them on next sync pass.
	versionsByProject := make(map[string][]models.ModVersion)
	for _, v := range versions {
		if v.ProjectID == "" {
			continue
		}
		key := models.ProjectKey(v.Platform, v.ProjectID)
		versionsByProject[key] = append(versionsByProject[key], v)
	}
	for key, projectVersions := range versionsByProject {
		platform, projectID := models.ParseProjectKey(key)
		if err := storage.SetPlatformVersions(platform, projectID, projectVersions); err != nil {
			logging.Error("cache resolved versions failed", "platform", platform, "projectID", projectID, "error", err)
		}
	}
}

// ResolveProjectsByHashes preserves the project-only API for existing core consumers.
func ResolveProjectsByHashes(hashes []string) map[string]models.ModProject {
	projects, _ := ResolveMetadataByHashes(hashes)
	return projects
}

func cacheSearchResults(results []models.ModProject) {
	for _, r := range results {
		if strings.TrimSpace(r.Title) == "" {
			continue
		}
		if err := storage.UpsertModPlatform(r); err != nil {
			logging.Error("cache search result failed", "platform", r.Platform, "projectID", r.ProjectID, "error", err)
		}
	}
}

func ListMatchingProjectVersions(result models.ModProject, minecraftVersion string, modLoader string) []models.ModVersion {
	provider, projectIDOrSlug, ok := providerAndProjectFromSearchResult(result)
	if !ok {
		return nil
	}

	startBackgroundTask(func() {
		refreshProjectMetadataIfStale(provider, provider.Name(), projectIDOrSlug)
	})
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

// StartBackgroundTasks marks background work as accepting and takes one
// reference. Every Start must be paired with a Stop; the last Stop waits for
// in-flight work and rejects new tasks.
func StartBackgroundTasks() {
	providerBackground.lifecycle.Lock()
	defer providerBackground.lifecycle.Unlock()
	providerBackground.mu.Lock()
	providerBackground.starts++
	providerBackground.accepting = true
	providerBackground.mu.Unlock()
}

// StopBackgroundTasks releases one reference. Once the last reference is
// released it waits for in-flight tasks and disables background work.
func StopBackgroundTasks() {
	providerBackground.lifecycle.Lock()
	defer providerBackground.lifecycle.Unlock()
	providerBackground.mu.Lock()
	if providerBackground.starts > 0 {
		providerBackground.starts--
	}
	if providerBackground.starts > 0 {
		providerBackground.mu.Unlock()
		return
	}
	providerBackground.accepting = false
	providerBackground.mu.Unlock()
	providerBackground.wg.Wait()
}

// DrainBackgroundTasks waits for in-flight background work while preserving
// the current accepting state and reference count. It is useful before closing
// shared resources such as storage without permanently disabling refreshes.
func DrainBackgroundTasks() {
	providerBackground.lifecycle.Lock()
	defer providerBackground.lifecycle.Unlock()
	providerBackground.mu.Lock()
	wasAccepting := providerBackground.accepting
	providerBackground.accepting = false
	providerBackground.mu.Unlock()
	providerBackground.wg.Wait()
	if !wasAccepting {
		return
	}
	providerBackground.mu.Lock()
	providerBackground.accepting = true
	providerBackground.mu.Unlock()
}

func startBackgroundTask(task func()) {
	if task == nil {
		return
	}
	providerBackground.mu.Lock()
	if !providerBackground.accepting {
		providerBackground.mu.Unlock()
		return
	}
	providerBackground.wg.Add(1)
	go func() {
		defer providerBackground.wg.Done()
		task()
	}()
	providerBackground.mu.Unlock()
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

func LookupProjectByPlatformSlug(platform, slug, mcVersion, modLoader string) (models.ModProject, bool) {
	return lookupProjectByPlatformMatching(platform, slug, mcVersion, modLoader, func(project models.ModProject, expected string) bool {
		return strings.EqualFold(strings.TrimSpace(project.Slug), expected)
	})
}

func LookupProjectByPlatformID(platform, projectID, mcVersion, modLoader string) (models.ModProject, bool) {
	return lookupProjectByPlatformMatching(platform, projectID, mcVersion, modLoader, func(project models.ModProject, expected string) bool {
		return strings.EqualFold(strings.TrimSpace(project.ProjectID), expected)
	})
}

func lookupProjectByPlatformMatching(platform, idOrSlug, mcVersion, modLoader string, match func(models.ModProject, string) bool) (models.ModProject, bool) {
	expected := strings.TrimSpace(idOrSlug)
	if expected == "" {
		return models.ModProject{}, false
	}
	project, ok := LookupProjectByPlatform(platform, expected, mcVersion, modLoader)
	if !ok || !match(project, expected) {
		return models.ModProject{}, false
	}
	return project, true
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
	startBackgroundTask(func() {
		refreshProjectMetadataIfStale(provider, provider.Name(), project)
	})
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
		if association, ok := storage.GetAssociationByCurseForge(project); ok && strings.TrimSpace(association.ModrinthProjectID) != "" {
			return []associatedPlatformProject{{provider: modrinthProvider{}, projectID: association.ModrinthProjectID}}
		}
	case "modrinth":
		if association, ok := storage.GetAssociationByModrinth(project); ok && strings.TrimSpace(association.CurseForgeProjectID) != "" {
			return []associatedPlatformProject{{provider: curseForgeProvider{}, projectID: association.CurseForgeProjectID}}
		}
	}
	return nil
}
