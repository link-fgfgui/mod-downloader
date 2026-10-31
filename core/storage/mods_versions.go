package storage

import (
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
)

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
	if strings.TrimSpace(p.Platform) == "" {
		p.Platform = key.Platform
	}
	if strings.TrimSpace(p.ProjectID) == "" {
		p.ProjectID = key.ProjectID
	}
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
	v.Platform = strings.TrimSpace(platform)
	v.ProjectID = projectID
	v.VersionID = strings.TrimSpace(v.VersionID)
	key := internVersionKey(pool, makeVersionKey(v.Platform, projectID, v.VersionID))

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
	platform = normalizePlatform(platform)
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
	platform = normalizePlatform(platform)
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

// FindCachedModVersionsByModID returns the newest compatible cached version for
// each platform project declaring modID. It never refreshes provider metadata.
func FindCachedModVersionsByModID(modID, minecraftVersion, modLoader string) []CachedModVersionCandidate {
	d, err := readyDB()
	if err != nil {
		return nil
	}
	modID = strings.ToLower(strings.TrimSpace(modID))
	minecraftVersion = strings.TrimSpace(minecraftVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))
	if modID == "" {
		return nil
	}

	byProject := make(map[platformKey]CachedModVersionCandidate)
	_ = d.view(func(state *cacheState) error {
		for key, version := range state.PlatformVersions {
			if !containsCachedValue(version.ModIDs, modID) ||
				!containsCachedValue(version.GameVersions, minecraftVersion) ||
				!containsCachedValue(version.Loaders, modLoader) {
				continue
			}
			if strings.TrimSpace(version.ID) == "" || strings.TrimSpace(version.DownloadURL) == "" {
				continue
			}
			projectKey := platformKey{Platform: key.Platform, ProjectID: key.ProjectID}
			current, ok := byProject[projectKey]
			if ok && !cachedVersionBefore(current.Version, version) {
				continue
			}
			project := state.ModPlatforms[projectKey]
			if strings.TrimSpace(project.Platform) == "" {
				project.Platform = key.Platform
			}
			if strings.TrimSpace(project.ProjectID) == "" {
				project.ProjectID = key.ProjectID
			}
			byProject[projectKey] = CachedModVersionCandidate{
				Project: copyProject(project),
				Version: copyVersion(version),
			}
		}
		return nil
	})

	out := make([]CachedModVersionCandidate, 0, len(byProject))
	for _, candidate := range byProject {
		out = append(out, candidate)
	}
	sort.Slice(out, func(i, j int) bool {
		left := models.ProjectKey(out[i].Project.Platform, out[i].Project.ProjectID)
		right := models.ProjectKey(out[j].Project.Platform, out[j].Project.ProjectID)
		return left < right
	})
	return out
}

func containsCachedValue(values []string, expected string) bool {
	expected = strings.ToLower(strings.TrimSpace(expected))
	if expected == "" {
		return true
	}
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == expected {
			return true
		}
	}
	return false
}

func cachedVersionBefore(current, next models.ModVersion) bool {
	if current.PublishedAt != next.PublishedAt {
		return current.PublishedAt < next.PublishedAt
	}
	return strings.TrimSpace(current.ID) < strings.TrimSpace(next.ID)
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

// GetVersionModIDs returns the persisted mod IDs for a platform version ID.
// Returns nil (no error) when the version is unknown or has no persisted IDs;
// callers should treat nil as a cache miss.
func GetVersionModIDs(platformVersionID string) ([]string, error) {
	d, err := readyDB()
	if err != nil {
		return nil, err
	}
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" {
		return nil, nil
	}

	var modIDs []string
	err = d.view(func(state *cacheState) error {
		key, ok := state.PlatformVersionKeyByID[platformVersionID]
		if !ok {
			return nil
		}
		if v, ok := state.PlatformVersions[key]; ok {
			modIDs = copyStringSlice(v.ModIDs)
		}
		return nil
	})
	if err != nil {
		logging.Error("get version mod IDs failed", "platformVersionID", platformVersionID, "error", err)
		return nil, err
	}
	logging.Debug("version mod IDs loaded", "platformVersionID", platformVersionID, "modIDCount", len(modIDs))
	return modIDs, nil
}
