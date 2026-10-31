package appcore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func (s *Service) SetMinecraftDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	previousDir := global.GetMinecraftDir()
	s.stopModsWatcher()
	simplified := minecraft.SimplifyPathWithEnv(dir)
	global.SetMinecraftDir(dir)
	global.ClearLocalMods()
	global.HardlinkIndexClear()
	global.InvalidateVersions()
	versions := s.LoadVersionsFromDisk(dir)
	if len(versions) == 0 {
		logging.Warn("chosen minecraft dir has no versions", "minecraftDir", dir)
		global.SetMinecraftDir(previousDir)
		global.ClearLocalMods()
		global.HardlinkIndexClear()
		global.InvalidateVersions()
		if strings.TrimSpace(previousDir) != "" {
			s.LoadVersionsFromDisk(previousDir)
			s.ensureModsWatcher(global.GetSelectedVersion())
		}
		return ""
	}
	if s.config != nil {
		s.config.Prefers.MinecraftDir = dir
	}
	s.emit(EventMinecraftDirChanged, simplified)
	s.emit(EventSelectedVersionChanged, global.GetSelectedVersion())
	return simplified
}

func (s *Service) GetMinecraftDir() string {
	return global.GetMinecraftDir()
}

func (s *Service) ValidateMinecraftDir() bool {
	resolvedDir := global.GetMinecraftDir()
	if resolvedDir == "" {
		return false
	}
	info, err := os.Stat(resolvedDir)
	return err == nil && info.IsDir()
}

func (s *Service) GetVersions() []structs.VersionInfo {
	mcDir := global.GetMinecraftDir()
	if versions, ok := global.GetVersionsForDir(mcDir); ok {
		// Cache population always goes through LoadVersionsFromDisk, which
		// selects the default instance; a read must not mutate selection.
		return versions
	}
	return s.LoadVersionsFromDisk(mcDir)
}

func (s *Service) GetSelectedVersion() structs.VersionInfo {
	return global.GetSelectedVersion()
}

func (s *Service) RefreshVersions() []structs.VersionInfo {
	return s.LoadVersionsFromDisk(global.GetMinecraftDir())
}

func (s *Service) RefreshSelectedVersionMods() structs.VersionInfo {
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return structs.VersionInfo{}
	}
	refreshed := s.refreshAndSelectVersionMods(selected, global.GetMinecraftDir())
	s.ensureModsWatcher(refreshed)
	return refreshed
}

func (s *Service) SelectVersion(versionKey string) (structs.VersionInfo, error) {
	versionKey = strings.TrimSpace(versionKey)
	if versionKey == "" {
		return structs.VersionInfo{}, errors.New("empty version key")
	}

	mcDir := global.GetMinecraftDir()
	if _, ok := global.GetVersionsForDir(mcDir); !ok {
		s.LoadVersionsFromDisk(mcDir)
	}

	if version, ok := global.GetVersionByKey(versionKey); ok {
		version = resetConnectorVersion(version)
		if !ValidMinecraftInstance(version) {
			return structs.VersionInfo{}, errors.New("invalid minecraft version or mod loader")
		}
		selected := s.refreshAndSelectVersionMods(version, mcDir)
		s.ensureModsWatcher(selected)
		return selected, nil
	}

	return structs.VersionInfo{}, fmt.Errorf("version not found: %s", versionKey)
}

func (s *Service) LocalMods(versionKey string) ([]structs.ModInfo, error) {
	if strings.TrimSpace(versionKey) != "" {
		if _, err := s.SelectVersion(versionKey); err != nil {
			return nil, err
		}
	}
	selected := s.RefreshSelectedVersionMods()
	return selected.Mods, nil
}

