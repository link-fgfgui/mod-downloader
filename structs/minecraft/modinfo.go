package structs

type JijModInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ModInfo struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Version     string       `json:"version"`
	Description string       `json:"description"`
	FileName    string       `json:"fileName"`
	Path        string       `json:"path"`
	SHA1        string       `json:"sha1,omitempty"`
	Enabled     bool         `json:"enabled"`
	JijMods     []JijModInfo `json:"jijMods,omitempty"`
}
