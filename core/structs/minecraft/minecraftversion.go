// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package structs defines the Minecraft version, manifest and mod projections
// shared by the GUI and TUI adapters.
package structs

// Patch is one entry in a Minecraft version JSON's patches array (used by
// Forge and NeoForge to record their loader version).
type Patch struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// MinecraftVersion maps the top-level structure of .minecraft/versions/<id>/<id>.json.
type MinecraftVersion struct {
	Name         string  `json:"name"`
	ID           string  `json:"id"`
	InheritsFrom string  `json:"inheritsFrom"`
	Jar          string  `json:"jar"`
	Patches      []Patch `json:"patches,omitempty"`
}

// VersionInfo is the adapter-facing projection of one Minecraft instance,
// including its resolved loader and scanned mods.
type VersionInfo struct {
	Name               string    `json:"name"`
	ID                 string    `json:"id"`
	MinecraftVersion   string    `json:"minecraftVersion"`
	ModLoader          string    `json:"modLoader"`
	ActualModLoader    string    `json:"actualModLoader,omitempty"`
	ConnectorAvailable bool      `json:"connectorAvailable,omitempty"`
	ConnectorVirtual   bool      `json:"connectorVirtual,omitempty"`
	Mods               []ModInfo `json:"mods,omitempty"`
}
