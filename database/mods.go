package database

import (
	"sort"
	"strings"
	"time"

	"mod-downloader/logging"
	"mod-downloader/models"
)

// --- record types ---

type PlatformAssociation struct {
	ID                  string `json:"id"`
	CurseForgeProjectID string `json:"curseforgeProjectId,omitempty"`
	ModrinthProjectID   string `json:"modrinthProjectId,omitempty"`
}

type PinnedMod struct {
	ID               string `json:"id"`
	Platform         string `json:"platform"`
	ModID            string `json:"modId"`
	VersionID        string `json:"versionId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
}

type ModPlatformVersionScope struct {
	MinecraftVersion string
	ModLoader        string
}

type storedVersionScope struct {
	Platform         string `json:"platform"`
	ProjectID        string `json:"projectId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
	UpdatedAt        int64  `json:"updatedAt"`
}

func normalizePinnedModKey(platform, modID, mcVersion, modLoader string) (string, string, string, string) {
	return strings.ToLower(strings.TrimSpace(platform)),
		strings.ToLower(strings.TrimSpace(modID)),
		strings.TrimSpace(mcVersion),
		strings.ToLower(strings.TrimSpace(modLoader))
}

func normalizePinnedMod(p PinnedMod) PinnedMod {
	p.Platform, p.ModID, p.MinecraftVersion, p.ModLoader = normalizePinnedModKey(p.Platform, p.ModID, p.MinecraftVersion, p.ModLoader)
	p.VersionID = strings.TrimSpace(p.VersionID)
	return p
}

func makePlatformKey(platform, projectID string) platformKey {
	return platformKey{Platform: strings.TrimSpace(platform), ProjectID: strings.TrimSpace(projectID)}
}

func makeVersionKey(platform, projectID, versionID string) versionKey {
	return versionKey{Platform: strings.TrimSpace(platform), ProjectID: strings.TrimSpace(projectID), VersionID: strings.TrimSpace(versionID)}
}

func makePinnedModKey(platform, modID, mcVersion, modLoader string) pinnedModKey {
	platform, modID, mcVersion, modLoader = normalizePinnedModKey(platform, modID, mcVersion, modLoader)
	return pinnedModKey{Platform: platform, ModID: modID, MinecraftVersion: mcVersion, ModLoader: modLoader}
}

func makeVersionScopeKey(platform, projectID string, scope ModPlatformVersionScope) versionScopeKey {
	return versionScopeKey{
		Platform:         strings.TrimSpace(platform),
		ProjectID:        strings.TrimSpace(projectID),
		MinecraftVersion: scope.MinecraftVersion,
		ModLoader:        scope.ModLoader,
	}
}

// --- mod platforms ---

func UpsertModPlatform(p models.ModProject) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	p.Platform = strings.TrimSpace(p.Platform)
	p.ProjectID = strings.TrimSpace(p.ProjectID)
	if p.Platform == "" || p.ProjectID == "" {
		return nil
	}

	isFullMetadata := strings.TrimSpace(p.Title) != ""

	err = d.update(func(state *cacheState, pool *stringPool) error {
		key := internPlatformKey(pool, makePlatformKey(p.Platform, p.ProjectID))
		if existing, ok := state.ModPlatforms[key]; ok {
			if strings.TrimSpace(p.Slug) == "" {
				p.Slug = existing.Slug
			}
			p.UpdatedAt = existing.UpdatedAt
			if isFullMetadata {
				p.CachedAt = time.Now().Unix()
			} else {
				p.Title = existing.Title
				p.Icon = existing.Icon
				p.IconURL = existing.IconURL
				p.Description = existing.Description
				p.Downloads = existing.Downloads
				p.CachedAt = existing.CachedAt
			}
		} else if isFullMetadata {
			p.CachedAt = time.Now().Unix()
		}
		state.ModPlatforms[key] = internModProject(pool, p)
		return nil
	})
	if err != nil {
		logging.Error("upsert mod platform failed", "platform", p.Platform, "projectID", p.ProjectID, "slug", p.Slug, "error", err)
		return err
	}
	logging.Info("mod platform upserted", "platform", p.Platform, "projectID", p.ProjectID, "slug", p.Slug)
	return nil
}

