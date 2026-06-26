package providers

import (
	"testing"
)

func TestProjectKey(t *testing.T) {
	tests := []struct {
		name      string
		platform  string
		projectID string
		want      string
	}{
		{
			name:      "modrinth with slug",
			platform:  "Modrinth",
			projectID: "fabric-api",
			want:      "modrinth:fabric-api",
		},
		{
			name:      "curseforge with numeric id",
			platform:  "CurseForge",
			projectID: "306612",
			want:      "curseforge:306612",
		},
		{
			name:      "empty platform",
			platform:  "",
			projectID: "fabric-api",
			want:      "",
		},
		{
			name:      "empty projectID",
			platform:  "Modrinth",
			projectID: "",
			want:      "",
		},
		{
			name:      "whitespace trimming",
			platform:  "  Modrinth  ",
			projectID: "  fabric-api  ",
			want:      "modrinth:fabric-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProjectKey(tt.platform, tt.projectID)
			if got != tt.want {
				t.Errorf("ProjectKey(%q, %q) = %q, want %q", tt.platform, tt.projectID, got, tt.want)
			}
		})
	}
}

func TestParseProjectKey(t *testing.T) {
	tests := []struct {
		name            string
		id              string
		wantPlatform    string
		wantProjectID   string
	}{
		{
			name:          "modrinth project",
			id:            "modrinth:fabric-api",
			wantPlatform:  "modrinth",
			wantProjectID: "fabric-api",
		},
		{
			name:          "curseforge project",
			id:            "curseforge:306612",
			wantPlatform:  "curseforge",
			wantProjectID: "306612",
		},
		{
			name:          "no colon - bare projectID",
			id:            "fabric-api",
			wantPlatform:  "",
			wantProjectID: "fabric-api",
		},
		{
			name:          "invalid platform prefix",
			id:            "unknown:fabric-api",
			wantPlatform:  "",
			wantProjectID: "unknown:fabric-api",
		},
		{
			name:          "empty string",
			id:            "",
			wantPlatform:  "",
			wantProjectID: "",
		},
		{
			name:          "whitespace trimming",
			id:            "  modrinth:fabric-api  ",
			wantPlatform:  "modrinth",
			wantProjectID: "fabric-api",
		},
		{
			name:          "uppercase platform normalized",
			id:            "Modrinth:fabric-api",
			wantPlatform:  "modrinth",
			wantProjectID: "fabric-api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlatform, gotProjectID := ParseProjectKey(tt.id)
			if gotPlatform != tt.wantPlatform || gotProjectID != tt.wantProjectID {
				t.Errorf("ParseProjectKey(%q) = (%q, %q), want (%q, %q)",
					tt.id, gotPlatform, gotProjectID, tt.wantPlatform, tt.wantProjectID)
			}
		})
	}
}

func TestVersionKey(t *testing.T) {
	tests := []struct {
		name      string
		platform  string
		projectID string
		versionID string
		want      string
	}{
		{
			name:      "modrinth version",
			platform:  "Modrinth",
			projectID: "fabric-api",
			versionID: "0.92.0+1.20.1",
			want:      "modrinth:fabric-api:0.92.0+1.20.1",
		},
		{
			name:      "curseforge version",
			platform:  "CurseForge",
			projectID: "306612",
			versionID: "4950231",
			want:      "curseforge:306612:4950231",
		},
		{
			name:      "empty platform",
			platform:  "",
			projectID: "fabric-api",
			versionID: "0.92.0",
			want:      "",
		},
		{
			name:      "empty versionID",
			platform:  "Modrinth",
			projectID: "fabric-api",
			versionID: "",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VersionKey(tt.platform, tt.projectID, tt.versionID)
			if got != tt.want {
				t.Errorf("VersionKey(%q, %q, %q) = %q, want %q",
					tt.platform, tt.projectID, tt.versionID, got, tt.want)
			}
		})
	}
}

