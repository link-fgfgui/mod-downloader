package structs

type LocalModBatchOperationRequest struct {
	Paths  []string `json:"paths"`
	Action string   `json:"action"`
}

type UnusedDependencyScanRequest struct {
	ExcludedPaths []string `json:"excludedPaths,omitempty"`
}

type UnusedDependencyCandidate struct {
	Path       string   `json:"path"`
	FileName   string   `json:"fileName"`
	ModIDs     []string `json:"modIds"`
	Name       string   `json:"name,omitempty"`
	OnlineName string   `json:"onlineName,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
}

type UnusedDependencyScanResult struct {
	Candidates []UnusedDependencyCandidate `json:"candidates"`
}

type LocalModDisableImpactRequest struct {
	Paths  []string `json:"paths"`
	Action string   `json:"action"`
}

type LocalModDependencyMod struct {
	Path     string   `json:"path"`
	FileName string   `json:"fileName,omitempty"`
	Name     string   `json:"name,omitempty"`
	ModIDs   []string `json:"modIds,omitempty"`
}

type CachedDependencyCandidate struct {
	ModID     string `json:"modId"`
	Platform  string `json:"platform"`
	ProjectID string `json:"projectId"`
	VersionID string `json:"versionId"`
	Title     string `json:"title,omitempty"`
}

type LocalModDependencyImpact struct {
	ModID             string                      `json:"modId"`
	AffectedMods      []LocalModDependencyMod     `json:"affectedMods"`
	DisabledProviders []LocalModDependencyMod     `json:"disabledProviders"`
	Candidates        []CachedDependencyCandidate `json:"candidates,omitempty"`
}

type LocalModDisableImpactResult struct {
	Impacts []LocalModDependencyImpact `json:"impacts"`
}

type RestoreCachedDependencyRequest struct {
	ModID     string `json:"modId"`
	Platform  string `json:"platform"`
	ProjectID string `json:"projectId"`
	VersionID string `json:"versionId"`
}
