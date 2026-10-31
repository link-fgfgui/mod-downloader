package structs

type JijModInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type LocalModDependency struct {
	ModID   string `json:"modId"`
	Type    string `json:"type,omitempty"`
	Version string `json:"version,omitempty"`
}

type ModInfo struct {
	ID                    string               `json:"id"`
	Name                  string               `json:"name"`
	Version               string               `json:"version"`
	Description           string               `json:"description"`
	FileName              string               `json:"fileName"`
	Path                  string               `json:"path"`
	SHA1                  string               `json:"sha1,omitempty"`
	CfFingerprint         uint32               `json:"cfFingerprint,omitempty"`
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
	Enabled               bool                 `json:"enabled"`
	JijMods               []JijModInfo         `json:"jijMods,omitempty"`
	Dependencies          []LocalModDependency `json:"dependencies,omitempty"`
}
