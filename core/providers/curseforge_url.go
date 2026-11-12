// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package providers

import (
	"net/url"
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/models"
)

// EffectiveDownloadURL returns the provider URL. When the CurseForge CDN
// fallback is compiled in, it reconstructs a missing CurseForge URL from the
// version's file ID and filename.
func EffectiveDownloadURL(version models.ModVersion) string {
	if downloadURL := strings.TrimSpace(version.DownloadURL); downloadURL != "" {
		return downloadURL
	}
	if !strings.EqualFold(strings.TrimSpace(version.Platform), "curseforge") {
		return ""
	}
	fileID := strings.TrimSpace(version.ID)
	if fileID == "" {
		fileID = strings.TrimSpace(version.VersionID)
	}
	return CurseForgeDownloadURL(fileID, version.FileName)
}

// VersionDownloadAvailable reports whether a version can be downloaded in the
// current build. With the CDN fallback compiled in, every version is considered
// available because EffectiveDownloadURL may reconstruct a CurseForge URL.
// Strict builds require a provider-supplied URL.
func VersionDownloadAvailable(version models.ModVersion) bool {
	if curseForgeCDNFallbackEnabled {
		return true
	}
	return strings.TrimSpace(EffectiveDownloadURL(version)) != ""
}

// ProjectHasDownloadableVersion reports whether a project has at least one
// version matching the requested scope that can be downloaded in this build.
// It returns true immediately when the CDN fallback is compiled in, preserving
// the normal search behavior without extra metadata requests.
func ProjectHasDownloadableVersion(result models.ModProject, minecraftVersion, modLoader string) bool {
	if curseForgeCDNFallbackEnabled {
		return true
	}
	for _, version := range ListMatchingProjectVersions(result, minecraftVersion, modLoader) {
		if VersionDownloadAvailable(version) {
			return true
		}
	}
	return false
}

// PreferDownloadableVersions filters out versions that the current build cannot
// download, such as CurseForge files without a provider URL in strict mode. When
// no version is downloadable the input is returned unchanged so callers can
// report the missing-URL reason instead of a generic "no matching version".
func PreferDownloadableVersions(versions []models.ModVersion) []models.ModVersion {
	downloadable := make([]models.ModVersion, 0, len(versions))
	for _, version := range versions {
		if VersionDownloadAvailable(version) {
			downloadable = append(downloadable, version)
		}
	}
	if len(downloadable) == 0 {
		return versions
	}
	return downloadable
}

// CurseForgeDownloadPageURL returns the official CurseForge file page used for
// manual browser downloads. The page is not a CDN URL; CurseForge redirects it
// to the CDN when the file is available.
func CurseForgeDownloadPageURL(slug, projectID, fileID string) string {
	fileID = strings.TrimSpace(fileID)
	if fileID == "" {
		return ""
	}
	if slug = strings.TrimSpace(slug); slug != "" {
		return "https://www.curseforge.com/minecraft/mc-mods/" + url.PathEscape(slug) + "/download/" + url.PathEscape(fileID)
	}
	if projectID = strings.TrimSpace(projectID); projectID != "" {
		return "https://www.curseforge.com/projects/" + url.PathEscape(projectID) + "/download/" + url.PathEscape(fileID)
	}
	return ""
}
