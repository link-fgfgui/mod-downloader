import { describe, expect, it } from "vitest";

import {
  collectMatchingTreeKeys,
  collectRelatedTreeKeys,
  flattenModDependencyTree,
  hydrateModDependencyGraph,
} from "@/lib/modDependencyGraph";
import type { ModDependencyGraph, ModGraphNode } from "@/types";

const node = (
  overrides: Partial<ModGraphNode> & Pick<ModGraphNode, "key" | "name">,
): ModGraphNode => {
  const { key, name, ...rest } = overrides;
  return {
    key,
    name,
    enabled: true,
    isRoot: false,
    isJij: false,
    isMissing: false,
    isOptional: false,
    searchText: name.toLowerCase(),
    ...rest,
  };
};

const graph: ModDependencyGraph = {
  nodes: [
    node({
      key: "/mods/host.jar",
      name: "Host Mod",
      isRoot: true,
      childKeys: ["missing:absent", "/mods/library.jar", "jij:/mods/host.jar:nested"],
      searchText: "host mod",
    }),
    node({
      key: "/mods/library.jar",
      name: "Library",
      parentKeys: ["/mods/host.jar"],
      searchText: "library",
    }),
    node({
      key: "missing:absent",
      name: "absent",
      isMissing: true,
      searchText: "absent",
    }),
    node({
      key: "jij:/mods/host.jar:nested",
      name: "Nested Bundled",
      isJij: true,
      parentKeys: ["/mods/host.jar"],
      searchText: "nested bundled",
    }),
  ],
  roots: ["/mods/host.jar"],
  missing: ["missing:absent"],
};

describe("modDependencyGraph", () => {
  it("hydrates core's flat graph into nested children", () => {
    const hydrated = hydrateModDependencyGraph(graph);
    expect(hydrated.roots).toHaveLength(1);
    expect(hydrated.roots[0].name).toBe("Host Mod");
    expect(hydrated.missing.map((item) => item.name)).toEqual(["absent"]);
    expect(hydrated.roots[0].children.map((item) => item.key)).toEqual([
      "missing:absent",
      "/mods/library.jar",
      "jij:/mods/host.jar:nested",
    ]);
  });

  it("flattens expanded branches and filters by visible keys", () => {
    const hydrated = hydrateModDependencyGraph(graph);
    const visible = collectMatchingTreeKeys(hydrated.roots, (item) => item.name.includes("Library"));
    const rows = flattenModDependencyTree(hydrated.roots, new Set(), {
      visibleKeys: visible,
      forceExpand: true,
    });
    expect(rows.map((row) => row.node.name)).toEqual(["Host Mod", "absent", "Library"]);
  });

  it("collects both directions around a selected node", () => {
    const hydrated = hydrateModDependencyGraph(graph);
    const relations = collectRelatedTreeKeys(hydrated.roots, "/mods/library.jar");
    expect(relations.dependents.has("/mods/host.jar")).toBe(true);
    expect(relations.dependencies.size).toBe(0);
  });
});
