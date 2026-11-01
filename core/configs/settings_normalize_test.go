// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package configs

import (
	"math"
	"testing"

	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

func TestNormalizeAnimationDurationMultiplier(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"nan", math.NaN(), DefaultAnimationDurationMultiplier},
		{"positive infinity", math.Inf(1), DefaultAnimationDurationMultiplier},
		{"negative infinity", math.Inf(-1), DefaultAnimationDurationMultiplier},
		{"zero", 0, DefaultAnimationDurationMultiplier},
		{"negative", -0.5, DefaultAnimationDurationMultiplier},
		{"below minimum", 0.05, MinAnimationDurationMultiplier},
		{"above maximum", 10, MaxAnimationDurationMultiplier},
		{"valid", 1.5, 1.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeAnimationDurationMultiplier(tt.in); got != tt.want {
				t.Fatalf("NormalizeAnimationDurationMultiplier(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeNetworkSettingsBounds(t *testing.T) {
	tests := []struct {
		name       string
		req        appstructs.SaveNetworkSettingsRequest
		wantFile   int
		wantConcur int
		wantRate   float64
		wantRPS    int
	}{
		{
			name:     "zero request falls back to defaults",
			req:      appstructs.SaveNetworkSettingsRequest{},
			wantFile: DefaultFileConcurrency, wantConcur: DefaultConcurrentDownloads,
			wantRate: DefaultTargetDownloadRateMiB, wantRPS: MinRequestsPerSecond,
		},
		{
			name:     "negative values fall back",
			req:      appstructs.SaveNetworkSettingsRequest{FileConcurrency: -2, ConcurrentDownloads: -3, TargetDownloadRateMiB: -1, RequestsPerSecond: -5},
			wantFile: DefaultFileConcurrency, wantConcur: DefaultConcurrentDownloads,
			wantRate: DefaultTargetDownloadRateMiB, wantRPS: MinRequestsPerSecond,
		},
		{
			name:     "above maximum clamps",
			req:      appstructs.SaveNetworkSettingsRequest{FileConcurrency: 999, ConcurrentDownloads: 500, TargetDownloadRateMiB: 9, RequestsPerSecond: 1000},
			wantFile: MaxFileConcurrency, wantConcur: MaxConcurrentDownloads,
			wantRate: MaxTargetDownloadRateMiB, wantRPS: MaxRequestsPerSecond,
		},
		{
			name:     "valid values pass through",
			req:      appstructs.SaveNetworkSettingsRequest{FileConcurrency: 8, ConcurrentDownloads: 32, TargetDownloadRateMiB: 2.5, RequestsPerSecond: 50},
			wantFile: 8, wantConcur: 32, wantRate: 2.5, wantRPS: 50,
		},
		{
			name:     "nan and infinity fall back",
			req:      appstructs.SaveNetworkSettingsRequest{TargetDownloadRateMiB: math.NaN()},
			wantFile: DefaultFileConcurrency, wantConcur: DefaultConcurrentDownloads,
			wantRate: DefaultTargetDownloadRateMiB, wantRPS: MinRequestsPerSecond,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeNetworkSettings(tt.req)
			if got.FileConcurrency != tt.wantFile {
				t.Fatalf("file concurrency = %d, want %d", got.FileConcurrency, tt.wantFile)
			}
			if got.ConcurrentDownloads != tt.wantConcur {
				t.Fatalf("concurrent downloads = %d, want %d", got.ConcurrentDownloads, tt.wantConcur)
			}
			if got.TargetDownloadRateMiB != tt.wantRate {
				t.Fatalf("target rate = %v, want %v", got.TargetDownloadRateMiB, tt.wantRate)
			}
			if got.RequestsPerSecond != tt.wantRPS {
				t.Fatalf("requests per second = %d, want %d", got.RequestsPerSecond, tt.wantRPS)
			}
		})
	}
}

func TestNormalizeNetworkSettingsRoundsTargetRate(t *testing.T) {
	got := NormalizeNetworkSettings(appstructs.SaveNetworkSettingsRequest{TargetDownloadRateMiB: 1.234})
	if got.TargetDownloadRateMiB != 1.2 {
		t.Fatalf("target rate = %v, want 1.2", got.TargetDownloadRateMiB)
	}
	got = NormalizeNetworkSettings(appstructs.SaveNetworkSettingsRequest{TargetDownloadRateMiB: 4.96})
	if got.TargetDownloadRateMiB != MaxTargetDownloadRateMiB {
		t.Fatalf("target rate = %v, want %v", got.TargetDownloadRateMiB, MaxTargetDownloadRateMiB)
	}
}

func TestNormalizeSettingsViewCanonicalizes(t *testing.T) {
	view := NormalizeSettingsView(appview.SettingsView{
		Theme:                       "bogus",
		Language:                    "bogus",
		AnimationDurationMultiplier: 0,
		PreferredReleaseType:        "bogus",
		FileConcurrency:             -1,
		ConcurrentDownloads:         9999,
		TargetDownloadRateMiB:       3.14159,
		RequestsPerSecond:           -2,
	})
	if view.Theme != ThemeDark.String() {
		t.Fatalf("theme = %q, want %q", view.Theme, ThemeDark)
	}
	if view.Language != LanguageSystem.String() {
		t.Fatalf("language = %q, want %q", view.Language, LanguageSystem)
	}
	if view.AnimationDurationMultiplier != DefaultAnimationDurationMultiplier {
		t.Fatalf("multiplier = %v, want %v", view.AnimationDurationMultiplier, DefaultAnimationDurationMultiplier)
	}
	if view.PreferredReleaseType != DefaultPreferredReleaseType {
		t.Fatalf("release type = %q, want %q", view.PreferredReleaseType, DefaultPreferredReleaseType)
	}
	if view.FileConcurrency != DefaultFileConcurrency {
		t.Fatalf("file concurrency = %d, want %d", view.FileConcurrency, DefaultFileConcurrency)
	}
	if view.ConcurrentDownloads != MaxConcurrentDownloads {
		t.Fatalf("concurrent downloads = %d, want %d", view.ConcurrentDownloads, MaxConcurrentDownloads)
	}
	if view.TargetDownloadRateMiB != 3.1 {
		t.Fatalf("target rate = %v, want 3.1", view.TargetDownloadRateMiB)
	}
	if view.RequestsPerSecond != MinRequestsPerSecond {
		t.Fatalf("requests per second = %d, want %d", view.RequestsPerSecond, MinRequestsPerSecond)
	}
}

func TestNormalizeSettingsViewKeepsValidValues(t *testing.T) {
	view := NormalizeSettingsView(appview.SettingsView{
		Theme:                       "light",
		Language:                    "zh-CN",
		AnimationDurationMultiplier: 2.5,
		PreferredReleaseType:        "beta",
		FileConcurrency:             8,
		ConcurrentDownloads:         32,
		TargetDownloadRateMiB:       2.5,
		RequestsPerSecond:           50,
		SimpleMode:                  true,
	})
	if view.Theme != ThemeLight.String() || view.Language != LanguageChinese.String() {
		t.Fatalf("theme/language = %q/%q", view.Theme, view.Language)
	}
	if view.AnimationDurationMultiplier != 2.5 || view.PreferredReleaseType != "beta" {
		t.Fatalf("animation/release = %v/%q", view.AnimationDurationMultiplier, view.PreferredReleaseType)
	}
	if view.FileConcurrency != 8 || view.ConcurrentDownloads != 32 || view.TargetDownloadRateMiB != 2.5 || view.RequestsPerSecond != 50 {
		t.Fatalf("network fields = %d/%d/%v/%d", view.FileConcurrency, view.ConcurrentDownloads, view.TargetDownloadRateMiB, view.RequestsPerSecond)
	}
	if !view.SimpleMode {
		t.Fatal("unrelated fields must be preserved")
	}
}
