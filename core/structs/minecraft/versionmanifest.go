package structs

type Latest struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

type Version struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type VersionManifest struct {
	Latest   Latest    `json:"latest"`
	Versions []Version `json:"versions"`
}
