// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package modbridge

import "testing"

// TestButtonStatusContract locks the wire values shared with the GUI and TUI.
// Adapters may centralize these strings, but they must never rename them.
func TestButtonStatusContract(t *testing.T) {
	want := map[string]string{
		"new":               BtnStatusNew,
		"installed":         BtnStatusInstalled,
		"update":            BtnStatusUpdate,
		"conflict":          BtnStatusConflict,
		"incompatible":      BtnStatusIncompatible,
		"metadataLoading":   BtnStatusMetadataLoading,
		"metadataFailed":    BtnStatusMetadataFailed,
		"noSelectedVersion": BtnStatusNoSelectedVersion,
		"invalidRequest":    BtnStatusInvalidRequest,
	}
	for expected, got := range want {
		if got != expected {
			t.Fatalf("status constant %q = %q, want %q", expected, got, expected)
		}
	}
}
