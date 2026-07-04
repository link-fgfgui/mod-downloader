package structs

type ModInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	FileName    string `json:"fileName"`
	Path        string `json:"path"`
	SHA1        string `json:"sha1,omitempty"`
	Enabled     bool   `json:"enabled"`
	// IsJij is true when this mod entry originates from a nested jar (jar-in-jar /
	// JIJ). These are weak references: they express what the host JAR bundles but
	// must not participate in install-conflict detection or replacement-archive
	// logic as equals of top-level mods.toml declarations (strong references).
	IsJij bool `json:"isJij,omitempty"`
}
