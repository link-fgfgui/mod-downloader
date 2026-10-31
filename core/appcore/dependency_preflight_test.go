package appcore

import (
	"testing"

	"github.com/link-fgfgui/mod-downloader-core/configs"
	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func TestBuildPreflightEntriesClassifiesRequiredOnly(t *testing.T) {
	target := models.ModVersion{
		Platform: "modrinth",
		Dependencies: []models.ModDependency{
			{DependencyProjectID: "fulfilled", DependencyType: "required", DependencyVersionID: "1.0"},
			{DependencyProjectID: "optional", DependencyType: "optional"},
			{DependencyProjectID: "conflict", DependencyType: "required"},
			{DependencyProjectID: "missing", DependencyType: "required", DependencyVersionID: "2.0"},
		},
	}
	parent := appstructs.ModDownloadRequest{
		ProjectID:        "modrinth:parent",
		MinecraftVersion: "1.21.1",
		ModLoader:        "fabric",
	}
	local := map[string]structs.ModInfo{
		localProjectKey("modrinth", "fulfilled"): {
			OnlineName:    "Fulfilled Mod",
			OnlineVersion: "1.0.0",
		},
	}
	classify := func(req appstructs.ModDownloadRequest) string {
		switch req.ProjectID {
		case "modrinth:fulfilled":
			return modbridge.BtnStatusInstalled
		case "modrinth:conflict":
			return modbridge.BtnStatusConflict
		default:
			return modbridge.BtnStatusNew
		}
	}

	entries := buildPreflightEntries(target, parent, local, classify)
	if len(entries) != 3 {
		t.Fatalf("entries = %#v, want 3 required dependencies", entries)
	}
	// Conflicts rank before missing, which rank before fulfilled.
	wantOrder := []string{"conflict", "missing", "fulfilled"}
	for index, want := range wantOrder {
		if entries[index].ProjectID != want {
			t.Fatalf("entry[%d] = %q, want %q (all: %#v)", index, entries[index].ProjectID, want, entries)
		}
	}
	if entries[0].Status != appstructs.DependencyPreflightConflict {
		t.Fatalf("conflict status = %q", entries[0].Status)
	}
	if entries[1].Status != appstructs.DependencyPreflightMissing || entries[1].RequiredVersion != "2.0" {
		t.Fatalf("missing entry = %#v", entries[1])
	}
	if entries[2].Status != appstructs.DependencyPreflightFulfilled {
		t.Fatalf("fulfilled status = %q", entries[2].Status)
	}
	if entries[2].Title != "Fulfilled Mod" || entries[2].LocalName != "Fulfilled Mod" || entries[2].LocalVersion != "1.0.0" {
		t.Fatalf("fulfilled local display = %#v", entries[2])
	}
}

func TestPreflightResultCountsStatuses(t *testing.T) {
	result := preflightResultFromEntries([]appstructs.DependencyPreflightEntry{
		{Status: appstructs.DependencyPreflightFulfilled},
		{Status: appstructs.DependencyPreflightFulfilled},
		{Status: appstructs.DependencyPreflightMissing},
		{Status: appstructs.DependencyPreflightConflict},
	})
	if result.Fulfilled != 2 || result.Missing != 1 || result.Conflicts != 1 {
		t.Fatalf("result = %#v", result)
	}
}

func TestPreflightStatusRank(t *testing.T) {
	if preflightStatusRank(appstructs.DependencyPreflightConflict) >= preflightStatusRank(appstructs.DependencyPreflightMissing) {
		t.Fatal("conflicts must rank before missing dependencies")
	}
	if preflightStatusRank(appstructs.DependencyPreflightMissing) >= preflightStatusRank(appstructs.DependencyPreflightFulfilled) {
		t.Fatal("missing dependencies must rank before fulfilled dependencies")
	}
}

func TestLocalProjectKeyNormalizes(t *testing.T) {
	if got := localProjectKey(" Modrinth ", " Sodium "); got != "modrinth|sodium" {
		t.Fatalf("localProjectKey = %q", got)
	}
}

func TestLocalModsByOnlineProjectSkipsMissingIDs(t *testing.T) {
	mods := localModsByOnlineProject([]structs.ModInfo{
		{OnlinePlatform: "modrinth", OnlineProjectID: "sodium", OnlineName: "Sodium"},
		{OnlinePlatform: "modrinth", OnlineProjectID: "sodium", OnlineName: "Sodium Duplicate"},
		{OnlinePlatform: "modrinth", OnlineName: "No Project ID"},
		{OnlinePlatform: "", OnlineProjectID: "orphan", OnlineName: "Orphan"},
	})
	if len(mods) != 2 {
		t.Fatalf("indexed mods = %#v, want 2", mods)
	}
	if mod, ok := mods[localProjectKey("modrinth", "sodium")]; !ok || mod.OnlineName != "Sodium" {
		t.Fatalf("sodium entry = %#v, %v", mod, ok)
	}
	if _, ok := mods[localProjectKey("", "orphan")]; !ok {
		t.Fatalf("orphan entry missing: %#v", mods)
	}
}

func TestHasPreflightDependency(t *testing.T) {
	if hasPreflightDependency(nil) {
		t.Fatal("nil dependencies reported as present")
	}
	if hasPreflightDependency([]models.ModDependency{{DependencyType: "embedded"}}) {
		t.Fatal("embedded dependency reported as preflight metadata")
	}
	for _, dependencyType := range []string{"required", "optional", "incompatible"} {
		if !hasPreflightDependency([]models.ModDependency{{DependencyType: dependencyType}}) {
			t.Fatalf("%s dependency not detected", dependencyType)
		}
	}
}

func TestCheckDependencyFulfillmentReturnsEmptyWithoutTarget(t *testing.T) {
	previous := global.GetSelectedVersion()
	t.Cleanup(func() {
		global.SetSelectedVersion(previous)
	})
	global.SetSelectedVersion(structs.VersionInfo{})

	svc := New(Options{Config: &configs.Config{}})
	result := svc.CheckDependencyFulfillment(appstructs.DependencyPreflightRequest{
		ProjectID: "",
		Result:    models.ModProject{Platform: "modrinth"},
	})
	if len(result.Entries) != 0 || result.Fulfilled != 0 || result.Missing != 0 || result.Conflicts != 0 {
		t.Fatalf("empty request result = %#v", result)
	}
}
