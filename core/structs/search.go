// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package structs

import "github.com/link-fgfgui/mod-downloader/core/models"

// SearchModsRequest is the adapter-independent search input. Empty Version or
// ModLoader means "any".
type SearchModsRequest struct {
	RequestID string `json:"requestId"`
	Query     string `json:"query"`
	Version   string `json:"version"`
	ModLoader string `json:"modLoader"`
	Offset    int    `json:"offset"`
	Limit     int    `json:"limit"`
}

// ModVersionPinRequest pins one platform version for a mod in a specific
// Minecraft version/loader scope.
type ModVersionPinRequest struct {
	Platform         string `json:"platform"`
	ModID            string `json:"modId"`
	VersionID        string `json:"versionId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
}

// ModDownloadRequest queues one mod (and its required dependencies) for
// install into TargetDir, or into the selected instance when TargetDir is
// empty.
type ModDownloadRequest struct {
	ProjectID        string            `json:"projectId"`
	Result           models.ModProject `json:"result"`
	MinecraftVersion string            `json:"minecraftVersion"`
	ModLoader        string            `json:"modLoader"`
	VersionID        string            `json:"versionId,omitempty"` // explicit platform version ID; empty means "best match"
	TargetDir        string            `json:"targetDir,omitempty"`
	InstanceID       string            `json:"instanceId,omitempty"`
}

// ModDownloadResult reports whether a queue request was accepted, skipped or
// rejected, and identifies the created queue entry.
type ModDownloadResult struct {
	Queued    bool   `json:"queued"`
	Skipped   bool   `json:"skipped"`
	Reason    string `json:"reason"`
	FileName  string `json:"fileName"`
	VersionID string `json:"versionId"`
	// QueueID identifies the queue entry created by this request so callers can
	// wait for or cancel exactly this download instead of the whole queue.
	QueueID string `json:"queueId,omitempty"`
}

// DownloadFailedEvent is emitted when one queue entry fails permanently.
type DownloadFailedEvent struct {
	FileName  string `json:"fileName"`
	VersionID string `json:"versionId"`
	Reason    string `json:"reason"`
}

// IncompatibleLocalMod identifies a local JAR that declares a conflict with a
// project about to be installed.
type IncompatibleLocalMod struct {
	Path     string `json:"path"`
	FileName string `json:"fileName"`
	SHA1     string `json:"sha1"`
}

// IncompatibleConflict pairs a requested project with the local mods that
// declare it incompatible.
type IncompatibleConflict struct {
	ProjectKey             string                 `json:"projectKey"`
	ProjectTitle           string                 `json:"projectTitle"`
	IncompatibleProjectKey string                 `json:"incompatibleProjectKey"`
	IncompatibleTitle      string                 `json:"incompatibleTitle"`
	IncompatibleVersionID  string                 `json:"incompatibleVersionId,omitempty"`
	Paths                  []IncompatibleLocalMod `json:"paths,omitempty"`
}

// DependencyPreflightRequest asks for a local dependency-fulfillment snapshot
// before a mod is installed.
type DependencyPreflightRequest struct {
	ProjectID        string            `json:"projectId"`
	Result           models.ModProject `json:"result"`
	MinecraftVersion string            `json:"minecraftVersion"`
	ModLoader        string            `json:"modLoader"`
	VersionID        string            `json:"versionId,omitempty"`
}

// DependencyPreflightStatus values describe how a required dependency resolves.
const (
	DependencyPreflightFulfilled = "fulfilled"
	DependencyPreflightMissing   = "missing"
	DependencyPreflightConflict  = "conflict"
)

// DependencyPreflightEntry is one dependency row in a preflight result.
type DependencyPreflightEntry struct {
	ProjectID       string `json:"projectId"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	RequiredVersion string `json:"requiredVersion,omitempty"`
	LocalName       string `json:"localName,omitempty"`
	LocalVersion    string `json:"localVersion,omitempty"`
}

// DependencyPreflightResult summarizes how many required dependencies are
// already fulfilled, missing, or blocked by a conflict.
type DependencyPreflightResult struct {
	Entries   []DependencyPreflightEntry `json:"entries"`
	Fulfilled int                        `json:"fulfilled"`
	Missing   int                        `json:"missing"`
	Conflicts int                        `json:"conflicts"`
}

// BatchDownloadRequest queues several projects for the same instance.
type BatchDownloadRequest struct {
	Results          []models.ModProject `json:"results"`
	MinecraftVersion string              `json:"minecraftVersion"`
	ModLoader        string              `json:"modLoader"`
	TargetDir        string              `json:"targetDir,omitempty"`
	InstanceID       string              `json:"instanceId,omitempty"`
}

// BatchIncompatibleConflict groups the conflicts found for one requested
// project in a batch.
type BatchIncompatibleConflict struct {
	Key       string                 `json:"key"`
	Title     string                 `json:"title"`
	Result    models.ModProject      `json:"result"`
	Conflicts []IncompatibleConflict `json:"conflicts,omitempty"`
}

// BatchIncompatibleAnalysis is the result of checking every project in a batch
// for local incompatibilities.
type BatchIncompatibleAnalysis struct {
	Conflicts []BatchIncompatibleConflict `json:"conflicts,omitempty"`
}

// DownloadStatesRequest asks for the per-project download button states of a
// result list.
type DownloadStatesRequest struct {
	Results          []models.ModProject `json:"results"`
	MinecraftVersion string              `json:"minecraftVersion"`
	ModLoader        string              `json:"modLoader"`
	TargetDir        string              `json:"targetDir,omitempty"`
	InstanceID       string              `json:"instanceId,omitempty"`
}