func GetModPlatform(platform, projectID string) (models.ModProject, bool) {
	d, err := readyDB()
	if err != nil {
		return models.ModProject{}, false
	}
	key := makePlatformKey(platform, projectID)
	if key.Platform == "" || key.ProjectID == "" {
		return models.ModProject{}, false
	}

	var p models.ModProject
	found := false
	err = d.view(func(state *cacheState) error {
		p, found = state.ModPlatforms[key]
		return nil
	})
	if err != nil || !found {
		logging.Debug("mod platform cache miss", "platform", key.Platform, "projectID", key.ProjectID)
		return models.ModProject{}, false
	}
	logging.Debug("mod platform cache hit", "platform", key.Platform, "projectID", key.ProjectID, "updatedAt", p.UpdatedAt)
	return p, true
}

func GetModPlatformBySlug(platform, slug string) (models.ModProject, bool) {
	d, err := readyDB()
	if err != nil {
		return models.ModProject{}, false
	}
	platform = strings.TrimSpace(platform)
	slug = strings.TrimSpace(slug)
	if platform == "" || slug == "" {
		return models.ModProject{}, false
	}

	var p models.ModProject
	found := false
	err = d.view(func(state *cacheState) error {
		for key, candidate := range state.ModPlatforms {
			if key.Platform == platform && candidate.Slug == slug {
				p = candidate
				found = true
				break
			}
		}
		return nil
	})
	if err != nil || !found {
		logging.Debug("mod platform slug cache miss", "platform", platform, "slug", slug)
		return models.ModProject{}, false
	}
	logging.Debug("mod platform slug cache hit", "platform", platform, "slug", slug, "projectID", p.ProjectID, "updatedAt", p.UpdatedAt)
	return p, true
}

func TouchModPlatform(platform, projectID string, updatedAt int64) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	key := makePlatformKey(platform, projectID)
	if key.Platform == "" || key.ProjectID == "" {
		return nil
	}

	err = d.update(func(state *cacheState, pool *stringPool) error {
		key := internPlatformKey(pool, key)
		p := state.ModPlatforms[key]
		p.Platform = key.Platform
		p.ProjectID = key.ProjectID
		p.UpdatedAt = updatedAt
		state.ModPlatforms[key] = internModProject(pool, p)
		return nil
	})
	if err != nil {
		logging.Error("touch mod platform failed", "platform", key.Platform, "projectID", key.ProjectID, "updatedAt", updatedAt, "error", err)
		return err
	}
	logging.Debug("mod platform touched", "platform", key.Platform, "projectID", key.ProjectID, "updatedAt", updatedAt)
	return nil
}

// --- platform associations ---

func UpsertPlatformAssociation(a PlatformAssociation) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	a.CurseForgeProjectID = strings.TrimSpace(a.CurseForgeProjectID)
	a.ModrinthProjectID = strings.TrimSpace(a.ModrinthProjectID)
	if a.ID == "" {
		a.ID = NewID()
	}

	err = d.update(func(state *cacheState, pool *stringPool) error {
		a = internPlatformAssociation(pool, a)
		state.PlatformAssociations[a.ID] = a
		return nil
	})
	if err != nil {
		logging.Error("upsert platform association failed", "id", a.ID, "curseforgeProjectID", a.CurseForgeProjectID, "modrinthProjectID", a.ModrinthProjectID, "error", err)
		return err
	}
	logging.Info("platform association upserted", "id", a.ID, "curseforgeProjectID", a.CurseForgeProjectID, "modrinthProjectID", a.ModrinthProjectID)
	return nil
}

