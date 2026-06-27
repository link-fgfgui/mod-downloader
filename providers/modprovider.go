package providers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"mod-downloader/database"
	"mod-downloader/global"
	"mod-downloader/logging"
	"mod-downloader/models"
	appstructs "mod-downloader/structs"

	modrinth "codeberg.org/jmansfield/go-modrinth/modrinth"

	cfFiles "github.com/sjet47/go-curseforge/api/files"
	cfMods "github.com/sjet47/go-curseforge/api/mods"
	cfSchema "github.com/sjet47/go-curseforge/schema"
	cfEnum "github.com/sjet47/go-curseforge/schema/enum"
)

const (
	projectMetadataTTL         = 30 * 24 * time.Hour
	projectVersionsSnapshotTTL = 15 * time.Minute
)

// modProvider defines the interface for mod platform providers.
type modProvider interface {
	Name() string
	ExactSearch(req appstructs.SearchModsRequest) ([]models.ModProject, error)
	Search(req appstructs.SearchModsRequest) ([]models.ModProject, error)
	ListVersions(projectIDOrSlug string, filter projectVersionFilter) ([]models.ModVersion, error)
}

type projectVersionFilter struct {
	MinecraftVersion string
	ModLoader        string
}

type curseForgeProvider struct{}

type modrinthProvider struct{}

func (curseForgeProvider) Name() string { return "CurseForge" }

func (p curseForgeProvider) ExactSearch(req appstructs.SearchModsRequest) ([]models.ModProject, error) {
	client := global.GetCurseForgeClient()
	if client == nil {
		return nil, nil
	}

	mods, err := p.exactCandidates(req.Query)
	if err != nil {
		return nil, err
	}

	results := make([]models.ModProject, 0, len(mods))
	for _, mod := range mods {
		result := p.modToModProject(mod)
		if projectHasExactMatchingVersion(p, result, req) {
			results = append(results, result)
		}
	}
	return results, nil
}

func (p curseForgeProvider) Search(req appstructs.SearchModsRequest) ([]models.ModProject, error) {
	client := global.GetCurseForgeClient()
	if client == nil {
		return nil, nil
	}

	search := client.SearchMod
	searchOptions := []cfMods.SearchModOption{
		search.WithClassID(cfEnum.ClassID(6)),
		search.WithSortOrder(cfEnum.SortOrderDescending),
		search.WithIndex(req.Offset),
		search.WithPageSize(req.Limit),
	}
	if req.Query != "" {
		searchOptions = append(searchOptions, search.WithSearchFilter(req.Query))
	}
	if req.Version != "" {
		searchOptions = append(searchOptions, search.WithGameVersion(cfSchema.GameVersionStr(req.Version)))
	}
	if req.ModLoader != "" {
		modLoader, err := cfEnum.ParseModLoader(req.ModLoader)
		if err == nil {
			searchOptions = append(searchOptions, search.WithModLoaderType(modLoader))
		}
	}

	response, err := search(cfEnum.MinecraftGameID, searchOptions...)
	if err != nil {
		return nil, err
	}

	results := make([]models.ModProject, 0, len(response.Data))
	for _, mod := range response.Data {
		results = append(results, p.modToModProject(mod))
	}

	return results, nil
}

func (p curseForgeProvider) ListVersions(projectIDOrSlug string, filter projectVersionFilter) ([]models.ModVersion, error) {
	client := global.GetCurseForgeClient()
	if client == nil {
		return nil, nil
	}

	modID, err := p.resolveModID(projectIDOrSlug)
	if err != nil {
		return nil, err
	}

	modFiles := client.ModFiles
	options := []cfFiles.ModFilesOption{modFiles.WithPageSize(50)}
	if filter.MinecraftVersion != "" {
		options = append(options, modFiles.WithGameVersion(cfSchema.GameVersionStr(filter.MinecraftVersion)))
	}
	if filter.ModLoader != "" {
		modLoader, err := cfEnum.ParseModLoader(filter.ModLoader)
		if err == nil {
			options = append(options, modFiles.WithModLoader(modLoader))
		}
	}

	response, err := modFiles(modID, options...)
	if err != nil {
		return nil, err
	}

	results := make([]models.ModVersion, 0, len(response.Data))
	for _, file := range response.Data {
		results = append(results, p.fileToModVersion(file))
	}
	return results, nil
}

