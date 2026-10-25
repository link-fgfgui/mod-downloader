package main

// ExportFavoritePackwizResult is adapter-specific because it reports a native
// save-dialog outcome rather than a domain result.
type ExportFavoritePackwizResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}
