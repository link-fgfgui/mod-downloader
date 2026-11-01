// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package strutil holds tiny string helpers shared inside the core module.
package strutil

import "strings"

// FirstNonEmpty returns the first value that is non-blank after trimming,
// preserving the original value. Callers that need the trimmed form should use
// FirstNonEmptyTrimmed.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// FirstNonEmptyTrimmed returns the first value that is non-blank after
// trimming, trimmed.
func FirstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