func (modrinthProvider) Name() string { return "Modrinth" }

func (p modrinthProvider) ExactSearch(req appstructs.SearchModsRequest) ([]models.ModProject, error) {
	result, err := p.searchExactMod(req)
	if err != nil || result.ID == "" {
		return nil, err
	}
	if !projectHasExactMatchingVersion(p, result, req) {
		return nil, nil
	}
	return []models.ModProject{result}, nil
}

func (p modrinthProvider) Search(req appstructs.SearchModsRequest) ([]models.ModProject, error) {
	client := global.GetModrinthClient()
	if client == nil {
		return nil, nil
	}

	facets := [][]string{{modrinth.SearchFacetProjectType + ":mod"}}
	if req.Version != "" {
		facets = append(facets, []string{modrinth.SearchFacetVersions + ":" + req.Version})
	}
	if req.ModLoader != "" {
		facets = append(facets, []string{modrinth.SearchFacetCategories + ":" + req.ModLoader})
	}

	response, err := client.Projects.Search(&modrinth.SearchOptions{
		Query:  req.Query,
		Facets: facets,
		Index:  modrinth.SearchIndexRelevance,
		Offset: req.Offset,
		Limit:  req.Limit,
	})
	if err != nil {
		return nil, err
	}

	results := make([]models.ModProject, 0, len(response.Hits))
	for _, hit := range response.Hits {
		results = append(results, p.searchHitToModProject(hit))
	}

	return results, nil
}

func (p modrinthProvider) ListVersions(projectIDOrSlug string, filter projectVersionFilter) ([]models.ModVersion, error) {
	client := global.GetModrinthClient()
	if client == nil {
		return nil, nil
	}

	options := modrinth.ListVersionsOptions{}
	if filter.MinecraftVersion != "" {
		options.GameVersions = []string{filter.MinecraftVersion}
	}
	if filter.ModLoader != "" {
		options.Loaders = []string{filter.ModLoader}
	}

	versions, err := client.Versions.ListVersions(projectIDOrSlug, options)
	if err != nil {
		return nil, err
	}

	results := make([]models.ModVersion, 0, len(versions))
	for _, version := range versions {
		if len(results) >= 50 {
			break
		}
		results = append(results, p.versionToModVersion(version))
	}
	return results, nil
}

var modProviders = []modProvider{
	curseForgeProvider{},
	modrinthProvider{},
}

// --- CurseForge helpers ---

func (p curseForgeProvider) exactCandidates(query string) ([]cfSchema.Mod, error) {
	type exactResult struct {
		mods []cfSchema.Mod
		err  error
	}

	results := make(chan exactResult, 2)
	started := 0
	if modID, err := strconv.Atoi(strings.TrimSpace(query)); err == nil {
		started++
		go func() {
			mod, ok, err := p.modByID(cfSchema.ModID(modID))
			if err != nil || !ok {
				results <- exactResult{err: err}
				return
			}
			results <- exactResult{mods: []cfSchema.Mod{mod}}
		}()
	}

	started++
	go func() {
		mods, err := p.exactModsBySlug(query)
		results <- exactResult{mods: mods, err: err}
	}()

	candidates := make([]cfSchema.Mod, 0, 2)
	var firstErr error
	for i := 0; i < started; i++ {
		result := <-results
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
		candidates = append(candidates, result.mods...)
	}
	if len(candidates) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return dedupeCurseForgeMods(candidates), nil
}

func (p curseForgeProvider) modByID(modID cfSchema.ModID) (cfSchema.Mod, bool, error) {
	client := global.GetCurseForgeClient()
	if client == nil {
		return cfSchema.Mod{}, false, nil
	}
	response, err := client.Mod(modID)
	if err != nil {
		return cfSchema.Mod{}, false, err
	}
	if response.Data.GameID != cfEnum.MinecraftGameID || response.Data.ClassID != cfEnum.ClassID(6) {
		return cfSchema.Mod{}, false, nil
	}
	return response.Data, true, nil
}

