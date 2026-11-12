// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package global

import (
	"sync"

	"github.com/link-fgfgui/mod-downloader/core/logging"
	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"

	modrinth "codeberg.org/jmansfield/go-modrinth/modrinth"
	curseforge "github.com/sjet47/go-curseforge"
)

var (
	mu                          sync.RWMutex
	minecraftDir                string
	cachedVersionsMinecraftDir  string
	versions                    []structs.VersionInfo
	versionMap                  map[string]structs.VersionInfo
	minecraftReleaseVersions    []string
	curseforgeClient            *curseforge.Client
	modrinthClient              *modrinth.Client
	selectedVersionKey          string
	selectedVersionMinecraftDir string
)

// GetMinecraftDir returns the currently selected .minecraft directory.
func GetMinecraftDir() string {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global minecraft dir read", "minecraftDir", minecraftDir)
	return minecraftDir
}

// SetMinecraftDir sets the selected .minecraft directory.
func SetMinecraftDir(dir string) {
	mu.Lock()
	defer mu.Unlock()
	previous := minecraftDir
	minecraftDir = dir
	logging.Info("global minecraft dir set", "previousDir", previous, "minecraftDir", dir)
}

// GetVersions returns a snapshot of the cached version list.
func GetVersions() []structs.VersionInfo {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global versions read", "minecraftDir", cachedVersionsMinecraftDir, "versionCount", len(versions))
	return cloneVersionInfos(versions)
}

// GetVersionsForDir returns a snapshot of the cached version list for dir when
// it matches the cache owner.
func GetVersionsForDir(dir string) ([]structs.VersionInfo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if cachedVersionsMinecraftDir == "" || cachedVersionsMinecraftDir != dir {
		logging.Debug("global versions cache miss", "requestedDir", dir, "cacheDir", cachedVersionsMinecraftDir)
		return nil, false
	}
	logging.Debug("global versions cache hit", "minecraftDir", dir, "versionCount", len(versions))
	return cloneVersionInfos(versions), true
}

// SetVersions updates the cached version list for the current Minecraft directory.
func SetVersions(v []structs.VersionInfo) {
	mu.Lock()
	defer mu.Unlock()
	setVersionsLocked(minecraftDir, v)
}

// SetVersionsForDir updates the cached version list for dir.
func SetVersionsForDir(dir string, v []structs.VersionInfo) {
	mu.Lock()
	defer mu.Unlock()
	setVersionsLocked(dir, v)
}

func setVersionsLocked(dir string, v []structs.VersionInfo) {
	cachedVersionsMinecraftDir = dir
	versions = cloneVersionInfos(v)
	versionMap = make(map[string]structs.VersionInfo, len(v)*2)
	for _, version := range versions {
		if version.Name != "" {
			versionMap[version.Name] = version
		}
		if version.ID != "" {
			versionMap[version.ID] = version
		}
	}
	if len(v) == 0 {
		selectedVersionKey = ""
		selectedVersionMinecraftDir = ""
	} else if selectedVersionMinecraftDir != dir {
		selectedVersionKey = ""
		selectedVersionMinecraftDir = dir
	} else if selectedVersionKey != "" {
		if _, ok := versionMap[selectedVersionKey]; !ok {
			selectedVersionKey = ""
		}
	}
	logging.Debug("global versions cache set", "minecraftDir", dir, "versionCount", len(v), "keyCount", len(versionMap))
}

func versionKey(version structs.VersionInfo) string {
	if version.ID != "" {
		return version.ID
	}
	return version.Name
}

// cloneVersionInfos returns a deep-enough copy for cross-goroutine readers.
// Strings are immutable; every slice reachable from VersionInfo is copied.
func cloneVersionInfos(v []structs.VersionInfo) []structs.VersionInfo {
	if v == nil {
		return nil
	}
	out := make([]structs.VersionInfo, len(v))
	for i, version := range v {
		out[i] = cloneVersionInfo(version)
	}
	return out
}

func cloneVersionInfo(version structs.VersionInfo) structs.VersionInfo {
	version.Mods = cloneModInfos(version.Mods)
	return version
}

