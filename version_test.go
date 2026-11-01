// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "testing"

func TestCurrentAppVersion(t *testing.T) {
	previous := appVersion
	t.Cleanup(func() { appVersion = previous })

	appVersion = " v1.2.3 "
	if got := currentAppVersion(); got != "v1.2.3" {
		t.Fatalf("currentAppVersion() = %q, want v1.2.3", got)
	}
	appVersion = "  "
	if got := currentAppVersion(); got != defaultAppVersion {
		t.Fatalf("currentAppVersion() = %q, want %q", got, defaultAppVersion)
	}
}
