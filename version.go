// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import "strings"

const defaultAppVersion = "dev"

var appVersion = defaultAppVersion

func currentAppVersion() string {
	version := strings.TrimSpace(appVersion)
	if version == "" {
		return defaultAppVersion
	}
	return version
}
