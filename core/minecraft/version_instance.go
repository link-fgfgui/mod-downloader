// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package minecraft

import (
	"strings"

	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

// VersionInstanceKey returns the stable key for an instance: the version ID
// when present, otherwise its directory/display name.
func VersionInstanceKey(version structs.VersionInfo) string {
	if strings.TrimSpace(version.ID) != "" {
		return strings.TrimSpace(version.ID)
	}
	return strings.TrimSpace(version.Name)
}
