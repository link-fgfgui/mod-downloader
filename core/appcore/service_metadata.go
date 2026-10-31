package appcore

import (
	"strings"

	"github.com/link-fgfgui/mod-downloader-core/global"
	"github.com/link-fgfgui/mod-downloader-core/modbridge"
	"github.com/link-fgfgui/mod-downloader-core/models"
	"github.com/link-fgfgui/mod-downloader-core/providers"
	structs "github.com/link-fgfgui/mod-downloader-core/structs/minecraft"
)

func (s *Service) resolveMissingModMetadata(refreshed structs.VersionInfo, missingFiles map[string]uint32) {
	if len(missingFiles) == 0 {
		return
	}
	s.startServiceBackground(func() {
		projects, versions := providers.ResolveMetadataByLocalFiles(missingFiles)
		updated := global.GetSelectedVersion()
		if updated.ID != refreshed.ID || updated.Name != refreshed.Name {
			return
		}
		mods, changed := applyResolvedModMetadata(updated.Mods, missingFiles, projects, versions)
		if !changed {
			return
		}
		updated.Mods = mods
		global.SetSelectedVersion(updated)
		s.emit(EventSelectedVersionChanged, updated)
	})
}

func applyResolvedModMetadata(mods []structs.ModInfo, missingFiles map[string]uint32, projects map[string]models.ModProject, versions map[string]models.ModVersion) ([]structs.ModInfo, bool) {
	updated := make([]structs.ModInfo, len(mods))
	copy(updated, mods)
	changed := false
	for i := range updated {
		sha1 := strings.ToLower(strings.TrimSpace(updated[i].SHA1))
		if _, pending := missingFiles[sha1]; !pending {
			continue
		}
		if project, ok := projects[sha1]; ok {
			enrichedMod := modbridge.ApplyPlatformMetadataToModInfo(updated[i], project, versions[sha1])
			if modInfoDisplayMetadataChanged(updated[i], enrichedMod) {
				updated[i] = enrichedMod
				changed = true
			}
		}
		if updated[i].OnlineMetadataLoading {
			updated[i].OnlineMetadataLoading = false
			changed = true
		}
	}
	return updated, changed
}

func setOnlineMetadataLoading(mods []structs.ModInfo, files map[string]uint32, loading bool) {
	for i := range mods {
		sha1 := strings.ToLower(strings.TrimSpace(mods[i].SHA1))
		if _, ok := files[sha1]; ok {
			mods[i].OnlineMetadataLoading = loading
		}
	}
}

func enrichModMetadata(mods []structs.ModInfo) ([]structs.ModInfo, map[string]uint32) {
	enriched := make([]structs.ModInfo, len(mods))
	copy(enriched, mods)
	missingFiles := make(map[string]uint32)
	for i := range enriched {
		sha1 := strings.ToLower(strings.TrimSpace(enriched[i].SHA1))
		if sha1 == "" {
			continue
		}
		project, version, ok := modbridge.PlatformMetadataForSHA1(sha1)
		if ok && hasProjectDisplayMetadata(project) {
			enriched[i] = modbridge.ApplyPlatformMetadataToModInfo(enriched[i], project, version)
		} else {
			missingFiles[sha1] = enriched[i].CfFingerprint
		}
	}
	return enriched, missingFiles
}

func hasProjectDisplayMetadata(project models.ModProject) bool {
	return strings.TrimSpace(project.Title) != "" ||
		strings.TrimSpace(project.IconURL) != "" ||
		len(project.Categories) > 0
}

func modInfoDisplayMetadataChanged(left, right structs.ModInfo) bool {
	if left.OnlineName != right.OnlineName ||
		left.OnlinePlatform != right.OnlinePlatform ||
		left.OnlineProjectID != right.OnlineProjectID ||
		left.OnlineSlug != right.OnlineSlug ||
		left.OnlineVersion != right.OnlineVersion ||
		left.OnlineVersionID != right.OnlineVersionID ||
		left.OnlineFileName != right.OnlineFileName ||
		left.OnlineMetadataLoading != right.OnlineMetadataLoading ||
		left.IconURL != right.IconURL ||
		len(left.Categories) != len(right.Categories) {
		return true
	}
	for i := range left.Categories {
		if left.Categories[i] != right.Categories[i] {
			return true
		}
	}
	return false
}
