// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package view

import "github.com/link-fgfgui/mod-downloader/core/models"

// SearchModsUpdate is one streamed search page. Append tells the adapter
// whether to replace or extend the current result list.
type SearchModsUpdate struct {
	RequestID string              `json:"requestId"`
	Results   []models.ModProject `json:"results"`
	Loading   bool                `json:"loading"`
	Append    bool                `json:"append"`
}

// DownloadQueueState is the aggregate download queue snapshot rendered by the
// adapters.
type DownloadQueueState struct {
	Active            bool                         `json:"active"`
	Pending           int                          `json:"pending"`
	Running           int                          `json:"running"`
	MessageCount      int                          `json:"messageCount"`
	BytesComplete     int64                        `json:"bytesComplete"`
	TotalBytes        int64                        `json:"totalBytes"`
	BytesPerSecond    int64                        `json:"bytesPerSecond"`
	Items             []DownloadQueueItem          `json:"items,omitempty"`
	OptionalReminders []OptionalDependencyReminder `json:"optionalReminders,omitempty"`
}

// DownloadQueueItem is one visible row in the download queue.
type DownloadQueueItem struct {
	ID               string `json:"id"`
	ProjectID        string `json:"projectId,omitempty"`
	Status           string `json:"status"`
	Title            string `json:"title"`
	FileName         string `json:"fileName"`
	VersionID        string `json:"versionId"`
	Platform         string `json:"platform"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
	Cancelable       bool   `json:"cancelable"`
	Retryable        bool   `json:"retryable"`
	Reason           string `json:"reason,omitempty"`
	BytesComplete    int64  `json:"bytesComplete"`
	TotalBytes       int64  `json:"totalBytes"`
}

// OptionalDependencyReminder lists the optional dependencies found for one
// installed mod so the user can install them later.
type OptionalDependencyReminder struct {
	ID               string                        `json:"id"`
	MainProjectKey   string                        `json:"mainProjectKey"`
	MainTitle        string                        `json:"mainTitle"`
	MainVersionID    string                        `json:"mainVersionId"`
	MinecraftVersion string                        `json:"minecraftVersion"`
	ModLoader        string                        `json:"modLoader"`
	Dependencies     []OptionalDependencyCandidate `json:"dependencies,omitempty"`
}

// OptionalDependencyCandidate is one optional dependency offered by a
// reminder, including whether it can currently be installed.
type OptionalDependencyCandidate struct {
	ProjectKey string `json:"projectKey"`
	Platform   string `json:"platform"`
	ProjectID  string `json:"projectId"`
	Title      string `json:"title"`
	VersionID  string `json:"versionId,omitempty"`
	Status     string `json:"status"`
	Disabled   bool   `json:"disabled"`
	Reason     string `json:"reason,omitempty"`
}

// ModDownloadButtonState is the projected download button state for one search
// result.
type ModDownloadButtonState struct {
	Key              string `json:"key"`
	Status           string `json:"status"`
	ConflictFileName string `json:"conflictFileName,omitempty"`
	Disabled         bool   `json:"disabled"`
	Loading          bool   `json:"loading"`
	Icon             string `json:"icon"`
	Color            string `json:"color"`
}
