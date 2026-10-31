package appcore

import (
	"path/filepath"
	"testing"

	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func TestAnalyzeLocalModDisableImpactFindsLostJijProvider(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{
			ID:      "dependent",
			Name:    "Dependent",
			Path:    "dependent.jar",
			Enabled: true,
			Dependencies: []mcstructs.LocalModDependency{{
				ModID: "architectury",
				Type:  "required",
			}},
		},
		{
			ID:      "host",
			Name:    "JIJ Host",
			Path:    "host.jar",
			Enabled: true,
			JijMods: []mcstructs.JijModInfo{{ID: "architectury"}},
		},
	}

	result := analyzeLocalModDisableImpact(mods, appstructs.LocalModDisableImpactRequest{
		Paths:  []string{"host.jar"},
		Action: LocalModBatchActionDisable,
	}, "1.21.1", "fabric")
	if len(result.Impacts) != 1 {
		t.Fatalf("impacts = %#v, want one", result.Impacts)
	}
	impact := result.Impacts[0]
	if impact.ModID != "architectury" || len(impact.AffectedMods) != 1 || impact.AffectedMods[0].Path != "dependent.jar" {
		t.Fatalf("impact = %#v", impact)
	}
	if len(impact.DisabledProviders) != 1 || impact.DisabledProviders[0].Path != "host.jar" {
		t.Fatalf("disabled providers = %#v", impact.DisabledProviders)
	}
}

func TestAnalyzeLocalModDisableImpactUsesProjectedBatchState(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{
			ID:      "dependent",
			Path:    "dependent.jar",
			Enabled: true,
			Dependencies: []mcstructs.LocalModDependency{{
				ModID: "architectury",
				Type:  "required",
			}},
		},
		{
			ID:      "host",
			Path:    "host.jar",
			Enabled: true,
			JijMods: []mcstructs.JijModInfo{{ID: "architectury"}},
		},
	}

	result := analyzeLocalModDisableImpact(mods, appstructs.LocalModDisableImpactRequest{
		Paths:  []string{"dependent.jar", "host.jar"},
		Action: LocalModBatchActionDisable,
	}, "1.21.1", "fabric")
	if len(result.Impacts) != 0 {
		t.Fatalf("impacts = %#v, want none when dependent is also disabled", result.Impacts)
	}

	mods = append(mods, mcstructs.ModInfo{ID: "architectury", Path: "standalone.jar", Enabled: true})
	result = analyzeLocalModDisableImpact(mods, appstructs.LocalModDisableImpactRequest{
		Paths:  []string{"host.jar"},
		Action: LocalModBatchActionInvert,
	}, "1.21.1", "fabric")
	if len(result.Impacts) != 0 {
		t.Fatalf("impacts = %#v, want none with another provider", result.Impacts)
	}
}

func TestAnalyzeLocalModDisableImpactIgnoresPreexistingMissingDependency(t *testing.T) {
	mods := []mcstructs.ModInfo{
		{
			ID:      "dependent",
			Path:    "dependent.jar",
			Enabled: true,
			Dependencies: []mcstructs.LocalModDependency{{
				ModID: "already-missing",
				Type:  "required",
			}},
		},
		{ID: "unrelated", Path: "unrelated.jar", Enabled: true},
	}

	result := analyzeLocalModDisableImpact(mods, appstructs.LocalModDisableImpactRequest{
		Paths:  []string{"unrelated.jar"},
		Action: LocalModBatchActionDisable,
	}, "1.21.1", "fabric")
	if len(result.Impacts) != 0 {
		t.Fatalf("impacts = %#v, want no warning for preexisting issue", result.Impacts)
	}
}

