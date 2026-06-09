package minecraft

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	structs "mod-downloader/structs/minecraft"
)

// SimplifyPathWithEnv takes an absolute path and simplifies it using the longest-matching environment variable.
func SimplifyPathWithEnv(dirPath2 string) string {
	// 1. Clean the input path to ensure a canonical format (normalize extra slashes, etc.)
	var dirPath = filepath.Clean(dirPath2)

	bestMatchEnv := ""
	bestMatchValue := ""

	// 2. Iterate over all environment variables of the current process.
	// os.Environ() returns a slice of "KEY=VALUE" strings.
	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}
		key, val := pair[0], pair[1]

		// Skip empty values, invalid paths, or paths that are too short
		// (e.g. root "/" has little value as a substitution).
		val = filepath.Clean(val)
		if val == "" || val == "." || val == "/" || len(val) <= 1 {
			continue
		}

		// 3. Check whether the env value is a prefix of the input path.
		// When using strings.HasPrefix, watch for boundary issues to avoid
		// "/Users/jack" falsely matching "/Users/jackson".
		// Ensure the next character after the match is a path separator,
		// or that the two paths are exactly equal.
		if strings.HasPrefix(dirPath, val) {
			rel, err := filepath.Rel(val, dirPath)
			if err != nil || strings.HasPrefix(rel, "..") {
				continue // Not a true sub-path
			}

			// 4. Pick the "best" match: the longest env value gives the most precise substitution.
			if len(val) > len(bestMatchValue) {
				bestMatchValue = val
				bestMatchEnv = key
			}
		}
	}

	// 5. If a suitable match was found, perform the substitution.
	if bestMatchEnv != "" {
		rel, _ := filepath.Rel(bestMatchValue, dirPath)
		if rel == "." {
			return "$" + bestMatchEnv
		}
		// Use filepath.Join to keep platform-native path separators.
		return filepath.Join("$"+bestMatchEnv, rel)
	}

	// No matching env variable found; return the cleaned path as-is.
	return dirPath2
}

func CheckManifest(jsonPath string) (structs.VersionInfo, bool) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return structs.VersionInfo{}, false
	}

	var mv structs.MinecraftVersion
	if err := json.Unmarshal(data, &mv); err != nil {
		return structs.VersionInfo{}, false
	}

	minecraftVersion := ""
	modLoader := ""
	for _, p := range mv.Patches {
		if p.ID == "game" && p.Version != "" {
			minecraftVersion = p.Version
		} else if p.ID != "" {
			modLoader = p.ID
		}
	}
	if minecraftVersion == "" || modLoader == "" {
		return structs.VersionInfo{}, false
	}

	// instanceID is the version folder name (== json filename), unique per instance.
	// It must NOT be the Minecraft version: multiple instances can share one MC
	// version (e.g. a Fabric and a NeoForge pack both on 1.21.1), and using the MC
	// version as ID collides them in the version lookup map.
	instanceID := strings.TrimSuffix(filepath.Base(jsonPath), filepath.Ext(jsonPath))
	name := mv.Name
	if name == "" {
		name = instanceID
	}

	return structs.VersionInfo{
		Name:             name,
		ID:               instanceID,
		MinecraftVersion: minecraftVersion,
		ModLoader:        modLoader,
	}, true
}
