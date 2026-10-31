package appcore

import (
	"fmt"
	"strings"
	"sync"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/logging"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	"github.com/link-fgfgui/mod-downloader-core/storage"
	appstructs "github.com/link-fgfgui/mod-downloader-core/structs"
)

type extensionAcceptedPayload struct {
	index     int
	platform  string
	idOrSlug  string
	versionID string
}

type extensionAcceptResult struct {
	index   int
	project models.ModProject
	pinned  bool
	skipped bool
	reason  string
}

// AcceptExtensionProjects receives browser-extension scan results, resolves
// their metadata, filters by the selected instance, auto-pins matching explicit
// versions and emits the accepted projects. HTTP transport must only decode and
// encode; this method owns the workflow and project-level concurrency.
func (s *Service) AcceptExtensionProjects(payloads []appstructs.ExtensionProjectPayload) []appstructs.ModDownloadResult {
	selected := global.GetSelectedVersion()
	mcVersion := strings.TrimSpace(selected.MinecraftVersion)
	modLoader := strings.ToLower(strings.TrimSpace(selected.ModLoader))

	results := make([]appstructs.ModDownloadResult, len(payloads))
	accepted := make([]extensionAcceptedPayload, 0, len(payloads))
	for i, payload := range payloads {
		platform, ok := extensionPlatform(payload.PlatformCode)
		if !ok {
			results[i] = appstructs.ModDownloadResult{
				Skipped: true,
				Reason:  fmt.Sprintf("payload[%d]: invalid platform %q", i, payload.PlatformCode),
			}
			continue
		}

		idOrSlug := ""
		if payload.ID != nil {
			idOrSlug = strings.TrimSpace(*payload.ID)
		}
		if idOrSlug == "" {
			idOrSlug = strings.TrimSpace(payload.Slug)
		}
		if idOrSlug == "" {
			results[i] = appstructs.ModDownloadResult{
				Skipped: true,
				Reason:  fmt.Sprintf("payload[%d]: empty project id and slug", i),
			}
			continue
		}

		accepted = append(accepted, extensionAcceptedPayload{
			index:     i,
			platform:  platform,
			idOrSlug:  idOrSlug,
			versionID: strings.TrimSpace(payload.VersionID),
		})
	}

	resolved := make([]extensionAcceptResult, len(accepted))
	var wg sync.WaitGroup
	for j, item := range accepted {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resolved[j] = s.resolveExtensionPayload(item, mcVersion, modLoader)
		}()
	}
	wg.Wait()

	filteredProjects := make([]models.ModProject, 0, len(resolved))
	seen := make(map[string]bool)
	for _, result := range resolved {
		results[result.index] = appstructs.ModDownloadResult{
			Queued:  !result.skipped,
			Skipped: result.skipped,
			Reason:  result.reason,
		}
		if result.skipped || result.project.ID == "" || seen[result.project.ID] {
			continue
		}
		seen[result.project.ID] = true
		filteredProjects = append(filteredProjects, result.project)
	}

	if len(filteredProjects) > 0 {
		s.emit(EventExtensionModsAccepted, appstructs.SearchModsUpdate{
			Results: filteredProjects,
			Loading: false,
		})
	}
	return results
}

func (s *Service) resolveExtensionPayload(payload extensionAcceptedPayload, mcVersion, modLoader string) extensionAcceptResult {
	project, ok := providers.LookupProjectByPlatform(payload.platform, payload.idOrSlug, mcVersion, modLoader)
	if !ok {
		logging.Warn("extension project lookup failed", "platform", payload.platform, "idOrSlug", payload.idOrSlug)
		return extensionAcceptResult{index: payload.index, skipped: true, reason: "project not found"}
	}

	versions := providers.ListMatchingProjectVersions(project, mcVersion, modLoader)
	if len(versions) == 0 {
		logging.Info("extension project has no matching versions", "platform", payload.platform, "idOrSlug", payload.idOrSlug, "mcVersion", mcVersion, "modLoader", modLoader)
		return extensionAcceptResult{index: payload.index, skipped: true, reason: "no matching version"}
	}

	result := extensionAcceptResult{index: payload.index, project: project}
	if payload.versionID == "" || mcVersion == "" || modLoader == "" {
		return result
	}
	for _, version := range versions {
		if version.ID != payload.versionID {
			continue
		}
		modID := extensionModID(project)
		if modID != "" {
			if err := storage.UpsertPinnedMod(storage.PinnedMod{
				Platform:         strings.ToLower(strings.TrimSpace(project.Platform)),
				ModID:            modID,
				VersionID:        payload.versionID,
				MinecraftVersion: mcVersion,
				ModLoader:        modLoader,
			}); err != nil {
				logging.Error("extension auto-pin failed", "platform", project.Platform, "modID", modID, "versionID", payload.versionID, "error", err)
			} else {
				result.pinned = true
				logging.Info("extension auto-pinned version", "platform", project.Platform, "modID", modID, "versionID", payload.versionID)
			}
		}
		break
	}
	return result
}

// extensionPlatform maps the extension's short platform code to the canonical
// provider platform name.
func extensionPlatform(code string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "mr":
		return "modrinth", true
	case "cf":
		return "curseforge", true
	}
	return "", false
}

// extensionModID returns the platform-specific project ID, stripping the
// "platform:" prefix from the composite ID.
func extensionModID(project models.ModProject) string {
	if project.ProjectID != "" {
		return strings.ToLower(strings.TrimSpace(project.ProjectID))
	}
	id := project.ID
	if idx := strings.Index(id, ":"); idx >= 0 {
		return strings.ToLower(strings.TrimSpace(id[idx+1:]))
	}
	return strings.ToLower(strings.TrimSpace(id))
}