func TestCachedDependencyCandidatesDeduplicatesAssociations(t *testing.T) {
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)

	projects := []models.ModProject{
		{Platform: "CurseForge", ProjectID: "419699", Title: "Architectury CF"},
		{Platform: "Modrinth", ProjectID: "architectury-api", Title: "Architectury MR"},
		{Platform: "Modrinth", ProjectID: "unrelated", Title: "Unrelated Provider"},
	}
	for _, project := range projects {
		if err := storage.UpsertModPlatform(project); err != nil {
			t.Fatal(err)
		}
	}
	if err := storage.UpsertPlatformAssociationByProjects("419699", "architectury-api"); err != nil {
		t.Fatal(err)
	}
	versions := []struct {
		platform  string
		projectID string
		version   models.ModVersion
	}{
		{"CurseForge", "419699", models.ModVersion{ID: "cf-version", VersionID: "cf-version", DownloadURL: "https://example.com/cf.jar", PublishedAt: 2, GameVersions: []string{"1.21.1"}, Loaders: []string{"fabric"}, ModIDs: []string{"architectury"}}},
		{"Modrinth", "architectury-api", models.ModVersion{ID: "mr-version", VersionID: "mr-version", DownloadURL: "https://example.com/mr.jar", PublishedAt: 1, GameVersions: []string{"1.21.1"}, Loaders: []string{"fabric"}, ModIDs: []string{"architectury"}}},
	}
	for _, item := range versions {
		if err := storage.SetPlatformVersions(item.platform, item.projectID, []models.ModVersion{item.version}); err != nil {
			t.Fatal(err)
		}
	}

	candidates := cachedDependencyCandidates("architectury", "1.21.1", "fabric")
	if len(candidates) != 1 || candidates[0].Platform != "curseforge" || candidates[0].ProjectID != "419699" {
		t.Fatalf("associated candidates = %#v, want newer CurseForge candidate", candidates)
	}

	if err := storage.SetPlatformVersions("Modrinth", "unrelated", []models.ModVersion{{
		ID: "unrelated-version", VersionID: "unrelated-version", DownloadURL: "https://example.com/unrelated.jar", PublishedAt: 3, GameVersions: []string{"1.21.1"}, Loaders: []string{"fabric"}, ModIDs: []string{"architectury"},
	}}); err != nil {
		t.Fatal(err)
	}
	candidates = cachedDependencyCandidates("architectury", "1.21.1", "fabric")
	if len(candidates) != 2 {
		t.Fatalf("unrelated candidates = %#v, want chooser with 2", candidates)
	}
}

func TestRestoreCachedDependencyRevalidatesCandidateBeforeQueue(t *testing.T) {
	_, _ = setupSelectedLocalModInstance(t)
	if err := storage.OpenAt(filepath.Join(t.TempDir(), storage.CacheFileName)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(storage.Close)
	if err := storage.UpsertModPlatform(models.ModProject{
		Platform: "Modrinth", ProjectID: "architectury-api", Title: "Architectury API",
	}); err != nil {
		t.Fatal(err)
	}
	if err := storage.SetPlatformVersions("Modrinth", "architectury-api", []models.ModVersion{{
		ID: "version-1", VersionID: "version-1", FileName: "architectury.jar", DownloadURL: "https://example.com/architectury.jar",
		GameVersions: []string{"1.21.1"}, Loaders: []string{"fabric"}, ModIDs: []string{"architectury"},
	}}); err != nil {
		t.Fatal(err)
	}

	originalQueue := queueRestoredCachedDependency
	queued := 0
	var queuedRequest appstructs.ModDownloadRequest
	queueRestoredCachedDependency = func(_ *Service, req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
		queued++
		queuedRequest = req
		return appstructs.ModDownloadResult{Queued: true, VersionID: req.VersionID}
	}
	t.Cleanup(func() { queueRestoredCachedDependency = originalQueue })

	svc := New(Options{})
	stale := svc.RestoreCachedDependency(appstructs.RestoreCachedDependencyRequest{
		ModID: "architectury", Platform: "modrinth", ProjectID: "architectury-api", VersionID: "stale",
	})
	if !stale.Skipped || queued != 0 {
		t.Fatalf("stale result = %#v, queued = %d", stale, queued)
	}

	result := svc.RestoreCachedDependency(appstructs.RestoreCachedDependencyRequest{
		ModID: "Architectury", Platform: "Modrinth", ProjectID: "architectury-api", VersionID: "version-1",
	})
	if !result.Queued || queued != 1 {
		t.Fatalf("result = %#v, queued = %d", result, queued)
	}
	if queuedRequest.ProjectID != "modrinth:architectury-api" || queuedRequest.VersionID != "version-1" ||
		queuedRequest.MinecraftVersion != "1.21.1" || queuedRequest.ModLoader != "fabric" {
		t.Fatalf("queued request = %#v", queuedRequest)
	}
}
