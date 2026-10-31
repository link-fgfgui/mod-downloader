package appcore

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/minecraft"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

const (
	unusedDependencyEvidenceOnlineLibrary     = "online-library"
	unusedDependencyEvidenceRequiredByDeleted = "required-by-excluded"
)

type localModDependencyGroup struct {
	path         string
	pathKey      string
	fileName     string
	name         string
	onlineName   string
	enabled      bool
	modIDs       []string
	jijModIDs    []string
	providedIDs  []string
	dependencies []string
	categories   []string
}

func (s *Service) ScanUnusedDependencies(req appstructs.UnusedDependencyScanRequest) (appstructs.UnusedDependencyScanResult, error) {
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return appstructs.UnusedDependencyScanResult{}, errors.New("no selected version")
	}
	if !minecraft.JijDetectionEnabled() {
		// Fail closed: without nested-jar metadata the provider index cannot see
		// IDs supplied by Jar-in-Jar children, so a jar could be reported as
		// unused while another mod still requires the bundled dependency.
		return appstructs.UnusedDependencyScanResult{}, nil
	}
	return scanUnusedDependencies(selected.Mods, req.ExcludedPaths), nil
}

func scanUnusedDependencies(mods []mcstructs.ModInfo, excludedPaths []string) appstructs.UnusedDependencyScanResult {
	groups := localModDependencyGroups(mods)
	excluded := scanPathKeySet(excludedPaths)

	requiredByActive := make(map[string]bool)
	requiredByExcluded := make(map[string]bool)
	providerCounts := make(map[string]int)
	for _, group := range groups {
		if !group.enabled {
			continue
		}
		isExcluded := excluded[group.pathKey]
		if !isExcluded {
			for _, providedID := range group.providedIDs {
				providerCounts[providedID]++
			}
		}
		for _, depID := range group.dependencies {
			if isExcluded {
				requiredByExcluded[depID] = true
			} else {
				requiredByActive[depID] = true
			}
		}
	}

	candidates := make([]appstructs.UnusedDependencyCandidate, 0)
	for _, group := range groups {
		if excluded[group.pathKey] {
			continue
		}
		if len(group.modIDs) == 0 {
			continue
		}
		if removesLastRequiredProvider(group, requiredByActive, providerCounts) {
			continue
		}
		evidence := unusedDependencyEvidence(group, requiredByExcluded)
		if len(evidence) == 0 {
			continue
		}
		candidates = append(candidates, appstructs.UnusedDependencyCandidate{
			Path:       group.path,
			FileName:   group.fileName,
			ModIDs:     append([]string(nil), group.modIDs...),
			Name:       group.name,
			OnlineName: group.onlineName,
			Evidence:   evidence,
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := strings.ToLower(firstNonEmpty(candidates[i].OnlineName, candidates[i].Name, candidates[i].FileName, candidates[i].Path))
		right := strings.ToLower(firstNonEmpty(candidates[j].OnlineName, candidates[j].Name, candidates[j].FileName, candidates[j].Path))
		if left == right {
			return candidates[i].Path < candidates[j].Path
		}
		return left < right
	})
	return appstructs.UnusedDependencyScanResult{Candidates: candidates}
}

func localModDependencyGroups(mods []mcstructs.ModInfo) []localModDependencyGroup {
	byPath := make(map[string]*localModDependencyGroup)
	order := make([]string, 0)
	for _, mod := range mods {
		path := strings.TrimSpace(mod.Path)
		if path == "" {
			continue
		}
		key := scanPathKey(path)
		if key == "" {
			continue
		}
		group := byPath[key]
		if group == nil {
			group = &localModDependencyGroup{
				path:       path,
				pathKey:    key,
				fileName:   strings.TrimSpace(mod.FileName),
				name:       strings.TrimSpace(mod.Name),
				onlineName: strings.TrimSpace(mod.OnlineName),
				enabled:    mod.Enabled,
			}
			byPath[key] = group
			order = append(order, key)
		}
		group.modIDs = appendNormalizedUnique(group.modIDs, mod.ID)
		group.jijModIDs = appendJijModIDs(group.jijModIDs, mod.JijMods)
		group.dependencies = appendDependencyIDs(group.dependencies, mod.Dependencies)
		group.categories = appendNormalizedUniqueMany(group.categories, mod.Categories)
		if group.fileName == "" {
			group.fileName = strings.TrimSpace(mod.FileName)
		}
		if group.name == "" {
			group.name = strings.TrimSpace(mod.Name)
		}
		if group.onlineName == "" {
			group.onlineName = strings.TrimSpace(mod.OnlineName)
		}
		group.enabled = group.enabled || mod.Enabled
	}

	groups := make([]localModDependencyGroup, 0, len(order))
	for _, key := range order {
		group := byPath[key]
		group.modIDs = uniqueLowerStrings(group.modIDs)
		group.jijModIDs = uniqueLowerStrings(group.jijModIDs)
		group.providedIDs = uniqueLowerStrings(append(append([]string(nil), group.modIDs...), group.jijModIDs...))
		group.dependencies = uniqueLowerStrings(group.dependencies)
		group.categories = uniqueLowerStrings(group.categories)
		groups = append(groups, *group)
	}
	return groups
}

func removesLastRequiredProvider(group localModDependencyGroup, required map[string]bool, providerCounts map[string]int) bool {
	for _, id := range group.providedIDs {
		if required[id] && providerCounts[id] <= 1 {
			return true
		}
	}
	return false
}

func unusedDependencyEvidence(group localModDependencyGroup, requiredByExcluded map[string]bool) []string {
	evidence := make([]string, 0, 2)
	if containsString(group.categories, "library") {
		evidence = append(evidence, unusedDependencyEvidenceOnlineLibrary)
	}
	if anyModIDRequired(group.modIDs, requiredByExcluded) {
		evidence = append(evidence, unusedDependencyEvidenceRequiredByDeleted)
	}
	return evidence
}

func anyModIDRequired(ids []string, required map[string]bool) bool {
	for _, id := range ids {
		if required[strings.ToLower(strings.TrimSpace(id))] {
			return true
		}
	}
	return false
}

func appendDependencyIDs(ids []string, deps []mcstructs.LocalModDependency) []string {
	for _, dep := range deps {
		if strings.EqualFold(strings.TrimSpace(dep.Type), "required") || strings.TrimSpace(dep.Type) == "" {
			ids = appendNormalizedUnique(ids, dep.ModID)
		}
	}
	return ids
}

func appendJijModIDs(ids []string, mods []mcstructs.JijModInfo) []string {
	for _, mod := range mods {
		ids = appendNormalizedUnique(ids, mod.ID)
	}
	return ids
}

func appendNormalizedUniqueMany(values []string, next []string) []string {
	for _, value := range next {
		values = appendNormalizedUnique(values, value)
	}
	return values
}

func appendNormalizedUnique(values []string, value string) []string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return values
	}
	if containsString(values, value) {
		return values
	}
	return append(values, value)
}

func uniqueLowerStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = appendNormalizedUnique(out, value)
	}
	return out
}

func containsString(values []string, value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	for _, existing := range values {
		if strings.ToLower(strings.TrimSpace(existing)) == value {
			return true
		}
	}
	return false
}

func scanPathKeySet(paths []string) map[string]bool {
	out := make(map[string]bool, len(paths))
	for _, path := range paths {
		if key := scanPathKey(path); key != "" {
			out[key] = true
		}
	}
	return out
}

func scanPathKey(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return strings.ToLower(filepath.ToSlash(filepath.Clean(path)))
}
