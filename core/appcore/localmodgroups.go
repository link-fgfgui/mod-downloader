// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/global"
	"github.com/link-fgfgui/mod-downloader/core/models"
	"github.com/link-fgfgui/mod-downloader/core/modgraph"
	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
	appview "github.com/link-fgfgui/mod-downloader/core/structs/view"
)

// BuildLocalModGroups aggregates every physical JAR record that shares one
// fileName/path/id into a single group and computes the UI-neutral projections
// adapters render: display name, categories, loader set, search text and the
// compatibility verdict for the selected loader.
//
// Group order follows first appearance so callers keep the scanner's order.
// Jar-in-Jar entries are de-duplicated case-insensitively by id.
func BuildLocalModGroups(mods []structs.ModInfo, selectedLoader string) []appview.LocalModGroup {
	order := make([]string, 0, len(mods))
	byKey := make(map[string]*appview.LocalModGroup, len(mods))
	hasPrimary := make(map[string]bool, len(mods))
	for i := range mods {
		mod := mods[i]
		key := mod.FileName
		if key == "" {
			key = mod.Path
		}
		if key == "" {
			key = mod.ID
		}
		group, ok := byKey[key]
		if !ok {
			group = &appview.LocalModGroup{}
			byKey[key] = group
			order = append(order, key)
		}
		group.Strong = append(group.Strong, mod)
		addJIJMods(group, mod.JIJMods)
		if !hasPrimary[key] {
			group.Primary = mod
			hasPrimary[key] = true
		}
	}

	selected := strings.ToLower(strings.TrimSpace(selectedLoader))
	groups := make([]appview.LocalModGroup, 0, len(order))
	for _, key := range order {
		group := byKey[key]
		finalizeLocalModGroup(group, selected)
		groups = append(groups, *group)
	}
	return groups
}

// GetLocalModView returns the Manage rows, dependency forest and reverse
// dependents in one scan so adapters do not rebuild those rules themselves.
// When the requested version is already selected the cached scan is reused so
// adapters can refresh the projection without touching the disk. A key that no
// longer resolves (for example during a Minecraft directory switch) falls back
// to the current selection instead of hiding its rows.
func (s *Service) GetLocalModView(versionKey, modLoader string) appview.LocalModView {
	mods, selected := s.resolveLocalModsForView(versionKey)
	groups := BuildLocalModGroups(mods, localModLoader(modLoader, selected))
	dependents := make(map[string][]modgraph.ReverseDependencyMatch, len(groups))
	for _, group := range groups {
		provided := make([]string, 0, len(group.Strong))
		excluded := make([]string, 0, len(group.Strong))
		for _, mod := range group.Strong {
			provided = append(provided, modgraph.ProvidedIDs(mod)...)
			if path := strings.TrimSpace(mod.Path); path != "" {
				excluded = append(excluded, path)
			}
		}
		dependents[group.RowKey] = modgraph.FindReverseDependents(mods, provided, excluded, true)
	}
	return appview.LocalModView{
		Groups:     groups,
		Graph:      modgraph.Build(mods),
		Dependents: dependents,
	}
}

func (s *Service) resolveLocalModsForView(versionKey string) ([]structs.ModInfo, structs.VersionInfo) {
	versionKey = strings.TrimSpace(versionKey)
	selected := global.GetSelectedVersion()
	switch {
	case versionKey == "" || versionKey == selected.ID || versionKey == selected.Name:
		return selected.Mods, selected
	default:
		localMods, err := s.LocalMods(versionKey)
		if err != nil {
			return selected.Mods, selected
		}
		return localMods, global.GetSelectedVersion()
	}
}

func localModLoader(requested string, selected structs.VersionInfo) string {
	if loader := strings.TrimSpace(requested); loader != "" {
		return loader
	}
	return selected.ModLoader
}

func finalizeLocalModGroup(group *appview.LocalModGroup, selectedLoader string) {
	group.DisplayName = displayName(group.Primary)
	group.Categories = modCategories(group.Primary)
	group.Loaders = groupLoaders(group.Strong)
	group.SearchText = searchableGroupText(group)
	group.Incompatible = groupLoadersIncompatible(group.Loaders, selectedLoader)
	group.RowKey = localModGroupRowKey(group)
	group.CanReplaceVersion = strings.TrimSpace(group.Primary.OnlinePlatform) != "" &&
		strings.TrimSpace(group.Primary.OnlineProjectID) != ""
	if group.CanReplaceVersion {
		group.OnlineProject = localModGroupOnlineProject(group)
	}
}

