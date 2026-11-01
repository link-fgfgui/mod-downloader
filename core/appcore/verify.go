// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package appcore

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/minecraft"
	structs "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

// VerifyIssue is one finding from VerifyLocalMods.
type VerifyIssue struct {
	Kind     string `json:"kind"`
	ModID    string `json:"modId,omitempty"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
	Suggest  string `json:"suggest,omitempty"`
	Advisory bool   `json:"advisory,omitempty"`
}

// VerifyReport summarizes a local mod verification pass.
type VerifyReport struct {
	Checked  int           `json:"checked"`
	Provided []string      `json:"provided"`
	Issues   []VerifyIssue `json:"issues,omitempty"`
	Passed   bool          `json:"passed"`
}

// builtinModIDs are dependency targets that are always present in a Minecraft
// instance and must not be reported as missing. A false value documents ids
// that look builtin but are ordinary downloadable projects.
var builtinModIDs = map[string]bool{
	"minecraft":    true,
	"java":         true,
	"fabricloader": true,
	"fabric":       true,
	"fabric-api":   false,
	"forge":        true,
	"neoforge":     true,
	"javafml":      true,
	"lowcodefml":   true,
	"fml":          true,
	"architectury": false,
}

func isBuiltinModID(id string) bool {
	return builtinModIDs[strings.ToLower(strings.TrimSpace(id))]
}

// VerifyLocalMods checks a scanned mod set for duplicate ids, missing required
// dependencies and loader/Minecraft version mismatches. The rules live here so
// the CLI, GUI and TUI all report the same verdict.
func VerifyLocalMods(modsDir string, mods []structs.ModInfo, minecraftVersion, modLoader string, strict bool) VerifyReport {
	provided := make(map[string]bool)
	jarPaths := make([]string, 0)
	seenPath := make(map[string]bool)
	idToPaths := make(map[string]map[string]bool)

	for _, mod := range mods {
		id := strings.ToLower(strings.TrimSpace(mod.ID))
		if id != "" {
			provided[id] = true
		}
		for _, jij := range mod.JIJMods {
			if jijID := strings.ToLower(strings.TrimSpace(jij.ID)); jijID != "" {
				provided[jijID] = true
			}
		}
		path := resolveLocalModPath(modsDir, mod)
		if path == "" {
			continue
		}
		if !seenPath[path] {
			seenPath[path] = true
			jarPaths = append(jarPaths, path)
		}
		if id != "" {
			if idToPaths[id] == nil {
				idToPaths[id] = make(map[string]bool)
			}
			idToPaths[id][path] = true
		}
	}

	report := VerifyReport{Checked: len(jarPaths)}
	for _, id := range sortedMapKeys(idToPaths) {
		if len(idToPaths[id]) > 1 {
			report.Issues = append(report.Issues, VerifyIssue{
				Kind:    "duplicate-mod",
				ModID:   id,
				Message: fmt.Sprintf("mod id %q is provided by %d different jars", id, len(idToPaths[id])),
			})
		}
	}

	// One verdict per physical jar: the first declared mod owns the metadata,
	// matching what a single-file parse would report.
	pathToMod := make(map[string]structs.ModInfo, len(jarPaths))
	for _, mod := range mods {
		path := resolveLocalModPath(modsDir, mod)
		if path == "" {
			continue
		}
		if _, ok := pathToMod[path]; !ok {
			pathToMod[path] = mod
		}
	}
	for _, path := range jarPaths {
		mod, ok := pathToMod[path]
		if !ok {
			continue
		}
		file := filepath.Base(path)
		for _, dep := range mod.Dependencies {
			depID := strings.ToLower(strings.TrimSpace(dep.ModID))
			if depID == "" || isBuiltinModID(depID) || provided[depID] {
				continue
			}
			report.Issues = append(report.Issues, VerifyIssue{
				Kind:    "missing-dependency",
				ModID:   dep.ModID,
				File:    file,
				Message: fmt.Sprintf("missing required dependency %q", dep.ModID),
				Suggest: fmt.Sprintf("mod-downloader-cli install modrinth:%s", dep.ModID),
			})
		}
		if modLoader != "" && !loadersCompatible(mod.Loaders, modLoader) {
			report.Issues = append(report.Issues, VerifyIssue{
				Kind:     "incompatible",
				ModID:    mod.ID,
				File:     file,
				Message:  fmt.Sprintf("mod targets loader %q but the instance uses %q", strings.Join(mod.Loaders, ", "), modLoader),
				Advisory: true,
			})
		}
		if minecraftVersion != "" && mod.MinecraftVersion != "" {
			if allowed, known := minecraft.VersionRangeAllows(mod.MinecraftVersion, minecraftVersion); known && !allowed {
				report.Issues = append(report.Issues, VerifyIssue{
					Kind:     "incompatible",
					ModID:    mod.ID,
					File:     file,
					Message:  fmt.Sprintf("mod supports Minecraft %s but the instance is %s", mod.MinecraftVersion, minecraftVersion),
					Advisory: true,
				})
			}
		}
	}

	report.Provided = sortedMapKeys(provided)
	sort.SliceStable(report.Issues, func(i, j int) bool {
		if report.Issues[i].Kind != report.Issues[j].Kind {
			return report.Issues[i].Kind < report.Issues[j].Kind
		}
		return report.Issues[i].ModID < report.Issues[j].ModID
	})

	blocking, advisory := 0, 0
	for _, issue := range report.Issues {
		if issue.Advisory {
			advisory++
		} else {
			blocking++
		}
	}
	report.Passed = blocking == 0 && (!strict || advisory == 0)
	return report
}

// loadersCompatible accepts an exact loader match and the Quilt-on-Fabric
// compatibility rule.
func loadersCompatible(loaders []string, instanceLoader string) bool {
	instanceLoader = strings.ToLower(strings.TrimSpace(instanceLoader))
	if instanceLoader == "" || len(loaders) == 0 {
		return true
	}
	for _, loader := range loaders {
		loader = strings.ToLower(strings.TrimSpace(loader))
		if loader == instanceLoader || (loader == "quilt" && instanceLoader == "fabric") {
			return true
		}
	}
	return false
}

func resolveLocalModPath(modsDir string, mod structs.ModInfo) string {
	path := strings.TrimSpace(mod.Path)
	if path == "" {
		if strings.TrimSpace(mod.FileName) == "" {
			return ""
		}
		path = filepath.Join(modsDir, mod.FileName+".jar")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(modsDir, path)
	}
	return filepath.Clean(path)
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
