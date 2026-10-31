package appcore

import (
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
	mcstructs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

// CheckDependencyFulfillment classifies the required dependencies of the version
// that would actually be installed, against the currently selected instance.
//
// It resolves the target version through modbridge.ResolveVersion (explicit
// version ID, then pinned version, then preferred release type) and classifies
// every required dependency through modbridge.InstallStatusPrecise, so the
// pre-flight result always matches what QueueModDownload would decide. It never
// installs anything; the queue still performs its own dependency resolution.
func (s *Service) CheckDependencyFulfillment(req appstructs.DependencyPreflightRequest) appstructs.DependencyPreflightResult {
	selected := global.GetSelectedVersion()
	downloadReq := appstructs.ModDownloadRequest{
		ProjectID:        strings.TrimSpace(req.ProjectID),
		Result:           req.Result,
		MinecraftVersion: strings.TrimSpace(req.MinecraftVersion),
		ModLoader:        strings.TrimSpace(req.ModLoader),
		VersionID:        strings.TrimSpace(req.VersionID),
	}
	if downloadReq.MinecraftVersion == "" {
		downloadReq.MinecraftVersion = selected.MinecraftVersion
	}
	if downloadReq.ModLoader == "" {
		downloadReq.ModLoader = selected.ModLoader
	}
	downloadReq = s.applyRuntimeDownloadRequest(downloadReq)
	if strings.TrimSpace(downloadReq.MinecraftVersion) == "" || strings.TrimSpace(downloadReq.ModLoader) == "" {
		return appstructs.DependencyPreflightResult{}
	}

	target, ok := modbridge.ResolveVersion(downloadReq)
	if !ok {
		return appstructs.DependencyPreflightResult{}
	}
	target = hydratePreflightDependencies(downloadReq, target)

	local := localModsByOnlineProject(selected.Mods)
	entries := buildPreflightEntries(target, downloadReq, local, modbridge.InstallStatusPrecise)
	return preflightResultFromEntries(entries)
}

type preflightStatusClassifier func(appstructs.ModDownloadRequest) string

// buildPreflightEntries classifies the required dependencies of a resolved
// target version. Optional/incompatible declarations are ignored because the
// downloader only queues required dependencies. The classifier is injectable so
// the status mapping and required-only filter can be unit tested without
// network/provider state.
func buildPreflightEntries(
	target models.ModVersion,
	parent appstructs.ModDownloadRequest,
	local map[string]mcstructs.ModInfo,
	classify preflightStatusClassifier,
) []appstructs.DependencyPreflightEntry {
	entries := make([]appstructs.DependencyPreflightEntry, 0, len(target.Dependencies))
	for _, dep := range target.Dependencies {
		if !modbridge.IsDependencyType(dep, "required") {
			continue
		}
		depReq, ok := modbridge.DependencyDownloadRequest(target.Platform, dep, parent)
		if !ok {
			continue
		}
		entry := appstructs.DependencyPreflightEntry{
			ProjectID:       strings.TrimSpace(dep.DependencyProjectID),
			Title:           strings.TrimSpace(depReq.Result.Title),
			RequiredVersion: strings.TrimSpace(dep.DependencyVersionID),
		}
		if entry.Title == "" {
			entry.Title = entry.ProjectID
		}
		switch classify(depReq) {
		case modbridge.BtnStatusNew:
			entry.Status = appstructs.DependencyPreflightMissing
		case modbridge.BtnStatusInstalled:
			entry.Status = appstructs.DependencyPreflightFulfilled
		default:
			entry.Status = appstructs.DependencyPreflightConflict
		}
		// Local identity is display-only; the status above comes from the same
		// mod-ID based resolution the downloader uses.
		if mod, found := local[localProjectKey(target.Platform, entry.ProjectID)]; found {
			entry.LocalName = firstNonEmpty(mod.OnlineName, mod.Name, mod.FileName)
			entry.LocalVersion = firstNonEmpty(mod.OnlineVersion, mod.Version)
			if entry.Title == entry.ProjectID {
				entry.Title = firstNonEmpty(mod.OnlineName, mod.Name, entry.ProjectID)
			}
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Status != entries[j].Status {
			return preflightStatusRank(entries[i].Status) < preflightStatusRank(entries[j].Status)
		}
		return strings.ToLower(entries[i].Title) < strings.ToLower(entries[j].Title)
	})
	return entries
}

func preflightResultFromEntries(entries []appstructs.DependencyPreflightEntry) appstructs.DependencyPreflightResult {
	result := appstructs.DependencyPreflightResult{Entries: entries}
	for _, entry := range entries {
		switch entry.Status {
		case appstructs.DependencyPreflightMissing:
			result.Missing++
		case appstructs.DependencyPreflightConflict:
			result.Conflicts++
		default:
			result.Fulfilled++
		}
	}
	return result
}

// hydratePreflightDependencies mirrors the downloader: when a version carries no
// dependency metadata at all, refresh the project versions once and retry.
func hydratePreflightDependencies(req appstructs.ModDownloadRequest, version models.ModVersion) models.ModVersion {
	if hasPreflightDependency(version.Dependencies) {
		return version
	}
	refreshed := providers.RefreshMatchingProjectVersions(req.Result, req.MinecraftVersion, req.ModLoader)
	if refreshedVersion, found := modbridge.FindVersionByID(refreshed, version.ID); found && hasPreflightDependency(refreshedVersion.Dependencies) {
		return refreshedVersion
	}
	return version
}

func hasPreflightDependency(deps []models.ModDependency) bool {
	for _, dep := range deps {
		if modbridge.IsDependencyType(dep, "required") || modbridge.IsDependencyType(dep, "optional") || modbridge.IsDependencyType(dep, "incompatible") {
			return true
		}
	}
	return false
}

func preflightStatusRank(status string) int {
	switch status {
	case appstructs.DependencyPreflightConflict:
		return 0
	case appstructs.DependencyPreflightMissing:
		return 1
	default:
		return 2
	}
}

func localProjectKey(platform, projectID string) string {
	return strings.ToLower(strings.TrimSpace(platform)) + "|" + strings.ToLower(strings.TrimSpace(projectID))
}

// localModsByOnlineProject indexes local mods (including disabled ones) by
// platform project ID for display purposes only.
func localModsByOnlineProject(mods []mcstructs.ModInfo) map[string]mcstructs.ModInfo {
	byProject := make(map[string]mcstructs.ModInfo)
	for _, mod := range mods {
		projectID := strings.TrimSpace(mod.OnlineProjectID)
		if projectID == "" {
			continue
		}
		key := localProjectKey(mod.OnlinePlatform, projectID)
		if _, exists := byProject[key]; !exists {
			byProject[key] = mod
		}
	}
	return byProject
}
