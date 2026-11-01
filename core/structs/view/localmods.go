// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package view

import (
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/modgraph"
	minecraft "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

// LocalModGroup aggregates every physical JAR record that belongs to one mod
// (same fileName, path or id) together with the projections both adapters
// render. Grouping, Jar-in-Jar de-duplication, the search text and the loader
// compatibility verdict are decided here so GUI and TUI cannot drift apart.
type LocalModGroup struct {
	Primary           minecraft.ModInfo      `json:"primary"`
	Strong            []minecraft.ModInfo    `json:"strong"`
	JIJ               []minecraft.JIJModInfo `json:"jij,omitempty"`
	DisplayName       string                 `json:"displayName"`
	Categories        []string               `json:"categories,omitempty"`
	Loaders           []string               `json:"loaders,omitempty"`
	SearchText        string                 `json:"searchText"`
	Incompatible      bool                   `json:"incompatible"`
	RowKey            string                 `json:"rowKey"`
	CanReplaceVersion bool                   `json:"canReplaceVersion"`
	OnlineProject     models.ModProject      `json:"onlineProject"`
}

// LocalModView is the full Manage projection for one instance: grouped rows,
// the dependency forest and the reverse-dependent index keyed by group row key.
type LocalModView struct {
	Groups     []LocalModGroup                              `json:"groups"`
	Graph      modgraph.Graph                               `json:"graph"`
	Dependents map[string][]modgraph.ReverseDependencyMatch `json:"dependents,omitempty"`
}
