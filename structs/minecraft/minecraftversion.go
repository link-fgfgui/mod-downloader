package structs

type Patch struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// MinecraftVersion maps the top-level structure of .minecraft/versions/<id>/<id>.json
type MinecraftVersion struct {
	Name         string  `json:"name"`
	ID           string  `json:"id"`
	InheritsFrom string  `json:"inheritsFrom"`
	Jar          string  `json:"jar"`
	Patches      []Patch `json:"patches,omitempty"`
}

// VersionInfo is a simplified structure returned to the frontend
type VersionInfo struct {
	Name             string    `json:"name"`
	ID               string    `json:"id"`
	MinecraftVersion string    `json:"minecraftVersion"`
	ModLoader        string    `json:"modLoader"`
	Mods             []ModInfo `json:"mods,omitempty"`
}