func UpsertPlatformAssociationByProjects(curseForgeProjectID, modrinthProjectID string) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	curseForgeProjectID = strings.TrimSpace(curseForgeProjectID)
	modrinthProjectID = strings.TrimSpace(modrinthProjectID)
	if curseForgeProjectID == "" || modrinthProjectID == "" {
		return nil
	}

	err = d.update(func(state *cacheState, pool *stringPool) error {
		ids := make([]string, 0, 2)
		for _, a := range state.PlatformAssociations {
			if a.CurseForgeProjectID == curseForgeProjectID || a.ModrinthProjectID == modrinthProjectID {
				ids = append(ids, a.ID)
			}
		}
		id := NewID()
		if len(ids) > 0 {
			id = ids[0]
			for _, duplicateID := range ids[1:] {
				delete(state.PlatformAssociations, duplicateID)
			}
		}
		state.PlatformAssociations[id] = internPlatformAssociation(pool, PlatformAssociation{
			ID:                  id,
			CurseForgeProjectID: curseForgeProjectID,
			ModrinthProjectID:   modrinthProjectID,
		})
		return nil
	})
	if err != nil {
		logging.Error("upsert platform association by projects failed", "curseforgeProjectID", curseForgeProjectID, "modrinthProjectID", modrinthProjectID, "error", err)
		return err
	}
	logging.Info("platform association upserted by projects", "curseforgeProjectID", curseForgeProjectID, "modrinthProjectID", modrinthProjectID)
	return nil
}

func GetAssociationByCurseForge(cfProjectID string) (PlatformAssociation, bool) {
	cfProjectID = strings.TrimSpace(cfProjectID)
	return getAssociationBy(func(a PlatformAssociation) bool {
		return a.CurseForgeProjectID == cfProjectID
	}, "curseforgeProjectID", cfProjectID)
}

func GetAssociationByModrinth(mrProjectID string) (PlatformAssociation, bool) {
	mrProjectID = strings.TrimSpace(mrProjectID)
	return getAssociationBy(func(a PlatformAssociation) bool {
		return a.ModrinthProjectID == mrProjectID
	}, "modrinthProjectID", mrProjectID)
}

func getAssociationBy(match func(PlatformAssociation) bool, logKey string, logValue string) (PlatformAssociation, bool) {
	d, err := readyDB()
	if err != nil || logValue == "" {
		return PlatformAssociation{}, false
	}
	var out PlatformAssociation
	found := false
	err = d.view(func(state *cacheState) error {
		for _, a := range state.PlatformAssociations {
			if match(a) {
				out = a
				found = true
				break
			}
		}
		return nil
	})
	if err != nil || !found {
		logging.Debug("platform association miss", logKey, logValue)
		return PlatformAssociation{}, false
	}
	logging.Debug("platform association hit", logKey, logValue, "id", out.ID)
	return out, true
}

func GetLatestProjectBySHA1(platform, sha1 string) (models.ModVersion, bool) {
	d, err := readyDB()
	if err != nil {
		return models.ModVersion{}, false
	}
	platform = strings.TrimSpace(platform)
	sha1 = strings.ToLower(strings.TrimSpace(sha1))
	if platform == "" || sha1 == "" {
		return models.ModVersion{}, false
	}

	var best models.ModVersion
	found := false
	err = d.view(func(state *cacheState) error {
		latestByProject := make(map[string]models.ModVersion)
		for key, v := range state.PlatformVersions {
			if key.Platform != platform {
				continue
			}
			if current, ok := latestByProject[v.ProjectID]; !ok || newerProjectVersion(v, current) {
				latestByProject[v.ProjectID] = v
			}
		}
		for _, v := range latestByProject {
			if strings.ToLower(strings.TrimSpace(v.SHA1)) != sha1 {
				continue
			}
			if !found || v.PublishedAt > best.PublishedAt || (v.PublishedAt == best.PublishedAt && v.ProjectID < best.ProjectID) {
				best = copyVersion(v)
				found = true
			}
		}
		return nil
	})
	if err != nil || !found {
		return models.ModVersion{}, false
	}
	return best, true
}

func newerProjectVersion(left, right models.ModVersion) bool {
	if left.PublishedAt != right.PublishedAt {
		return left.PublishedAt > right.PublishedAt
	}
	return left.VersionID > right.VersionID
}

// --- mod platform versions ---

