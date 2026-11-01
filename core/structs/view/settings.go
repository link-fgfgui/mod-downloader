// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package view

// SettingsView is a client-safe settings snapshot. API keys are represented
// by existence and masks; plaintext keys never cross the Wails boundary.
type SettingsView struct {
	Theme                       string  `json:"theme"`
	Language                    string  `json:"language"`
	SimpleMode                  bool    `json:"simpleMode"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
	AutoScanUnusedDependencies  bool    `json:"autoScanUnusedDependencies"`
	MCIMEnabled                 bool    `json:"mcimEnabled"`
	EnableJIJDetection          bool    `json:"enableJijDetection"`
	HardwareAcceleration        bool    `json:"hardwareAcceleration"`
	RequireCtrlForMultiSelect   bool    `json:"requireCtrlForMultiSelect"`
	PreferredReleaseType        string  `json:"preferredReleaseType"`
	MinecraftDir                string  `json:"minecraftDir"`
	CacheDir                    string  `json:"cacheDir"`
	CachePath                   string  `json:"cachePath"`
	HasCurseForgeKey            bool    `json:"hasCurseforgeKey"`
	CurseForgeKeyMask           string  `json:"curseforgeKeyMask"`
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
