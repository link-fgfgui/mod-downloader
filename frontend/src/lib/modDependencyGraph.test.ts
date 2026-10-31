import { describe, expect, it } from "vitest";

import {
  buildModDependencyGraph,
  collectMatchingTreeKeys,
  collectRelatedTreeKeys,
  flattenModDependencyTree,
} from "@/lib/modDependencyGraph";

const mods = [
  {
    id: "host",
    name: "Host Mod",
    path: "/mods/host.jar",
    enabled: true,
    dependencies: [
      { modId: "library", type: "required" },
      { modId: "absent", type: "required" },
      { modId: "optional-local", type: "optional" },
      { modId: "optional-missing", type: "optional" },
    ],
    jijMods: [{ id: "nested", name: "Nested Bundled" }],
  },
  {
    id: "library",
    name: "Library",
    path: "/mods/library.jar",
    enabled: true,
  },
  {
    id: "optional-local",
    name: "Optional Local",
    path: "/mods/optional.jar",
    enabled: false,
  },
];

describe("modDependencyGraph", () => {
  it("builds roots, missing leaves and JIJ children", () => {
    const graph = buildModDependencyGraph(mods);
    expect(graph.roots).toHaveLength(1);
    expect(graph.roots[0].name).toBe("Host Mod");
    expect(graph.missing.map((node) => node.name)).toEqual(["absent"]);
    const children = graph.roots[0].children.map((node) => node.key);
    expect(children).toContain("/mods/library.jar");
    expect(children).toContain("missing:absent");
    expect(children.some((key) => key.startsWith("jij:"))).toBe(true);
  });

  it("flattens expanded branches and filters by visible keys", () => {
    const graph = buildModDependencyGraph(mods);
    const visible = collectMatchingTreeKeys(graph.roots, (node) => node.name.includes("Library"));
    const rows = flattenModDependencyTree(graph.roots, new Set(), { visibleKeys: visible, forceExpand: true });
    expect(rows.map((row) => row.node.name)).toEqual(["Host Mod", "absent", "Library"]);
  });

  it("collects both directions around a selected node", () => {
    const graph = buildModDependencyGraph(mods);
    const relations = collectRelatedTreeKeys(graph.roots, "/mods/library.jar");
    expect(relations.dependents.has("/mods/host.jar")).toBe(true);
    expect(relations.dependencies.size).toBe(0);
  });
});
