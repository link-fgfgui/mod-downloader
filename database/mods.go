package database

import (
	"database/sql"
	"encoding/json"
	"strings"

	"mod-downloader/logging"
	structs "mod-downloader/structs/minecraft"
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

func jsonStringArray(values []string) string {
	if values == nil {
		values = []string{}
	}
	data, _ := json.Marshal(values)
	return string(data)
}

// --- mod platforms ---

func UpsertModPlatform(p ModPlatform) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	_, err = d.Exec(`
		INSERT INTO mod_platforms (platform, project_id, slug, name, description, mcmod_url)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(platform, project_id) DO UPDATE SET
			slug        = CASE
				WHEN excluded.slug IS NOT NULL AND excluded.slug <> '' THEN excluded.slug
				ELSE mod_platforms.slug
			END,
			name        = excluded.name,
			description = excluded.description,
			mcmod_url   = excluded.mcmod_url
	`, p.Platform, p.ProjectID, p.Slug, p.Name, p.Description, p.McmodURL)
	if err != nil {
		logging.Error("upsert mod platform failed", "platform", p.Platform, "projectID", p.ProjectID, "slug", p.Slug, "error", err)
		return err
	}
	logging.Info("mod platform upserted", "platform", p.Platform, "projectID", p.ProjectID, "slug", p.Slug)
	return err
}

