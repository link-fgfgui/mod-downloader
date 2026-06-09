package database

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"mod-downloader/logging"
	structs "mod-downloader/structs/minecraft"

	"github.com/tidwall/buntdb"
)

const (
	kindMetadata       = "meta"
	kindModPlatform    = "mod-platform"
	kindAssociation    = "association"
	kindVersion        = "version"
	kindVersionByID    = "version-id"
	kindVersionScope   = "version-scope"
	kindPinnedMod      = "pinned-mod"
	kindJarMetadata    = "jar-metadata"
	jarMetadataVersion = "recursive-jar-mod-id-v4"
)

// --- record types ---

type ModPlatform struct {
	Platform    string `json:"platform"`
	ProjectID   string `json:"projectId"`
	Slug        string `json:"slug,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	McmodURL    string `json:"mcmodUrl,omitempty"`
	UpdatedAt   int64  `json:"updatedAt"`
}

type PlatformAssociation struct {
	ID                  string `json:"id"`
	CurseForgeProjectID string `json:"curseforgeProjectId,omitempty"`
	ModrinthProjectID   string `json:"modrinthProjectId,omitempty"`
}

type ModPlatformVersion struct {
	ID           string          `json:"id"`
	Platform     string          `json:"platform"`
	ProjectID    string          `json:"projectId"`
	VersionID    string          `json:"versionId"`
	Name         string          `json:"name,omitempty"`
	Version      string          `json:"version,omitempty"`
	FileName     string          `json:"fileName,omitempty"`
	DownloadURL  string          `json:"downloadUrl,omitempty"`
	SHA1         string          `json:"sha1,omitempty"`
	PublishedAt  int64           `json:"publishedAt,omitempty"`
	Downloads    int64           `json:"downloads"`
	GameVersions []string        `json:"gameVersions,omitempty"`
	Loaders      []string        `json:"loaders,omitempty"`
	Dependencies []ModDependency `json:"dependencies,omitempty"`
}

type ModDependency struct {
	ID                  string `json:"id"`
	PlatformVersionID   string `json:"platformVersionId"`
	DependencyProjectID string `json:"dependencyProjectId"`
	DependencyVersionID string `json:"dependencyVersionId,omitempty"`
	DependencyType      string `json:"dependencyType,omitempty"`
}

type PinnedMod struct {
	ID               string `json:"id"`
	Platform         string `json:"platform"`
	ModID            string `json:"modId"`
	VersionID        string `json:"versionId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
}

