// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package netutil holds small URL helpers shared inside the core module.
package netutil

import "net/url"

// RemoteHost returns the host component of rawURL, or "" when it cannot be
// parsed.
func RemoteHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
