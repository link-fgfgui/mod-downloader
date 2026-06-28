package minecraft

import (
	"os"
	"path/filepath"
	"strings"

	structs "mod-downloader/structs/minecraft"
)

// PrismInstanceGameDirName is the conventional Prism Launcher subfolder that
// holds the actual Minecraft game files (versions/, mods/, etc.). Prism creates
// this folder inside each instance directory by default.
const PrismInstanceGameDirName = ".minecraft"

// prismInstanceConfigFiles are marker files that uniquely identify a Prism
// Launcher instance directory. Either one is sufficient for detection.
var prismInstanceConfigFiles = []string{"mmc-pack.json", "instance.cfg"}

// prismVersionIDSeparator separates the Prism instance name from the version
// folder name in a composite version ID. A forward slash is used regardless of
// the host OS so composite IDs stay stable across platforms; Prism instance
// names and Minecraft version folder names never contain "/" on any OS.
const prismVersionIDSeparator = "/"

// IsPrismInstancesDir reports whether dir is the parent "instances/" folder
// used by Prism Launcher: it must contain at least one subdirectory that looks
// like a Prism instance (has a .minecraft/ subfolder or one of the Prism marker
// files). The per-instance detection is integrated into this scan and not
// exposed separately — there is no single-instance selection support.
func IsPrismInstancesDir(dir string) bool {
	if dir == "" {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if isPrismInstanceDir(filepath.Join(dir, entry.Name())) {
			return true
		}
	}
	return false
}

// isPrismInstanceDir reports whether dir is a single Prism Launcher instance
// directory. It returns true when dir contains a .minecraft/ subfolder, or
// contains one of the Prism marker files (mmc-pack.json / instance.cfg).
// This is a private helper used only within the Prism instances/ scan.
func isPrismInstanceDir(dir string) bool {
	if dir == "" {
		return false
	}
	if info, err := os.Stat(filepath.Join(dir, PrismInstanceGameDirName)); err == nil && info.IsDir() {
		return true
	}
	for _, marker := range prismInstanceConfigFiles {
		if info, err := os.Stat(filepath.Join(dir, marker)); err == nil && !info.IsDir() {
			return true
		}
	}
	return false
}

// PrismInstanceGameDir returns the directory holding versions/ and mods/ for a
// Prism instance directory. It prefers <instanceDir>/.minecraft and falls back
// to <instanceDir> itself when no .minecraft/ subfolder exists (some Prism
// instances are configured to use the instance root as the game dir).
func PrismInstanceGameDir(instanceDir string) string {
	if instanceDir == "" {
		return ""
	}
	gameDir := filepath.Join(instanceDir, PrismInstanceGameDirName)
	if info, err := os.Stat(gameDir); err == nil && info.IsDir() {
		return gameDir
	}
	return instanceDir
}

// MakePrismVersionID builds the composite version ID for a version that lives
// inside a Prism instance directory. Returns versionFolder unchanged when
// instanceName is empty.
func MakePrismVersionID(instanceName, versionFolder string) string {
	instanceName = strings.TrimSpace(instanceName)
	versionFolder = strings.TrimSpace(versionFolder)
	if instanceName == "" || versionFolder == "" {
		return versionFolder
	}
	return instanceName + prismVersionIDSeparator + versionFolder
}

// SplitPrismVersionID splits a composite Prism version ID of the form
// "<instanceName>/<versionFolder>" into its parts. ok is false when id is not
// in the composite form (no separator or empty component).
func SplitPrismVersionID(id string) (instanceName, versionFolder string, ok bool) {
	id = strings.TrimSpace(id)
	idx := strings.Index(id, prismVersionIDSeparator)
	if idx <= 0 || idx == len(id)-1 {
		return "", "", false
	}
	return id[:idx], id[idx+1:], true
}

// VersionFolderName returns the version folder name (the directory under
// versions/ that contains the manifest and mods/) for the given version. For
// composite Prism version IDs this is the part after the separator; otherwise
// it falls back to the version's ID, then its Name.
func VersionFolderName(version structs.VersionInfo) string {
	if _, folder, ok := SplitPrismVersionID(version.ID); ok {
		return folder
	}
	if id := strings.TrimSpace(version.ID); id != "" {
		return id
	}
	return strings.TrimSpace(version.Name)
}

// VersionDirPath returns the absolute path to the version directory for the
// given version, given the user-selected minecraft directory. The result is
// where versions/<folder>/mods/ lives. Behavior:
//   - For composite Prism version IDs (when mcDir is a Prism "instances/"
//     folder): mcDir/<instanceName>/.minecraft/versions/<folder>, with the
//     .minecraft part skipped when the instance has no such subfolder.
//   - Otherwise: mcDir/versions/<folder> (the standard .minecraft layout).
func VersionDirPath(mcDir string, version structs.VersionInfo) string {
	if mcDir == "" {
		return ""
	}
	for _, layout := range launcherLayouts {
		if versionDir := layout.VersionDir(mcDir, version); versionDir != "" {
			return versionDir
		}
	}
	return ""
}
