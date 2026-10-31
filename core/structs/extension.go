package structs

// ExtensionProjectPayload mirrors the browser extension's scan payload:
// {"p":"mr"|"cf","id":null,"slug":"...","file":"..."}.
// The HTTP adapter decodes the raw JSON into this shared type; appcore owns the
// platform mapping, metadata resolution, filtering and auto-pin workflow.
type ExtensionProjectPayload struct {
	PlatformCode string  `json:"p"`
	ID           *string `json:"id"`
	Slug         string  `json:"slug"`
	VersionID    string  `json:"file"`
}
