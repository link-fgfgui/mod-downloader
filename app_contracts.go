// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import appview "github.com/link-fgfgui/mod-downloader/core/structs/view"

// ExportFavoritePackwizResult is adapter-specific because it reports a native
// save-dialog outcome rather than a domain result.
type ExportFavoritePackwizResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

// DirectorySelectionResult reports a native folder-dialog outcome. Cancel and
// invalid selections stay separate so the UI can remain silent on cancel and
// explain why a chosen folder could not be used.
type DirectorySelectionResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
	Invalid  bool   `json:"invalid"`
}

// CacheDirSelectionResult reports a native cache-folder selection together
// with the settings snapshot. Cancel is explicit, and a failed cache switch is
// carried in Error instead of being indistinguishable from cancel.
type CacheDirSelectionResult struct {
	Settings appview.SettingsView `json:"settings"`
	Canceled bool                 `json:"canceled"`
	Error    string               `json:"error,omitempty"`
}
