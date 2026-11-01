// SPDX-FileCopyrightText: 2026 link-fgfgui
// SPDX-License-Identifier: LGPL-3.0-or-later WITH LGPL-3.0-linking-exception

package modgraph

import (
	"testing"

	minecraft "github.com/link-fgfgui/mod-downloader/core/structs/minecraft"
)

func TestBuildRootsMissingAndJijChildren(t *testing.T) {
	graph := Build([]minecraft.ModInfo{
		{
			ID:      "host",
			Name:    "Host Mod",
			Path:    "/mods/host.jar",
			Enabled: true,
			Dependencies: []minecraft.LocalModDependency{
				{ModID: "library", Type: "required"},
				{ModID: "absent", Type: "required"},
				{ModID: "optional-local", Type: "optional"},
				{ModID: "optional-missing", Type: "optional"},
				{ModID: "minecraft", Type: "required"},
			},
			JIJMods: []minecraft.JIJModInfo{{ID: "nested", Name: "Nested Bundled"}},
		},
		{ID: "library", Name: "Library", Path: "/mods/library.jar", Enabled: true},
		{ID: "optional-local", Name: "Optional Local", Path: "/mods/optional.jar", Enabled: false},
	})

	nodes := graphNodes(graph)
	if len(graph.Roots) != 1 || graph.Roots[0] != "/mods/host.jar" {
		t.Fatalf("roots = %#v, want host jar", graph.Roots)
	}
	if len(graph.Missing) != 1 || nodes[graph.Missing[0]].Name != "absent" {
		t.Fatalf("missing = %#v", graph.Missing)
	}
	host := nodes["/mods/host.jar"]
	if host == nil || !host.IsRoot || host.Name != "Host Mod" {
		t.Fatalf("host node = %#v", host)
	}
	children := keysToNames(nodes, host.ChildKeys)
	if !containsString(children, "Library") || !containsString(children, "absent") || !containsString(children, "Nested Bundled") {
		t.Fatalf("host children = %#v", children)
	}
	if !containsString(children, "Optional Local") {
		t.Fatal("optional dependency with a local provider must become a child")
	}
	optional := nodes["/mods/optional.jar"]
	if optional == nil || !optional.IsOptional {
		t.Fatalf("optional node = %#v", optional)
	}
	if got := nodes["/mods/library.jar"].ParentKeys; len(got) != 1 || got[0] != "/mods/host.jar" {
		t.Fatalf("library parents = %#v", got)
	}
}

func TestBuildHandlesCycles(t *testing.T) {
	graph := Build([]minecraft.ModInfo{
		{ID: "a", Name: "A", Path: "/mods/a.jar", Enabled: true, Dependencies: []minecraft.LocalModDependency{{ModID: "b", Type: "required"}}},
		{ID: "b", Name: "B", Path: "/mods/b.jar", Enabled: true, Dependencies: []minecraft.LocalModDependency{{ModID: "a", Type: "required"}}},
	})
	if len(graph.Roots) != 1 {
		t.Fatalf("roots = %#v, want one condensed root", graph.Roots)
	}
}

func TestFindReverseDependents(t *testing.T) {
	mods := []minecraft.ModInfo{
		{ID: "library", Name: "Library", Path: "/mods/library.jar", Enabled: true},
		{ID: "user", Name: "User", Path: "/mods/user.jar", Enabled: true, Dependencies: []minecraft.LocalModDependency{{ModID: "library", Type: "required", Version: ">=1.0"}}},
		{ID: "disabled", Name: "Disabled", Path: "/mods/disabled.jar", Enabled: false, Dependencies: []minecraft.LocalModDependency{{ModID: "library", Type: "required"}}},
		{ID: "self", Name: "Self", Path: "/mods/self.jar", Enabled: true, Dependencies: []minecraft.LocalModDependency{{ModID: "self", Type: "required"}}},
	}

	matches := FindReverseDependents(mods, []string{"library"}, nil, true)
	if len(matches) != 1 || matches[0].Name != "User" || len(matches[0].Requirements) != 1 || matches[0].Requirements[0].Version != ">=1.0" {
		t.Fatalf("matches = %#v", matches)
	}

	excluded := FindReverseDependents(mods, []string{"library"}, []string{"/mods/user.jar"}, true)
	if len(excluded) != 0 {
		t.Fatalf("excluded matches = %#v", excluded)
	}

	all := FindReverseDependents(mods, []string{"library"}, nil, false)
	if len(all) != 2 {
		t.Fatalf("include-disabled matches = %#v", all)
	}
}

func TestProvidedIDsIncludesJij(t *testing.T) {
	ids := ProvidedIDs(minecraft.ModInfo{
		ID:      "Host",
		JIJMods: []minecraft.JIJModInfo{{ID: "Nested"}, {ID: "nested"}, {ID: ""}},
	})
	if len(ids) != 2 || ids[0] != "host" || ids[1] != "nested" {
		t.Fatalf("ProvidedIDs() = %#v", ids)
	}
}

func graphNodes(graph Graph) map[string]*Node {
	nodes := make(map[string]*Node, len(graph.Nodes))
	for i := range graph.Nodes {
		nodes[graph.Nodes[i].Key] = &graph.Nodes[i]
	}
	return nodes
}

func keysToNames(nodes map[string]*Node, keys []string) []string {
	names := make([]string, 0, len(keys))
	for _, key := range keys {
		if node := nodes[key]; node != nil {
			names = append(names, node.Name)
		}
	}
	return names
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
