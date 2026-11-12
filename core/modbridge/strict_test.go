// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

//go:build no_curseforge_cdn_fallback

package modbridge

import (
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader/core/global"
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

func TestStrictModeResolveVersionsPrefersDownloadableVersion(t *testing.T) {
	openModbridgeTestDB(t)

	project := models.ModProject{
		ID:        "curseforge:238222",
		Platform:  "CurseForge",
		ProjectID: "238222",
		Slug:      "jei",
		Title:     "JEI",
	}
	versions := []models.ModVersion{
		{
			ID:           "new-no-url",
			VersionID:    "new-no-url",
			Platform:     "CurseForge",
			ProjectID:    "238222",
			FileName:     "jei-new.jar",
			PublishedAt:  2,
			GameVersions: []string{"1.21.1"},
			Loaders:      []string{"fabric"},
		},
		{
			ID:           "old-with-url",
			VersionID:    "old-with-url",
			Platform:     "CurseForge",
			ProjectID:    "238222",
			FileName:     "jei-old.jar",
			DownloadURL:  "https://edge.forgecdn.net/files/4593/550/jei-old.jar",
			PublishedAt:  1,
			GameVersions: []string{"1.21.1"},
			Loaders:      []string{"fabric"},
		},
	}
	if err := storage.SetPlatformVersionSnapshot("CurseForge", "jei", versions, time.Now().Unix(), []storage.PlatformVersionScope{{
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}}); err != nil {
		t.Fatal(err)
	}

	got := ResolveVersions(appstructs.ModDownloadRequest{
		ProjectID:        "curseforge:238222",
		Result:           project,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	})
	if len(got) != 1 || got[0].ID != "old-with-url" {
		t.Fatalf("ResolveVersions() = %#v, want the downloadable version", got)
	}
}

func TestStrictModeMarksSearchResultWithoutDownloadURL(t *testing.T) {
	openModbridgeTestDB(t)
	global.ClearLocalMods()
	t.Cleanup(global.ClearLocalMods)

	const (
		projectID   = "238222"
		projectSlug = "jei"
		instanceID  = "test:strict"
	)
	if err := storage.SetPlatformVersionSnapshot("CurseForge", projectSlug, []models.ModVersion{{
		ID:           "4593550",
		VersionID:    "4593550",
		Platform:     "CurseForge",
		ProjectID:    projectID,
		FileName:     "jei.jar",
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
	}}, time.Now().Unix(), []storage.PlatformVersionScope{{
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}}); err != nil {
		t.Fatal(err)
	}
	// Keep the instance in the scanned state so DownloadStates takes the
	// per-result goroutine path instead of the early default-state path.
	global.UpsertLocalMod(mcstructs.ModInfo{
		ID:       "othermod",
		FileName: "othermod.jar",
		Path:     "mods/othermod.jar",
		SHA1:     "other-sha1",
		Enabled:  true,
	}, instanceID, "1.21.1", "fabric")

	targetDir := t.TempDir()
	states := DownloadStates(appstructs.DownloadStatesRequest{
		Results: []models.ModProject{{
			ID:        "curseforge:" + projectID,
			Platform:  "CurseForge",
			ProjectID: projectID,
			Slug:      projectSlug,
			Title:     "JEI",
		}},
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		TargetDir:        targetDir,
		InstanceID:       instanceID,
	}, nil)
	if len(states) != 1 {
		t.Fatalf("state count = %d, want 1", len(states))
	}
	if !states[0].Disabled || states[0].Status != BtnStatusMissingDownloadURL {
		t.Fatalf("strict state = %#v, want missing-URL state", states[0])
	}
	if states[0].Icon != "mdi-close" || states[0].Color != "surface-variant" {
		t.Fatalf("strict state visuals = %#v, want disabled close icon", states[0])
	}
}