func SetPlatformVersions(platform, projectID string, versions []models.ModVersion) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	if platform == "" || projectID == "" {
		return nil
	}

	logging.Debug("set platform versions started", "platform", platform, "projectID", projectID, "versionCount", len(versions))
	err = d.update(func(state *cacheState, pool *stringPool) error {
		for _, v := range versions {
			savePlatformVersion(state, pool, platform, projectID, v)
		}
		return nil
	})
	if err != nil {
		logging.Error("set platform versions failed", "platform", platform, "projectID", projectID, "versionCount", len(versions), "error", err)
		return err
	}
	logging.Info("platform versions set", "platform", platform, "projectID", projectID, "versionCount", len(versions))
	return nil
}

func SetPlatformVersionSnapshot(platform, projectID string, versions []models.ModVersion, updatedAt int64, scopes []ModPlatformVersionScope) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	if platform == "" || projectID == "" {
		return nil
	}
	scopes = normalizePlatformVersionScopes(scopes)

	logging.Debug("set platform version snapshot started", "platform", platform, "projectID", projectID, "versionCount", len(versions), "updatedAt", updatedAt, "scopeCount", len(scopes))
	err = d.update(func(state *cacheState, pool *stringPool) error {
		touchSnapshotPlatform(state, pool, platform, projectID, updatedAt, len(scopes) == 0)
		for _, v := range versions {
			savePlatformVersion(state, pool, platform, projectID, v)
		}
		for _, scope := range scopes {
			key := internVersionScopeKey(pool, makeVersionScopeKey(platform, projectID, scope))
			state.PlatformVersionScopes[key] = internStoredVersionScope(pool, storedVersionScope{
				Platform:         platform,
				ProjectID:        projectID,
				MinecraftVersion: scope.MinecraftVersion,
				ModLoader:        scope.ModLoader,
				UpdatedAt:        updatedAt,
			})
		}
		return nil
	})
	if err != nil {
		logging.Error("set platform version snapshot failed", "platform", platform, "projectID", projectID, "versionCount", len(versions), "error", err)
		return err
	}
	logging.Info("platform version snapshot set", "platform", platform, "projectID", projectID, "versionCount", len(versions), "updatedAt", updatedAt)
	return nil
}

func touchSnapshotPlatform(state *cacheState, pool *stringPool, platform, projectID string, updatedAt int64, updateProjectTimestamp bool) {
	key := internPlatformKey(pool, makePlatformKey(platform, projectID))
	p := state.ModPlatforms[key]
	p.Platform = key.Platform
	p.ProjectID = key.ProjectID
	if updateProjectTimestamp {
		p.UpdatedAt = updatedAt
	}
	state.ModPlatforms[key] = internModProject(pool, p)
}

