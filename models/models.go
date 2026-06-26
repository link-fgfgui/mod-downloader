package models

import (
	"strings"
)

// ModProject represents a mod project on any platform.
// Replaces both SearchModResult (from structs/search.go) and ModPlatform (from database/mods.go).
type ModProject struct {
	ID          string `json:"id"`          // Composite: "platform:projectID" or just projectID
	Platform    string `json:"platform"`    // "CurseForge" | "Modrinth"
	ProjectID   string `json:"projectId"`   // Numeric ID for CF, slug for MR
	Slug        string `json:"slug"`        // URL slug
	Title       string `json:"title"`       // Display name
	Icon        string `json:"icon"`        // Material Design Icon name (e.g. "mdi-package-variant")
	IconURL     string `json:"iconUrl"`     // Avatar/logo URL
	Description string `json:"description"` // Short description
	Downloads   int64  `json:"downloads"`   // Total download count
	UpdatedAt   int64  `json:"updatedAt"`   // Last fetched timestamp (Unix seconds)
}

// ModVersion represents a specific version file.
// Replaces both ProjectVersionResult (from structs/search.go) and ModPlatformVersion (from database/mods.go).
type ModVersion struct {
	ID           string          `json:"id"`           // Platform-specific version ID
	Platform     string          `json:"platform"`     // "CurseForge" | "Modrinth"
	ProjectID    string          `json:"projectId"`    // Parent project ID
	VersionID    string          `json:"versionId"`    // Alias for ID (frontend compatibility)
	Name         string          `json:"name"`         // Display name
	Version      string          `json:"version"`      // Version number string
	FileName     string          `json:"fileName"`     // JAR filename
	DownloadURL  string          `json:"downloadUrl"`  // Direct download link
	SHA1         string          `json:"sha1"`         // File hash (lowercase)
	PublishedAt  int64           `json:"publishedAt"`  // Unix timestamp (seconds)
	Downloads    int64           `json:"downloads"`    // Version-specific download count
	GameVersions []string        `json:"gameVersions"` // e.g. ["1.20.1", "1.20.2"]
	Loaders      []string        `json:"loaders"`      // e.g. ["fabric", "forge"]
	Dependencies []ModDependency `json:"dependencies,omitempty"`
}

// ModDependency represents a dependency link between versions.
// Replaces both ProjectDependency (from structs/search.go) and ModDependency (from database/mods.go).
type ModDependency struct {
	ID                  string `json:"id,omitempty"`              // Internal database ID
	PlatformVersionID   string `json:"platformVersionId,omitempty"` // Parent version ID
	DependencyProjectID string `json:"projectId"`                   // Target project ID (JSON alias for frontend)
	DependencyVersionID string `json:"versionId,omitempty"`         // Target version ID (JSON alias for frontend)
	DependencyType      string `json:"type,omitempty"`              // "required" | "optional" | "embedded" | etc (JSON alias for frontend)
}

// ProjectKey generates a composite project ID from platform and projectID.
// Format: "platform:projectID" (e.g. "modrinth:fabric-api" or "curseforge:306612")
func ProjectKey(platform, projectID string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	projectID = strings.TrimSpace(projectID)
	if platform == "" || projectID == "" {
		return ""
	}
	return platform + ":" + projectID
}

// ParseProjectKey parses a composite project ID into platform and projectID.
// Input: "platform:projectID" or just "projectID"
// Returns: (platform, projectID)
func ParseProjectKey(id string) (string, string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", ""
	}

	platform, projectID, found := strings.Cut(id, ":")
	if !found {
		// No colon: treat entire string as projectID with unknown platform
		return "", id
	}

	platform = strings.ToLower(strings.TrimSpace(platform))
	projectID = strings.TrimSpace(projectID)

	// Validate platform
	if platform != "curseforge" && platform != "modrinth" {
		// Invalid platform prefix: treat entire string as projectID
		return "", id
	}

	return platform, projectID
}

// VersionKey generates a composite version ID from platform, projectID, and versionID.
// Format: "platform:projectID:versionID"
func VersionKey(platform, projectID, versionID string) string {
	platform = strings.ToLower(strings.TrimSpace(platform))
	projectID = strings.TrimSpace(projectID)
	versionID = strings.TrimSpace(versionID)
	if platform == "" || projectID == "" || versionID == "" {
		return ""
	}
	return platform + ":" + projectID + ":" + versionID
}

// ParseVersionKey parses a composite version ID.
// Input: "platform:projectID:versionID"
// Returns: (platform, projectID, versionID)
func ParseVersionKey(id string) (string, string, string) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", "", ""
	}

	parts := strings.SplitN(id, ":", 3)
	if len(parts) != 3 {
		return "", "", ""
	}

	platform := strings.ToLower(strings.TrimSpace(parts[0]))
	projectID := strings.TrimSpace(parts[1])
	versionID := strings.TrimSpace(parts[2])

	// Validate platform
	if platform != "curseforge" && platform != "modrinth" {
		return "", "", ""
	}

	return platform, projectID, versionID
}
