package minecraft

import (
	"os"
	"path/filepath"
	"strings"

	structs "mod-downloader/structs/minecraft"
)

// GameDirVersionLoader loads recognized version entries from a directory that
// directly contains a versions/ subfolder.
type GameDirVersionLoader func(gameDir string) []structs.VersionInfo

type launcherLayout interface {
	Matches(root string) bool
	LoadVersions(root string, loadGameDir GameDirVersionLoader) []structs.VersionInfo
	VersionDir(root string, version structs.VersionInfo) string
}

var launcherLayouts = []launcherLayout{
	prismLayout{},
	standardMinecraftLayout{},
}

// LoadLauncherVersions loads recognized app instances from a selected launcher
// root. It checks launcher-specific layouts first and falls back to a standard
// .minecraft directory.
func LoadLauncherVersions(root string, loadGameDir GameDirVersionLoader) []structs.VersionInfo {
	root = strings.TrimSpace(root)
	if root == "" || loadGameDir == nil {
		return nil
	}
	return launcherLayoutFor(root).LoadVersions(root, loadGameDir)
}

func launcherLayoutFor(root string) launcherLayout {
	for _, layout := range launcherLayouts {
		if layout.Matches(root) {
			return layout
		}
	}
	return standardMinecraftLayout{}
}

type standardMinecraftLayout struct{}

func (standardMinecraftLayout) Matches(root string) bool {
	return false
}

func (standardMinecraftLayout) LoadVersions(root string, loadGameDir GameDirVersionLoader) []structs.VersionInfo {
	return loadGameDir(root)
}

func (standardMinecraftLayout) VersionDir(root string, version structs.VersionInfo) string {
	folder := VersionFolderName(version)
	if root == "" || folder == "" {
		return ""
	}
	return filepath.Join(root, "versions", folder)
}

type prismLayout struct{}

func (prismLayout) Matches(root string) bool {
	return IsPrismInstancesDir(root)
}

func (prismLayout) LoadVersions(root string, loadGameDir GameDirVersionLoader) []structs.VersionInfo {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}

	var infos []structs.VersionInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		instanceName := entry.Name()
		instanceDir := filepath.Join(root, instanceName)
		if !isPrismInstanceDir(instanceDir) {
			continue
		}
		gameDir := PrismInstanceGameDir(instanceDir)
		instanceVersions := loadGameDir(gameDir)
		if len(instanceVersions) == 0 {
			continue
		}
		info := instanceVersions[0]
		info.ID = MakePrismVersionID(instanceName, info.ID)
		info.Name = instanceName
		infos = append(infos, info)
	}
	return infos
}

func (prismLayout) VersionDir(root string, version structs.VersionInfo) string {
	instanceName, folder, ok := SplitPrismVersionID(version.ID)
	if root == "" || !ok || folder == "" {
		return ""
	}
	instanceDir := filepath.Join(root, instanceName)
	gameDir := PrismInstanceGameDir(instanceDir)
	return filepath.Join(gameDir, "versions", folder)
}