func (p curseForgeProvider) exactModsBySlug(slug string) ([]cfSchema.Mod, error) {
	mods, err := p.searchBySlug(slug)
	if err != nil {
		return nil, err
	}
	results := make([]cfSchema.Mod, 0, len(mods))
	for _, mod := range mods {
		if strings.EqualFold(mod.Slug, slug) {
			results = append(results, mod)
		}
	}
	return results, nil
}

func (p curseForgeProvider) searchBySlug(slug string) ([]cfSchema.Mod, error) {
	client := global.GetCurseForgeClient()
	if client == nil {
		return nil, nil
	}
	search := client.SearchMod
	response, err := search(
		cfEnum.MinecraftGameID,
		search.WithClassID(cfEnum.ClassID(6)),
		search.WithSlug(slug),
		search.WithPageSize(1),
	)
	if err != nil {
		return nil, err
	}
	return response.Data, nil
}

func dedupeCurseForgeMods(mods []cfSchema.Mod) []cfSchema.Mod {
	seen := make(map[cfSchema.ModID]struct{}, len(mods))
	results := make([]cfSchema.Mod, 0, len(mods))
	for _, mod := range mods {
		if _, ok := seen[mod.ID]; ok {
			continue
		}
		seen[mod.ID] = struct{}{}
		results = append(results, mod)
	}
	return results
}

// --- Modrinth helpers ---

func (p modrinthProvider) searchExactMod(req appstructs.SearchModsRequest) (models.ModProject, error) {
	client := global.GetModrinthClient()
	if client == nil {
		return models.ModProject{}, nil
	}

	project, err := client.Projects.Get(req.Query)
	if err != nil {
		return models.ModProject{}, nil
	}
	if project.ProjectType == nil || *project.ProjectType != "mod" {
		return models.ModProject{}, nil
	}

	return p.projectToModProject(project), nil
}

// --- Search result helpers ---

func mergeProviderSearchResults(providerResults map[string][]models.ModProject) []models.ModProject {
	results := make([]models.ModProject, 0)
	for _, providerResult := range providerResults {
		results = append(results, providerResult...)
	}
	return sortSearchResultsByTitle(results)
}

func mergeSearchResultsByTitle(left, right []models.ModProject) []models.ModProject {
	results := make([]models.ModProject, 0, len(left)+len(right))
	results = append(results, left...)
	results = append(results, right...)
	return sortSearchResultsByTitle(results)
}

func sortSearchResultsByTitle(results []models.ModProject) []models.ModProject {
	sort.SliceStable(results, func(i, j int) bool {
		leftTitle := strings.ToLower(results[i].Title)
		rightTitle := strings.ToLower(results[j].Title)
		if leftTitle != rightTitle {
			return leftTitle < rightTitle
		}
		if results[i].Platform != results[j].Platform {
			return results[i].Platform < results[j].Platform
		}
		return results[i].ID < results[j].ID
	})
	return results
}

func projectHasMatchingVersion(provider modProvider, result models.ModProject, req appstructs.SearchModsRequest) bool {
	projectIDOrSlug := result.Slug
	if result.ID != "" {
		_, projectIDOrSlug = splitProjectReference(result.ID)
	}
	if projectIDOrSlug == "" {
		return false
	}

	versions := listProjectVersionsForSearch(provider, projectIDOrSlug, req)
	for _, version := range versions {
		if versionMatchesSearchRequest(version, req) {
			return true
		}
	}
	return false
}

func projectHasExactMatchingVersion(provider modProvider, result models.ModProject, req appstructs.SearchModsRequest) bool {
	if strings.TrimSpace(req.Version) == "" || strings.TrimSpace(req.ModLoader) == "" {
		return false
	}
	projectIDOrSlug := result.Slug
	if result.ID != "" {
		_, projectIDOrSlug = splitProjectReference(result.ID)
	}
	if projectIDOrSlug == "" {
		return false
	}

	versions := listProjectVersionsForSearch(provider, projectIDOrSlug, req)
	for _, version := range versions {
		if versionMatchesSearchRequest(version, req) {
			return true
		}
	}
	return false
}

