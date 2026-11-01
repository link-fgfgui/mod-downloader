// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

// Package modgraph builds the local mod dependency forest shared by the GUI and
// TUI. Edges come from each mod's declared required dependencies (the parser
// already strips minecraft/java/loader pseudo-deps). Optional dependencies only
// become edges when a local provider exists, so an uninstalled optional library
// never turns into a false "missing" gap. JIJ sub-mods become synthetic children
// of their host. A dependency that no local mod provides becomes a synthetic
// "missing" leaf so gaps are visible. Roots are the source components of the
// graph, i.e. mods no other local mod depends on. Strongly connected components
// are condensed first so a dependency cycle still yields a usable tree instead
// of an empty root list.
//
// The graph is serialized as a flat node list plus key references so recursive
// children cannot produce a cyclic JSON payload.
package modgraph

import (
	"sort"
	"strings"

	"github.com/link-fgfgui/mod-downloader/core/internal/strutil"
	minecraft "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

var pseudoDependencies = map[string]bool{
	"minecraft":    true,
	"java":         true,
	"fabricloader": true,
	"quilt_loader": true,
	"forge":        true,
	"neoforge":     true,
	"modloader":    true,
}

// Requirement is one declared dependency that a reverse-dependent mod needs.
type Requirement struct {
	ModID   string `json:"modId"`
	Version string `json:"version,omitempty"`
}

// Node is one renderable row of the dependency forest. ChildKeys and ParentKeys
// reference other Node.Key values; the adapter assembles the tree.
type Node struct {
	Key               string   `json:"key"`
	Name              string   `json:"name"`
	Version           string   `json:"version,omitempty"`
	Enabled           bool     `json:"enabled"`
	IsRoot            bool     `json:"isRoot"`
	IsJij             bool     `json:"isJij"`
	IsMissing         bool     `json:"isMissing"`
	IsOptional        bool     `json:"isOptional"`
	Path              string   `json:"path,omitempty"`
	IconURL           string   `json:"iconUrl,omitempty"`
	OnlinePlatform    string   `json:"onlinePlatform,omitempty"`
	OnlineProjectID   string   `json:"onlineProjectId,omitempty"`
	SearchText        string   `json:"searchText"`
	SearchPinyinSeeds []string `json:"searchPinyinSeeds,omitempty"`
	ChildKeys         []string `json:"childKeys,omitempty"`
	ParentKeys        []string `json:"parentKeys,omitempty"`
}

// Graph is the serialized dependency forest.
type Graph struct {
	Nodes   []Node   `json:"nodes"`
	Roots   []string `json:"roots"`
	Missing []string `json:"missing"`
}

// ReverseDependencyMatch describes an installed mod that requires one of the
// provided mod IDs.
type ReverseDependencyMatch struct {
	Key          string        `json:"key"`
	Name         string        `json:"name"`
	Version      string        `json:"version,omitempty"`
	Requirements []Requirement `json:"requirements"`
}

type buildableNode struct {
	Node
	mod              minecraft.ModInfo
	requiredIDs      []string
	optionalIDs      []string
	requiredIncoming int
	optionalIncoming int
}

// Build constructs the dependency forest for a scanned instance.
func Build(mods []minecraft.ModInfo) Graph {
	nodes := make(map[string]*buildableNode, len(mods))
	providerIndex := make(map[string][]string, len(mods))
	for _, mod := range mods {
		key := strutil.FirstNonEmptyTrimmed(mod.Path, mod.FileName, mod.ID)
		if key == "" {
			continue
		}
		if _, ok := nodes[key]; ok {
			continue
		}
		node := &buildableNode{
			Node: Node{
				Key:               key,
				Name:              displayName(mod),
				Version:           strings.TrimSpace(mod.Version),
				Enabled:           mod.Enabled,
				Path:              strings.TrimSpace(mod.Path),
				IconURL:           strings.TrimSpace(mod.IconURL),
				OnlinePlatform:    strings.TrimSpace(mod.OnlinePlatform),
				OnlineProjectID:   strings.TrimSpace(mod.OnlineProjectID),
				SearchText:        searchTextOf(mod),
				SearchPinyinSeeds: pinyinSeedsOf(mod),
			},
			mod: mod,
		}
		required := requiredDependenciesOf(mod)
		node.requiredIDs = make([]string, 0, len(required))
		for _, dep := range required {
			node.requiredIDs = append(node.requiredIDs, dep.ModID)
		}
		node.optionalIDs = optionalDependencyIDsOf(mod)
		nodes[key] = node
		for _, id := range ProvidedIDs(mod) {
			providerIndex[id] = append(providerIndex[id], key)
		}
	}

	missingNodes := make(map[string]*Node)
	for _, node := range nodes {
		for _, depID := range node.requiredIDs {
			if linkProviders(node, depID, false, nodes, providerIndex) {
				continue
			}
			missingKey := "missing:" + depID
			missing, ok := missingNodes[missingKey]
			if !ok {
				missing = &Node{
					Key:        missingKey,
					Name:       depID,
					IsMissing:  true,
					SearchText: depID,
				}
				missingNodes[missingKey] = missing
			}
			if !containsKey(node.ChildKeys, missingKey) {
				node.ChildKeys = append(node.ChildKeys, missingKey)
			}
		}
		for _, depID := range node.optionalIDs {
			linkProviders(node, depID, true, nodes, providerIndex)
		}
	}

	for _, node := range nodes {
		node.IsOptional = node.requiredIncoming == 0 && node.optionalIncoming > 0
	}

	jijNodes := make(map[string]*Node)
	for _, node := range nodes {
		for _, jij := range node.mod.JIJMods {
			jijID := normalizeID(jij.ID)
			if jijID == "" {
				continue
			}
			key := jijNodeKey(node.Key, jijID)
			if _, ok := jijNodes[key]; ok {
				continue
			}
			name := strutil.FirstNonEmptyTrimmed(jij.Name, jij.ID, jijID)
			jijNode := &Node{
				Key:               key,
				Name:              name,
				Enabled:           node.Enabled,
				IsJij:             true,
				SearchText:        strings.ToLower(strings.Join(nonEmpty(jij.ID, jij.Name), "\n")),
				SearchPinyinSeeds: nonEmpty(strings.TrimSpace(jij.Name)),
				ParentKeys:        []string{node.Key},
			}
			jijNodes[key] = jijNode
			node.ChildKeys = append(node.ChildKeys, key)
		}
	}

	rootKeys := sourceRootKeys(nodes)
	for _, key := range rootKeys {
		if node, ok := nodes[key]; ok {
			node.IsRoot = true
		}
	}

	allNodes := make(map[string]*Node, len(nodes)+len(missingNodes)+len(jijNodes))
	for key, node := range nodes {
		allNodes[key] = &node.Node
	}
	for key, node := range missingNodes {
		allNodes[key] = node
	}
	for key, node := range jijNodes {
		allNodes[key] = node
	}
	for _, node := range nodes {
		sortKeysByNode(node.ChildKeys, allNodes)
	}

	graph := Graph{
		Nodes:   make([]Node, 0, len(allNodes)),
		Roots:   append([]string(nil), rootKeys...),
		Missing: make([]string, 0, len(missingNodes)),
	}
	for _, node := range allNodes {
		graph.Nodes = append(graph.Nodes, *node)
	}
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].Key < graph.Nodes[j].Key })
	for key := range missingNodes {
		graph.Missing = append(graph.Missing, key)
	}
	sort.Slice(graph.Roots, func(i, j int) bool {
		left, right := allNodes[graph.Roots[i]], allNodes[graph.Roots[j]]
		if left.Enabled != right.Enabled {
			return left.Enabled
		}
		return byName(left, right) < 0
	})
	sort.Slice(graph.Missing, func(i, j int) bool {
		return byName(allNodes[graph.Missing[i]], allNodes[graph.Missing[j]]) < 0
	})
	return graph
}

