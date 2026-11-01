// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package global

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/link-fgfgui/mod-downloader/core/logging"
)

var (
	hardlinkIndexMu         sync.RWMutex
	hardlinkIndex           = make(map[string]string)
	hardlinkIndexGeneration uint64
)

// HardlinkIndexAdd records the newest path for sha1 in the current scan
// generation.
func HardlinkIndexAdd(sha1, absolutePath string) {
	HardlinkIndexAddForGeneration(sha1, absolutePath, HardlinkIndexGeneration())
}

// HardlinkIndexAddForGeneration records absolutePath for sha1 only when
// generation still matches the active scan, preventing stale scans from
// overwriting newer results.
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

// HardlinkIndexRemove drops every SHA1 entry that points at absolutePath.
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

// HardlinkIndexLookup returns the indexed path for sha1, dropping the entry
// when the file no longer exists.
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

// HardlinkIndexClear empties the index and starts a new scan generation.
func HardlinkIndexClear() {
	hardlinkIndexMu.Lock()
	defer hardlinkIndexMu.Unlock()
	count := len(hardlinkIndex)
	hardlinkIndex = make(map[string]string)
	hardlinkIndexGeneration++
	logging.Info("hardlink index cleared", "previousCount", count, "generation", hardlinkIndexGeneration)
}

// HardlinkIndexGeneration returns the current scan generation.
func HardlinkIndexGeneration() uint64 {
	hardlinkIndexMu.RLock()
	defer hardlinkIndexMu.RUnlock()
	return hardlinkIndexGeneration
}
