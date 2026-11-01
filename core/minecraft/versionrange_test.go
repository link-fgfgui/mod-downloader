// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package minecraft

import "testing"

func TestVersionRangeAllows(t *testing.T) {
	tests := []struct {
		declared string
		version  string
		want     bool
		known    bool
	}{
		{"", "1.21.1", true, false},
		{"*", "1.21.1", true, false},
		{">=1.21", "1.21.1", true, true},
		{">=1.22", "1.21.1", false, true},
		{"~1.21", "1.21.1", true, true},
		{"~1.21.1", "1.21.4", true, true},
		{"~1.21.1", "1.22.0", false, true},
		{"^1.21", "1.21.9", true, true},
		{"^1.21", "2.0", false, true},
		{"[1.20.1,1.21)", "1.21.1", false, true},
		{"[1.20.1,1.21.1]", "1.21.1", true, true},
		{"1.21.x", "1.21.1", true, true},
		{"1.21.1", "1.21.1", true, true},
		{"1.21.2", "1.21.1", false, true},
		{">=1.21- <1.21.2-", "1.21.1", true, true},
		{">=1.21- <1.21.2-", "1.21.2", false, true},
		{">=1.21- <1.21.2-", "1.20.6", false, true},
		{">=1.20.1, <1.21.2-", "1.21.1", true, true},
		{">=1.20.1, <1.21.2-", "1.22", false, true},
		{">=1.21-", "1.21.4", true, true},
		{"<1.21.2-", "1.21.1", true, true},
	}
	for _, tt := range tests {
		got, known := VersionRangeAllows(tt.declared, tt.version)
		if got != tt.want || known != tt.known {
			t.Fatalf("VersionRangeAllows(%q, %q) = (%v, %v), want (%v, %v)", tt.declared, tt.version, got, known, tt.want, tt.known)
		}
	}
}
