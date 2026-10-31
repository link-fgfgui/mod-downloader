package structs

import "github.com/link-fgfgui/mod-downloader-core/models"

type SearchModsRequest struct {
	RequestID string `json:"requestId"`
	Query     string `json:"query"`
	Version   string `json:"version"`
	ModLoader string `json:"modLoader"`
	Offset    int    `json:"offset"`
	Limit     int    `json:"limit"`
}

type SearchModsUpdate struct {
	RequestID string              `json:"requestId"`
	Results   []models.ModProject `json:"results"`
	Loading   bool                `json:"loading"`
	Append    bool                `json:"append"`
}

type ModVersionPinRequest struct {
	Platform         string `json:"platform"`
	ModID            string `json:"modId"`
	VersionID        string `json:"versionId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
}

type ModDownloadRequest struct {
	ProjectID        string            `json:"projectId"`
	Result           models.ModProject `json:"result"`
	MinecraftVersion string            `json:"minecraftVersion"`
	ModLoader        string            `json:"modLoader"`
	VersionID        string            `json:"versionId,omitempty"` // explicit platform version ID; empty means "best match"
	TargetDir        string            `json:"targetDir,omitempty"`
	InstanceID       string            `json:"instanceId,omitempty"`
}

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

type DownloadFailedEvent struct {
	FileName  string `json:"fileName"`
	VersionID string `json:"versionId"`
	Reason    string `json:"reason"`
}

type OptionalDependencyReminder struct {
	ID               string                        `json:"id"`
	MainProjectKey   string                        `json:"mainProjectKey"`
	MainTitle        string                        `json:"mainTitle"`
	MainVersionID    string                        `json:"mainVersionId"`
	MinecraftVersion string                        `json:"minecraftVersion"`
	ModLoader        string                        `json:"modLoader"`
	Dependencies     []OptionalDependencyCandidate `json:"dependencies,omitempty"`
}

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

type IncompatibleLocalMod struct {
	Path     string `json:"path"`
	FileName string `json:"fileName"`
	SHA1     string `json:"sha1"`
}

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

type DependencyPreflightEntry struct {
	ProjectID       string `json:"projectId"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	RequiredVersion string `json:"requiredVersion,omitempty"`
	LocalName       string `json:"localName,omitempty"`
	LocalVersion    string `json:"localVersion,omitempty"`
}

type DependencyPreflightResult struct {
	Entries   []DependencyPreflightEntry `json:"entries"`
	Fulfilled int                        `json:"fulfilled"`
	Missing   int                        `json:"missing"`
	Conflicts int                        `json:"conflicts"`
}

type BatchDownloadRequest struct {
	Results          []models.ModProject `json:"results"`
	MinecraftVersion string              `json:"minecraftVersion"`
	ModLoader        string              `json:"modLoader"`
	TargetDir        string              `json:"targetDir,omitempty"`
	InstanceID       string              `json:"instanceId,omitempty"`
}

type BatchIncompatibleConflict struct {
	Key       string                 `json:"key"`
	Title     string                 `json:"title"`
	Result    models.ModProject      `json:"result"`
	Conflicts []IncompatibleConflict `json:"conflicts,omitempty"`
}

type BatchIncompatibleAnalysis struct {
	Conflicts []BatchIncompatibleConflict `json:"conflicts,omitempty"`
}

type DownloadStatesRequest struct {
	Results          []models.ModProject `json:"results"`
	MinecraftVersion string              `json:"minecraftVersion"`
	ModLoader        string              `json:"modLoader"`
	TargetDir        string              `json:"targetDir,omitempty"`
	InstanceID       string              `json:"instanceId,omitempty"`
}

type ModDownloadButtonState struct {
	Key              string `json:"key"`
	Status           string `json:"status"`
	ConflictFileName string `json:"conflictFileName,omitempty"`
	Disabled         bool   `json:"disabled"`
	Loading          bool   `json:"loading"`
	Icon             string `json:"icon"`
	Color            string `json:"color"`
}
