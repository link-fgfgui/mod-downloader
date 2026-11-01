// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package providers integrates the Modrinth and CurseForge APIs behind a
// single provider interface and persists their metadata to the cache.
package providers

import (
	"github.com/link-fgfgui/mod-downloader/core/logging"
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	"strings"
)

// GetProjectByKey returns the cached project for a "platform:projectID"
// composite key.
func GetProjectByKey(id string) (models.ModProject, bool) {
	platform, projectID := models.ParseProjectKey(id)
	if platform == "" || projectID == "" {
		logging.Debug("invalid project ID format", "id", id)
		return models.ModProject{}, false
	}

	return storage.GetModProject(platform, projectID)
}

// GetProjectsByKeys returns every cached project matching the given composite
// keys, preserving input order and skipping misses.
func GetProjectsByKeys(ids []string) []models.ModProject {
	if len(ids) == 0 {
		return nil
	}

	results := make([]models.ModProject, 0, len(ids))
	for _, id := range ids {
		if project, ok := GetProjectByKey(id); ok {
			results = append(results, project)
		}
	}
	return results
}

// GetVersionByID returns one cached platform version by its platform version
// ID.
func GetVersionByID(platform, projectID, versionID string) (models.ModVersion, bool) {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)
	versionID = strings.TrimSpace(versionID)

	if platform == "" || projectID == "" || versionID == "" {
		return models.ModVersion{}, false
	}

	versions, err := storage.GetPlatformVersions(platform, projectID)
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

// GetVersionsByProject returns the cached versions of one platform project.
func GetVersionsByProject(platform, projectID string) []models.ModVersion {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)

	if platform == "" || projectID == "" {
		return nil
	}

	versions, err := storage.GetPlatformVersions(platform, projectID)
	if err != nil || len(versions) == 0 {
		return nil
	}

	return versions
}

// StoreProject caches one platform project.
func StoreProject(project models.ModProject) error {
	return storage.UpsertModProject(project)
}

// StoreVersion caches one platform version under its project.
func StoreVersion(version models.ModVersion) error {
	platform := strings.TrimSpace(version.Platform)
	projectID := strings.TrimSpace(version.ProjectID)

	if platform == "" || projectID == "" {
		return nil
	}

	return storage.SetPlatformVersions(platform, projectID, []models.ModVersion{version})
}

// StoreVersions replaces the cached version list of one platform project.
func StoreVersions(platform, projectID string, versions []models.ModVersion) error {
	platform = strings.TrimSpace(platform)
	projectID = strings.TrimSpace(projectID)

	if platform == "" || projectID == "" || len(versions) == 0 {
		return nil
	}

	return storage.SetPlatformVersions(platform, projectID, versions)
}