type ModJarMetadata struct {
	SHA1        string `json:"sha1"`
	ModID       string `json:"modId"`
	Name        string `json:"name,omitempty"`
	Version     string `json:"version,omitempty"`
	Description string `json:"description,omitempty"`
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

func keyPart(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func dbKey(kind string, parts ...string) string {
	var b strings.Builder
	b.WriteString(kind)
	for _, part := range parts {
		b.WriteByte(':')
		b.WriteString(keyPart(part))
	}
	return b.String()
}

func dbPrefix(kind string, parts ...string) string {
	return dbKey(kind, parts...) + ":"
}

func dbPattern(kind string, parts ...string) string {
	return dbPrefix(kind, parts...) + "*"
}

func getJSON(tx *buntdb.Tx, key string, out any) (bool, error) {
	value, err := tx.Get(key)
	if err != nil {
		if errors.Is(err, buntdb.ErrNotFound) {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(value), out); err != nil {
		return false, err
	}
	return true, nil
}

func setJSON(tx *buntdb.Tx, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, _, err = tx.Set(key, string(data), nil)
	return err
}

func deleteKey(tx *buntdb.Tx, key string) error {
	if _, err := tx.Delete(key); err != nil && !errors.Is(err, buntdb.ErrNotFound) {
		return err
	}
	return nil
}

func deletePattern(tx *buntdb.Tx, pattern string) error {
	keys := make([]string, 0)
	if err := tx.AscendKeys(pattern, func(key, value string) bool {
		keys = append(keys, key)
		return true
	}); err != nil {
		return err
	}
	for _, key := range keys {
		if err := deleteKey(tx, key); err != nil {
			return err
		}
	}
	return nil
}

func resetJarMetadataCacheWhenNeeded(tx *buntdb.Tx) error {
	metaKey := dbKey(kindMetadata, "jar_metadata_version")
	version, err := tx.Get(metaKey)
	if err == nil && version == jarMetadataVersion {
		return nil
	}
	if err != nil && !errors.Is(err, buntdb.ErrNotFound) {
		return err
	}
	if err := deletePattern(tx, dbPattern(kindJarMetadata)); err != nil {
		return err
	}
	_, _, err = tx.Set(metaKey, jarMetadataVersion, nil)
	return err
}

// --- mod platforms ---

func UpsertModPlatform(p ModPlatform) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	p.Platform = strings.TrimSpace(p.Platform)
	p.ProjectID = strings.TrimSpace(p.ProjectID)
	if p.Platform == "" || p.ProjectID == "" {
		return nil
	}

	err = d.Update(func(tx *buntdb.Tx) error {
		key := dbKey(kindModPlatform, p.Platform, p.ProjectID)
		var existing ModPlatform
		if ok, err := getJSON(tx, key, &existing); err != nil {
			return err
		} else if ok {
			if strings.TrimSpace(p.Slug) == "" {
				p.Slug = existing.Slug
			}
			p.UpdatedAt = existing.UpdatedAt
		}
		return setJSON(tx, key, p)
	})
	if err != nil {
		logging.Error("upsert mod platform failed", "platform", p.Platform, "projectID", p.ProjectID, "slug", p.Slug, "error", err)
		return err
	}
	logging.Info("mod platform upserted", "platform", p.Platform, "projectID", p.ProjectID, "slug", p.Slug)
	return nil
}

func GetModPlatform(platform, projectID string) (ModPlatform, bool) {
	d, err := readyDB()
	if err != nil {
		return ModPlatform{}, false
	}
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	if platform == "" || projectID == "" {
		return ModPlatform{}, false
	}

	var p ModPlatform
	err = d.View(func(tx *buntdb.Tx) error {
		ok, err := getJSON(tx, dbKey(kindModPlatform, platform, projectID), &p)
		if err != nil {
			return err
		}
		if !ok {
			return buntdb.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, buntdb.ErrNotFound) {
			logging.Error("get mod platform failed", "platform", platform, "projectID", projectID, "error", err)
		} else {
			logging.Debug("mod platform cache miss", "platform", platform, "projectID", projectID)
		}
		return ModPlatform{}, false
	}
	logging.Debug("mod platform cache hit", "platform", platform, "projectID", projectID, "updatedAt", p.UpdatedAt)
	return p, true
}

func GetModPlatformBySlug(platform, slug string) (ModPlatform, bool) {
	d, err := readyDB()
	if err != nil {
		return ModPlatform{}, false
	}
	platform = strings.TrimSpace(platform)
	slug = strings.TrimSpace(slug)
	if platform == "" || slug == "" {
		return ModPlatform{}, false
	}

	var p ModPlatform
	err = d.View(func(tx *buntdb.Tx) error {
		found := false
		var scanErr error
		err := tx.AscendKeys(dbPattern(kindModPlatform, platform), func(key, value string) bool {
			var candidate ModPlatform
			if err := json.Unmarshal([]byte(value), &candidate); err != nil {
				scanErr = err
				return false
			}
			if candidate.Slug == slug {
				p = candidate
				found = true
				return false
			}
			return true
		})
		if err != nil {
			return err
		}
		if scanErr != nil {
			return scanErr
		}
		if !found {
			return buntdb.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, buntdb.ErrNotFound) {
			logging.Error("get mod platform by slug failed", "platform", platform, "slug", slug, "error", err)
		} else {
			logging.Debug("mod platform slug cache miss", "platform", platform, "slug", slug)
		}
		return ModPlatform{}, false
	}
	logging.Debug("mod platform slug cache hit", "platform", platform, "slug", slug, "projectID", p.ProjectID, "updatedAt", p.UpdatedAt)
	return p, true
}

func TouchModPlatform(platform, projectID string, updatedAt int64) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	if platform == "" || projectID == "" {
		return nil
	}

	err = d.Update(func(tx *buntdb.Tx) error {
		key := dbKey(kindModPlatform, platform, projectID)
		p := ModPlatform{Platform: platform, ProjectID: projectID}
		if _, err := getJSON(tx, key, &p); err != nil {
			return err
		}
		p.Platform = platform
		p.ProjectID = projectID
		p.UpdatedAt = updatedAt
		return setJSON(tx, key, p)
	})
	if err != nil {
		logging.Error("touch mod platform failed", "platform", platform, "projectID", projectID, "updatedAt", updatedAt, "error", err)
		return err
	}
	logging.Debug("mod platform touched", "platform", platform, "projectID", projectID, "updatedAt", updatedAt)
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

	err = d.Update(func(tx *buntdb.Tx) error {
		return setJSON(tx, dbKey(kindAssociation, a.ID), a)
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

	err = d.Update(func(tx *buntdb.Tx) error {
		ids := make([]string, 0, 2)
		var scanErr error
		if err := tx.AscendKeys(dbPattern(kindAssociation), func(key, value string) bool {
			var a PlatformAssociation
			if err := json.Unmarshal([]byte(value), &a); err != nil {
				scanErr = err
				return false
			}
			if a.CurseForgeProjectID == curseForgeProjectID || a.ModrinthProjectID == modrinthProjectID {
				ids = append(ids, a.ID)
			}
			return true
		}); err != nil {
			return err
		}
		if scanErr != nil {
			return scanErr
		}

		id := NewID()
		if len(ids) > 0 {
			id = ids[0]
			for _, duplicateID := range ids[1:] {
				if err := deleteKey(tx, dbKey(kindAssociation, duplicateID)); err != nil {
					logging.Error("delete duplicate platform association failed", "id", duplicateID, "error", err)
					return err
				}
			}
		}
		return setJSON(tx, dbKey(kindAssociation, id), PlatformAssociation{
			ID:                  id,
			CurseForgeProjectID: curseForgeProjectID,
			ModrinthProjectID:   modrinthProjectID,
		})
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
	err = d.View(func(tx *buntdb.Tx) error {
		found := false
		var scanErr error
		if err := tx.AscendKeys(dbPattern(kindAssociation), func(key, value string) bool {
			var a PlatformAssociation
			if err := json.Unmarshal([]byte(value), &a); err != nil {
				scanErr = err
				return false
			}
			if match(a) {
				out = a
				found = true
				return false
			}
			return true
		}); err != nil {
			return err
		}
		if scanErr != nil {
			return scanErr
		}
		if !found {
			return buntdb.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, buntdb.ErrNotFound) {
			logging.Error("get platform association failed", logKey, logValue, "error", err)
		} else {
			logging.Debug("platform association miss", logKey, logValue)
		}
		return PlatformAssociation{}, false
	}
	logging.Debug("platform association hit", logKey, logValue, "id", out.ID)
	return out, true
}

func GetLatestProjectBySHA1(platform, sha1 string) (ModPlatformVersion, bool) {
	d, err := readyDB()
	if err != nil {
		return ModPlatformVersion{}, false
	}
	platform = strings.TrimSpace(platform)
	sha1 = strings.ToLower(strings.TrimSpace(sha1))
	if platform == "" || sha1 == "" {
		return ModPlatformVersion{}, false
	}

	var best ModPlatformVersion
	err = d.View(func(tx *buntdb.Tx) error {
		latestByProject := make(map[string]ModPlatformVersion)
		var scanErr error
		if err := tx.AscendKeys(dbPattern(kindVersion, platform), func(key, value string) bool {
			var v ModPlatformVersion
			if err := json.Unmarshal([]byte(value), &v); err != nil {
				scanErr = err
				return false
			}
			if current, ok := latestByProject[v.ProjectID]; !ok || newerProjectVersion(v, current) {
				latestByProject[v.ProjectID] = v
			}
			return true
		}); err != nil {
			return err
		}
		if scanErr != nil {
			return scanErr
		}

		found := false
		for _, v := range latestByProject {
			if strings.ToLower(strings.TrimSpace(v.SHA1)) != sha1 {
				continue
			}
			if !found || v.PublishedAt > best.PublishedAt || (v.PublishedAt == best.PublishedAt && v.ProjectID < best.ProjectID) {
				best = v
				found = true
			}
		}
		if !found {
			return buntdb.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, buntdb.ErrNotFound) {
			logging.Error("get latest project by sha1 failed", "platform", platform, "sha1", sha1, "error", err)
		}
		return ModPlatformVersion{}, false
	}
	return best, true
}

func newerProjectVersion(left, right ModPlatformVersion) bool {
	if left.PublishedAt != right.PublishedAt {
		return left.PublishedAt > right.PublishedAt
	}
	return left.VersionID > right.VersionID
}

// --- mod platform versions ---

func SetPlatformVersions(platform, projectID string, versions []ModPlatformVersion) error {
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
	err = d.Update(func(tx *buntdb.Tx) error {
		if err := deleteProjectVersionsTx(tx, platform, projectID); err != nil {
			return err
		}
		for _, v := range versions {
			if err := savePlatformVersionTx(tx, platform, projectID, v); err != nil {
				logging.Error("insert platform version failed", "platform", platform, "projectID", projectID, "versionID", v.VersionID, "error", err)
				return err
			}
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

func SetPlatformVersionSnapshot(platform, projectID string, versions []ModPlatformVersion, updatedAt int64, scopes []ModPlatformVersionScope) error {
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
	err = d.Update(func(tx *buntdb.Tx) error {
		if err := touchSnapshotPlatformTx(tx, platform, projectID, updatedAt, len(scopes) == 0); err != nil {
			return err
		}
		if len(scopes) == 0 {
			if err := deleteProjectVersionsTx(tx, platform, projectID); err != nil {
				return err
			}
		} else if err := deletePlatformVersionSnapshotScopesTx(tx, platform, projectID, scopes); err != nil {
			return err
		}
		for _, v := range versions {
			if err := savePlatformVersionTx(tx, platform, projectID, v); err != nil {
				logging.Error("insert snapshot platform version failed", "platform", platform, "projectID", projectID, "versionID", v.VersionID, "error", err)
				return err
			}
		}
		for _, scope := range scopes {
			rec := storedVersionScope{
				Platform:         platform,
				ProjectID:        projectID,
				MinecraftVersion: scope.MinecraftVersion,
				ModLoader:        scope.ModLoader,
				UpdatedAt:        updatedAt,
			}
			if err := setJSON(tx, dbKey(kindVersionScope, platform, projectID, scope.MinecraftVersion, scope.ModLoader), rec); err != nil {
				return err
			}
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

func touchSnapshotPlatformTx(tx *buntdb.Tx, platform, projectID string, updatedAt int64, updateProjectTimestamp bool) error {
	key := dbKey(kindModPlatform, platform, projectID)
	p := ModPlatform{Platform: platform, ProjectID: projectID}
	if _, err := getJSON(tx, key, &p); err != nil {
		return err
	}
	p.Platform = platform
	p.ProjectID = projectID
	if updateProjectTimestamp {
		p.UpdatedAt = updatedAt
	}
	return setJSON(tx, key, p)
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

func deleteProjectVersionsTx(tx *buntdb.Tx, platform, projectID string) error {
	return deleteMatchingVersionsTx(tx, dbPattern(kindVersion, platform, projectID), func(ModPlatformVersion) bool {
		return true
	})
}

func deletePlatformVersionSnapshotScopesTx(tx *buntdb.Tx, platform, projectID string, scopes []ModPlatformVersionScope) error {
	return deleteMatchingVersionsTx(tx, dbPattern(kindVersion, platform, projectID), func(v ModPlatformVersion) bool {
		return platformVersionMatchesAnyScope(v.GameVersions, v.Loaders, scopes)
	})
}

type versionDelete struct {
	key string
	id  string
}

func deleteMatchingVersionsTx(tx *buntdb.Tx, pattern string, shouldDelete func(ModPlatformVersion) bool) error {
	deletes := make([]versionDelete, 0)
	var scanErr error
	if err := tx.AscendKeys(pattern, func(key, value string) bool {
		var v ModPlatformVersion
		if err := json.Unmarshal([]byte(value), &v); err != nil {
			scanErr = err
			return false
		}
		if shouldDelete(v) {
			deletes = append(deletes, versionDelete{key: key, id: v.ID})
		}
		return true
	}); err != nil {
		return err
	}
	if scanErr != nil {
		return scanErr
	}
	for _, item := range deletes {
		if err := deleteKey(tx, item.key); err != nil {
			return err
		}
		if item.id != "" {
			if err := deleteKey(tx, dbKey(kindVersionByID, item.id)); err != nil {
				return err
			}
		}
	}
	return nil
}

func platformVersionMatchesAnyScope(gameVersions, loaders []string, scopes []ModPlatformVersionScope) bool {
	for _, scope := range scopes {
		if scope.MinecraftVersion != "" && !containsFoldDB(gameVersions, scope.MinecraftVersion) {
			continue
		}
		if scope.ModLoader != "" && !containsFoldDB(loaders, scope.ModLoader) {
			continue
		}
		return true
	}
	return false
}

func containsFoldDB(values []string, expected string) bool {
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), expected) {
			return true
		}
	}
	return false
}

func savePlatformVersionTx(tx *buntdb.Tx, platform, projectID string, v ModPlatformVersion) error {
	v.Platform = platform
	v.ProjectID = projectID
	v.VersionID = strings.TrimSpace(v.VersionID)
	key := dbKey(kindVersion, platform, projectID, v.VersionID)

	var existing ModPlatformVersion
	if ok, err := getJSON(tx, key, &existing); err != nil {
		return err
	} else if ok && existing.ID != "" {
		v.ID = existing.ID
	}
	if v.ID == "" {
		v.ID = NewID()
	}
	v.Dependencies = normalizeDependencies(v.ID, v.Dependencies)
	if err := setJSON(tx, key, v); err != nil {
		return err
	}
	_, _, err := tx.Set(dbKey(kindVersionByID, v.ID), key, nil)
	return err
}

func normalizeDependencies(platformVersionID string, deps []ModDependency) []ModDependency {
	out := make([]ModDependency, 0, len(deps))
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
	scope = scopes[0]

	var rec storedVersionScope
	err = d.View(func(tx *buntdb.Tx) error {
		ok, err := getJSON(tx, dbKey(kindVersionScope, platform, projectID, scope.MinecraftVersion, scope.ModLoader), &rec)
		if err != nil {
			return err
		}
		if !ok {
			return buntdb.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, buntdb.ErrNotFound) {
			logging.Error("get platform version scope failed", "platform", platform, "projectID", projectID, "minecraftVersion", scope.MinecraftVersion, "modLoader", scope.ModLoader, "error", err)
		}
		return 0, false
	}
	return rec.UpdatedAt, true
}

func GetPlatformVersions(platform, projectID string) ([]ModPlatformVersion, error) {
	d, err := readyDB()
	if err != nil {
		return nil, err
	}
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	if platform == "" || projectID == "" {
		return nil, nil
	}

	versions := make([]ModPlatformVersion, 0)
	err = d.View(func(tx *buntdb.Tx) error {
		var scanErr error
		if err := tx.AscendKeys(dbPattern(kindVersion, platform, projectID), func(key, value string) bool {
			var v ModPlatformVersion
			if err := json.Unmarshal([]byte(value), &v); err != nil {
				scanErr = err
				return false
			}
			versions = append(versions, v)
			return true
		}); err != nil {
			return err
		}
		return scanErr
	})
	if err != nil {
		logging.Error("get platform versions failed", "platform", platform, "projectID", projectID, "error", err)
		return nil, err
	}
	logging.Debug("platform versions loaded", "platform", platform, "projectID", projectID, "versionCount", len(versions))
	return versions, nil
}

// --- mod dependencies ---

func SetVersionDependencies(platformVersionID string, deps []ModDependency) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" {
		return nil
	}

	logging.Debug("set version dependencies started", "platformVersionID", platformVersionID, "dependencyCount", len(deps))
	err = d.Update(func(tx *buntdb.Tx) error {
		versionKey, err := tx.Get(dbKey(kindVersionByID, platformVersionID))
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				return nil
			}
			return err
		}
		var v ModPlatformVersion
		if ok, err := getJSON(tx, versionKey, &v); err != nil {
			return err
		} else if !ok {
			return nil
		}
		v.Dependencies = normalizeDependencies(platformVersionID, deps)
		return setJSON(tx, versionKey, v)
	})
	if err != nil {
		logging.Error("set version dependencies failed", "platformVersionID", platformVersionID, "dependencyCount", len(deps), "error", err)
		return err
	}
	logging.Info("version dependencies set", "platformVersionID", platformVersionID, "dependencyCount", len(deps))
	return nil
}

func GetVersionDependencies(platformVersionID string) ([]ModDependency, error) {
	d, err := readyDB()
	if err != nil {
		return nil, err
	}
	platformVersionID = strings.TrimSpace(platformVersionID)
	if platformVersionID == "" {
		return nil, nil
	}

	var deps []ModDependency
	err = d.View(func(tx *buntdb.Tx) error {
		versionKey, err := tx.Get(dbKey(kindVersionByID, platformVersionID))
		if err != nil {
			if errors.Is(err, buntdb.ErrNotFound) {
				return nil
			}
			return err
		}
		var v ModPlatformVersion
		if ok, err := getJSON(tx, versionKey, &v); err != nil {
			return err
		} else if ok {
			deps = append(deps, v.Dependencies...)
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

	err = d.Update(func(tx *buntdb.Tx) error {
		key := dbKey(kindPinnedMod, p.Platform, p.ModID, p.MinecraftVersion, p.ModLoader)
		var existing PinnedMod
		if ok, err := getJSON(tx, key, &existing); err != nil {
			return err
		} else if ok && existing.ID != "" {
			p.ID = existing.ID
		}
		if p.ID == "" {
			p.ID = NewID()
		}
		return setJSON(tx, key, p)
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
	platform, modID, mcVersion, modLoader = normalizePinnedModKey(platform, modID, mcVersion, modLoader)
	if platform == "" || modID == "" || mcVersion == "" || modLoader == "" {
		return PinnedMod{}, false
	}

	var p PinnedMod
	err = d.View(func(tx *buntdb.Tx) error {
		ok, err := getJSON(tx, dbKey(kindPinnedMod, platform, modID, mcVersion, modLoader), &p)
		if err != nil {
			return err
		}
		if !ok {
			return buntdb.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, buntdb.ErrNotFound) {
			logging.Error("get pinned mod failed", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader, "error", err)
		} else {
			logging.Debug("pinned mod miss", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader)
		}
		return PinnedMod{}, false
	}
	logging.Debug("pinned mod hit", "platform", platform, "modID", modID, "versionID", p.VersionID, "minecraftVersion", mcVersion, "modLoader", modLoader)
	return p, true
}

func DeletePinnedMod(platform, modID, mcVersion, modLoader string) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	platform, modID, mcVersion, modLoader = normalizePinnedModKey(platform, modID, mcVersion, modLoader)
	if platform == "" || modID == "" || mcVersion == "" || modLoader == "" {
		return nil
	}

	err = d.Update(func(tx *buntdb.Tx) error {
		return deleteKey(tx, dbKey(kindPinnedMod, platform, modID, mcVersion, modLoader))
	})
	if err != nil {
		logging.Error("delete pinned mod failed", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader, "error", err)
		return err
	}
	logging.Info("pinned mod deleted", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader)
	return nil
}

// --- jar metadata cache ---

func GetJarMetadata(sha1 string) ([]structs.ModInfo, bool) {
	d, err := readyDB()
	if err != nil {
		return nil, false
	}
	sha1 = strings.TrimSpace(sha1)
	if sha1 == "" {
		logging.Debug("jar metadata skipped for empty sha1")
		return nil, false
	}

	var mods []structs.ModInfo
	err = d.View(func(tx *buntdb.Tx) error {
		ok, err := getJSON(tx, dbKey(kindJarMetadata, sha1), &mods)
		if err != nil {
			return err
		}
		if !ok || len(mods) == 0 {
			return buntdb.ErrNotFound
		}
		return nil
	})
	if err != nil {
		if !errors.Is(err, buntdb.ErrNotFound) {
			logging.Error("get jar metadata failed", "sha1", sha1, "error", err)
		} else {
			logging.Debug("jar metadata miss", "sha1", sha1)
		}
		return nil, false
	}
	logging.Debug("jar metadata hit", "sha1", sha1, "modCount", len(mods))
	return mods, true
}

func SetJarMetadata(sha1 string, mods []structs.ModInfo) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	sha1 = strings.TrimSpace(sha1)
	if sha1 == "" || len(mods) == 0 {
		logging.Debug("set jar metadata skipped", "sha1", sha1, "modCount", len(mods))
		return nil
	}

	filtered := make([]structs.ModInfo, 0, len(mods))
	seen := make(map[string]struct{}, len(mods))
	for _, mod := range mods {
		mod.ID = strings.TrimSpace(mod.ID)
		if mod.ID == "" {
			continue
		}
		key := strings.ToLower(mod.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, mod)
	}

	err = d.Update(func(tx *buntdb.Tx) error {
		key := dbKey(kindJarMetadata, sha1)
		if len(filtered) == 0 {
			return deleteKey(tx, key)
		}
		return setJSON(tx, key, filtered)
	})
	if err != nil {
		logging.Error("set jar metadata failed", "sha1", sha1, "modCount", len(mods), "error", err)
		return err
	}
	logging.Info("jar metadata set", "sha1", sha1, "modCount", len(filtered))
	return nil
}