func localModGroupRowKey(group *appview.LocalModGroup) string {
	mod := group.Primary
	parts := make([]string, 0, 4)
	for _, value := range []string{mod.ID, mod.SHA1, mod.Path, mod.FileName} {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, "|")
}

func localModGroupOnlineProject(group *appview.LocalModGroup) models.ModProject {
	mod := group.Primary
	platform := strings.TrimSpace(mod.OnlinePlatform)
	projectID := strings.TrimSpace(mod.OnlineProjectID)
	return models.ModProject{
		ID:          models.ProjectKey(platform, projectID),
		Platform:    platform,
		ProjectID:   projectID,
		Slug:        mod.OnlineSlug,
		Title:       group.DisplayName,
		Icon:        "mdi-package-variant",
		IconURL:     mod.IconURL,
		Description: mod.Description,
		Categories:  group.Categories,
	}
}

func addJIJMods(group *appview.LocalModGroup, mods []structs.JIJModInfo) {
	for _, mod := range mods {
		id := strings.TrimSpace(mod.ID)
		if id == "" {
			continue
		}
		duplicate := false
		for _, existing := range group.JIJ {
			if strings.EqualFold(existing.ID, id) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			group.JIJ = append(group.JIJ, mod)
		}
	}
}

// groupLoaders returns the sorted, lowercased loader set declared by strong
// records.
func groupLoaders(mods []structs.ModInfo) []string {
	seen := map[string]bool{}
	loaders := []string{}
	for _, mod := range mods {
		for _, loader := range mod.Loaders {
			name := strings.ToLower(strings.TrimSpace(loader))
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			loaders = append(loaders, name)
		}
	}
	sort.Strings(loaders)
	return loaders
}

// groupLoadersIncompatible reports whether the selected loader is missing from
// a non-empty declared loader set.
func groupLoadersIncompatible(loaders []string, selectedLoader string) bool {
	if len(loaders) == 0 || selectedLoader == "" {
		return false
	}
	for _, loader := range loaders {
		if loader == selectedLoader {
			return false
		}
	}
	return true
}

// displayName mirrors the adapter-facing name precedence.
func displayName(mod structs.ModInfo) string {
	if mod.OnlineName != "" {
		return mod.OnlineName
	}
	if mod.Name != "" {
		return mod.Name
	}
	return mod.ID
}

// modCategories returns trimmed, de-duplicated, lowercased categories in
// declaration order.
func modCategories(mod structs.ModInfo) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, category := range mod.Categories {
		key := strings.ToLower(strings.TrimSpace(category))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, key)
	}
	return out
}

// searchableGroupText flattens the human-facing fields of the group so adapters
// can run substring matching. Paths and dependency declarations are excluded on
// purpose: they embed the Minecraft/loader version and would make every mod
// match a version query.
func searchableGroupText(group *appview.LocalModGroup) string {
	values := []string{}
	for _, mod := range group.Strong {
		collectSearchValues(mod, &values)
	}
	for _, jij := range group.JIJ {
		appendSearchValues(&values, jij.ID, jij.Name)
	}
	return strings.ToLower(strings.Join(values, "\n"))
}

func collectSearchValues(mod structs.ModInfo, out *[]string) {
	appendSearchValues(out, mod.ID, mod.Name, mod.Version, mod.Description, mod.FileName,
		mod.OnlineName, mod.OnlinePlatform, mod.OnlineProjectID, mod.OnlineSlug, mod.OnlineVersion,
		mod.OnlineVersionID, mod.OnlineFileName)
	appendSearchValues(out, mod.Categories...)
	for _, jij := range mod.JIJMods {
		appendSearchValues(out, jij.ID, jij.Name)
	}
}

func appendSearchValues(out *[]string, values ...string) {
	for _, value := range values {
		if value != "" {
			*out = append(*out, value)
		}
	}
}
