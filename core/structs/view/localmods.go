// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package view

import (
	minecraft "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

// LocalModGroup aggregates every physical JAR record that belongs to one mod
// (same fileName, path or id) together with the projections both adapters
// render. Grouping, Jar-in-Jar de-duplication, the search text and the loader
// compatibility verdict are decided here so GUI and TUI cannot drift apart.
type LocalModGroup struct {
	Primary      minecraft.ModInfo      `json:"primary"`
	Strong       []minecraft.ModInfo    `json:"strong"`
	JIJ          []minecraft.JIJModInfo `json:"jij,omitempty"`
	DisplayName  string                 `json:"displayName"`
	Categories   []string               `json:"categories,omitempty"`
	Loaders      []string               `json:"loaders,omitempty"`
	SearchText   string                 `json:"searchText"`
	Incompatible bool                   `json:"incompatible"`
}
