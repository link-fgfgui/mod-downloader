// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/link-fgfgui/mod-downloader/core/appcore"
)

var expectedAppMethods = []string{
	"GetStartupStatus", "Quit",
	"SearchMods", "ListMatchingProjectVersions", "LookupProjectBySlug",
	"GetPinnedModVersion", "PinModVersion", "ListPinnedMods", "UnpinMod",
	"ListFavoriteLists", "CreateFavoriteListForScope", "RenameFavoriteList", "DeleteFavoriteList",
	"UpdateFavoriteListMetadata", "ReorderFavoriteLists",
	"ListFavoriteOverview", "ListFavoriteContentsForScope",
	"AddFavoriteMod", "AddFavoriteModsToLists", "CopyFavoriteListToList",
	"PreviewFavoriteListCrossVersionCopy", "ApplyFavoriteListCrossVersionCopy",
	"AddFavoriteListReference",
	"RemoveFavoriteMod", "ExportFavoriteListPackwizZipForScope",
	"GetMinecraftReleaseVersions", "GetSettings", "ChooseCacheDir",
	"SaveTheme", "SaveLanguage", "SaveAnimationSettings", "SaveSimpleModeSettings",
	"SaveUnusedDependencyCleanupSettings", "SaveMCIMSettings", "SaveJIJDetectionSettings",
	"SaveMultiSelectSettings",
	"SavePreferredReleaseTypeSettings", "SaveHardwareAccelerationSettings", "SaveNetworkSettings",
	"SaveCacheDirPreference", "SaveAPIKeys",
	"QueueModDownload", "CheckDependencyFulfillment", "RestoreCachedDependency",
	"GetDownloadQueueState", "GetUsageStats", "CancelDownload", "RetryDownload", "RemoveDownload",
	"DismissOptionalDependencyReminder", "ClearOptionalDependencyReminders", "InstallOptionalDependencies",
	"AnalyzeBatchIncompatibleConflicts", "GetDownloadStates",
	"ChooseMinecraftDir", "GetMinecraftDir", "ValidateMinecraftDir", "GetVersions",
	"GetSelectedVersion", "RefreshVersions", "RefreshSelectedVersionMods", "GetLocalModView", "SelectVersion",
	"ToggleConnectorLoader", "ApplyLocalModBatchOperation", "AnalyzeLocalModDisableImpact",
	"OpenLocalModLocation", "ScanUnusedDependencies",
}

func TestAppBindingContract(t *testing.T) {
	if len(expectedAppMethods) != 68 {
		t.Fatalf("contract contains %d business methods, want 68", len(expectedAppMethods))
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

// appBindingSignatureHash pins the full method surface (names plus parameter
// and result types). Update the constant only for an intentional breaking
// change to the binding contract; additive methods also change the hash.
const appBindingSignatureHash = "f0bd29565c3140bf9307650d7fe0d511e25ef010892310c9c6c4335445dbc4f2"

func TestAppBindingSignatureContract(t *testing.T) {
	typeOfApp := reflect.TypeOf((*App)(nil))
	signatures := make([]string, 0, typeOfApp.NumMethod())
	for i := 0; i < typeOfApp.NumMethod(); i++ {
		method := typeOfApp.Method(i)
		signatures = append(signatures, fmt.Sprintf("%s%s", method.Name, method.Type))
	}
	sort.Strings(signatures)
	hash := sha256.Sum256([]byte(strings.Join(signatures, "\n")))
	got := hex.EncodeToString(hash[:])
	if got != appBindingSignatureHash {
		t.Fatalf("App binding signature hash = %q, want %q\nsignatures:\n%s", got, appBindingSignatureHash, strings.Join(signatures, "\n"))
	}
}

func TestAppEventContract(t *testing.T) {
	events := []string{
		appcore.EventMinecraftDirChanged.Name(),
		appcore.EventSelectedVersionChanged.Name(),
		appcore.EventSearchModsUpdated.Name(),
		appcore.EventExtensionModsAccepted.Name(),
		appcore.EventDownloadStatesUpdated.Name(),
		appcore.EventDownloadQueueUpdated.Name(),
		appcore.EventDownloadFailed.Name(),
		appcore.EventDownloadCompleted.Name(),
		appcore.EventUsageStatsUpdated.Name(),
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
