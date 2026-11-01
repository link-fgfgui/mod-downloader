// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import "testing"

// TestEventKindNames locks the wire names: the GUI registers frontend events
// with these strings, so changing one silently breaks the renderer.
func TestEventKindNames(t *testing.T) {
	want := map[EventKind]string{
		EventSearchModsUpdated:      "search-mods-updated",
		EventExtensionModsAccepted:  "extension-mods-accepted",
		EventDownloadStatesUpdated:  "download-states-updated",
		EventDownloadQueueUpdated:   "download-queue-updated",
		EventDownloadFailed:         "download-failed",
		EventDownloadCompleted:      "download-completed",
		EventMinecraftDirChanged:    "minecraft-dir-changed",
		EventSelectedVersionChanged: "selected-version-changed",
		EventUsageStatsUpdated:      "usage-stats-updated",
	}
	if len(want) != 9 {
		t.Fatalf("contract covers %d events, want 9", len(want))
	}
	seen := map[string]bool{}
	for kind, name := range want {
		if got := kind.Name(); got != name {
			t.Errorf("%s.Name() = %q, want %q", kind, got, name)
		}
		if seen[name] {
			t.Errorf("duplicate wire name %q", name)
		}
		seen[name] = true
	}
	if got := EventKind("future-event").Name(); got != "future-event" {
		t.Errorf("unknown kind name = %q, want passthrough", got)
	}
}
