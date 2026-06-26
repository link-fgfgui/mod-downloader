package providers

import (
	"mod-downloader/database"
	"mod-downloader/logging"
	"strings"
)

// GetProjectByID fetches a ModProject from the database cache by composite ID.
// ID format: "platform:projectID" (e.g. "modrinth:fabric-api" or "curseforge:306612")
func GetProjectByID(id string) (ModProject, bool) {
	platform, projectID := ParseProjectKey(id)
	if platform == "" || projectID == "" {
		logging.Debug("invalid project ID format", "id", id)
		return ModProject{}, false
	}

	dbProject, ok := database.GetModPlatform(platform, projectID)
	if !ok {
		return ModProject{}, false
	}

	return modPlatformToModProject(dbProject), true
}

// GetProjectsByIDs batch fetches multiple ModProjects by their composite IDs.
// Returns a slice in the same order as input IDs; missing projects are omitted.
func GetProjectsByIDs(ids []string) []ModProject {
	if len(ids) == 0 {
		return nil
	}

	results := make([]ModProject, 0, len(ids))
	for _, id := range ids {
		if project, ok := GetProjectByID(id); ok {
			results = append(results, project)
		}
	}
	return results
}

// GetVersionByID fetches a ModVersion from the database cache.
func GetVersionByID(platform, projectID, versionID string) (ModVersion, bool) {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	versionID = strings.TrimSpace(versionID)

	if platform == "" || projectID == "" || versionID == "" {
		return ModVersion{}, false
	}

	versions, err := database.GetPlatformVersions(platform, projectID)
	if err != nil || len(versions) == 0 {
		return ModVersion{}, false
	}

	for _, v := range versions {
		if v.VersionID == versionID {
			return modPlatformVersionToModVersion(v), true
		}
	}

	return ModVersion{}, false
}

// GetVersionsByProject fetches all ModVersions for a given project.
func GetVersionsByProject(platform, projectID string) []ModVersion {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)

	if platform == "" || projectID == "" {
		return nil
	}

	dbVersions, err := database.GetPlatformVersions(platform, projectID)
	if err != nil || len(dbVersions) == 0 {
		return nil
	}

	results := make([]ModVersion, 0, len(dbVersions))
	for _, dbV := range dbVersions {
		results = append(results, modPlatformVersionToModVersion(dbV))
	}
	return results
}

// StoreProject saves a ModProject to the database cache.
func StoreProject(project ModProject) error {
	dbProject := modProjectToModPlatform(project)
	return database.UpsertModPlatform(dbProject)
}

// StoreVersion saves a ModVersion to the database cache.
func StoreVersion(version ModVersion) error {
	platform := strings.TrimSpace(version.Platform)
	projectID := strings.TrimSpace(version.ProjectID)

	if platform == "" || projectID == "" {
		return nil
	}

	dbVersion := modVersionToModPlatformVersion(version)
	return database.SetPlatformVersions(platform, projectID, []database.ModPlatformVersion{dbVersion})
}

// StoreVersions saves multiple ModVersions for a project to the database cache.
func StoreVersions(platform, projectID string, versions []ModVersion) error {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)

	if platform == "" || projectID == "" || len(versions) == 0 {
		return nil
	}

	dbVersions := make([]database.ModPlatformVersion, 0, len(versions))
	for _, v := range versions {
		dbVersions = append(dbVersions, modVersionToModPlatformVersion(v))
	}

	return database.SetPlatformVersions(platform, projectID, dbVersions)
}

// --- Conversion functions ---

func modPlatformToModProject(dbProject database.ModPlatform) ModProject {
	// Icon is derived from platform
	icon := "mdi-package-variant"
	if dbProject.Platform == "Modrinth" {
		icon = "mdi-leaf"
	}

	return ModProject{
		ID:          ProjectKey(dbProject.Platform, dbProject.ProjectID),
		Platform:    dbProject.Platform,
		ProjectID:   dbProject.ProjectID,
		Slug:        dbProject.Slug,
		Title:       dbProject.Name,
		Icon:        icon,
		IconURL:     dbProject.McmodURL, // Reuse McmodURL field for IconURL
		Description: dbProject.Description,
		Downloads:   0, // Not stored in old structure
		UpdatedAt:   dbProject.UpdatedAt,
	}
}

func modProjectToModPlatform(project ModProject) database.ModPlatform {
	return database.ModPlatform{
		Platform:    strings.TrimSpace(project.Platform),
		ProjectID:   strings.TrimSpace(project.ProjectID),
		Slug:        project.Slug,
		Name:        project.Title,
		Description: project.Description,
		McmodURL:    project.IconURL, // Store IconURL in McmodURL field
		UpdatedAt:   project.UpdatedAt,
	}
}

func modPlatformVersionToModVersion(dbVersion database.ModPlatformVersion) ModVersion {
	deps := make([]ModDependency, 0, len(dbVersion.Dependencies))
	for _, dbDep := range dbVersion.Dependencies {
		deps = append(deps, ModDependency{
			ID:                  dbDep.ID,
			PlatformVersionID:   dbDep.PlatformVersionID,
			DependencyProjectID: dbDep.DependencyProjectID,
			DependencyVersionID: dbDep.DependencyVersionID,
			DependencyType:      dbDep.DependencyType,
		})
	}

	return ModVersion{
		ID:           dbVersion.VersionID,
		Platform:     dbVersion.Platform,
		ProjectID:    dbVersion.ProjectID,
		VersionID:    dbVersion.VersionID,
		Name:         dbVersion.Name,
		Version:      dbVersion.Version,
		FileName:     dbVersion.FileName,
		DownloadURL:  dbVersion.DownloadURL,
		SHA1:         dbVersion.SHA1,
		PublishedAt:  dbVersion.PublishedAt,
		Downloads:    dbVersion.Downloads,
		GameVersions: append([]string(nil), dbVersion.GameVersions...),
		Loaders:      append([]string(nil), dbVersion.Loaders...),
		Dependencies: deps,
	}
}

func modVersionToModPlatformVersion(version ModVersion) database.ModPlatformVersion {
	deps := make([]database.ModDependency, 0, len(version.Dependencies))
	for _, dep := range version.Dependencies {
		deps = append(deps, database.ModDependency{
			ID:                  dep.ID,
			PlatformVersionID:   dep.PlatformVersionID,
			DependencyProjectID: dep.DependencyProjectID,
			DependencyVersionID: dep.DependencyVersionID,
			DependencyType:      dep.DependencyType,
		})
	}

	return database.ModPlatformVersion{
		ID:           version.ID,
		Platform:     strings.TrimSpace(version.Platform),
		ProjectID:    strings.TrimSpace(version.ProjectID),
		VersionID:    strings.TrimSpace(version.VersionID),
		Name:         version.Name,
		Version:      version.Version,
		FileName:     version.FileName,
		DownloadURL:  version.DownloadURL,
		SHA1:         version.SHA1,
		PublishedAt:  version.PublishedAt,
		Downloads:    version.Downloads,
		GameVersions: append([]string(nil), version.GameVersions...),
		Loaders:      append([]string(nil), version.Loaders...),
		Dependencies: deps,
	}
}
