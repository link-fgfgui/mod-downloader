// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"strings"
	"testing"

	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

func TestVerifyLocalModsReportsMissingDependency(t *testing.T) {
	mods := []structs.ModInfo{
		{
			ID: "a", Path: "/mods/a.jar",
			Dependencies: []structs.LocalModDependency{
				{ModID: "fabric-api", Type: "required"},
				{ModID: "minecraft", Type: "required"},
				{ModID: "b", Type: "required"},
			},
		},
		{ID: "b", Path: "/mods/b.jar"},
	}
	report := VerifyLocalMods("/mods", mods, "1.21.1", "fabric", false)
	if report.Passed {
		t.Fatalf("expected failure: %#v", report)
	}
	if len(report.Issues) != 1 || report.Issues[0].Kind != "missing-dependency" || report.Issues[0].ModID != "fabric-api" {
		t.Fatalf("issues = %#v", report.Issues)
	}
	if !strings.Contains(report.Issues[0].Suggest, "fabric-api") {
		t.Fatalf("missing suggestion: %#v", report.Issues[0])
	}
	if len(report.Provided) != 2 {
		t.Fatalf("provided = %#v", report.Provided)
	}

	// Installing the provider clears the issue.
	mods = append(mods, structs.ModInfo{ID: "fabric-api", Path: "/mods/fabric-api.jar"})
	if report := VerifyLocalMods("/mods", mods, "1.21.1", "fabric", false); !report.Passed {
		t.Fatalf("expected pass: %#v", report.Issues)
	}
}

func TestVerifyLocalModsDetectsDuplicateIDs(t *testing.T) {
	mods := []structs.ModInfo{
		{ID: "dup", Path: "/mods/dup1.jar"},
		{ID: "dup", Path: "/mods/dup2.jar"},
	}
	report := VerifyLocalMods("/mods", mods, "", "", false)
	if report.Passed || len(report.Issues) != 1 || report.Issues[0].Kind != "duplicate-mod" {
		t.Fatalf("report = %#v", report)
	}
}

func TestVerifyLocalModsLoaderMismatchIsAdvisory(t *testing.T) {
	mods := []structs.ModInfo{{ID: "forgemod", Path: "/mods/forgemod.jar", Loaders: []string{"forge"}}}
	report := VerifyLocalMods("/mods", mods, "", "fabric", false)
	if !report.Passed {
		t.Fatalf("advisory issue must not fail default verify: %#v", report)
	}
	if len(report.Issues) != 1 || !report.Issues[0].Advisory || report.Issues[0].Kind != "incompatible" {
		t.Fatalf("issues = %#v", report.Issues)
	}
	if report := VerifyLocalMods("/mods", mods, "", "fabric", true); report.Passed {
		t.Fatalf("strict verify must fail: %#v", report)
	}

	// Quilt mods run on Fabric.
	quilt := []structs.ModInfo{{ID: "qmod", Path: "/mods/qmod.jar", Loaders: []string{"quilt"}}}
	if report := VerifyLocalMods("/mods", quilt, "", "fabric", true); !report.Passed {
		t.Fatalf("quilt-on-fabric must pass: %#v", report.Issues)
	}
}

func TestVerifyLocalModsMinecraftRange(t *testing.T) {
	mods := []structs.ModInfo{{ID: "old", Path: "/mods/old.jar", MinecraftVersion: "[1.20,1.21)"}}
	report := VerifyLocalMods("/mods", mods, "1.21.1", "", false)
	if len(report.Issues) != 1 || !report.Issues[0].Advisory {
		t.Fatalf("issues = %#v", report.Issues)
	}

	mods[0].MinecraftVersion = "[1.20,1.22)"
	if report := VerifyLocalMods("/mods", mods, "1.21.1", "", false); len(report.Issues) != 0 {
		t.Fatalf("supported range reported: %#v", report.Issues)
	}
}