func normalizePlatformVersionScopes(scopes []ModPlatformVersionScope) []ModPlatformVersionScope {
	seen := make(map[string]struct{}, len(scopes))
	out := make([]ModPlatformVersionScope, 0, len(scopes))
	for _, scope := range scopes {
		scope.MinecraftVersion = strings.TrimSpace(scope.MinecraftVersion)
		scope.ModLoader = strings.ToLower(strings.TrimSpace(scope.ModLoader))
		if scope.MinecraftVersion == "" && scope.ModLoader == "" {
			continue
		}
		key := scope.MinecraftVersion + "|" + scope.ModLoader
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func savePlatformVersion(state *cacheState, pool *stringPool, platform, projectID string, v models.ModVersion) {
	v = copyVersion(v)
	v.Platform = platform
	v.ProjectID = projectID
	v.VersionID = strings.TrimSpace(v.VersionID)
	key := internVersionKey(pool, makeVersionKey(platform, projectID, v.VersionID))

	if existing, ok := state.PlatformVersions[key]; ok && existing.ID != "" {
		v.ID = existing.ID
	}
	if v.ID == "" {
		v.ID = NewID()
	}
	v.Dependencies = normalizeDependencies(v.ID, v.Dependencies)
	v = internModVersion(pool, v)
	state.PlatformVersions[key] = v
	state.PlatformVersionKeyByID[v.ID] = key
}

func normalizeDependencies(platformVersionID string, deps []models.ModDependency) []models.ModDependency {
	out := make([]models.ModDependency, 0, len(deps))
	for _, dep := range deps {
		projectID := strings.TrimSpace(dep.DependencyProjectID)
		versionID := strings.TrimSpace(dep.DependencyVersionID)
		if projectID == "" && versionID == "" {
			continue
		}
		if dep.ID == "" {
			dep.ID = NewID()
		}
		dep.PlatformVersionID = platformVersionID
		dep.DependencyProjectID = projectID
		dep.DependencyVersionID = versionID
		dep.DependencyType = strings.TrimSpace(dep.DependencyType)
		out = append(out, dep)
	}
	return out
}

func GetPlatformVersionScopeUpdatedAt(platform, projectID string, scope ModPlatformVersionScope) (int64, bool) {
	d, err := readyDB()
	if err != nil {
		return 0, false
	}
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	scopes := normalizePlatformVersionScopes([]ModPlatformVersionScope{scope})
	if platform == "" || projectID == "" || len(scopes) == 0 {
		return 0, false
	}
	key := makeVersionScopeKey(platform, projectID, scopes[0])

	var rec storedVersionScope
	found := false
	err = d.view(func(state *cacheState) error {
		rec, found = state.PlatformVersionScopes[key]
		return nil
	})
	if err != nil || !found {
		return 0, false
	}
	return rec.UpdatedAt, true
}

func GetPlatformVersions(platform, projectID string) ([]models.ModVersion, error) {
	d, err := readyDB()
	if err != nil {
		return nil, err
	}
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	if platform == "" || projectID == "" {
		return nil, nil
	}

	versions := make([]models.ModVersion, 0)
	err = d.view(func(state *cacheState) error {
		for key, v := range state.PlatformVersions {
			if key.Platform == platform && key.ProjectID == projectID {
				versions = append(versions, copyVersion(v))
			}
		}
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].VersionID < versions[j].VersionID
		})
		return nil
	})
	if err != nil {
		logging.Error("get platform versions failed", "platform", platform, "projectID", projectID, "error", err)
		return nil, err
	}
	logging.Debug("platform versions loaded", "platform", platform, "projectID", projectID, "versionCount", len(versions))
	return versions, nil
}

// --- mod dependencies ---

func SetVersionDependencies(platformVersionID string, deps []models.ModDependency) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" {
		return nil
	}

	logging.Debug("set version dependencies started", "platformVersionID", platformVersionID, "dependencyCount", len(deps))
	err = d.update(func(state *cacheState, pool *stringPool) error {
		key, ok := state.PlatformVersionKeyByID[platformVersionID]
		if !ok {
			return nil
		}
		v, ok := state.PlatformVersions[key]
		if !ok {
			return nil
		}
		v.Dependencies = internDependencies(pool, normalizeDependencies(platformVersionID, deps))
		state.PlatformVersions[key] = v
		return nil
	})
	if err != nil {
		logging.Error("set version dependencies failed", "platformVersionID", platformVersionID, "dependencyCount", len(deps), "error", err)
		return err
	}
	logging.Info("version dependencies set", "platformVersionID", platformVersionID, "dependencyCount", len(deps))
	return nil
}

func GetVersionDependencies(platformVersionID string) ([]models.ModDependency, error) {
	d, err := readyDB()
	if err != nil {
		return nil, err
	}
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" {
		return nil, nil
	}

	var deps []models.ModDependency
	err = d.view(func(state *cacheState) error {
		key, ok := state.PlatformVersionKeyByID[platformVersionID]
		if !ok {
			return nil
		}
		if v, ok := state.PlatformVersions[key]; ok {
			deps = copyDependencies(v.Dependencies)
		}
		return nil
	})
	if err != nil {
		logging.Error("get version dependencies failed", "platformVersionID", platformVersionID, "error", err)
		return nil, err
	}
	logging.Debug("version dependencies loaded", "platformVersionID", platformVersionID, "dependencyCount", len(deps))
	return deps, nil
}

// --- pinned mods ---

