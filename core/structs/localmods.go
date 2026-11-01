// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package structs

// LocalModBatchOperationRequest asks the core to enable, disable, delete or
// rename the local JARs at Paths.
type LocalModBatchOperationRequest struct {
	Paths  []string `json:"paths"`
	Action string   `json:"action"`
}

// UnusedDependencyScanRequest scopes an unused-dependency scan. Paths that are
// still enabled should be passed in ExcludedPaths so they are never reported.
type UnusedDependencyScanRequest struct {
	ExcludedPaths []string `json:"excludedPaths,omitempty"`
}

// UnusedDependencyCandidate is a local JAR that appears to exist only to
// satisfy other mods and can be removed.
type UnusedDependencyCandidate struct {
	Path       string   `json:"path"`
	FileName   string   `json:"fileName"`
	ModIDs     []string `json:"modIds"`
	Name       string   `json:"name,omitempty"`
	OnlineName string   `json:"onlineName,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
}

// UnusedDependencyScanResult lists every candidate found by a scan.
type UnusedDependencyScanResult struct {
	Candidates []UnusedDependencyCandidate `json:"candidates"`
}

// LocalModDisableImpactRequest asks which mods would lose a dependency if the
// given JARs were disabled or deleted.
type LocalModDisableImpactRequest struct {
	Paths  []string `json:"paths"`
	Action string   `json:"action"`
}

// LocalModDependencyMod identifies a local mod affected by disabling a
// dependency provider.
type LocalModDependencyMod struct {
	Path     string   `json:"path"`
	FileName string   `json:"fileName,omitempty"`
	Name     string   `json:"name,omitempty"`
	ModIDs   []string `json:"modIds,omitempty"`
}

// CachedDependencyCandidate is a download-cache version that can restore a
// dependency without another network request.
type CachedDependencyCandidate struct {
	ModID     string `json:"modId"`
	Platform  string `json:"platform"`
	ProjectID string `json:"projectId"`
	VersionID string `json:"versionId"`
	Title     string `json:"title,omitempty"`
}

// LocalModDependencyImpact describes what disabling one dependency provider
// would break and which cached versions could restore it.
type LocalModDependencyImpact struct {
	ModID             string                      `json:"modId"`
	AffectedMods      []LocalModDependencyMod     `json:"affectedMods"`
	DisabledProviders []LocalModDependencyMod     `json:"disabledProviders"`
	Candidates        []CachedDependencyCandidate `json:"candidates,omitempty"`
}

// LocalModDisableImpactResult lists the dependency impacts of a batch
// disable/delete operation.
type LocalModDisableImpactResult struct {
	Impacts []LocalModDependencyImpact `json:"impacts"`
}

// RestoreCachedDependencyRequest restores one dependency version from the
// local download cache.
type RestoreCachedDependencyRequest struct {
	ModID     string `json:"modId"`
	Platform  string `json:"platform"`
	ProjectID string `json:"projectId"`
	VersionID string `json:"versionId"`
}
