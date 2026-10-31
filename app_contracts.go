package main

// ExportFavoritePackwizResult is adapter-specific because it reports a native
// save-dialog outcome rather than a domain result.
type ExportFavoritePackwizResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

// DirectorySelectionResult reports a native folder-dialog outcome. Cancel and
// invalid selections stay separate so the UI can remain silent on cancel and
// explain why a chosen folder could not be used.
type DirectorySelectionResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
	Invalid  bool   `json:"invalid"`
}
