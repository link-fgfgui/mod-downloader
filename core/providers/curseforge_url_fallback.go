// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

//go:build !no_curseforge_cdn_fallback

package providers

import (
	"net/url"
	"strings"
)

const curseForgeCDNFallbackEnabled = true

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