func versionMatchesSearchRequest(version models.ModVersion, req appstructs.SearchModsRequest) bool {
	if req.Version != "" && !containsFold(version.GameVersions, req.Version) {
		return false
	}
	if req.ModLoader == "" {
		return true
	}
	return containsFold(version.Loaders, req.ModLoader)
}

func containsFold(values []string, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), expected) {
			return true
		}
	}
	return false
}

func filterProjectVersionsForSearch(versions []models.ModVersion, req appstructs.SearchModsRequest) []models.ModVersion {
	results := make([]models.ModVersion, 0, len(versions))
	for _, version := range versions {
		if versionMatchesSearchRequest(version, req) {
			results = append(results, version)
		}
	}
	return results
}

// --- Version listing ---

func (p curseForgeProvider) resolveModID(projectIDOrSlug string) (cfSchema.ModID, error) {
	projectIDOrSlug = strings.TrimSpace(projectIDOrSlug)
	if id, err := strconv.Atoi(projectIDOrSlug); err == nil {
		return cfSchema.ModID(id), nil
	}

	mods, err := p.searchBySlug(projectIDOrSlug)
	if err != nil {
		return 0, err
	}
	for _, mod := range mods {
		if strings.EqualFold(mod.Slug, projectIDOrSlug) {
			return mod.ID, nil
		}
	}
	if len(mods) == 0 {
		return 0, fmt.Errorf("curseforge project not found: %s", projectIDOrSlug)
	}
	return mods[0].ID, nil
}

func curseForgeRelationType(relation cfEnum.FileRelationType) string {
	switch relation {
	case cfEnum.EmbeddedLibrary:
		return "embedded"
	case cfEnum.OptionalDependency:
		return "optional"
	case cfEnum.RequiredDependency:
		return "required"
	case cfEnum.Tool:
		return "tool"
	case cfEnum.Incompatible:
		return "incompatible"
	case cfEnum.Include:
		return "include"
	default:
		return ""
	}
}

func curseForgeFileSHA1(file cfSchema.File) string {
	for _, hash := range file.Hashes {
		if hash.Algo == cfEnum.HashAlgoSHA1 {
			return strings.TrimSpace(hash.Value)
		}
	}
	return ""
}

func (p curseForgeProvider) gameVersionsToStrings(gameVersions []cfSchema.GameVersionStr) []string {
	versions := make([]string, 0, len(gameVersions))
	for _, version := range gameVersions {
		versions = append(versions, string(version))
	}
	return versions
}

