// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

//go:build no_curseforge_cdn_fallback

package downloader

import (
	"testing"
	"time"

	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader/core/structs"
)

func TestStrictModeBuildDownloadJobReportsOfficialCurseForgePage(t *testing.T) {
	openDownloadTestDatabase(t)

	project := models.ModProject{
		ID:        "curseforge:238222",
		Platform:  "CurseForge",
		ProjectID: "238222",
		Slug:      "jei",
		Title:     "JEI",
	}
	if err := storage.UpsertModProject(project); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersionSnapshot("CurseForge", "jei", []models.ModVersion{{
		ID:           "4593550",
		VersionID:    "4593550",
		Platform:     "CurseForge",
		ProjectID:    "238222",
		FileName:     "jei.jar",
		GameVersions: []string{"1.21.1"},
		Loaders:      []string{"fabric"},
	}}, time.Now().Unix(), []storage.PlatformVersionScope{{
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}}); err != nil {
		t.Fatal(err)
	}

	targetDir := t.TempDir()
	_, result, ok := buildDownloadJob(appstructs.ModDownloadRequest{
		ProjectID:        "curseforge:238222",
		Result:           project,
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
		TargetDir:        targetDir,
		InstanceID:       "test:" + targetDir,
	}, "", "", nil)
	if ok {
		t.Fatal("buildDownloadJob() queued a version without a download URL")
	}
	if !result.Skipped {
		t.Fatalf("result = %#v, want skipped", result)
	}
	want := "https://www.curseforge.com/minecraft/mc-mods/jei/download/4593550"
	if result.Reason != want {
		t.Fatalf("skip reason = %q, want %q", result.Reason, want)
	}
}
