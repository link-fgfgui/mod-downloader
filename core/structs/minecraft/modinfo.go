// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package structs

// JIJModInfo identifies a Jar-in-Jar mod bundled inside another mod JAR.
type JIJModInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// LocalModDependency is a dependency declared by a local mod's metadata.
type LocalModDependency struct {
	ModID   string `json:"modId"`
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

// ModInfo is the normalized metadata parsed from one local mod JAR. Online*
// fields are filled in later when platform metadata is resolved.
type ModInfo struct {
	ID                    string               `json:"id"`
	Name                  string               `json:"name"`
	Version               string               `json:"version"`
	Description           string               `json:"description"`
	FileName              string               `json:"fileName"`
	Path                  string               `json:"path"`
	SHA1                  string               `json:"sha1,omitempty"`
	CurseForgeFingerprint uint32               `json:"cfFingerprint,omitempty"`
	OnlineName            string               `json:"onlineName,omitempty"`
	OnlinePlatform        string               `json:"onlinePlatform,omitempty"`
	OnlineProjectID       string               `json:"onlineProjectId,omitempty"`
	OnlineSlug            string               `json:"onlineSlug,omitempty"`
	OnlineVersion         string               `json:"onlineVersion,omitempty"`
	OnlineVersionID       string               `json:"onlineVersionId,omitempty"`
	OnlineFileName        string               `json:"onlineFileName,omitempty"`
	OnlineMetadataLoading bool                 `json:"onlineMetadataLoading,omitempty"`
	IconURL               string               `json:"iconUrl,omitempty"`
	Categories            []string             `json:"categories,omitempty"`
	Loaders               []string             `json:"loaders,omitempty"`
	Authors               []string             `json:"authors,omitempty"`
	License               string               `json:"license,omitempty"`
	Environment           string               `json:"environment,omitempty"`
	MinecraftVersion      string               `json:"minecraftVersion,omitempty"`
	Provides              []string             `json:"provides,omitempty"`
	Enabled               bool                 `json:"enabled"`
	JIJMods               []JIJModInfo         `json:"jijMods,omitempty"`
	Dependencies          []LocalModDependency `json:"dependencies,omitempty"`
	Recommends            []LocalModDependency `json:"recommends,omitempty"`
	Breaks                []LocalModDependency `json:"breaks,omitempty"`
}
