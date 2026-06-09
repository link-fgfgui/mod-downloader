package structs

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

type SearchModResult struct {
	ID          string `json:"id"`
	Platform    string `json:"platform"`
	Title       string `json:"title"`
	Icon        string `json:"icon"`
	IconURL     string `json:"iconUrl"`
	Description string `json:"description"`
	Downloads   int64  `json:"downloads"`
	Slug        string `json:"slug"`
}

type ProjectVersionResult struct {
	ID           string              `json:"id"`
	Platform     string              `json:"platform"`
	ProjectID    string              `json:"projectId"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	FileName     string              `json:"fileName"`
	DownloadURL  string              `json:"downloadUrl"`
	SHA1         string              `json:"sha1"`
	PublishedAt  int64               `json:"publishedAt"`
	Downloads    int64               `json:"downloads"`
	GameVersions []string            `json:"gameVersions"`
	Loaders      []string            `json:"loaders"`
	Dependencies []ProjectDependency `json:"dependencies,omitempty"`
}

type ProjectDependency struct {
	ProjectID string `json:"projectId"`
	VersionID string `json:"versionId,omitempty"`
	Type      string `json:"type,omitempty"`
}

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
	Active  bool `json:"active"`
	Pending int  `json:"pending"`
	Running int  `json:"running"`
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