func (s *Service) LocalModsInDir(dir, minecraftVersion, modLoader string) ([]structs.ModInfo, error) {
	dir = s.resolveRuntimePath(dir)
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("empty mods dir")
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("mods directory does not exist: %s", dir)
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("mods path is not a directory: %s", dir)
	}
	minecraftVersion = firstNonEmpty(strings.TrimSpace(minecraftVersion), s.options.Runtime.MinecraftVersion, s.Config().Runtime.MinecraftVersion)
	modLoader = strings.ToLower(firstNonEmpty(strings.TrimSpace(modLoader), s.options.Runtime.ModLoader, s.Config().Runtime.ModLoader))
	instanceID := modbridge.InstanceIDForTargetDir(dir)
	global.ClearLocalModsByInstance(instanceID)
	mods := minecraft.ScanModsDir(dir, instanceID, minecraftVersion, modLoader, "")
	enriched, missingFiles := enrichModMetadata(mods)
	if len(missingFiles) > 0 {
		logging.Debug("local mods in dir have unresolved online metadata", "dir", dir, "count", len(missingFiles))
	}
	return enriched, nil
}

func (s *Service) ApplyLocalModBatchOperation(req appstructs.LocalModBatchOperationRequest) (structs.VersionInfo, error) {
	action := strings.ToLower(strings.TrimSpace(req.Action))
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return structs.VersionInfo{}, errors.New("no selected version")
	}

	versionDir := minecraft.VersionDirPath(global.GetMinecraftDir(), selected)
	if strings.TrimSpace(versionDir) == "" {
		return s.RefreshSelectedVersionMods(), errors.New("selected version directory not found")
	}
	modsDir := filepath.Join(versionDir, "mods")

	ops, err := planLocalModBatchOperations(req.Paths, action, modsDir, global.GetMinecraftDir())
	if err != nil {
		return s.RefreshSelectedVersionMods(), err
	}
	if err := applyLocalModFileOperations(ops); err != nil {
		return s.RefreshSelectedVersionMods(), err
	}
	s.recordLocalModUsage(ops)
	return s.refreshChangedLocalMods(ops), nil
}

func (s *Service) refreshChangedLocalMods(ops []localModFileOperation) structs.VersionInfo {
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return selected
	}
	instanceID := versionInstanceDir(selected)
	mcDir := global.GetMinecraftDir()
	changed := make(map[string]bool)
	for _, op := range ops {
		removeLocalModPathVariants(op.source, mcDir)
		removeLocalModPathVariants(op.target, mcDir)
		changed[op.source], changed[op.target] = true, true
		if op.action == LocalModBatchActionDelete {
			continue
		}
		if strings.HasSuffix(op.target, ".jar") {
			minecraft.ScanModFile(op.target, instanceID, selected.MinecraftVersion, selected.ActualModLoader, mcDir)
		}
	}
	mods := make([]structs.ModInfo, 0, len(selected.Mods))
	for _, mod := range selected.Mods {
		absPath := mod.Path
		if !filepath.IsAbs(absPath) {
			absPath = filepath.Join(mcDir, absPath)
		}
		if !changed[absPath] && !changed[mod.Path] {
			mods = append(mods, mod)
		}
	}
	for _, op := range ops {
		if op.action == LocalModBatchActionDelete || !isLocalModJarPath(op.target) {
			continue
		}
		fresh := minecraft.ScanModFile(op.target, instanceID, selected.MinecraftVersion, selected.ActualModLoader, mcDir)
		mods = append(mods, fresh...)
	}
	enriched, missing := enrichModMetadata(mods)
	setOnlineMetadataLoading(enriched, missing, true)
	selected.Mods = enriched
	selected = reconcileConnectorVersion(selected)
	global.SetSelectedVersion(selected)
	s.emit(EventSelectedVersionChanged, selected)
	s.resolveMissingModMetadata(selected, missing)
	return selected
}

