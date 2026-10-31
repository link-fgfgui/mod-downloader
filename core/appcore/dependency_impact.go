package appcore

import (
	"errors"
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

var queueRestoredCachedDependency = func(s *Service, req appstructs.ModDownloadRequest) appstructs.ModDownloadResult {
	return s.QueueModDownload(req)
}

func (s *Service) AnalyzeLocalModDisableImpact(req appstructs.LocalModDisableImpactRequest) (appstructs.LocalModDisableImpactResult, error) {
	selected := global.GetSelectedVersion()
	if selected.ID == "" && selected.Name == "" {
		return appstructs.LocalModDisableImpactResult{}, errors.New("no selected version")
	}
	return analyzeLocalModDisableImpact(selected.Mods, req, selected.MinecraftVersion, selected.ModLoader), nil
}

func (s *Service) RestoreCachedDependency(req appstructs.RestoreCachedDependencyRequest) appstructs.ModDownloadResult {
	selected := global.GetSelectedVersion()
	modID := strings.ToLower(strings.TrimSpace(req.ModID))
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	projectID := strings.TrimSpace(req.ProjectID)
	versionID := strings.TrimSpace(req.VersionID)
	if (selected.ID == "" && selected.Name == "") || modID == "" || platform == "" || projectID == "" || versionID == "" {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "invalid cached dependency"}
	}

	valid := false
	for _, candidate := range cachedDependencyCandidates(modID, selected.MinecraftVersion, selected.ModLoader) {
		if candidate.Platform == platform && candidate.ProjectID == projectID && candidate.VersionID == versionID {
			valid = true
			break
		}
	}
	if !valid {
		return appstructs.ModDownloadResult{Skipped: true, Reason: "cached dependency unavailable"}
	}

	project, ok := storage.GetModPlatform(platform, projectID)
	if !ok {
		project = models.ModProject{Platform: platform, ProjectID: projectID}
	}
	project.ID = models.ProjectKey(platform, projectID)
	return queueRestoredCachedDependency(s, appstructs.ModDownloadRequest{
		ProjectID:        project.ID,
		Result:           project,
		MinecraftVersion: selected.MinecraftVersion,
		ModLoader:        selected.ModLoader,
		VersionID:        versionID,
	})
}

func analyzeLocalModDisableImpact(mods []mcstructs.ModInfo, req appstructs.LocalModDisableImpactRequest, minecraftVersion, modLoader string) appstructs.LocalModDisableImpactResult {
	groups := localModDependencyGroups(mods)
	selected := scanPathKeySet(req.Paths)
	action := strings.ToLower(strings.TrimSpace(req.Action))
	if action != LocalModBatchActionDisable && action != LocalModBatchActionInvert {
		return appstructs.LocalModDisableImpactResult{}
	}

	baselineProviders := dependencyProviderCounts(groups)
	projectedDisabled := make(map[string]bool)
	for i := range groups {
		group := &groups[i]
		if !selected[group.pathKey] {
			continue
		}
		switch action {
		case LocalModBatchActionDisable:
			projectedDisabled[group.pathKey] = group.enabled
			group.enabled = false
		case LocalModBatchActionInvert:
			wasEnabled := group.enabled
			group.enabled = !group.enabled
			projectedDisabled[group.pathKey] = wasEnabled
		}
	}
	projectedProviders := dependencyProviderCounts(groups)

	byModID := make(map[string]*appstructs.LocalModDependencyImpact)
	for _, group := range groups {
		if !group.enabled {
			continue
		}
		for _, depID := range group.dependencies {
			if baselineProviders[depID] == 0 || projectedProviders[depID] > 0 {
				continue
			}
			impact := byModID[depID]
			if impact == nil {
				impact = &appstructs.LocalModDependencyImpact{ModID: depID}
				byModID[depID] = impact
			}
			impact.AffectedMods = appendLocalDependencyMod(impact.AffectedMods, group)
		}
	}

	for depID, impact := range byModID {
		for _, group := range groups {
			if projectedDisabled[group.pathKey] && containsString(group.providedIDs, depID) {
				impact.DisabledProviders = appendLocalDependencyMod(impact.DisabledProviders, group)
			}
		}
		impact.Candidates = cachedDependencyCandidates(depID, minecraftVersion, modLoader)
	}

	impacts := make([]appstructs.LocalModDependencyImpact, 0, len(byModID))
	for _, impact := range byModID {
		sortLocalDependencyMods(impact.AffectedMods)
		sortLocalDependencyMods(impact.DisabledProviders)
		impacts = append(impacts, *impact)
	}
	sort.Slice(impacts, func(i, j int) bool { return impacts[i].ModID < impacts[j].ModID })
	return appstructs.LocalModDisableImpactResult{Impacts: impacts}
}

