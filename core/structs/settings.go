package structs

// StartupStatus is the typed core readiness snapshot exposed to adapters.
// Stage is one of "config", "storage", "releases" or "unknown". A non-empty
// ExtensionServerError is non-fatal and must be presented separately from Error.
type StartupStatus struct {
	Ready                bool   `json:"ready"`
	Stage                string `json:"stage,omitempty"`
	Error                string `json:"error,omitempty"`
	ExtensionServerError string `json:"extensionServerError,omitempty"`
}

// AppPreferences is the persisted preference subset exposed to clients.
type AppPreferences struct {
	Theme                       string  `json:"theme"`
	Language                    string  `json:"language"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
}

// SettingsView is a client-safe settings snapshot. API keys are represented
// by existence and masks; plaintext keys never cross the Wails boundary.
type SettingsView struct {
	Theme                       string  `json:"theme"`
	Language                    string  `json:"language"`
	SimpleMode                  bool    `json:"simpleMode"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
	AutoScanUnusedDependencies  bool    `json:"autoScanUnusedDependencies"`
	MCIMEnabled                 bool    `json:"mcimEnabled"`
	EnableJijDetection          bool    `json:"enableJijDetection"`
	HardwareAcceleration        bool    `json:"hardwareAcceleration"`
	RequireCtrlForMultiSelect   bool    `json:"requireCtrlForMultiSelect"`
	PreferredReleaseType        string  `json:"preferredReleaseType"`
	MinecraftDir                string  `json:"minecraftDir"`
	CacheDir                    string  `json:"cacheDir"`
	CachePath                   string  `json:"cachePath"`
	HasCurseforgeKey            bool    `json:"hasCurseforgeKey"`
	CurseforgeKeyMask           string  `json:"curseforgeKeyMask"`
	HasModrinthKey              bool    `json:"hasModrinthKey"`
	ModrinthKeyMask             string  `json:"modrinthKeyMask"`
	FileConcurrency             int     `json:"fileConcurrency"`
	ConcurrentDownloads         int     `json:"concurrentDownloads"`
	AdaptiveFileConcurrency     bool    `json:"adaptiveFileConcurrency"`
	TargetDownloadRateMiB       float64 `json:"targetDownloadRateMiB"`
	VerifySHA1                  bool    `json:"verifySha1"`
	RequestsPerSecond           int     `json:"requestsPerSecond"`
	// SaveError carries the last failed config write. Empty means the last
	// persist attempt succeeded.
	SaveError string `json:"saveError,omitempty"`
}

type SaveApiKeysRequest struct {
	CurseforgeApiKey string `json:"curseforgeApiKey"`
	ModrinthApiKey   string `json:"modrinthApiKey"`
}

type SaveAnimationSettingsRequest struct {
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
}

type SaveSimpleModeSettingsRequest struct {
	SimpleMode bool `json:"simpleMode"`
}

type SaveUnusedDependencyCleanupSettingsRequest struct {
	AutoScanUnusedDependencies bool `json:"autoScanUnusedDependencies"`
}

type SaveMCIMSettingsRequest struct {
	MCIMEnabled bool `json:"mcimEnabled"`
}

type SaveJijDetectionSettingsRequest struct {
	EnableJijDetection bool `json:"enableJijDetection"`
}

type SavePreferredReleaseTypeSettingsRequest struct {
	PreferredReleaseType string `json:"preferredReleaseType"`
}

type SaveHardwareAccelerationSettingsRequest struct {
	HardwareAcceleration bool `json:"hardwareAcceleration"`
}

type SaveNetworkSettingsRequest struct {
	FileConcurrency         int     `json:"fileConcurrency"`
	ConcurrentDownloads     int     `json:"concurrentDownloads"`
	AdaptiveFileConcurrency bool    `json:"adaptiveFileConcurrency"`
	TargetDownloadRateMiB   float64 `json:"targetDownloadRateMiB"`
	VerifySHA1              bool    `json:"verifySha1"`
	RequestsPerSecond       int     `json:"requestsPerSecond"`
}
