package global

import (
	"strings"
	"sync"

	"mod-downloader/logging"
	structs "mod-downloader/structs/minecraft"
)

type LocalModFile struct {
	SHA1           string `json:"sha1"`
	FileName       string `json:"fileName"`
	CfFingerprint  int64  `json:"cfFingerprint,omitempty"`
	ModID          string `json:"modId,omitempty"`
	ModName        string `json:"modName,omitempty"`
	ModVersion     string `json:"modVersion,omitempty"`
	ModDescription string `json:"modDescription,omitempty"`
}

type LocalModFilePath struct {
	ID               string `json:"id"`
	FileSHA1         string `json:"fileSha1"`
	Path             string `json:"path"`
	InstanceID       string `json:"instanceId"`
	MinecraftVersion string `json:"minecraftVersion"`
	ModLoader        string `json:"modLoader"`
	Enabled          bool   `json:"enabled"`
}

var (
	localModsMu       sync.RWMutex
	localModFiles     = make(map[string][]LocalModFile)
	localModFilePaths = make(map[string]LocalModFilePath)
)

func UpsertLocalMod(info structs.ModInfo, instanceID, mcVersion, modLoader string) {
	sha1 := strings.TrimSpace(info.SHA1)
	path := strings.TrimSpace(info.Path)
	if sha1 == "" || path == "" {
		logging.Debug("local mod upsert skipped", "hasSHA1", sha1 != "", "hasPath", path != "", "instanceID", instanceID)
		return
	}

	file := LocalModFile{
		SHA1:           sha1,
		FileName:       info.FileName,
		ModID:          info.ID,
		ModName:        info.Name,
		ModVersion:     info.Version,
		ModDescription: info.Description,
	}
	filePath := LocalModFilePath{
		ID:               path,
		FileSHA1:         sha1,
		Path:             path,
		InstanceID:       strings.TrimSpace(instanceID),
		MinecraftVersion: strings.TrimSpace(mcVersion),
		ModLoader:        strings.ToLower(strings.TrimSpace(modLoader)),
		Enabled:          info.Enabled,
	}

	localModsMu.Lock()
	defer localModsMu.Unlock()
	localModFiles[sha1] = upsertLocalModFile(localModFiles[sha1], file)
	localModFilePaths[path] = filePath
	logging.Debug("local mod upserted", "sha1", sha1, "path", path, "modID", file.ModID, "instanceID", filePath.InstanceID, "minecraftVersion", filePath.MinecraftVersion, "modLoader", filePath.ModLoader)
}

func GetLocalModFileBySHA1(sha1 string) (LocalModFile, bool) {
	localModsMu.RLock()
	defer localModsMu.RUnlock()
	normalizedSHA1 := strings.TrimSpace(sha1)
	files := localModFiles[normalizedSHA1]
	ok := len(files) > 0
	logging.Debug("local mod file read by sha1", "sha1", normalizedSHA1, "hit", ok)
	if !ok {
		return LocalModFile{}, false
	}
	return files[0], true
}

func GetLocalModFilePathsBySHA1(sha1 string) []LocalModFilePath {
	sha1 = strings.TrimSpace(sha1)
	localModsMu.RLock()
	defer localModsMu.RUnlock()

	paths := make([]LocalModFilePath, 0)
	for _, path := range localModFilePaths {
		if path.FileSHA1 == sha1 {
			paths = append(paths, path)
		}
	}
	logging.Debug("local mod paths read by sha1", "sha1", sha1, "pathCount", len(paths))
	return paths
}

func GetLocalModFilePathsByVersionLoader(mcVersion, modLoader string) []LocalModFilePath {
	mcVersion = strings.TrimSpace(mcVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))
	localModsMu.RLock()
	defer localModsMu.RUnlock()

	paths := make([]LocalModFilePath, 0)
	for _, path := range localModFilePaths {
		if path.MinecraftVersion == mcVersion && path.ModLoader == modLoader {
			paths = append(paths, path)
		}
	}
	logging.Debug("local mod paths read by version loader", "minecraftVersion", mcVersion, "modLoader", modLoader, "pathCount", len(paths))
	return paths
}

// LocalModPathsInInstanceByModID 返回某实例内、所属 jar 的 ModID 匹配 modID 的本地文件路径记录。
// 每条记录带有 FileSHA1(用于按 sha1 判定状态）与 Path（用于替换时删除磁盘文件）。
func LocalModPathsInInstanceByModID(instanceID, modID string) []LocalModFilePath {
	modID = strings.ToLower(strings.TrimSpace(modID))
	instanceID = strings.TrimSpace(instanceID)
	if modID == "" {
		return nil
	}

	localModsMu.RLock()
	defer localModsMu.RUnlock()
	out := make([]LocalModFilePath, 0)
	for _, path := range localModFilePaths {
		if path.InstanceID != instanceID {
			continue
		}
		for _, file := range localModFiles[path.FileSHA1] {
			if strings.ToLower(strings.TrimSpace(file.ModID)) == modID {
				out = append(out, path)
				break
			}
		}
	}
	return out
}

// RemoveLocalModByPath 从内存表移除某个本地 mod 路径记录（替换旧版本时配合磁盘删除使用）。
func RemoveLocalModByPath(path string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	localModsMu.Lock()
	defer localModsMu.Unlock()
	if _, ok := localModFilePaths[path]; ok {
		delete(localModFilePaths, path)
		logging.Debug("local mod path removed", "path", path)
	}
}

func ClearLocalModsByInstance(instanceID string) {
	instanceID = strings.TrimSpace(instanceID)
	if instanceID == "" {
		return
	}

	localModsMu.Lock()
	defer localModsMu.Unlock()
	deleted := 0
	for path, modPath := range localModFilePaths {
		if modPath.InstanceID == instanceID {
			delete(localModFilePaths, path)
			deleted++
		}
	}
	logging.Info("local mods cleared by instance", "instanceID", instanceID, "deletedPathCount", deleted)
}

func ClearLocalMods() {
	localModsMu.Lock()
	defer localModsMu.Unlock()
	fileCount := len(localModFiles)
	pathCount := len(localModFilePaths)
	localModFiles = make(map[string][]LocalModFile)
	localModFilePaths = make(map[string]LocalModFilePath)
	logging.Info("local mods cleared", "fileCount", fileCount, "pathCount", pathCount)
}

func ClearLocalModsByVersionLoader(mcVersion, modLoader string) {
	mcVersion = strings.TrimSpace(mcVersion)
	modLoader = strings.ToLower(strings.TrimSpace(modLoader))

	localModsMu.Lock()
	defer localModsMu.Unlock()
	deleted := 0
	for path, modPath := range localModFilePaths {
		if modPath.MinecraftVersion == mcVersion && modPath.ModLoader == modLoader {
			delete(localModFilePaths, path)
			deleted++
		}
	}
	logging.Info("local mods cleared by version loader", "minecraftVersion", mcVersion, "modLoader", modLoader, "deletedPathCount", deleted)
}

func upsertLocalModFile(files []LocalModFile, file LocalModFile) []LocalModFile {
	modID := strings.ToLower(strings.TrimSpace(file.ModID))
	for i := range files {
		if strings.ToLower(strings.TrimSpace(files[i].ModID)) == modID {
			files[i] = file
			return files
		}
	}
	return append(files, file)
}
