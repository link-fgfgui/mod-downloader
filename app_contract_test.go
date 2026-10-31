package main

import (
	"reflect"
	"testing"
)

var expectedAppMethods = []string{
	"GetAppVersion", "GetStartupStatus", "Quit",
	"SearchMods", "ListMatchingProjectVersions", "LookupProjectBySlug",
	"GetPinnedModVersion", "PinModVersion", "ListPinnedMods", "UnpinMod",
	"ListFavoriteLists", "CreateFavoriteList", "RenameFavoriteList", "DeleteFavoriteList",
	"UpdateFavoriteListMetadata", "ReorderFavoriteLists", "ListFavoriteMods", "ListFavoriteContents",
	"AddFavoriteMod", "AddFavoriteModsToLists", "CopyFavoriteListToList",
	"PreviewFavoriteListCrossVersionCopy", "ApplyFavoriteListCrossVersionCopy",
	"AddFavoriteListReference", "RemoveFavoriteListReference", "ListFavoriteListRefs",
	"RemoveFavoriteMod", "ExportFavoriteListPackwizZip",
	"GetMinecraftReleaseVersions", "GetPreferences", "GetSettings", "ChooseCacheDir",
	"SaveTheme", "SaveLanguage", "SaveAnimationSettings", "SaveSimpleModeSettings",
	"SaveUnusedDependencyCleanupSettings", "SaveMCIMSettings", "SaveJijDetectionSettings",
	"SavePreferredReleaseTypeSettings", "SaveHardwareAccelerationSettings", "SaveNetworkSettings",
	"SaveCacheDirPreference", "SaveApiKeys",
	"QueueModDownload", "CheckDependencyFulfillment", "RestoreCachedDependency",
	"GetDownloadQueueState", "GetUsageStats", "CancelDownload", "RetryDownload", "RemoveDownload",
	"DismissOptionalDependencyReminder", "ClearOptionalDependencyReminders", "InstallOptionalDependencies",
	"AnalyzeBatchIncompatibleConflicts", "GetDownloadStates",
	"ChooseMinecraftDir", "GetMinecraftDir", "ValidateMinecraftDir", "GetVersions",
	"GetSelectedVersion", "RefreshVersions", "RefreshSelectedVersionMods", "SelectVersion",
	"ToggleConnectorLoader", "ApplyLocalModBatchOperation", "AnalyzeLocalModDisableImpact",
	"OpenLocalModLocation", "ScanUnusedDependencies",
}

func TestAppBindingContract(t *testing.T) {
	if len(expectedAppMethods) != 70 {
		t.Fatalf("contract contains %d business methods, want 70", len(expectedAppMethods))
	}
	expected := make(map[string]bool, len(expectedAppMethods)+2)
	for _, name := range expectedAppMethods {
		expected[name] = true
	}
	expected["ServiceStartup"] = true
	expected["ServiceShutdown"] = true

	typeOfApp := reflect.TypeOf((*App)(nil))
	if typeOfApp.NumMethod() != len(expected) {
		t.Fatalf("App exports %d methods, want %d", typeOfApp.NumMethod(), len(expected))
	}
	for i := 0; i < typeOfApp.NumMethod(); i++ {
		name := typeOfApp.Method(i).Name
		if !expected[name] {
			t.Errorf("unexpected exported App method %q", name)
		}
		delete(expected, name)
	}
	for name := range expected {
		t.Errorf("missing App method %q", name)
	}
}

func TestAppEventContract(t *testing.T) {
	events := []string{
		minecraftDirChangedEvent,
		selectedVersionChangedEvent,
		searchModsUpdatedEvent,
		extensionModsAcceptedEvent,
		downloadStatesUpdatedEvent,
		downloadQueueUpdatedEvent,
		downloadFailedEvent,
		downloadCompletedEvent,
		usageStatsUpdatedEvent,
	}
	if len(events) != 9 {
		t.Fatalf("event contract contains %d events, want 9", len(events))
	}
	seen := make(map[string]bool, len(events))
	for _, name := range events {
		if name == "" || seen[name] {
			t.Fatalf("empty or duplicate event %q", name)
		}
		seen[name] = true
	}
}