// ProvidedIDs returns the normalized mod IDs a physical JAR provides, including
// its Jar-in-Jar entries.
func ProvidedIDs(mod minecraft.ModInfo) []string {
	seen := make(map[string]bool)
	ids := make([]string, 0, 1+len(mod.JIJMods))
	add := func(value string) {
		id := normalizeID(value)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	add(mod.ID)
	for _, jij := range mod.JIJMods {
		add(jij.ID)
	}
	return ids
}

// FindReverseDependents returns enabled installed mods whose required
// dependencies intersect providedIDs, excluding jars under excludePaths.
func FindReverseDependents(mods []minecraft.ModInfo, providedIDs []string, excludePaths []string, requireEnabled bool) []ReverseDependencyMatch {
	provided := make(map[string]bool, len(providedIDs))
	for _, id := range providedIDs {
		if normalized := normalizeID(id); normalized != "" {
			provided[normalized] = true
		}
	}
	if len(provided) == 0 {
		return nil
	}
	excluded := make(map[string]bool, len(excludePaths))
	for _, path := range excludePaths {
		if trimmed := strings.TrimSpace(path); trimmed != "" {
			excluded[trimmed] = true
		}
	}

	matches := make(map[string]ReverseDependencyMatch)
	for _, mod := range mods {
		if requireEnabled && !mod.Enabled {
			continue
		}
		path := strings.TrimSpace(mod.Path)
		if path != "" && excluded[path] {
			continue
		}
		requirements := make([]Requirement, 0, 2)
		for _, dep := range requiredDependenciesOf(mod) {
			if provided[dep.ModID] {
				requirements = append(requirements, dep)
			}
		}
		if len(requirements) == 0 {
			continue
		}
		key := strutil.FirstNonEmptyTrimmed(path, mod.FileName, mod.ID)
		if key == "" {
			continue
		}
		matches[key] = ReverseDependencyMatch{
			Key:          key,
			Name:         displayName(mod),
			Version:      strings.TrimSpace(mod.Version),
			Requirements: requirements,
		}
	}

	out := make([]ReverseDependencyMatch, 0, len(matches))
	for _, match := range matches {
		out = append(out, match)
	}
	sort.Slice(out, func(i, j int) bool {
		left, right := strings.ToLower(out[i].Name), strings.ToLower(out[j].Name)
		if left != right {
			return left < right
		}
		return out[i].Key < out[j].Key
	})
	return out
}

func linkProviders(node *buildableNode, dependencyID string, optional bool, nodes map[string]*buildableNode, providerIndex map[string][]string) bool {
	providers := providerIndex[dependencyID]
	if len(providers) == 0 {
		return false
	}
	for _, providerKey := range providers {
		if providerKey == node.Key {
			continue
		}
		provider, ok := nodes[providerKey]
		if !ok {
			continue
		}
		if containsKey(node.ChildKeys, providerKey) {
			continue
		}
		node.ChildKeys = append(node.ChildKeys, providerKey)
		provider.ParentKeys = append(provider.ParentKeys, node.Key)
		if optional {
			provider.optionalIncoming++
		} else {
			provider.requiredIncoming++
		}
	}
	return true
}

func requiredDependenciesOf(mod minecraft.ModInfo) []Requirement {
	seen := make(map[string]bool)
	out := make([]Requirement, 0, len(mod.Dependencies))
	for _, dep := range mod.Dependencies {
		kind := normalizeID(dep.Type)
		if kind != "" && kind != "required" {
			continue
		}
		id := normalizeID(dep.ModID)
		if id == "" || pseudoDependencies[id] || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, Requirement{ModID: id, Version: strings.TrimSpace(dep.Version)})
	}
	return out
}