func dependencyProviderCounts(groups []localModDependencyGroup) map[string]int {
	counts := make(map[string]int)
	for _, group := range groups {
		if !group.enabled {
			continue
		}
		for _, id := range group.providedIDs {
			counts[id]++
		}
	}
	return counts
}

func appendLocalDependencyMod(values []appstructs.LocalModDependencyMod, group localModDependencyGroup) []appstructs.LocalModDependencyMod {
	for _, value := range values {
		if scanPathKey(value.Path) == group.pathKey {
			return values
		}
	}
	return append(values, appstructs.LocalModDependencyMod{
		Path:     group.path,
		FileName: group.fileName,
		Name:     firstNonEmpty(group.onlineName, group.name),
		ModIDs:   append([]string(nil), group.modIDs...),
	})
}

func sortLocalDependencyMods(values []appstructs.LocalModDependencyMod) {
	sort.Slice(values, func(i, j int) bool {
		left := strings.ToLower(firstNonEmpty(values[i].Name, values[i].FileName, values[i].Path))
		right := strings.ToLower(firstNonEmpty(values[j].Name, values[j].FileName, values[j].Path))
		if left == right {
			return values[i].Path < values[j].Path
		}
		return left < right
	})
}

func cachedDependencyCandidates(modID, minecraftVersion, modLoader string) []appstructs.CachedDependencyCandidate {
	cached := storage.FindCachedModVersionsByModID(modID, minecraftVersion, modLoader)
	byLogicalProject := make(map[string]storage.CachedModVersionCandidate)
	for _, candidate := range cached {
		key := cachedLogicalProjectKey(candidate.Project)
		current, ok := byLogicalProject[key]
		if !ok || cachedCandidateBefore(current, candidate) {
			byLogicalProject[key] = candidate
		}
	}

	out := make([]appstructs.CachedDependencyCandidate, 0, len(byLogicalProject))
	for _, candidate := range byLogicalProject {
		out = append(out, appstructs.CachedDependencyCandidate{
			ModID:     strings.ToLower(strings.TrimSpace(modID)),
			Platform:  strings.ToLower(strings.TrimSpace(candidate.Project.Platform)),
			ProjectID: strings.TrimSpace(candidate.Project.ProjectID),
			VersionID: strings.TrimSpace(candidate.Version.ID),
			Title:     firstNonEmpty(candidate.Project.Title, candidate.Version.Name, candidate.Project.ProjectID),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Platform + ":" + out[i].ProjectID
		right := out[j].Platform + ":" + out[j].ProjectID
		return left < right
	})
	return out
}

func cachedLogicalProjectKey(project models.ModProject) string {
	platform := strings.ToLower(strings.TrimSpace(project.Platform))
	projectID := strings.TrimSpace(project.ProjectID)
	switch platform {
	case "curseforge":
		if association, ok := storage.GetAssociationByCurseForge(projectID); ok {
			return "association:" + association.ID
		}
	case "modrinth":
		if association, ok := storage.GetAssociationByModrinth(projectID); ok {
			return "association:" + association.ID
		}
	}
	return models.ProjectKey(platform, projectID)
}

func cachedCandidateBefore(current, next storage.CachedModVersionCandidate) bool {
	if current.Version.PublishedAt != next.Version.PublishedAt {
		return current.Version.PublishedAt < next.Version.PublishedAt
	}
	left := models.ProjectKey(current.Project.Platform, current.Project.ProjectID)
	right := models.ProjectKey(next.Project.Platform, next.Project.ProjectID)
	return left > right
}
