// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package minecraft

import "testing"

func TestCompareVersions(t *testing.T) {
	if got := CompareVersions("1.2.3", "1.2.4"); got != -1 {
		t.Fatalf("CompareVersions(1.2.3, 1.2.4) = %d", got)
	}
	if got := CompareVersions("1.10.0", "1.9.9"); got != 1 {
		t.Fatalf("CompareVersions(1.10.0, 1.9.9) = %d", got)
	}
	if got := CompareVersions("v2.0.0", "2.0.0"); got != 0 {
		t.Fatalf("CompareVersions(v2.0.0, 2.0.0) = %d", got)
	}
	if !IsVersionDowngrade("sodium-fabric-0.5.8", "0.5.11") {
		t.Fatal("0.5.8 should be a downgrade from 0.5.11")
	}
	if IsVersionDowngrade("sodium-fabric-0.6.0", "0.5.11") {
		t.Fatal("0.6.0 should not be a downgrade from 0.5.11")
	}
	if got := CompareVersions("snapshot", "1.0.0"); got != 0 {
		t.Fatalf("text vs numeric = %d, want 0", got)
	}
}

func TestVersionMajorAndChange(t *testing.T) {
	if got, ok := VersionMajor("mod-1.20.1"); !ok || got != 1 {
		t.Fatalf("VersionMajor(mod-1.20.1) = %d, %v", got, ok)
	}
	if !HasMajorVersionChange("1.0.0", "0.9.0") {
		t.Fatal("1.0.0 -> 0.9.0 should be a major change")
	}
	if HasMajorVersionChange("0.10.0", "0.9.0") {
		t.Fatal("0.10.0 -> 0.9.0 is not a major change")
	}
	if HasMajorVersionChange("snapshot", "0.9.0") {
		t.Fatal("non-numeric versions must not report a major change")
	}
}