func (p curseForgeProvider) loadersFromGameVersions(gameVersions []cfSchema.GameVersionStr) []string {
	loaders := make([]string, 0)
	seen := make(map[string]struct{})
	for _, version := range gameVersions {
		loader, err := cfEnum.ParseModLoader(string(version))
		if err != nil || loader == cfEnum.ModLoaderAny {
			continue
		}
		name := loader.String()
		key := strings.ToLower(name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		loaders = append(loaders, name)
	}
	return loaders
}

// --- Version helpers ---

func normalizeProjectVersionFilter(filter projectVersionFilter) projectVersionFilter {
	return projectVersionFilter{
		MinecraftVersion: strings.TrimSpace(filter.MinecraftVersion),
		ModLoader:        strings.ToLower(strings.TrimSpace(filter.ModLoader)),
	}
}

func projectVersionFilterFromSearchRequest(req appstructs.SearchModsRequest) projectVersionFilter {
	return normalizeProjectVersionFilter(projectVersionFilter{
		MinecraftVersion: req.Version,
		ModLoader:        req.ModLoader,
	})
}

func isFilteredProjectVersionRequest(filter projectVersionFilter) bool {
	filter = normalizeProjectVersionFilter(filter)
	return filter.MinecraftVersion != "" || filter.ModLoader != ""
}

func projectVersionSnapshotScope(filter projectVersionFilter) database.ModPlatformVersionScope {
	filter = normalizeProjectVersionFilter(filter)
	return database.ModPlatformVersionScope{
		MinecraftVersion: filter.MinecraftVersion,
		ModLoader:        filter.ModLoader,
	}
}

func projectVersionSnapshotScopes(filter projectVersionFilter) []database.ModPlatformVersionScope {
	if !isFilteredProjectVersionRequest(filter) {
		return nil
	}
	return []database.ModPlatformVersionScope{projectVersionSnapshotScope(filter)}
}

func filterProjectVersionsForFilter(versions []models.ModVersion, filter projectVersionFilter) []models.ModVersion {
	if !isFilteredProjectVersionRequest(filter) {
		return versions
	}
	return filterProjectVersionsForSearch(versions, appstructs.SearchModsRequest{
		Version:   filter.MinecraftVersion,
		ModLoader: filter.ModLoader,
	})
}

func splitProjectReference(projectIDOrSlug string) (string, string) {
	projectIDOrSlug = strings.TrimSpace(projectIDOrSlug)
	platform, project, ok := strings.Cut(projectIDOrSlug, ":")
	if !ok {
		return "", projectIDOrSlug
	}
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform != "modrinth" && platform != "curseforge" {
		return "", projectIDOrSlug
	}
	return platform, strings.TrimSpace(project)
}

func sortModVersions(results []models.ModVersion) []models.ModVersion {
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].PublishedAt != results[j].PublishedAt {
			return results[i].PublishedAt > results[j].PublishedAt
		}
		leftName := strings.ToLower(results[i].Name)
		rightName := strings.ToLower(results[j].Name)
		if leftName != rightName {
			return leftName < rightName
		}
		if results[i].Platform != results[j].Platform {
			return results[i].Platform < results[j].Platform
		}
		return results[i].ID < results[j].ID
	})
	return results
}

func refreshProjectMetadataIfStale(provider modProvider, platform, projectIDOrSlug string) {
	record, ok := getProjectSnapshotPlatform(platform, projectIDOrSlug)
	if ok && record.CachedAt > 0 && time.Since(time.Unix(record.CachedAt, 0)) <= projectMetadataTTL {
		return
	}
	results, err := provider.ExactSearch(appstructs.SearchModsRequest{Query: projectIDOrSlug})
	if err != nil || len(results) == 0 {
		return
	}
	for _, r := range results {
		if err := database.UpsertModPlatform(r); err != nil {
			logging.Error("refresh project metadata failed", "platform", platform, "projectID", projectIDOrSlug, "error", err)
		}
	}
}

func listProjectVersions(provider modProvider, projectIDOrSlug string) []models.ModVersion {
	return listProjectVersionsWithFilter(provider, projectIDOrSlug, projectVersionFilter{})
}

func listProjectVersionsForSearch(provider modProvider, projectIDOrSlug string, req appstructs.SearchModsRequest) []models.ModVersion {
	return listProjectVersionsWithFilter(provider, projectIDOrSlug, projectVersionFilterFromSearchRequest(req))
}

