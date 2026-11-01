// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package configs

import (
	"math"

	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// NormalizeNetworkSettings applies the documented download/API bounds and the
// one-decimal precision rule for the target rate. Both adapters send raw
// drafts here; Service.GetSettings uses the same helpers for its snapshot.
func NormalizeNetworkSettings(req appstructs.SaveNetworkSettingsRequest) appstructs.SaveNetworkSettingsRequest {
	return appstructs.SaveNetworkSettingsRequest{
		FileConcurrency:         normalizePositiveInt(req.FileConcurrency, MinFileConcurrency, MaxFileConcurrency, DefaultFileConcurrency),
		ConcurrentDownloads:     normalizePositiveInt(req.ConcurrentDownloads, MinConcurrentDownloads, MaxConcurrentDownloads, DefaultConcurrentDownloads),
		AdaptiveFileConcurrency: req.AdaptiveFileConcurrency,
		TargetDownloadRateMiB:   normalizeTargetDownloadRateMiB(req.TargetDownloadRateMiB),
		VerifySHA1:              req.VerifySHA1,
		RequestsPerSecond:       normalizePositiveInt(req.RequestsPerSecond, MinRequestsPerSecond, MaxRequestsPerSecond, MinRequestsPerSecond),
	}
}

// NormalizeSettingsView clamps and canonicalizes a settings snapshot before it
// crosses the adapter boundary. Adapters render the result as-is.
func NormalizeSettingsView(view appview.SettingsView) appview.SettingsView {
	view.Theme = ParseTheme(view.Theme).Normalized().String()
	view.Language = ParseLanguage(view.Language).Normalized().String()
	view.AnimationDurationMultiplier = NormalizeAnimationDurationMultiplier(view.AnimationDurationMultiplier)
	view.PreferredReleaseType = Preferences{PreferredReleaseType: view.PreferredReleaseType}.PreferredReleaseTypeValue()

	network := NormalizeNetworkSettings(appstructs.SaveNetworkSettingsRequest{
		FileConcurrency:         view.FileConcurrency,
		ConcurrentDownloads:     view.ConcurrentDownloads,
		AdaptiveFileConcurrency: view.AdaptiveFileConcurrency,
		TargetDownloadRateMiB:   view.TargetDownloadRateMiB,
		VerifySHA1:              view.VerifySHA1,
		RequestsPerSecond:       view.RequestsPerSecond,
	})
	view.FileConcurrency = network.FileConcurrency
	view.ConcurrentDownloads = network.ConcurrentDownloads
	view.TargetDownloadRateMiB = network.TargetDownloadRateMiB
	view.RequestsPerSecond = network.RequestsPerSecond
	return view
}

// normalizePositiveInt treats non-positive values as "unset" so a zero-value
// request falls back to the documented default instead of the minimum.
func normalizePositiveInt(value, minimum, maximum, fallback int) int {
	if value <= 0 {
		return fallback
	}
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

// normalizeTargetDownloadRateMiB clamps to the documented range and keeps one
// decimal so the stored value matches what the settings UI can express.
func normalizeTargetDownloadRateMiB(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return DefaultTargetDownloadRateMiB
	}
	value = math.Min(MaxTargetDownloadRateMiB, math.Max(MinTargetDownloadRateMiB, value))
	return math.Round(value*10) / 10
}
