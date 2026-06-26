package structs

import "mod-downloader/models"

type SearchModsRequest struct {
	RequestID string `json:"requestId"`
	Query     string `json:"query"`
	Version   string `json:"version"`
	ModLoader string `json:"modLoader"`
	Offset    int    `json:"offset"`
	Limit     int    `json:"limit"`
}

type SearchModsUpdate struct {
	RequestID string            `json:"requestId"`
	Results   []SearchModResult `json:"results"`
	Loading   bool              `json:"loading"`
	Append    bool              `json:"append"`
}

// SearchModResult is now an alias to the unified ModProject type
type SearchModResult = models.ModProject

// ProjectVersionResult is now an alias to the unified ModVersion type
type ProjectVersionResult = models.ModVersion

// ProjectDependency is now an alias to the unified ModDependency type
type ProjectDependency = models.ModDependency

type ModVersionPinRequest struct {
	Platform         string `json:"platform"`
	ModID            string `json:"modId"`
	VersionID        string `json:"versionId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
}

type ModDownloadRequest struct {
	ProjectID        string          `json:"projectId"`
	Result           SearchModResult `json:"result"`
	MinecraftVersion string          `json:"minecraftVersion"`
	ModLoader        string          `json:"modLoader"`
}

type ModDownloadResult struct {
	Queued    bool   `json:"queued"`
	Skipped   bool   `json:"skipped"`
	Reason    string `json:"reason"`
	FileName  string `json:"fileName"`
	VersionID string `json:"versionId"`
}

type DownloadQueueState struct {
	Active  bool                `json:"active"`
	Pending int                 `json:"pending"`
	Running int                 `json:"running"`
	Items   []DownloadQueueItem `json:"items,omitempty"`
}

type DownloadQueueItem struct {
	ID               string `json:"id"`
	Status           string `json:"status"`
	Title            string `json:"title"`
	FileName         string `json:"fileName"`
	VersionID        string `json:"versionId"`
	Platform         string `json:"platform"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
	Cancelable       bool   `json:"cancelable"`
}

type DownloadFailedEvent struct {
	FileName  string `json:"fileName"`
	VersionID string `json:"versionId"`
	Reason    string `json:"reason"`
}

type DownloadStatesRequest struct {
	Results          []SearchModResult `json:"results"`
	MinecraftVersion string            `json:"minecraftVersion"`
	ModLoader        string            `json:"modLoader"`
}

type ModDownloadButtonState struct {
	Key      string `json:"key"`
	Status   string `json:"status"`
	Disabled bool   `json:"disabled"`
	Icon     string `json:"icon"`
	Color    string `json:"color"`
}