func (s *Service) refreshChangedPaths(paths []string) {
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return
	}
	instanceID := versionInstanceDir(selected)
	mcDir := global.GetMinecraftDir()
	changed := make(map[string]bool)
	for _, path := range paths {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		removeLocalModPathVariants(abs, mcDir)
		changed[abs] = true
	}
	mods := make([]structs.ModInfo, 0, len(selected.Mods))
	for _, mod := range selected.Mods {
		abs := mod.Path
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(mcDir, abs)
		}
		if !changed[abs] {
			mods = append(mods, mod)
		}
	}
	for _, path := range paths {
		if isLocalModJarPath(path) {
			mods = append(mods, minecraft.ScanModFile(path, instanceID, selected.MinecraftVersion, selected.ActualModLoader, mcDir)...)
		}
	}
	enriched, missing := enrichModMetadata(mods)
	setOnlineMetadataLoading(enriched, missing, true)
	selected.Mods = enriched
	selected = reconcileConnectorVersion(selected)
	global.SetSelectedVersion(selected)
	s.emit(EventSelectedVersionChanged, selected)
	s.resolveMissingModMetadata(selected, missing)
}

func removeLocalModPathVariants(path, mcDir string) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	global.RemoveLocalModByPath(path)
	if abs, err := filepath.Abs(path); err == nil {
		global.RemoveLocalModByPath(abs)
		if strings.TrimSpace(mcDir) != "" {
			if rel, err := filepath.Rel(mcDir, abs); err == nil {
				global.RemoveLocalModByPath(rel)
			}
		}
	}
}

func (s *Service) recordLocalModUsage(ops []localModFileOperation) {
	counts := make(map[storage.UsageStatKey]int64)
	for _, op := range ops {
		switch op.action {
		case LocalModBatchActionEnable:
			counts[storage.UsageStatModsEnabled]++
		case LocalModBatchActionDisable:
			counts[storage.UsageStatModsDisabled]++
		case LocalModBatchActionDelete:
			counts[storage.UsageStatModsDeleted]++
		}
	}
	for key, count := range counts {
		s.incrementUsageStat(key, count)
	}
}

type localModFileOperation struct {
	action string
	source string
	target string
}

func planLocalModBatchOperations(paths []string, action string, modsDir string, minecraftDir string) ([]localModFileOperation, error) {
	switch action {
	case LocalModBatchActionEnable, LocalModBatchActionDisable, LocalModBatchActionInvert, LocalModBatchActionDelete:
	default:
		return nil, fmt.Errorf("unsupported local mod action: %s", action)
	}
	if len(paths) == 0 {
		return nil, errors.New("no local mod paths selected")
	}

	modsDir, err := filepath.Abs(filepath.Clean(modsDir))
	if err != nil {
		return nil, err
	}
	minecraftDir = strings.TrimSpace(minecraftDir)
	if minecraftDir != "" {
		if abs, err := filepath.Abs(filepath.Clean(minecraftDir)); err == nil {
			minecraftDir = abs
		}
	}

	seen := make(map[string]bool, len(paths))
	targets := make(map[string]string, len(paths))
	ops := make([]localModFileOperation, 0, len(paths))
	for _, rawPath := range paths {
		source, err := resolveSelectedLocalModPath(rawPath, modsDir, minecraftDir)
		if err != nil {
			return nil, err
		}
		if seen[source] {
			continue
		}
		seen[source] = true

		info, err := os.Stat(source)
		if err != nil {
			return nil, fmt.Errorf("local mod file not found: %s", rawPath)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("local mod path is a directory: %s", rawPath)
		}

		fileAction := action
		if action == LocalModBatchActionInvert {
			if localModPathDisabled(source) {
				fileAction = LocalModBatchActionEnable
			} else {
				fileAction = LocalModBatchActionDisable
			}
		}

		op := localModFileOperation{action: fileAction, source: source}
		switch fileAction {
		case LocalModBatchActionEnable:
			if !localModPathDisabled(source) {
				continue
			}
			op.target = strings.TrimSuffix(source, ".disabled")
			if err := validateLocalModRenameTarget(source, op.target, targets); err != nil {
				return nil, err
			}
		case LocalModBatchActionDisable:
			if localModPathDisabled(source) {
				continue
			}
			op.target = source + ".disabled"
			if err := validateLocalModRenameTarget(source, op.target, targets); err != nil {
				return nil, err
			}
		case LocalModBatchActionDelete:
		default:
			return nil, fmt.Errorf("unsupported local mod action: %s", fileAction)
		}
		ops = append(ops, op)
	}
	return ops, nil
}