func GetModPlatform(platform, projectID string) (ModPlatform, bool) {
	d, err := readyDB()
	if err != nil {
		return ModPlatform{}, false
	}
	var p ModPlatform
	err = d.QueryRow(`SELECT platform, project_id, slug, name, description, mcmod_url, updated_at FROM mod_platforms WHERE platform = ? AND project_id = ?`,
		platform, projectID).Scan(&p.Platform, &p.ProjectID, &p.Slug, &p.Name, &p.Description, &p.McmodURL, &p.UpdatedAt)
	if err != nil {
		if err != sql.ErrNoRows {
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
	var p ModPlatform
	err = d.QueryRow(`SELECT platform, project_id, slug, name, description, mcmod_url, updated_at FROM mod_platforms WHERE platform = ? AND slug = ?`,
		platform, slug).Scan(&p.Platform, &p.ProjectID, &p.Slug, &p.Name, &p.Description, &p.McmodURL, &p.UpdatedAt)
	if err != nil {
		if err != sql.ErrNoRows {
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
	_, err = d.Exec(`
		INSERT INTO mod_platforms (platform, project_id, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(platform, project_id) DO UPDATE SET
			updated_at = excluded.updated_at
	`, platform, projectID, updatedAt)
	if err != nil {
		logging.Error("touch mod platform failed", "platform", platform, "projectID", projectID, "updatedAt", updatedAt, "error", err)
		return err
	}
	logging.Debug("mod platform touched", "platform", platform, "projectID", projectID, "updatedAt", updatedAt)
	return err
}

// --- platform associations ---

func UpsertPlatformAssociation(a PlatformAssociation) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	if a.ID == "" {
		a.ID = NewID()
	}
	_, err = d.Exec(`
		INSERT INTO platform_associations (id, curseforge_project_id, modrinth_project_id)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			curseforge_project_id = excluded.curseforge_project_id,
			modrinth_project_id   = excluded.modrinth_project_id
	`, a.ID, a.CurseForgeProjectID, a.ModrinthProjectID)
	if err != nil {
		logging.Error("upsert platform association failed", "id", a.ID, "curseforgeProjectID", a.CurseForgeProjectID, "modrinthProjectID", a.ModrinthProjectID, "error", err)
		return err
	}
	logging.Info("platform association upserted", "id", a.ID, "curseforgeProjectID", a.CurseForgeProjectID, "modrinthProjectID", a.ModrinthProjectID)
	return err
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

	rows, err := d.Query(`
		SELECT id FROM platform_associations
		WHERE curseforge_project_id = ? OR modrinth_project_id = ?
	`, curseForgeProjectID, modrinthProjectID)
	if err != nil {
		logging.Error("query platform association failed", "curseforgeProjectID", curseForgeProjectID, "modrinthProjectID", modrinthProjectID, "error", err)
		return err
	}
	defer rows.Close()

	ids := make([]string, 0, 2)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	id := NewID()
	if len(ids) > 0 {
		id = ids[0]
		for _, duplicateID := range ids[1:] {
			if _, err := d.Exec(`DELETE FROM platform_associations WHERE id = ?`, duplicateID); err != nil {
				logging.Error("delete duplicate platform association failed", "id", duplicateID, "error", err)
				return err
			}
		}
	}

	_, err = d.Exec(`
		INSERT INTO platform_associations (id, curseforge_project_id, modrinth_project_id)
		VALUES (?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			curseforge_project_id = excluded.curseforge_project_id,
			modrinth_project_id   = excluded.modrinth_project_id
	`, id, curseForgeProjectID, modrinthProjectID)
	if err != nil {
		logging.Error("upsert platform association by projects failed", "id", id, "curseforgeProjectID", curseForgeProjectID, "modrinthProjectID", modrinthProjectID, "error", err)
		return err
	}
	logging.Info("platform association upserted by projects", "id", id, "curseforgeProjectID", curseForgeProjectID, "modrinthProjectID", modrinthProjectID)
	return nil
}

func GetAssociationByCurseForge(cfProjectID string) (PlatformAssociation, bool) {
	d, err := readyDB()
	if err != nil {
		return PlatformAssociation{}, false
	}
	var a PlatformAssociation
	err = d.QueryRow(`SELECT id, curseforge_project_id, modrinth_project_id FROM platform_associations WHERE curseforge_project_id = ?`,
		cfProjectID).Scan(&a.ID, &a.CurseForgeProjectID, &a.ModrinthProjectID)
	if err != nil {
		if err != sql.ErrNoRows {
			logging.Error("get platform association by curseforge failed", "curseforgeProjectID", cfProjectID, "error", err)
		} else {
			logging.Debug("platform association curseforge miss", "curseforgeProjectID", cfProjectID)
		}
		return PlatformAssociation{}, false
	}
	logging.Debug("platform association curseforge hit", "curseforgeProjectID", cfProjectID, "id", a.ID, "modrinthProjectID", a.ModrinthProjectID)
	return a, true
}

func GetAssociationByModrinth(mrProjectID string) (PlatformAssociation, bool) {
	d, err := readyDB()
	if err != nil {
		return PlatformAssociation{}, false
	}
	var a PlatformAssociation
	err = d.QueryRow(`SELECT id, curseforge_project_id, modrinth_project_id FROM platform_associations WHERE modrinth_project_id = ?`,
		mrProjectID).Scan(&a.ID, &a.CurseForgeProjectID, &a.ModrinthProjectID)
	if err != nil {
		if err != sql.ErrNoRows {
			logging.Error("get platform association by modrinth failed", "modrinthProjectID", mrProjectID, "error", err)
		} else {
			logging.Debug("platform association modrinth miss", "modrinthProjectID", mrProjectID)
		}
		return PlatformAssociation{}, false
	}
	logging.Debug("platform association modrinth hit", "modrinthProjectID", mrProjectID, "id", a.ID, "curseforgeProjectID", a.CurseForgeProjectID)
	return a, true
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

	var v ModPlatformVersion
	err = d.QueryRow(`
		SELECT v.id, v.platform, v.project_id, v.version_id, v.name, v.version, v.file_name, v.download_url, v.sha1, v.published_at, v.downloads, v.game_versions, v.loaders
		FROM mod_platform_versions v
		WHERE v.platform = ? AND lower(v.sha1) = ?
			AND NOT EXISTS (
				SELECT 1 FROM mod_platform_versions newer
				WHERE newer.platform = v.platform
					AND newer.project_id = v.project_id
					AND (
						newer.published_at > v.published_at
						OR (newer.published_at = v.published_at AND newer.version_id > v.version_id)
					)
			)
		ORDER BY v.published_at DESC, v.project_id
		LIMIT 1
	`, platform, sha1).Scan(&v.ID, &v.Platform, &v.ProjectID, &v.VersionID, &v.Name, &v.Version, &v.FileName, &v.DownloadURL, &v.SHA1, &v.PublishedAt, &v.Downloads, new(sql.NullString), new(sql.NullString))
	if err != nil {
		if err != sql.ErrNoRows {
			logging.Error("get latest project by sha1 failed", "platform", platform, "sha1", sha1, "error", err)
		}
		return ModPlatformVersion{}, false
	}
	return v, true
}

// --- mod platform versions ---

func SetPlatformVersions(platform, projectID string, versions []ModPlatformVersion) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	logging.Debug("set platform versions started", "platform", platform, "projectID", projectID, "versionCount", len(versions))
	tx, err := d.Begin()
	if err != nil {
		logging.Error("begin set platform versions transaction failed", "platform", platform, "projectID", projectID, "error", err)
		return err
	}
	defer tx.Rollback()

	_, _ = tx.Exec(`DELETE FROM mod_platform_versions WHERE platform = ? AND project_id = ?`, platform, projectID)
	for _, v := range versions {
		if v.ID == "" {
			v.ID = NewID()
		}
		_, err := tx.Exec(`
			INSERT INTO mod_platform_versions (id, platform, project_id, version_id, name, version, file_name, download_url, sha1, published_at, downloads, game_versions, loaders)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, v.ID, platform, projectID, v.VersionID, v.Name, v.Version, v.FileName, v.DownloadURL, v.SHA1, v.PublishedAt, v.Downloads, jsonStringArray(v.GameVersions), jsonStringArray(v.Loaders))
		if err != nil {
			logging.Error("insert platform version failed", "platform", platform, "projectID", projectID, "versionID", v.VersionID, "error", err)
			return err
		}
		if err := insertVersionDependenciesTx(tx, v.ID, v.Dependencies); err != nil {
			logging.Error("insert platform version dependencies failed", "platform", platform, "projectID", projectID, "versionID", v.VersionID, "dependencyCount", len(v.Dependencies), "error", err)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		logging.Error("commit platform versions failed", "platform", platform, "projectID", projectID, "versionCount", len(versions), "error", err)
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
	scopes = normalizePlatformVersionScopes(scopes)
	logging.Debug("set platform version snapshot started", "platform", platform, "projectID", projectID, "versionCount", len(versions), "updatedAt", updatedAt, "scopeCount", len(scopes))
	tx, err := d.Begin()
	if err != nil {
		logging.Error("begin platform version snapshot transaction failed", "platform", platform, "projectID", projectID, "error", err)
		return err
	}
	defer tx.Rollback()

	if len(scopes) == 0 {
		_, err = tx.Exec(`
			INSERT INTO mod_platforms (platform, project_id, updated_at)
			VALUES (?, ?, ?)
			ON CONFLICT(platform, project_id) DO UPDATE SET
				updated_at = excluded.updated_at
		`, platform, projectID, updatedAt)
	} else {
		_, err = tx.Exec(`
			INSERT INTO mod_platforms (platform, project_id, updated_at)
			VALUES (?, ?, 0)
			ON CONFLICT(platform, project_id) DO NOTHING
		`, platform, projectID)
	}
	if err != nil {
		logging.Error("upsert platform snapshot timestamp failed", "platform", platform, "projectID", projectID, "updatedAt", updatedAt, "error", err)
		return err
	}

	if len(scopes) == 0 {
		_, _ = tx.Exec(`DELETE FROM mod_platform_versions WHERE platform = ? AND project_id = ?`, platform, projectID)
	} else if err := deletePlatformVersionSnapshotScopesTx(tx, platform, projectID, scopes); err != nil {
		logging.Error("delete scoped platform version snapshot failed", "platform", platform, "projectID", projectID, "scopeCount", len(scopes), "error", err)
		return err
	}
	for _, v := range versions {
		if v.ID == "" {
			v.ID = existingPlatformVersionIDTx(tx, platform, projectID, v.VersionID)
			if v.ID == "" {
				v.ID = NewID()
			}
		}
		_, _ = tx.Exec(`DELETE FROM mod_dependencies WHERE platform_version_id = ?`, v.ID)
		_, err := tx.Exec(`
			INSERT INTO mod_platform_versions (id, platform, project_id, version_id, name, version, file_name, download_url, sha1, published_at, downloads, game_versions, loaders)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(platform, project_id, version_id) DO UPDATE SET
				name          = excluded.name,
				version       = excluded.version,
				file_name     = excluded.file_name,
				download_url  = excluded.download_url,
				sha1          = excluded.sha1,
				published_at  = excluded.published_at,
				downloads     = excluded.downloads,
				game_versions = excluded.game_versions,
				loaders       = excluded.loaders
		`, v.ID, platform, projectID, v.VersionID, v.Name, v.Version, v.FileName, v.DownloadURL, v.SHA1, v.PublishedAt, v.Downloads, jsonStringArray(v.GameVersions), jsonStringArray(v.Loaders))
		if err != nil {
			logging.Error("insert snapshot platform version failed", "platform", platform, "projectID", projectID, "versionID", v.VersionID, "error", err)
			return err
		}
		if err := insertVersionDependenciesTx(tx, v.ID, v.Dependencies); err != nil {
			logging.Error("insert snapshot dependencies failed", "platform", platform, "projectID", projectID, "versionID", v.VersionID, "dependencyCount", len(v.Dependencies), "error", err)
			return err
		}
	}
	for _, scope := range scopes {
		if _, err := tx.Exec(`
			INSERT INTO mod_platform_version_scopes (platform, project_id, minecraft_version, mod_loader, updated_at)
			VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(platform, project_id, minecraft_version, mod_loader) DO UPDATE SET
				updated_at = excluded.updated_at
		`, platform, projectID, scope.MinecraftVersion, scope.ModLoader, updatedAt); err != nil {
			logging.Error("upsert platform version scope failed", "platform", platform, "projectID", projectID, "minecraftVersion", scope.MinecraftVersion, "modLoader", scope.ModLoader, "updatedAt", updatedAt, "error", err)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		logging.Error("commit platform version snapshot failed", "platform", platform, "projectID", projectID, "versionCount", len(versions), "error", err)
		return err
	}
	logging.Info("platform version snapshot set", "platform", platform, "projectID", projectID, "versionCount", len(versions), "updatedAt", updatedAt)
	return nil
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

func deletePlatformVersionSnapshotScopesTx(tx *sql.Tx, platform, projectID string, scopes []ModPlatformVersionScope) error {
	rows, err := tx.Query(`
		SELECT id, game_versions, loaders
		FROM mod_platform_versions WHERE platform = ? AND project_id = ?
	`, platform, projectID)
	if err != nil {
		return err
	}
	defer rows.Close()

	deleteIDs := make([]string, 0)
	for rows.Next() {
		var id string
		var gameVersionsJSON, loadersJSON sql.NullString
		if err := rows.Scan(&id, &gameVersionsJSON, &loadersJSON); err != nil {
			return err
		}
		gameVersions := decodeStringArray(gameVersionsJSON)
		loaders := decodeStringArray(loadersJSON)
		if platformVersionMatchesAnyScope(gameVersions, loaders, scopes) {
			deleteIDs = append(deleteIDs, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range deleteIDs {
		if _, err := tx.Exec(`DELETE FROM mod_platform_versions WHERE id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

func decodeStringArray(value sql.NullString) []string {
	if !value.Valid || strings.TrimSpace(value.String) == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(value.String), &out)
	return out
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

func existingPlatformVersionIDTx(tx *sql.Tx, platform, projectID, versionID string) string {
	var id string
	err := tx.QueryRow(`
		SELECT id FROM mod_platform_versions
		WHERE platform = ? AND project_id = ? AND version_id = ?
	`, platform, projectID, versionID).Scan(&id)
	if err != nil {
		return ""
	}
	return id
}

func GetPlatformVersionScopeUpdatedAt(platform, projectID string, scope ModPlatformVersionScope) (int64, bool) {
	d, err := readyDB()
	if err != nil {
		return 0, false
	}
	scopes := normalizePlatformVersionScopes([]ModPlatformVersionScope{scope})
	if len(scopes) == 0 {
		return 0, false
	}
	scope = scopes[0]
	var updatedAt int64
	err = d.QueryRow(`
		SELECT updated_at FROM mod_platform_version_scopes
		WHERE platform = ? AND project_id = ? AND minecraft_version = ? AND mod_loader = ?
	`, platform, projectID, scope.MinecraftVersion, scope.ModLoader).Scan(&updatedAt)
	if err != nil {
		if err != sql.ErrNoRows {
			logging.Error("get platform version scope failed", "platform", platform, "projectID", projectID, "minecraftVersion", scope.MinecraftVersion, "modLoader", scope.ModLoader, "error", err)
		}
		return 0, false
	}
	return updatedAt, true
}

func GetPlatformVersions(platform, projectID string) ([]ModPlatformVersion, error) {
	d, err := readyDB()
	if err != nil {
		return nil, err
	}
	rows, err := d.Query(`
		SELECT id, platform, project_id, version_id, name, version, file_name, download_url, sha1, published_at, downloads, game_versions, loaders
		FROM mod_platform_versions WHERE platform = ? AND project_id = ?
	`, platform, projectID)
	if err != nil {
		logging.Error("get platform versions failed", "platform", platform, "projectID", projectID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var versions []ModPlatformVersion
	for rows.Next() {
		var v ModPlatformVersion
		var gv, ld sql.NullString
		if err := rows.Scan(&v.ID, &v.Platform, &v.ProjectID, &v.VersionID, &v.Name, &v.Version,
			&v.FileName, &v.DownloadURL, &v.SHA1, &v.PublishedAt, &v.Downloads, &gv, &ld); err != nil {
			return nil, err
		}
		if gv.Valid {
			_ = json.Unmarshal([]byte(gv.String), &v.GameVersions)
		}
		if ld.Valid {
			_ = json.Unmarshal([]byte(ld.String), &v.Loaders)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		logging.Error("iterate platform versions failed", "platform", platform, "projectID", projectID, "error", err)
		return nil, err
	}
	if err := attachVersionDependencies(d, versions); err != nil {
		logging.Error("attach platform version dependencies failed", "platform", platform, "projectID", projectID, "versionCount", len(versions), "error", err)
		return nil, err
	}
	logging.Debug("platform versions loaded", "platform", platform, "projectID", projectID, "versionCount", len(versions))
	return versions, nil
}

// --- mod dependencies ---

func insertVersionDependenciesTx(tx *sql.Tx, platformVersionID string, deps []ModDependency) error {
	for _, dep := range deps {
		projectID := strings.TrimSpace(dep.DependencyProjectID)
		versionID := strings.TrimSpace(dep.DependencyVersionID)
		if projectID == "" && versionID == "" {
			continue
		}
		id := dep.ID
		if id == "" {
			id = NewID()
		}
		if _, err := tx.Exec(`
			INSERT INTO mod_dependencies (id, platform_version_id, dependency_project_id, dependency_version_id, dependency_type)
			VALUES (?, ?, ?, ?, ?)
		`, id, platformVersionID, projectID, versionID, dep.DependencyType); err != nil {
			return err
		}
	}
	return nil
}

func attachVersionDependencies(d *sql.DB, versions []ModPlatformVersion) error {
	if len(versions) == 0 {
		return nil
	}

	ids := make([]any, 0, len(versions))
	index := make(map[string]int, len(versions))
	for i, v := range versions {
		ids = append(ids, v.ID)
		index[v.ID] = i
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	rows, err := d.Query(`
		SELECT id, platform_version_id, dependency_project_id, dependency_version_id, dependency_type
		FROM mod_dependencies WHERE platform_version_id IN (`+placeholders+`)
	`, ids...)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var dep ModDependency
		var versionID sql.NullString
		if err := rows.Scan(&dep.ID, &dep.PlatformVersionID, &dep.DependencyProjectID, &versionID, &dep.DependencyType); err != nil {
			return err
		}
		dep.DependencyVersionID = versionID.String
		if i, ok := index[dep.PlatformVersionID]; ok {
			versions[i].Dependencies = append(versions[i].Dependencies, dep)
		}
	}
	return rows.Err()
}

func SetVersionDependencies(platformVersionID string, deps []ModDependency) error {
	d, err := readyDB()
	if err != nil {
		return err
	}
	logging.Debug("set version dependencies started", "platformVersionID", platformVersionID, "dependencyCount", len(deps))
	tx, err := d.Begin()
	if err != nil {
		logging.Error("begin set version dependencies transaction failed", "platformVersionID", platformVersionID, "error", err)
		return err
	}
	defer tx.Rollback()

	_, _ = tx.Exec(`DELETE FROM mod_dependencies WHERE platform_version_id = ?`, platformVersionID)
	if err := insertVersionDependenciesTx(tx, platformVersionID, deps); err != nil {
		logging.Error("insert version dependency failed", "platformVersionID", platformVersionID, "error", err)
		return err
	}
	if err := tx.Commit(); err != nil {
		logging.Error("commit version dependencies failed", "platformVersionID", platformVersionID, "dependencyCount", len(deps), "error", err)
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
	rows, err := d.Query(`
		SELECT id, platform_version_id, dependency_project_id, dependency_version_id, dependency_type
		FROM mod_dependencies WHERE platform_version_id = ?
	`, platformVersionID)
	if err != nil {
		logging.Error("get version dependencies failed", "platformVersionID", platformVersionID, "error", err)
		return nil, err
	}
	defer rows.Close()

	var deps []ModDependency
	for rows.Next() {
		var dep ModDependency
		var versionID sql.NullString
		if err := rows.Scan(&dep.ID, &dep.PlatformVersionID, &dep.DependencyProjectID, &versionID, &dep.DependencyType); err != nil {
			return nil, err
		}
		dep.DependencyVersionID = versionID.String
		deps = append(deps, dep)
	}
	if err := rows.Err(); err != nil {
		logging.Error("iterate version dependencies failed", "platformVersionID", platformVersionID, "error", err)
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
	if p.ID == "" {
		p.ID = NewID()
	}
	_, err = d.Exec(`
		INSERT INTO pinned_mods (id, platform, project_id, version_id, minecraft_version, mod_loader)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(platform, project_id, minecraft_version, mod_loader) DO UPDATE SET
			version_id = excluded.version_id
	`, p.ID, p.Platform, p.ModID, p.VersionID, p.MinecraftVersion, p.ModLoader)
	if err != nil {
		logging.Error("upsert pinned mod failed", "platform", p.Platform, "modID", p.ModID, "versionID", p.VersionID, "minecraftVersion", p.MinecraftVersion, "modLoader", p.ModLoader, "error", err)
		return err
	}
	logging.Info("pinned mod upserted", "platform", p.Platform, "modID", p.ModID, "versionID", p.VersionID, "minecraftVersion", p.MinecraftVersion, "modLoader", p.ModLoader)
	return err
}

func GetPinnedMod(platform, modID, mcVersion, modLoader string) (PinnedMod, bool) {
	d, err := readyDB()
	if err != nil {
		return PinnedMod{}, false
	}
	platform, modID, mcVersion, modLoader = normalizePinnedModKey(platform, modID, mcVersion, modLoader)
	var p PinnedMod
	err = d.QueryRow(`
		SELECT id, platform, project_id, version_id, minecraft_version, mod_loader
		FROM pinned_mods WHERE platform = ? AND project_id = ? AND minecraft_version = ? AND mod_loader = ?
	`, platform, modID, mcVersion, modLoader).
		Scan(&p.ID, &p.Platform, &p.ModID, &p.VersionID, &p.MinecraftVersion, &p.ModLoader)
	if err != nil {
		if err != sql.ErrNoRows {
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
	_, err = d.Exec(`
		DELETE FROM pinned_mods WHERE platform = ? AND project_id = ? AND minecraft_version = ? AND mod_loader = ?
	`, platform, modID, mcVersion, modLoader)
	if err != nil {
		logging.Error("delete pinned mod failed", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader, "error", err)
		return err
	}
	logging.Info("pinned mod deleted", "platform", platform, "modID", modID, "minecraftVersion", mcVersion, "modLoader", modLoader)
	return err
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

	rows, err := d.Query(`
		SELECT mod_id, name, version, description
		FROM mod_jar_metadata WHERE sha1 = ?
	`, sha1)
	if err != nil {
		logging.Error("get jar metadata failed", "sha1", sha1, "error", err)
		return nil, false
	}
	defer rows.Close()

	var mods []structs.ModInfo
	for rows.Next() {
		var mod structs.ModInfo
		if err := rows.Scan(&mod.ID, &mod.Name, &mod.Version, &mod.Description); err != nil {
			return nil, false
		}
		mods = append(mods, mod)
	}
	if err := rows.Err(); err != nil || len(mods) == 0 {
		if err != nil {
			logging.Error("iterate jar metadata failed", "sha1", sha1, "error", err)
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

	tx, err := d.Begin()
	if err != nil {
		logging.Error("begin set jar metadata transaction failed", "sha1", sha1, "modCount", len(mods), "error", err)
		return err
	}
	defer tx.Rollback()

	_, _ = tx.Exec(`DELETE FROM mod_jar_metadata WHERE sha1 = ?`, sha1)
	seen := make(map[string]struct{}, len(mods))
	for _, mod := range mods {
		modID := strings.TrimSpace(mod.ID)
		if modID == "" {
			continue
		}
		key := strings.ToLower(modID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		_, err := tx.Exec(`
			INSERT INTO mod_jar_metadata (sha1, mod_id, name, version, description)
			VALUES (?, ?, ?, ?, ?)
		`, sha1, modID, mod.Name, mod.Version, mod.Description)
		if err != nil {
			logging.Error("insert jar metadata failed", "sha1", sha1, "modID", modID, "error", err)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		logging.Error("commit jar metadata failed", "sha1", sha1, "modCount", len(mods), "error", err)
		return err
	}
	logging.Info("jar metadata set", "sha1", sha1, "modCount", len(mods))
	return nil
}