func listProjectVersionsWithFilter(provider modProvider, projectIDOrSlug string, filter projectVersionFilter) []models.ModVersion {
	projectIDOrSlug = strings.TrimSpace(projectIDOrSlug)
	if projectIDOrSlug == "" {
		return nil
	}

	_, projectIDOrSlug = splitProjectReference(projectIDOrSlug)
	filter = normalizeProjectVersionFilter(filter)
	platform := provider.Name()
	if versions, ok := getFreshProjectVersionsSnapshot(platform, projectIDOrSlug, filter); ok {
		versions = filterProjectVersionsForFilter(versions, filter)
		versions = sortModVersions(versions)
		return versions
	}

	versions, err := provider.ListVersions(projectIDOrSlug, filter)
	if err != nil || len(versions) == 0 {
		logging.Error("list project versions failed", "provider", provider.Name(), "projectIDOrSlug", projectIDOrSlug, "minecraftVersion", filter.MinecraftVersion, "modLoader", filter.ModLoader, "error", err)
		if versions, ok := getProjectVersionsSnapshot(platform, projectIDOrSlug, filter); ok {
			versions = filterProjectVersionsForFilter(versions, filter)
			versions = sortModVersions(versions)
			return versions
		}
		return nil
	}

	versions = sortModVersions(versions)
	projectID := projectIDFromVersions(versions, projectIDOrSlug)
	if err := saveProjectVersionsSnapshot(platform, projectIDOrSlug, projectID, versions, filter); err != nil {
		logging.Error("save project versions snapshot failed", "platform", platform, "requestedProject", projectIDOrSlug, "projectID", projectID, "versionCount", len(versions), "error", err)
	} else {
		associateEquivalentPlatformProject(platform, projectID, versions)
	}
	return versions
}

func associateEquivalentPlatformProject(platform, projectID string, versions []models.ModVersion) {
	projectID = strings.TrimSpace(projectID)
	latestSHA1 := latestVersionSHA1(versions)
	if projectID == "" || latestSHA1 == "" {
		return
	}

	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "curseforge":
		if match, ok := database.GetLatestProjectBySHA1("Modrinth", latestSHA1); ok && match.ProjectID != "" {
			if err := database.UpsertPlatformAssociationByProjects(projectID, match.ProjectID); err != nil {
				logging.Error("associate equivalent platform projects failed", "curseforgeProjectID", projectID, "modrinthProjectID", match.ProjectID, "sha1", latestSHA1, "error", err)
			}
		}
	case "modrinth":
		if match, ok := database.GetLatestProjectBySHA1("CurseForge", latestSHA1); ok && match.ProjectID != "" {
			if err := database.UpsertPlatformAssociationByProjects(match.ProjectID, projectID); err != nil {
				logging.Error("associate equivalent platform projects failed", "curseforgeProjectID", match.ProjectID, "modrinthProjectID", projectID, "sha1", latestSHA1, "error", err)
			}
		}
	}
}

func latestVersionSHA1(versions []models.ModVersion) string {
	for _, version := range versions {
		if sha1 := strings.ToLower(strings.TrimSpace(version.SHA1)); sha1 != "" {
			return sha1
		}
	}
	return ""
}

func getFreshProjectVersionsSnapshot(platform, projectID string, filter projectVersionFilter) ([]models.ModVersion, bool) {
	record, ok := getProjectSnapshotPlatform(platform, projectID)
	if !ok {
		return nil, false
	}
	updatedAt := record.UpdatedAt
	if isFilteredProjectVersionRequest(filter) {
		if updatedAt > 0 && time.Since(time.Unix(updatedAt, 0)) <= projectVersionsSnapshotTTL {
			updatedAt = record.UpdatedAt
		} else if ts, ok := database.GetPlatformVersionScopeUpdatedAt(platform, record.ProjectID, projectVersionSnapshotScope(filter)); ok {
			updatedAt = ts
		} else {
			return nil, false
		}
	}
	if updatedAt <= 0 || time.Since(time.Unix(updatedAt, 0)) > projectVersionsSnapshotTTL {
		return nil, false
	}

	versions, err := database.GetPlatformVersions(platform, record.ProjectID)
	if err != nil || len(versions) == 0 {
		if err != nil {
			logging.Error("read fresh project versions snapshot failed", "platform", platform, "projectID", record.ProjectID, "error", err)
		}
		return nil, false
	}
	return versions, true
}

func getProjectVersionsSnapshot(platform, projectID string, filter projectVersionFilter) ([]models.ModVersion, bool) {
	record, ok := getProjectSnapshotPlatform(platform, projectID)
	if !ok {
		return nil, false
	}
	versions, err := database.GetPlatformVersions(platform, record.ProjectID)
	if err != nil || len(versions) == 0 {
		if err != nil {
			logging.Error("read stale project versions snapshot failed", "platform", platform, "projectID", record.ProjectID, "error", err)
		}
		return nil, false
	}
	return versions, true
}