func TestParseVersionKey(t *testing.T) {
	tests := []struct {
		name            string
		id              string
		wantPlatform    string
		wantProjectID   string
		wantVersionID   string
	}{
		{
			name:          "modrinth version",
			id:            "modrinth:fabric-api:0.92.0+1.20.1",
			wantPlatform:  "modrinth",
			wantProjectID: "fabric-api",
			wantVersionID: "0.92.0+1.20.1",
		},
		{
			name:          "curseforge version",
			id:            "curseforge:306612:4950231",
			wantPlatform:  "curseforge",
			wantProjectID: "306612",
			wantVersionID: "4950231",
		},
		{
			name:          "invalid format - only 2 parts",
			id:            "modrinth:fabric-api",
			wantPlatform:  "",
			wantProjectID: "",
			wantVersionID: "",
		},
		{
			name:          "invalid platform",
			id:            "unknown:fabric-api:0.92.0",
			wantPlatform:  "",
			wantProjectID: "",
			wantVersionID: "",
		},
		{
			name:          "empty string",
			id:            "",
			wantPlatform:  "",
			wantProjectID: "",
			wantVersionID: "",
		},
		{
			name:          "version with colon",
			id:            "modrinth:fabric-api:0.92.0:extra",
			wantPlatform:  "modrinth",
			wantProjectID: "fabric-api",
			wantVersionID: "0.92.0:extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlatform, gotProjectID, gotVersionID := ParseVersionKey(tt.id)
			if gotPlatform != tt.wantPlatform || gotProjectID != tt.wantProjectID || gotVersionID != tt.wantVersionID {
				t.Errorf("ParseVersionKey(%q) = (%q, %q, %q), want (%q, %q, %q)",
					tt.id, gotPlatform, gotProjectID, gotVersionID,
					tt.wantPlatform, tt.wantProjectID, tt.wantVersionID)
			}
		})
	}
}

func TestModProjectRoundtrip(t *testing.T) {
	project := ModProject{
		ID:          "modrinth:fabric-api",
		Platform:    "Modrinth",
		ProjectID:   "fabric-api",
		Slug:        "fabric-api",
		Title:       "Fabric API",
		Description: "Essential hooks for modding with Fabric",
		IconURL:     "https://example.com/icon.png",
		Downloads:   1000000,
		UpdatedAt:   1234567890,
	}

	// Parse the ID
	platform, projectID := ParseProjectKey(project.ID)
	if platform != "modrinth" || projectID != "fabric-api" {
		t.Errorf("ParseProjectKey failed: got (%q, %q), want (modrinth, fabric-api)", platform, projectID)
	}

	// Reconstruct the ID
	reconstructed := ProjectKey(platform, projectID)
	if reconstructed != project.ID {
		t.Errorf("ProjectKey roundtrip failed: got %q, want %q", reconstructed, project.ID)
	}
}

func TestModVersionRoundtrip(t *testing.T) {
	version := ModVersion{
		ID:           "0.92.0+1.20.1",
		Platform:     "Modrinth",
		ProjectID:    "fabric-api",
		VersionID:    "0.92.0+1.20.1",
		Name:         "Fabric API 0.92.0+1.20.1",
		Version:      "0.92.0+1.20.1",
		FileName:     "fabric-api-0.92.0+1.20.1.jar",
		DownloadURL:  "https://example.com/download",
		SHA1:         "abc123",
		PublishedAt:  1234567890,
		Downloads:    50000,
		GameVersions: []string{"1.20.1"},
		Loaders:      []string{"fabric"},
	}

	// Build composite key
	key := VersionKey(version.Platform, version.ProjectID, version.ID)
	expected := "modrinth:fabric-api:0.92.0+1.20.1"
	if key != expected {
		t.Errorf("VersionKey failed: got %q, want %q", key, expected)
	}

	// Parse back
	platform, projectID, versionID := ParseVersionKey(key)
	if platform != "modrinth" || projectID != "fabric-api" || versionID != "0.92.0+1.20.1" {
		t.Errorf("ParseVersionKey failed: got (%q, %q, %q), want (modrinth, fabric-api, 0.92.0+1.20.1)",
			platform, projectID, versionID)
	}
}