func UpsertPinnedMod(p PinnedMod) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	p = normalizePinnedMod(p)
	if p.Platform == "" || p.ModID == "" || p.MinecraftVersion == "" || p.ModLoader == "" {
		return nil
	}

	err = d.update(func(state *cacheState, pool *stringPool) error {
		key := internPinnedModKey(pool, makePinnedModKey(p.Platform, p.ModID, p.MinecraftVersion, p.ModLoader))
		if existing, ok := state.PinnedMods[key]; ok && existing.ID != "" {
			p.ID = existing.ID
		}
		if p.ID == "" {
			p.ID = NewID()
		}
		state.PinnedMods[key] = internPinnedMod(pool, p)
		return nil
	})
	if err != nil {
		logging.Error("upsert pinned mod failed", "platform", p.Platform, "modID", p.ModID, "versionID", p.VersionID, "minecraftVersion", p.MinecraftVersion, "modLoader", p.ModLoader, "error", err)
		return err
	}
	logging.Info("pinned mod upserted", "platform", p.Platform, "modID", p.ModID, "versionID", p.VersionID, "minecraftVersion", p.MinecraftVersion, "modLoader", p.ModLoader)
	return nil
}

func GetPinnedMod(platform, modID, mcVersion, modLoader string) (PinnedMod, bool) {
	d, err := readyDB()
	if err != nil {
		return PinnedMod{}, false
	}
	key := makePinnedModKey(platform, modID, mcVersion, modLoader)
	if key.Platform == "" || key.ModID == "" || key.MinecraftVersion == "" || key.ModLoader == "" {
		return PinnedMod{}, false
	}

	var p PinnedMod
	found := false
	err = d.view(func(state *cacheState) error {
		p, found = state.PinnedMods[key]
		return nil
	})
	if err != nil || !found {
		logging.Debug("pinned mod miss", "platform", key.Platform, "modID", key.ModID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader)
		return PinnedMod{}, false
	}
	logging.Debug("pinned mod hit", "platform", key.Platform, "modID", key.ModID, "versionID", p.VersionID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader)
	return p, true
}

func DeletePinnedMod(platform, modID, mcVersion, modLoader string) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	key := makePinnedModKey(platform, modID, mcVersion, modLoader)
	if key.Platform == "" || key.ModID == "" || key.MinecraftVersion == "" || key.ModLoader == "" {
		return nil
	}

	err = d.update(func(state *cacheState, pool *stringPool) error {
		delete(state.PinnedMods, key)
		return nil
	})
	if err != nil {
		logging.Error("delete pinned mod failed", "platform", key.Platform, "modID", key.ModID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader, "error", err)
		return err
	}
	logging.Info("pinned mod deleted", "platform", key.Platform, "modID", key.ModID, "minecraftVersion", key.MinecraftVersion, "modLoader", key.ModLoader)
	return nil
}

// --- version mod IDs ---

func SetVersionModIDs(platformVersionID string, modIDs []string) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" {
		return nil
	}

	logging.Debug("set version mod IDs started", "platformVersionID", platformVersionID, "modIDCount", len(modIDs))
	err = d.update(func(state *cacheState, pool *stringPool) error {
		key, ok := state.PlatformVersionKeyByID[platformVersionID]
		if !ok {
			return nil
		}
		v, ok := state.PlatformVersions[key]
		if !ok {
			return nil
		}
		seen := make(map[string]struct{}, len(modIDs))
		out := make([]string, 0, len(modIDs))
		for _, id := range modIDs {
			id = strings.ToLower(strings.TrimSpace(id))
			if id == "" {
				continue
			}
			if _, dup := seen[id]; dup {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, pool.Intern(id))
		}
		v.ModIDs = out
		state.PlatformVersions[key] = v
		return nil
	})
	if err != nil {
		logging.Error("set version mod IDs failed", "platformVersionID", platformVersionID, "modIDCount", len(modIDs), "error", err)
		return err
	}
	logging.Info("version mod IDs set", "platformVersionID", platformVersionID, "modIDCount", len(modIDs))
	return nil
}
