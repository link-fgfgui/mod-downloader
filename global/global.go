package global

import (
	"sync"

	"mod-downloader/logging"
	structs "mod-downloader/structs/minecraft"

	modrinth "codeberg.org/jmansfield/go-modrinth/modrinth"
	curseforge "github.com/sjet47/go-curseforge"
)

var (
	mu                       sync.RWMutex
	mcDir                    string
	versionsMinecraftDir     string
	versions                 []structs.VersionInfo
	versionMap               map[string]structs.VersionInfo
	minecraftReleaseVersions []string
	curseforgeClient         *curseforge.Client
	modrinthClient           *modrinth.Client
	selectedVersionKey       string
	selectedVersionDir       string
)

// GetMinecraftDir returns the currently selected .minecraft directory.
func GetMinecraftDir() string {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global minecraft dir read", "minecraftDir", mcDir)
	return mcDir
}

// SetMinecraftDir sets the selected .minecraft directory.
func SetMinecraftDir(dir string) {
	mu.Lock()
	defer mu.Unlock()
	previous := mcDir
	mcDir = dir
	logging.Info("global minecraft dir set", "previousDir", previous, "minecraftDir", dir)
}

// GetVersions returns the cached version list.
func GetVersions() []structs.VersionInfo {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global versions read", "minecraftDir", versionsMinecraftDir, "versionCount", len(versions))
	return versions
}

// GetVersionsForDir returns the cached version list for dir when it matches the cache owner.
func GetVersionsForDir(dir string) ([]structs.VersionInfo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if versionsMinecraftDir == "" || versionsMinecraftDir != dir {
		logging.Debug("global versions cache miss", "requestedDir", dir, "cacheDir", versionsMinecraftDir)
		return nil, false
	}
	logging.Debug("global versions cache hit", "minecraftDir", dir, "versionCount", len(versions))
	return versions, true
}

// SetVersions updates the cached version list for the current Minecraft directory.
func SetVersions(v []structs.VersionInfo) {
	mu.Lock()
	defer mu.Unlock()
	setVersionsLocked(mcDir, v)
}

// SetVersionsForDir updates the cached version list for dir.
func SetVersionsForDir(dir string, v []structs.VersionInfo) {
	mu.Lock()
	defer mu.Unlock()
	setVersionsLocked(dir, v)
}

func setVersionsLocked(dir string, v []structs.VersionInfo) {
	versionsMinecraftDir = dir
	versions = v
	versionMap = make(map[string]structs.VersionInfo, len(v)*2)
	for _, version := range v {
		if version.Name != "" {
			versionMap[version.Name] = version
		}
		if version.ID != "" {
			versionMap[version.ID] = version
		}
	}
	if len(v) == 0 {
		selectedVersionKey = ""
		selectedVersionDir = ""
	} else if selectedVersionDir != dir || selectedVersionKey == "" {
		selectedVersionKey = versionKey(v[0])
		selectedVersionDir = dir
	} else if _, ok := versionMap[selectedVersionKey]; !ok {
		selectedVersionKey = versionKey(v[0])
		selectedVersionDir = dir
	}
	logging.Info("global versions cache set", "minecraftDir", dir, "versionCount", len(v), "keyCount", len(versionMap))
}

func versionKey(version structs.VersionInfo) string {
	if version.Name != "" {
		return version.Name
	}
	return version.ID
}

// InvalidateVersions clears the cached version list and lookup map.
func InvalidateVersions() {
	mu.Lock()
	defer mu.Unlock()
	previousDir := versionsMinecraftDir
	previousCount := len(versions)
	versionsMinecraftDir = ""
	versions = nil
	versionMap = nil
	logging.Info("global versions cache invalidated", "previousDir", previousDir, "previousVersionCount", previousCount)
}

func GetVersionByKey(key string) (structs.VersionInfo, bool) {
	mu.RLock()
	defer mu.RUnlock()
	version, ok := versionMap[key]
	logging.Debug("global version read by key", "key", key, "hit", ok)
	return version, ok
}

func GetSelectedVersion() structs.VersionInfo {
	mu.RLock()
	defer mu.RUnlock()
	if selectedVersionDir != mcDir || selectedVersionKey == "" {
		logging.Debug("global selected version read", "hit", false, "key", selectedVersionKey, "minecraftDir", mcDir, "selectedDir", selectedVersionDir)
		return structs.VersionInfo{}
	}
	if version, ok := versionMap[selectedVersionKey]; ok {
		logging.Debug("global selected version read", "hit", true, "key", selectedVersionKey, "id", version.ID, "name", version.Name, "modLoader", version.ModLoader)
		return version
	}
	logging.Debug("global selected version read", "hit", false, "key", selectedVersionKey, "minecraftDir", mcDir)
	return structs.VersionInfo{ID: selectedVersionKey, Name: selectedVersionKey, ModLoader: "vanilla"}
}

func SetSelectedVersion(version structs.VersionInfo) {
	mu.Lock()
	defer mu.Unlock()
	previousKey := selectedVersionKey
	selectedVersionKey = versionKey(version)
	selectedVersionDir = mcDir
	logging.Info("global selected version set", "previousKey", previousKey, "key", selectedVersionKey, "minecraftDir", selectedVersionDir)
}

// GetMinecraftReleaseVersions returns the cached official Minecraft release ids.
func GetMinecraftReleaseVersions() []string {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global minecraft release versions read", "versionCount", len(minecraftReleaseVersions))
	return minecraftReleaseVersions
}

// SetMinecraftReleaseVersions updates the cached official Minecraft release ids.
func SetMinecraftReleaseVersions(versions []string) {
	mu.Lock()
	defer mu.Unlock()
	minecraftReleaseVersions = versions
	logging.Info("global minecraft release versions set", "versionCount", len(versions))
}

func GetCurseForgeClient() *curseforge.Client {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global curseforge client read", "configured", curseforgeClient != nil)
	return curseforgeClient
}

func SetCurseForgeClient(client *curseforge.Client) {
	mu.Lock()
	defer mu.Unlock()
	curseforgeClient = client
	logging.Info("global curseforge client set", "configured", client != nil)
}

func GetModrinthClient() *modrinth.Client {
	mu.RLock()
	defer mu.RUnlock()
	logging.Debug("global modrinth client read", "configured", modrinthClient != nil)
	return modrinthClient
}

func SetModrinthClient(client *modrinth.Client) {
	mu.Lock()
	defer mu.Unlock()
	modrinthClient = client
	logging.Info("global modrinth client set", "configured", client != nil)
}