func getProjectSnapshotPlatform(platform, projectIDOrSlug string) (models.ModProject, bool) {
	projectIDOrSlug = strings.TrimSpace(projectIDOrSlug)
	if projectIDOrSlug == "" {
		return models.ModProject{}, false
	}
	if record, ok := database.GetModPlatform(platform, projectIDOrSlug); ok {
		return record, true
	}
	return database.GetModPlatformBySlug(platform, projectIDOrSlug)
}

func saveProjectVersionsSnapshot(platform, requestedProject string, projectID string, versions []models.ModVersion, filter projectVersionFilter) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil
	}
	requestedProject = strings.TrimSpace(requestedProject)
	meta := models.ModProject{
		Platform:  platform,
		ProjectID: projectID,
	}
	if requestedProject != "" && !strings.EqualFold(requestedProject, projectID) {
		meta.Slug = requestedProject
	}
	if err := database.UpsertModPlatform(meta); err != nil {
		return err
	}
	return database.SetPlatformVersionSnapshot(platform, projectID, versions, time.Now().Unix(), projectVersionSnapshotScopes(filter))
}

func projectIDFromVersions(versions []models.ModVersion, fallback string) string {
	for _, version := range versions {
		if strings.TrimSpace(version.ProjectID) != "" {
			return strings.TrimSpace(version.ProjectID)
		}
	}
	return strings.TrimSpace(fallback)
}

func curseForgeLogoURL(logo cfSchema.ModAsset) string {
	if url := strings.TrimSpace(logo.ThumbnailUrl); url != "" {
		return url
	}
	return strings.TrimSpace(logo.URL)
}

// --- Conversion methods (SDK → Unified Model) ---

// CurseForge SDK → ModProject
func (p curseForgeProvider) modToModProject(mod cfSchema.Mod) models.ModProject {
	id := strconv.Itoa(int(mod.ID))
	return models.ModProject{
		ID:          models.ProjectKey("curseforge", id),
		Platform:    "CurseForge",
		ProjectID:   id,
		Slug:        mod.Slug,
		Title:       mod.Name,
		Icon:        "mdi-package-variant",
		IconURL:     curseForgeLogoURL(mod.Logo),
		Description: mod.Summary,
		Downloads:   mod.DownloadCount,
		UpdatedAt:   0, // Set by caller if needed
	}
}

// CurseForge SDK → ModVersion
func (p curseForgeProvider) fileToModVersion(file cfSchema.File) models.ModVersion {
	return models.ModVersion{
		ID:           strconv.Itoa(int(file.ID)),
		Platform:     "CurseForge",
		ProjectID:    strconv.Itoa(int(file.ModID)),
		VersionID:    strconv.Itoa(int(file.ID)),
		Name:         file.DisplayName,
		Version:      file.DisplayName,
		FileName:     file.FileName,
		DownloadURL:  file.DownloadURL,
		SHA1:         curseForgeFileSHA1(file),
		PublishedAt:  file.FileDate.Unix(),
		Downloads:    file.DownloadCount,
		GameVersions: p.gameVersionsToStrings(file.GameVersions),
		Loaders:      p.loadersFromGameVersions(file.GameVersions),
		Dependencies: p.dependenciesFromFileToModDeps(file.Dependencies),
	}
}

func (p curseForgeProvider) dependenciesFromFileToModDeps(deps []cfSchema.FileDependency) []models.ModDependency {
	results := make([]models.ModDependency, 0, len(deps))
	for _, dep := range deps {
		if dep.ModID == 0 {
			continue
		}
		results = append(results, models.ModDependency{
			DependencyProjectID: strconv.Itoa(int(dep.ModID)),
			DependencyVersionID: "",
			DependencyType:      curseForgeRelationType(dep.RelationType),
		})
	}
	return results
}