func applyLocalModFileOperations(ops []localModFileOperation) error {
	for _, op := range ops {
		switch op.action {
		case LocalModBatchActionEnable, LocalModBatchActionDisable:
			if err := os.Rename(op.source, op.target); err != nil {
				return err
			}
			global.RemoveLocalModByPath(op.source)
		case LocalModBatchActionDelete:
			if err := os.Remove(op.source); err != nil {
				return err
			}
			global.RemoveLocalModByPath(op.source)
		default:
			return fmt.Errorf("unsupported local mod action: %s", op.action)
		}
	}
	return nil
}

func resolveSelectedLocalModPath(rawPath string, modsDir string, minecraftDir string) (string, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return "", errors.New("empty local mod path")
	}

	candidates := []string{rawPath}
	if !filepath.IsAbs(rawPath) {
		if minecraftDir != "" {
			candidates = append(candidates, filepath.Join(minecraftDir, rawPath))
		}
		candidates = append(candidates, filepath.Join(modsDir, rawPath))
	}

	missingInside := false
	for _, candidate := range candidates {
		abs, err := filepath.Abs(filepath.Clean(candidate))
		if err != nil {
			continue
		}
		if !pathInsideDir(abs, modsDir) {
			continue
		}
		if !isLocalModJarPath(abs) {
			return "", fmt.Errorf("local mod path is not a jar: %s", rawPath)
		}
		if _, err := os.Stat(abs); err == nil {
			return abs, nil
		} else if os.IsNotExist(err) {
			missingInside = true
		}
	}

	if missingInside {
		return "", fmt.Errorf("local mod file not found: %s", rawPath)
	}
	return "", fmt.Errorf("local mod path is outside selected instance: %s", rawPath)
}

func validateLocalModRenameTarget(source string, target string, plannedTargets map[string]string) error {
	if existingSource, ok := plannedTargets[target]; ok && existingSource != source {
		return fmt.Errorf("multiple local mod files target the same path: %s", target)
	}
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("local mod target already exists: %s", target)
	} else if !os.IsNotExist(err) {
		return err
	}
	plannedTargets[target] = source
	return nil
}

func isLocalModJarPath(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".jar") || strings.HasSuffix(lower, ".jar.disabled")
}

func localModPathDisabled(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".jar.disabled")
}

func pathInsideDir(path string, dir string) bool {
	rel, err := filepath.Rel(dir, path)
	if err != nil || rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (s *Service) LoadVersionsFromDisk(mcDir string) []structs.VersionInfo {
	if strings.TrimSpace(mcDir) == "" {
		global.SetVersionsForDir(mcDir, nil)
		return nil
	}

	infos := minecraft.LoadLauncherVersions(mcDir, loadMinecraftDirVersions)
	for i := range infos {
		infos[i] = resetConnectorVersion(infos[i])
	}
	global.SetVersionsForDir(mcDir, infos)
	ensureSelectedVersion(infos)
	generation := global.HardlinkIndexGeneration()
	go ScanAllModDirsForHardlinkIndex(mcDir, infos, generation)
	return infos
}

func (s *Service) refreshVersionMods(version structs.VersionInfo, mcDir string) structs.VersionInfo {
	versionDirName := versionInstanceDir(version)
	global.ClearLocalModsByInstance(versionDirName)
	refreshed := ScanVersionMods(version, mcDir)
	if versions, ok := global.GetVersionsForDir(mcDir); ok {
		next := make([]structs.VersionInfo, len(versions))
		copy(next, versions)
		for i, cached := range next {
			if cached.Name == version.Name || cached.ID == version.ID {
				next[i] = refreshed
			}
		}
		global.SetVersionsForDir(mcDir, next)
	}
	return refreshed
}
