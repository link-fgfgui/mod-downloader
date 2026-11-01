// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

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

// SaveAPIKeysRequest carries the API keys to persist. An empty key clears the
// stored value; the keep sentinel leaves it untouched.
type SaveAPIKeysRequest struct {
	CurseForgeAPIKey string `json:"curseforgeApiKey"`
	ModrinthAPIKey   string `json:"modrinthApiKey"`
}

// SaveAnimationSettingsRequest updates the UI animation speed multiplier.
type SaveAnimationSettingsRequest struct {
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
}

// SaveSimpleModeSettingsRequest toggles simple mode.
type SaveSimpleModeSettingsRequest struct {
	SimpleMode bool `json:"simpleMode"`
}

// SaveUnusedDependencyCleanupSettingsRequest toggles automatic unused
// dependency scans.
type SaveUnusedDependencyCleanupSettingsRequest struct {
	AutoScanUnusedDependencies bool `json:"autoScanUnusedDependencies"`
}

// SaveMCIMSettingsRequest toggles the MCIM CurseForge mirror.
type SaveMCIMSettingsRequest struct {
	MCIMEnabled bool `json:"mcimEnabled"`
}

// SaveJIJDetectionSettingsRequest toggles Jar-in-Jar detection.
type SaveJIJDetectionSettingsRequest struct {
	EnableJIJDetection bool `json:"enableJijDetection"`
}

// SavePreferredReleaseTypeSettingsRequest sets the preferred release channel
// ("latest", "release", "beta" or "alpha").
type SavePreferredReleaseTypeSettingsRequest struct {
	PreferredReleaseType string `json:"preferredReleaseType"`
}

// SaveHardwareAccelerationSettingsRequest toggles GPU rendering. The change
// takes effect on the next launch.
type SaveHardwareAccelerationSettingsRequest struct {
	HardwareAcceleration bool `json:"hardwareAcceleration"`
}

// SaveNetworkSettingsRequest updates download concurrency, rate limiting and
// API request pacing; the core clamps every field to its documented range.
type SaveNetworkSettingsRequest struct {
	FileConcurrency         int     `json:"fileConcurrency"`
	ConcurrentDownloads     int     `json:"concurrentDownloads"`
	AdaptiveFileConcurrency bool    `json:"adaptiveFileConcurrency"`
	TargetDownloadRateMiB   float64 `json:"targetDownloadRateMiB"`
	VerifySHA1              bool    `json:"verifySha1"`
	RequestsPerSecond       int     `json:"requestsPerSecond"`
}