func optionalDependencyIDsOf(mod minecraft.ModInfo) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(mod.Dependencies))
	for _, dep := range mod.Dependencies {
		if normalizeID(dep.Type) != "optional" {
			continue
		}
		id := normalizeID(dep.ModID)
		if id == "" || pseudoDependencies[id] || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

func sourceRootKeys(nodes map[string]*buildableNode) []string {
	index := make(map[string]int, len(nodes))
	lowLink := make(map[string]int, len(nodes))
	onStack := make(map[string]bool, len(nodes))
	stack := make([]string, 0, len(nodes))
	component := make(map[string]int, len(nodes))
	nextIndex := 0
	componentCount := 0

	var visit func(key string)
	visit = func(key string) {
		index[key] = nextIndex
		lowLink[key] = nextIndex
		nextIndex++
		stack = append(stack, key)
		onStack[key] = true
		node := nodes[key]
		for _, childKey := range node.ChildKeys {
			if _, ok := nodes[childKey]; !ok {
				continue
			}
			if _, seen := index[childKey]; !seen {
				visit(childKey)
				if lowLink[childKey] < lowLink[key] {
					lowLink[key] = lowLink[childKey]
				}
			} else if onStack[childKey] && index[childKey] < lowLink[key] {
				lowLink[key] = index[childKey]
			}
		}
		if lowLink[key] != index[key] {
			return
		}
		for len(stack) > 0 {
			member := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[member] = false
			component[member] = componentCount
			if member == key {
				break
			}
		}
		componentCount++
	}

	for key := range nodes {
		if _, seen := index[key]; !seen {
			visit(key)
		}
	}

	componentInDegree := make([]int, componentCount)
	for _, node := range nodes {
		from := component[node.Key]
		for _, childKey := range node.ChildKeys {
			to, ok := component[childKey]
			if !ok || to == from {
				continue
			}
			componentInDegree[to]++
		}
	}

	representatives := make(map[int]*buildableNode, componentCount)
	for _, node := range nodes {
		id := component[node.Key]
		if componentInDegree[id] != 0 {
			continue
		}
		current, ok := representatives[id]
		if !ok || byName(&node.Node, &current.Node) < 0 {
			representatives[id] = node
		}
	}
	roots := make([]string, 0, len(representatives))
	for _, node := range representatives {
		roots = append(roots, node.Key)
	}
	return roots
}

func displayName(mod minecraft.ModInfo) string {
	return strutil.FirstNonEmptyTrimmed(mod.OnlineName, mod.Name, mod.ID, mod.FileName)
}

func searchTextOf(mod minecraft.ModInfo) string {
	values := nonEmpty(
		mod.ID,
		mod.Name,
		mod.OnlineName,
		mod.Version,
		mod.FileName,
		mod.Description,
	)
	values = append(values, nonEmpty(mod.Categories...)...)
	for _, jij := range mod.JIJMods {
		values = append(values, nonEmpty(jij.ID, jij.Name)...)
	}
	return strings.ToLower(strings.Join(values, "\n"))
}

func pinyinSeedsOf(mod minecraft.ModInfo) []string {
	values := nonEmpty(mod.Name, mod.OnlineName)
	values = append(values, nonEmpty(mod.Categories...)...)
	for _, jij := range mod.JIJMods {
		values = append(values, nonEmpty(jij.Name)...)
	}
	return values
}

func nonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func normalizeID(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func containsKey(keys []string, target string) bool {
	for _, key := range keys {
		if key == target {
			return true
		}
	}
	return false
}

func jijNodeKey(hostKey, jijID string) string {
	return "jij:" + hostKey + ":" + jijID
}

func sortKeysByNode(keys []string, nodes map[string]*Node) {
	sort.Slice(keys, func(i, j int) bool {
		return byName(nodes[keys[i]], nodes[keys[j]]) < 0
	})
}

func byName(left, right *Node) int {
	leftName, rightName := strings.ToLower(left.Name), strings.ToLower(right.Name)
	if leftName != rightName {
		if leftName < rightName {
			return -1
		}
		return 1
	}
	if left.Key < right.Key {
		return -1
	}
	if left.Key > right.Key {
		return 1
	}
	return 0
}
