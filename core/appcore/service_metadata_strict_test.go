// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

//go:build no_curseforge_cdn_fallback

package appcore

import (
	"path/filepath"
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

func openMetadataTestDB(t *testing.T) {
	t.Helper()
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
}

func TestStrictModeSkipsLocalMetadataWithoutDownloadURL(t *testing.T) {
	openMetadataTestDB(t)

	project := models.ModProject{
		Platform:  "CurseForge",
		ProjectID: "238222",
		Slug:      "jei",
		Title:     "JEI",
	}
	if err := storage.UpsertModProject(project); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersions("CurseForge", "238222", []models.ModVersion{{
		ID:        "4593550",
		VersionID: "4593550",
		Platform:  "CurseForge",
		ProjectID: "238222",
		Version:   "1.0.0",
		FileName:  "jei.jar",
		SHA1:      "abc123",
	}}); err != nil {
		t.Fatal(err)
	}

	enriched, misses := enrichModMetadata([]structs.ModInfo{{
		ID:   "local",
		SHA1: "abc123",
	}})
	if len(enriched) != 1 {
		t.Fatalf("enriched count = %d, want 1", len(enriched))
	}
	if enriched[0].OnlineName != "" || enriched[0].OnlinePlatform != "" ||
		enriched[0].OnlineProjectID != "" || enriched[0].OnlineVersionID != "" {
		t.Fatalf("online fields = %#v, want unresolved metadata", enriched[0])
	}
	if len(misses) != 0 {
		t.Fatalf("misses = %#v, want no retry for a known URL-less version", misses)
	}
}

func TestStrictModeRejectsResolvedMetadataWithoutDownloadURL(t *testing.T) {
	projects := map[string]models.ModProject{
		"abc123": {Platform: "CurseForge", ProjectID: "238222", Title: "JEI"},
	}
	versions := map[string]models.ModVersion{
		"abc123": {ID: "4593550", Platform: "CurseForge", ProjectID: "238222", FileName: "jei.jar"},
	}
	updated, changed := applyResolvedModMetadata(
		[]structs.ModInfo{{ID: "local", SHA1: "abc123", OnlineMetadataLoading: true}},
		map[string]uint32{"abc123": 42},
		projects,
		versions,
	)
	if !changed || updated[0].OnlineMetadataLoading {
		t.Fatalf("resolved metadata = changed=%v mods=%#v, want loading cleared", changed, updated)
	}
	if updated[0].OnlineName != "" || updated[0].OnlinePlatform != "" || updated[0].OnlineProjectID != "" {
		t.Fatalf("online fields = %#v, want unresolved metadata", updated[0])
	}
}
