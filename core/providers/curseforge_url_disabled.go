// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

//go:build no_curseforge_cdn_fallback

package providers

const curseForgeCDNFallbackEnabled = false

// CurseForgeDownloadURL is compiled out in strict builds. No CDN URL is
// reconstructed; callers must use a provider-supplied download URL.
func CurseForgeDownloadURL(fileID, fileName string) string {
	return ""
}
