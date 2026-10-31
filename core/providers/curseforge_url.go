package providers

import (
	"net/url"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/models"
)

// CurseForgeDownloadURL reconstructs the public CDN URL used by CurseForge
// when the API omits a file's direct download URL.
func CurseForgeDownloadURL(fileID, fileName string) string {
	fileID = strings.TrimSpace(fileID)
	fileName = strings.TrimSpace(fileName)
	if len(fileID) <= 4 || fileName == "" {
		return ""
	}
	for _, r := range fileID {
		if r < '0' || r > '9' {
			return ""
		}
	}
	return "https://edge.forgecdn.net/files/" + fileID[:4] + "/" + fileID[4:] + "/" + url.PathEscape(fileName)
}

// EffectiveDownloadURL returns the provider URL, reconstructing a missing
// CurseForge CDN URL when the version contains enough identifying fields.
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
