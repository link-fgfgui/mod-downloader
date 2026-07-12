package main

import "github.com/link-fgfgui/mod-downloader-core/appcore"

// AppPreferences is the persisted preference subset exposed to the frontend.
type AppPreferences struct {
	Theme                       string  `json:"theme"`
	Language                    string  `json:"language"`
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
}

// SettingsView is a frontend-safe settings snapshot. API keys are represented
// by existence and masks; plaintext keys never cross the Wails boundary.
type SettingsView struct {
	Theme                       string  `json:"theme"`
	Language                    string  `json:"language"`
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
	AutoScanUnusedDependencies  bool    `json:"autoScanUnusedDependencies"`
	MCIMEnabled                 bool    `json:"mcimEnabled"`
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
	RequestsPerSecond           int     `json:"requestsPerSecond"`
}

type SaveApiKeysRequest struct {
	CurseforgeApiKey string `json:"curseforgeApiKey"`
	ModrinthApiKey   string `json:"modrinthApiKey"`
}

type SaveAnimationSettingsRequest struct {
	AnimationMode               string  `json:"animationMode"`
	AnimationEnabled            bool    `json:"animationEnabled"`
	AnimationDurationMultiplier float64 `json:"animationDurationMultiplier"`
}

type ExportFavoritePackwizResult struct {
	Path     string `json:"path"`
	Canceled bool   `json:"canceled"`
}

type SaveUnusedDependencyCleanupSettingsRequest struct {
	AutoScanUnusedDependencies bool `json:"autoScanUnusedDependencies"`
}

type SaveMCIMSettingsRequest struct {
	MCIMEnabled bool `json:"mcimEnabled"`
}

type SaveNetworkSettingsRequest struct {
	FileConcurrency         int     `json:"fileConcurrency"`
	ConcurrentDownloads     int     `json:"concurrentDownloads"`
	AdaptiveFileConcurrency bool    `json:"adaptiveFileConcurrency"`
	TargetDownloadRateMiB   float64 `json:"targetDownloadRateMiB"`
	RequestsPerSecond       int     `json:"requestsPerSecond"`
}

const apiKeyKeepSentinel = appcore.APIKeyKeepSentinel
