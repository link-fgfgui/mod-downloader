package global

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"mod-downloader/logging"
)

var (
	hardlinkIndexMu         sync.RWMutex
	hardlinkIndex           = make(map[string]string)
	hardlinkIndexGeneration uint64
)

func HardlinkIndexAdd(sha1, absolutePath string) {
	HardlinkIndexAddForGeneration(sha1, absolutePath, HardlinkIndexGeneration())
}

func HardlinkIndexAddForGeneration(sha1, absolutePath string, generation uint64) bool {
	sha1 = strings.ToLower(strings.TrimSpace(sha1))
	absolutePath = filepath.Clean(strings.TrimSpace(absolutePath))
	if sha1 == "" || absolutePath == "" {
		return false
	}
	hardlinkIndexMu.Lock()
	defer hardlinkIndexMu.Unlock()
	if generation != hardlinkIndexGeneration {
		return false
	}
	if existing, ok := hardlinkIndex[sha1]; ok && existing == absolutePath {
		return false
	}
	hardlinkIndex[sha1] = absolutePath
	logging.Debug("hardlink index add", "sha1", sha1, "path", absolutePath)
	return true
}

func HardlinkIndexRemove(absolutePath string) {
	absolutePath = filepath.Clean(strings.TrimSpace(absolutePath))
	if absolutePath == "" {
		return
	}
	hardlinkIndexMu.Lock()
	defer hardlinkIndexMu.Unlock()
	for sha1, p := range hardlinkIndex {
		if filepath.Clean(p) == absolutePath {
			delete(hardlinkIndex, sha1)
			logging.Debug("hardlink index remove", "sha1", sha1, "path", absolutePath)
		}
	}
}

func HardlinkIndexLookup(sha1 string) (string, bool) {
	sha1 = strings.ToLower(strings.TrimSpace(sha1))
	if sha1 == "" {
		return "", false
	}
	hardlinkIndexMu.RLock()
	path, ok := hardlinkIndex[sha1]
	hardlinkIndexMu.RUnlock()
	if !ok {
		return "", false
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			hardlinkIndexMu.Lock()
			if stale, ok := hardlinkIndex[sha1]; ok && filepath.Clean(stale) == filepath.Clean(path) {
				delete(hardlinkIndex, sha1)
				logging.Debug("hardlink index stale entry removed", "sha1", sha1, "path", path)
			}
			hardlinkIndexMu.Unlock()
		}
		return "", false
	}
	return path, true
}

func HardlinkIndexClear() {
	hardlinkIndexMu.Lock()
	defer hardlinkIndexMu.Unlock()
	count := len(hardlinkIndex)
	hardlinkIndex = make(map[string]string)
	hardlinkIndexGeneration++
	logging.Info("hardlink index cleared", "previousCount", count, "generation", hardlinkIndexGeneration)
}

func HardlinkIndexGeneration() uint64 {
	hardlinkIndexMu.RLock()
	defer hardlinkIndexMu.RUnlock()
	return hardlinkIndexGeneration
}
