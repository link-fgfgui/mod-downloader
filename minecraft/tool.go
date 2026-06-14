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
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		return structs.VersionInfo{}, false
	}

	var mv structs.MinecraftVersion
	if err := json.Unmarshal(raw, &mv); err != nil {
		return structs.VersionInfo{}, false
	}

	instanceID := strings.TrimSuffix(filepath.Base(jsonPath), filepath.Ext(jsonPath))
	if id := strings.TrimSpace(mv.ID); id != "" {
		instanceID = id
	}

	name := strings.TrimSpace(mv.Name)
	if name == "" {
		name = instanceID
	}

	minecraftVersion, modLoader := manifestMinecraftVersionAndLoader(mv, raw)
	if minecraftVersion == "" {
		if detected, ok := detectMinecraftVersionForManifest(jsonPath, mv); ok {
			minecraftVersion = detected
		}
	}
	if modLoader == "" {
		modLoader = "vanilla"
	}
	if strings.TrimSpace(instanceID) == "" || strings.TrimSpace(minecraftVersion) == "" {
		return structs.VersionInfo{}, false
	}

	// ID is the version folder name / manifest id, unique per instance.
	// It must NOT be the Minecraft version: multiple instances can share one MC
	// version (e.g. a Fabric and a NeoForge pack both on 1.21.1), and using the MC
	// version as ID collides them in the version lookup map.
	return structs.VersionInfo{
		Name:             name,
		ID:               instanceID,
		MinecraftVersion: minecraftVersion,
		ModLoader:        modLoader,
	}, true
}

func manifestMinecraftVersionAndLoader(mv structs.MinecraftVersion, raw []byte) (string, string) {
	minecraftVersion := strings.TrimSpace(mv.InheritsFrom)
	modLoader := ""

	for _, p := range mv.Patches {
		id := strings.TrimSpace(p.ID)
		version := strings.TrimSpace(p.Version)
		if id == "" {
			continue
		}
		if id == "game" {
			if version != "" {
				minecraftVersion = version
			}
			continue
		}
		modLoader = id
	}

	if modLoader == "" {
		modLoader = detectModLoaderFromRawJSON(raw)
	}

	return minecraftVersion, strings.ToLower(modLoader)
}

// detectModLoaderFromRawJSON infers the mod loader by scanning the raw manifest
// JSON for known Maven coordinates, avoiding the need to parse mainClass or
// libraries into structured fields.
func detectModLoaderFromRawJSON(raw []byte) string {
	s := string(raw)
	if strings.Contains(s, "net.fabricmc:fabric-loader") {
		return "fabric"
	}
	if strings.Contains(s, "net.neoforged.fancymodloader:loader") {
		return "neoforge"
	}
	if strings.Contains(s, "net.minecraftforge:fmlloader") {
		return "forge"
	}
	return ""
}

func detectMinecraftVersionForManifest(jsonPath string, mv structs.MinecraftVersion) (string, bool) {
	for _, jarPath := range manifestJarPaths(jsonPath, mv) {
		if detected, ok := DetectMinecraftVersionFromJar(jarPath); ok {
			return detected, true
		}
	}
	return "", false
}

func manifestJarPaths(jsonPath string, mv structs.MinecraftVersion) []string {
	jarID := strings.TrimSpace(mv.Jar)
	if jarID == "" {
		jarID = strings.TrimSuffix(filepath.Base(jsonPath), filepath.Ext(jsonPath))
	}

	versionDir := filepath.Dir(jsonPath)
	paths := []string{filepath.Join(versionDir, jarID+".jar")}

	instanceID := strings.TrimSuffix(filepath.Base(jsonPath), filepath.Ext(jsonPath))
	if jarID != "" && jarID != instanceID {
		versionsDir := filepath.Dir(versionDir)
		paths = append(paths, filepath.Join(versionsDir, jarID, jarID+".jar"))
	}
	return paths
}
