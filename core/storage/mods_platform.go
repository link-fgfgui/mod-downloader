package storage

import (
	"strings"
	"time"

	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
)

func UpsertModPlatform(p models.ModProject) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	p = copyProject(p)
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
				p.Categories = copyStringSlice(existing.Categories)
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
	return copyProject(p), true
}

func GetModPlatformBySlug(platform, slug string) (models.ModProject, bool) {
	d, err := readyDB()
	if err != nil {
		return models.ModProject{}, false
	}
	platform = normalizePlatform(platform)
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
	return copyProject(p), true
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
	platform = normalizePlatform(platform)
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

// GetVersionBySHA1 looks up any cached platform version whose SHA1 matches,
// regardless of whether it is the latest version for its project.
func GetVersionBySHA1(platform, sha1 string) (models.ModVersion, bool) {
	d, err := readyDB()
	if err != nil {
		return models.ModVersion{}, false
	}
	platform = normalizePlatform(platform)
	sha1 = strings.ToLower(strings.TrimSpace(sha1))
	if platform == "" || sha1 == "" {
		return models.ModVersion{}, false
	}

	var best models.ModVersion
	found := false
	err = d.view(func(state *cacheState) error {
		for key, v := range state.PlatformVersions {
			if key.Platform != platform {
				continue
			}
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

func GetLocalMetadataMissCheckedAt(provider, identity string) (int64, bool) {
	d, err := readyDB()
	if err != nil {
		return 0, false
	}
	key := makeLocalMetadataMissKey(provider, identity)
	if key.Provider == "" || key.Identity == "" {
		return 0, false
	}

	var checkedAt int64
	var found bool
	err = d.view(func(state *cacheState) error {
		checkedAt, found = state.LocalMetadataMisses[key]
		return nil
	})
	return checkedAt, err == nil && found
}

func SetLocalMetadataMisses(provider string, identities []string, checkedAt int64) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	if checkedAt <= 0 {
		return nil
	}
	keys := make([]localMetadataMissKey, 0, len(identities))
	seen := make(map[localMetadataMissKey]struct{}, len(identities))
	for _, identity := range identities {
		key := makeLocalMetadataMissKey(provider, identity)
		if key.Provider == "" || key.Identity == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return nil
	}
	return d.update(func(state *cacheState, pool *stringPool) error {
		for _, key := range keys {
			key.Provider = pool.Intern(key.Provider)
			key.Identity = pool.Intern(key.Identity)
			state.LocalMetadataMisses[key] = checkedAt
		}
		return nil
	})
}

func makeLocalMetadataMissKey(provider, identity string) localMetadataMissKey {
	return localMetadataMissKey{
		Provider: strings.ToLower(strings.TrimSpace(provider)),
		Identity: strings.ToLower(strings.TrimSpace(identity)),
	}
}

func GetMinecraftReleaseVersions() ([]string, int64) {
	d, err := readyDB()
	if err != nil {
		return nil, 0
	}
	var versions []string
	var updatedAt int64
	if err := d.view(func(state *cacheState) error {
		versions = copyStringSlice(state.MinecraftReleases)
		updatedAt = state.MinecraftReleasesAt
		return nil
	}); err != nil {
		return nil, 0
	}
	return versions, updatedAt
}

func SetMinecraftReleaseVersions(versions []string, updatedAt int64) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	if len(versions) == 0 || updatedAt <= 0 {
		return nil
	}
	return d.update(func(state *cacheState, pool *stringPool) error {
		state.MinecraftReleases = make([]string, 0, len(versions))
		seen := make(map[string]struct{}, len(versions))
		for _, version := range versions {
			version = strings.TrimSpace(version)
			if version == "" {
				continue
			}
			if _, ok := seen[version]; ok {
				continue
			}
			seen[version] = struct{}{}
			state.MinecraftReleases = append(state.MinecraftReleases, pool.Intern(version))
		}
		state.MinecraftReleasesAt = updatedAt
		return nil
	})
}

// --- mod platform versions ---
