package appcore

import (
	"reflect"
	"testing"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func TestScanUnusedDependenciesFindsLibraryRequiredOnlyByExcludedMod(t *testing.T) {
	version := structs.VersionInfo{
		ID:   "test",
		Name: "test",
		Mods: []structs.ModInfo{
			{
				ID:      "main",
				Name:    "Main",
				Path:    "versions/test/mods/main.jar",
				Enabled: true,
				Dependencies: []structs.LocalModDependency{{
					ModID: "lib-one",
					Type:  "required",
				}},
			},
			{
				ID:         "lib-one",
				Name:       "Library One",
				Path:       "versions/test/mods/lib-one.jar",
				FileName:   "lib-one",
				Enabled:    true,
				Categories: []string{"library"},
			},
			{
				ID:         "ordinary",
				Name:       "Ordinary",
				Path:       "versions/test/mods/ordinary.jar",
				FileName:   "ordinary",
				Enabled:    true,
				Categories: []string{"adventure"},
			},
		},
	}
	setSelectedVersionForScanTest(t, version)

	result, err := New(Options{}).ScanUnusedDependencies(appstructs.UnusedDependencyScanRequest{
		ExcludedPaths: []string{"versions/test/mods/main.jar"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1: %#v", len(result.Candidates), result.Candidates)
	}
	got := result.Candidates[0]
	if got.Path != "versions/test/mods/lib-one.jar" {
		t.Fatalf("candidate path = %q, want lib-one path", got.Path)
	}
	if !reflect.DeepEqual(got.ModIDs, []string{"lib-one"}) {
		t.Fatalf("candidate mod IDs = %#v, want lib-one", got.ModIDs)
	}
	if !reflect.DeepEqual(got.Evidence, []string{unusedDependencyEvidenceOnlineLibrary, unusedDependencyEvidenceRequiredByDeleted}) {
		t.Fatalf("candidate evidence = %#v", got.Evidence)
	}
}

func TestScanUnusedDependenciesKeepsLibrariesUsedByEnabledMods(t *testing.T) {
	mods := []structs.ModInfo{
		{
			ID:      "main",
			Path:    "main.jar",
			Enabled: true,
			Dependencies: []structs.LocalModDependency{{
				ModID: "lib-one",
				Type:  "required",
			}},
		},
		{
			ID:      "other",
			Path:    "other.jar",
			Enabled: true,
			Dependencies: []structs.LocalModDependency{{
				ModID: "lib-one",
				Type:  "required",
			}},
		},
		{
			ID:         "lib-one",
			Path:       "lib-one.jar",
			Enabled:    true,
			Categories: []string{"library"},
		},
	}

	result := scanUnusedDependencies(mods, []string{"main.jar"})
	if len(result.Candidates) != 0 {
		t.Fatalf("candidates = %#v, want none", result.Candidates)
	}
}

func TestScanUnusedDependenciesIgnoresDisabledDependentsAndOrdinaryMods(t *testing.T) {
	mods := []structs.ModInfo{
		{
			ID:      "disabled-main",
			Path:    "disabled-main.jar.disabled",
			Enabled: false,
			Dependencies: []structs.LocalModDependency{{
				ModID: "lib-one",
				Type:  "required",
			}},
		},
		{
			ID:         "lib-one",
			Path:       "lib-one.jar",
			Enabled:    true,
			Categories: []string{"library"},
		},
		{
			ID:      "ordinary",
			Path:    "ordinary.jar",
			Enabled: true,
		},
	}

	result := scanUnusedDependencies(mods, nil)
	if len(result.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want only library candidate: %#v", len(result.Candidates), result.Candidates)
	}
	if result.Candidates[0].Path != "lib-one.jar" {
		t.Fatalf("candidate path = %q, want lib-one.jar", result.Candidates[0].Path)
	}
}

func TestScanUnusedDependenciesTreatsEnabledJijAsProvider(t *testing.T) {
	mods := []structs.ModInfo{
		{
			ID:      "main",
			Path:    "main.jar",
			Enabled: true,
			Dependencies: []structs.LocalModDependency{{
				ModID: "architectury",
				Type:  "required",
			}},
		},
		{
			ID:         "architectury",
			Name:       "Architectury API",
			Path:       "architectury.jar",
			Enabled:    true,
			Categories: []string{"library"},
		},
		{
			ID:      "host",
			Path:    "host.jar",
			Enabled: true,
			JijMods: []structs.JijModInfo{{ID: "Architectury", Name: "Architectury API"}},
		},
	}

	result := scanUnusedDependencies(mods, nil)
	if len(result.Candidates) != 1 || result.Candidates[0].Path != "architectury.jar" {
		t.Fatalf("candidates = %#v, want standalone architectury", result.Candidates)
	}

	mods[2].Enabled = false
	result = scanUnusedDependencies(mods, nil)
	if len(result.Candidates) != 0 {
		t.Fatalf("candidates with disabled JIJ host = %#v, want none", result.Candidates)
	}
}

func TestScanUnusedDependenciesFailsClosedWhenJijDetectionDisabled(t *testing.T) {
	setSelectedVersionForScanTest(t, structs.VersionInfo{
		ID:   "jij-off",
		Name: "jij-off",
		Mods: []structs.ModInfo{
			{
				ID:         "lib-one",
				Path:       "mods/lib-one.jar",
				Enabled:    true,
				Categories: []string{"library"},
			},
		},
	})

	minecraft.SetJijDetectionEnabled(false)
	t.Cleanup(func() { minecraft.SetJijDetectionEnabled(true) })

	result, err := New(Options{}).ScanUnusedDependencies(appstructs.UnusedDependencyScanRequest{})
	if err != nil {
		t.Fatalf("scan error = %v", err)
	}
	if len(result.Candidates) != 0 {
		t.Fatalf("candidates = %#v, want none while JIJ detection is off", result.Candidates)
	}
}

func setSelectedVersionForScanTest(t *testing.T, version structs.VersionInfo) {
	t.Helper()
	root := t.TempDir()
	global.SetMinecraftDir(root)
	global.SetVersionsForDir(root, []structs.VersionInfo{version})
	global.SetSelectedVersion(version)
	t.Cleanup(func() {
		global.SetMinecraftDir("")
		global.InvalidateVersions()
	})
}