// Modrinth SDK → ModProject (from SearchResult)
func (p modrinthProvider) searchHitToModProject(hit *modrinth.SearchResult) models.ModProject {
	project := models.ModProject{
		Platform: "Modrinth",
		Icon:     "mdi-leaf",
	}
	if hit.ProjectID != nil {
		project.ID = models.ProjectKey("modrinth", *hit.ProjectID)
		project.ProjectID = *hit.ProjectID
	}
	if hit.Title != nil {
		project.Title = *hit.Title
	}
	if hit.Description != nil {
		project.Description = *hit.Description
	}
	if hit.IconURL != nil {
		project.IconURL = *hit.IconURL
	}
	if hit.Downloads != nil {
		project.Downloads = int64(*hit.Downloads)
	}
	if hit.Slug != nil {
		project.Slug = *hit.Slug
	}
	return project
}

// Modrinth SDK → ModProject (from Project)
func (p modrinthProvider) projectToModProject(project *modrinth.Project) models.ModProject {
	result := models.ModProject{
		Platform: "Modrinth",
		Icon:     "mdi-leaf",
	}
	if project.ID != nil {
		result.ID = models.ProjectKey("modrinth", *project.ID)
		result.ProjectID = *project.ID
	}
	if project.Title != nil {
		result.Title = *project.Title
	}
	if project.Description != nil {
		result.Description = *project.Description
	}
	if project.IconURL != nil {
		result.IconURL = *project.IconURL
	}
	if project.Downloads != nil {
		result.Downloads = int64(*project.Downloads)
	}
	if project.Slug != nil {
		result.Slug = *project.Slug
	}
	return result
}

// Modrinth SDK → ModVersion
func (p modrinthProvider) versionToModVersion(version *modrinth.Version) models.ModVersion {
	result := models.ModVersion{Platform: "Modrinth"}
	if version == nil {
		return result
	}
	if version.ID != nil {
		result.ID = *version.ID
		result.VersionID = *version.ID
	}
	if version.ProjectID != nil {
		result.ProjectID = *version.ProjectID
	}
	if version.Name != nil {
		result.Name = *version.Name
	}
	if version.VersionNumber != nil {
		result.Version = *version.VersionNumber
	}
	if version.Downloads != nil {
		result.Downloads = int64(*version.Downloads)
	}
	if version.DatePublished != nil {
		result.PublishedAt = version.DatePublished.Unix()
	}
	result.GameVersions = version.GameVersions
	result.Loaders = version.Loaders
	result.Dependencies = p.dependenciesFromVersionToModDeps(version.Dependencies)

	for _, file := range version.Files {
		if file == nil {
			continue
		}
		if file.Primary != nil && *file.Primary {
			p.setModVersionFileFields(&result, file)
			return result
		}
	}
	if len(version.Files) > 0 {
		p.setModVersionFileFields(&result, version.Files[0])
	}
	return result
}

func (p modrinthProvider) setModVersionFileFields(result *models.ModVersion, file *modrinth.File) {
	if file == nil {
		return
	}
	if file.Filename != nil {
		result.FileName = *file.Filename
	}
	if file.URL != nil {
		result.DownloadURL = *file.URL
	}
	if file.Hashes != nil {
		result.SHA1 = strings.TrimSpace(file.Hashes["sha1"])
	}
}

func (p modrinthProvider) dependenciesFromVersionToModDeps(deps []*modrinth.Dependency) []models.ModDependency {
	results := make([]models.ModDependency, 0, len(deps))
	for _, dep := range deps {
		if dep == nil {
			continue
		}
		projectID := ""
		if dep.ProjectID != nil {
			projectID = strings.TrimSpace(*dep.ProjectID)
		}
		versionID := ""
		if dep.VersionID != nil {
			versionID = strings.TrimSpace(*dep.VersionID)
		}
		if projectID == "" && versionID == "" {
			continue
		}
		depType := ""
		if dep.DependencyType != nil {
			depType = strings.ToLower(strings.TrimSpace(*dep.DependencyType))
		}
		results = append(results, models.ModDependency{
			DependencyProjectID: projectID,
			DependencyVersionID: versionID,
			DependencyType:      depType,
		})
	}
	return results
}
