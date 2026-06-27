package providers

import (
	"mod-downloader/database"
	"mod-downloader/logging"
	"mod-downloader/models"
	"strings"
)

func GetProjectByID(id string) (ModProject, bool) {
	platform, projectID := ParseProjectKey(id)
	if platform == "" || projectID == "" {
		logging.Debug("invalid project ID format", "id", id)
		return ModProject{}, false
	}

	return database.GetModPlatform(platform, projectID)
}

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
			return v, true
		}
	}

	return ModVersion{}, false
}

func GetVersionsByProject(platform, projectID string) []ModVersion {
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

func StoreProject(project ModProject) error {
	return database.UpsertModPlatform(project)
}

func StoreVersion(version ModVersion) error {
	platform := strings.TrimSpace(version.Platform)
	projectID := strings.TrimSpace(version.ProjectID)

	if platform == "" || projectID == "" {
		return nil
	}

	return database.SetPlatformVersions(platform, projectID, []models.ModVersion{version})
}

func StoreVersions(platform, projectID string, versions []ModVersion) error {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)

	if platform == "" || projectID == "" || len(versions) == 0 {
		return nil
	}

	return database.SetPlatformVersions(platform, projectID, versions)
}