func cloneModInfos(mods []structs.ModInfo) []structs.ModInfo {
	if mods == nil {
		return nil
	}
	out := make([]structs.ModInfo, len(mods))
	for i, mod := range mods {
		out[i] = mod
		out[i].Categories = cloneStrings(mod.Categories)
		out[i].Loaders = cloneStrings(mod.Loaders)
		out[i].Authors = cloneStrings(mod.Authors)
		out[i].Provides = cloneStrings(mod.Provides)
		out[i].JIJMods = append([]structs.JIJModInfo(nil), mod.JIJMods...)
		out[i].Dependencies = append([]structs.LocalModDependency(nil), mod.Dependencies...)
		out[i].Recommends = append([]structs.LocalModDependency(nil), mod.Recommends...)
		out[i].Breaks = append([]structs.LocalModDependency(nil), mod.Breaks...)
	}
	return out
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

// InvalidateVersions clears the cached version list and lookup map.
func InvalidateVersions() {
	mu.Lock()
	defer mu.Unlock()
	previousDir := cachedVersionsMinecraftDir
	previousCount := len(versions)
	cachedVersionsMinecraftDir = ""
	versions = nil
	versionMap = nil
	logging.Info("global versions cache invalidated", "previousDir", previousDir, "previousVersionCount", previousCount)
}

// GetVersionByKey returns the cached version with the given ID or display
// name.
func GetVersionByKey(key string) (structs.VersionInfo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	version, ok := versionMap[key]
	logging.Debug("global version read by key", "key", key, "hit", ok)
	return cloneVersionInfo(version), ok
}

// GetSelectedVersion returns the selected instance, or the zero value when the
// selection belongs to another Minecraft directory.
func GetSelectedVersion() structs.VersionInfo {
	mu.RLock()
	defer mu.RUnlock()
	if selectedVersionMinecraftDir != minecraftDir || selectedVersionKey == "" {
		logging.Debug("global selected version read", "hit", false, "key", selectedVersionKey, "minecraftDir", minecraftDir, "selectedDir", selectedVersionMinecraftDir)
		return structs.VersionInfo{}
	}
	if version, ok := versionMap[selectedVersionKey]; ok {
		logging.Debug("global selected version read", "hit", true, "key", selectedVersionKey, "id", version.ID, "name", version.Name, "modLoader", version.ModLoader)
		return cloneVersionInfo(version)
	}
	logging.Debug("global selected version read", "hit", false, "key", selectedVersionKey, "minecraftDir", minecraftDir)
	return structs.VersionInfo{}
}

// SetSelectedVersion records the selected instance and refreshes its cached
// entry when the instance is already known.
func SetSelectedVersion(version structs.VersionInfo) {
	mu.Lock()
	defer mu.Unlock()
	previousKey := selectedVersionKey
	selectedVersionKey = versionKey(version)
	selectedVersionMinecraftDir = minecraftDir
	if _, ok := versionMap[selectedVersionKey]; ok {
		next := append([]structs.VersionInfo(nil), versions...)
		for i := range next {
			if versionKey(next[i]) == selectedVersionKey {
				next[i] = cloneVersionInfo(version)
				break
			}
		}
		versions = next
		if version.Name != "" {
			versionMap[version.Name] = cloneVersionInfo(version)
		}
		if version.ID != "" {
			versionMap[version.ID] = cloneVersionInfo(version)
		}
	}
	logging.Debug("global selected version set", "previousKey", previousKey, "key", selectedVersionKey, "minecraftDir", selectedVersionMinecraftDir)
}

// GetMinecraftReleaseVersions returns a snapshot of the cached official
// Minecraft release ids.
func GetMinecraftReleaseVersions() []string {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global minecraft release versions read", "versionCount", len(minecraftReleaseVersions))
	return cloneStrings(minecraftReleaseVersions)
}

// SetMinecraftReleaseVersions updates the cached official Minecraft release ids.
func SetMinecraftReleaseVersions(versions []string) {
	mu.Lock()
	defer mu.Unlock()
	minecraftReleaseVersions = cloneStrings(versions)
	logging.Info("global minecraft release versions set", "versionCount", len(versions))
}

// GetCurseForgeClient returns the configured CurseForge client, or nil.
func GetCurseForgeClient() *curseforge.Client {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global curseforge client read", "configured", curseforgeClient != nil)
	return curseforgeClient
}

// SetCurseForgeClient installs the process-wide CurseForge client.
func SetCurseForgeClient(client *curseforge.Client) {
	mu.Lock()
	defer mu.Unlock()
	curseforgeClient = client
	logging.Info("global curseforge client set", "configured", client != nil)
}

// GetModrinthClient returns the configured Modrinth client, or nil.
func GetModrinthClient() *modrinth.Client {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global modrinth client read", "configured", modrinthClient != nil)
	return modrinthClient
}

// SetModrinthClient installs the process-wide Modrinth client.
func SetModrinthClient(client *modrinth.Client) {
	mu.Lock()
	defer mu.Unlock()
	modrinthClient = client
	logging.Info("global modrinth client set", "configured", client != nil)
}
