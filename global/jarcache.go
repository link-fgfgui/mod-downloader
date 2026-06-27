package global

import (
	"strings"
	"sync"

	"mod-downloader/logging"
	structs "mod-downloader/structs/minecraft"
)

var (
	jarCacheMu sync.RWMutex
	jarCache   = make(map[string][]structs.ModInfo)
)

// GetJarMetadata returns cached JAR mod metadata for the given file SHA1.
// This is a purely in-memory cache for local JAR metadata (not persisted across sessions).
func GetJarMetadata(sha1 string) ([]structs.ModInfo, bool) {
	sha1 = strings.TrimSpace(sha1)
	if sha1 == "" {
		return nil, false
	}
	jarCacheMu.RLock()
	defer jarCacheMu.RUnlock()
	mods, ok := jarCache[sha1]
	if !ok || len(mods) == 0 {
		return nil, false
	}
	out := make([]structs.ModInfo, len(mods))
	copy(out, mods)
	return out, true
}

// SetJarMetadata stores parsed JAR mod metadata for the given file SHA1.
// This is a purely in-memory cache for local JAR metadata (not persisted across sessions).
func SetJarMetadata(sha1 string, mods []structs.ModInfo) {
	sha1 = strings.TrimSpace(sha1)
	if sha1 == "" || len(mods) == 0 {
		return
	}

	filtered := make([]structs.ModInfo, 0, len(mods))
	seen := make(map[string]struct{}, len(mods))
	for _, mod := range mods {
		mod.ID = strings.TrimSpace(mod.ID)
		if mod.ID == "" {
			continue
		}
		key := strings.ToLower(mod.ID)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, mod)
	}
	if len(filtered) == 0 {
		return
	}

	jarCacheMu.Lock()
	defer jarCacheMu.Unlock()
	out := make([]structs.ModInfo, len(filtered))
	copy(out, filtered)
	jarCache[sha1] = out
	logging.Debug("jar metadata set in memory cache", "sha1", sha1, "modCount", len(filtered))
}

// ClearJarMetadata clears all in-memory JAR metadata cache.
func ClearJarMetadata() {
	jarCacheMu.Lock()
	defer jarCacheMu.Unlock()
	count := len(jarCache)
	jarCache = make(map[string][]structs.ModInfo)
	logging.Info("jar metadata memory cache cleared", "entryCount", count)
}
