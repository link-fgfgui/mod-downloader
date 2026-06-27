package providers

import (
	"mod-downloader/database"
	"mod-downloader/logging"
	"mod-downloader/models"
	"strings"
)

func GetProjectByID(id string) (models.ModProject, bool) {
	platform, projectID := models.ParseProjectKey(id)
	if platform == "" || projectID == "" {
		logging.Debug("invalid project ID format", "id", id)
		return models.ModProject{}, false
	}

	return database.GetModPlatform(platform, projectID)
}

func GetProjectsByIDs(ids []string) []models.ModProject {
	if len(ids) == 0 {
		return nil
	}

	results := make([]models.ModProject, 0, len(ids))
	for _, id := range ids {
		if project, ok := GetProjectByID(id); ok {
			results = append(results, project)
		}
	}
	return results
}

func GetVersionByID(platform, projectID, versionID string) (models.ModVersion, bool) {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	versionID = strings.TrimSpace(versionID)

	if platform == "" || projectID == "" || versionID == "" {
		return models.ModVersion{}, false
	}

	versions, err := database.GetPlatformVersions(platform, projectID)
	if err != nil || len(versions) == 0 {
		return models.ModVersion{}, false
	}

	for _, v := range versions {
		if v.VersionID == versionID {
			return v, true
		}
	}

	return models.ModVersion{}, false
}

func GetVersionsByProject(platform, projectID string) []models.ModVersion {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)

	if platform == "" || projectID == "" {
		return nil
	}

	versions, err := database.GetPlatformVersions(platform, projectID)
	if err != nil || len(versions) == 0 {
		return nil
	}

	return versions
}

func StoreProject(project models.ModProject) error {
	return database.UpsertModPlatform(project)
}

func StoreVersion(version models.ModVersion) error {
	platform := strings.TrimSpace(version.Platform)
	projectID := strings.TrimSpace(version.ProjectID)

	if platform == "" || projectID == "" {
		return nil
	}

	return database.SetPlatformVersions(platform, projectID, []models.ModVersion{version})
}

func StoreVersions(platform, projectID string, versions []models.ModVersion) error {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)

	if platform == "" || projectID == "" || len(versions) == 0 {
		return nil
	}

	return database.SetPlatformVersions(platform, projectID, versions)
}
