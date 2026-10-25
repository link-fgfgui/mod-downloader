import assert from "node:assert/strict";
import { test } from "node:test";

import {
    buildModDependencyGraph,
    collectMatchingTreeKeys,
    collectRelatedTreeKeys,
    findReverseDependents,
    flattenModDependencyTree,
} from "../src/composables/useModDependencyGraph.ts";

const mod = (over: Record<string, unknown>) => ({
    id: "",
    name: "",
    version: "",
    fileName: "",
    path: "",
    enabled: true,
    dependencies: [],
    jijMods: [],
    ...over,
});

test("roots are mods nothing else depends on", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "app", path: "/mods/app.jar", name: "App", dependencies: [{ modId: "library", type: "required" }] }),
        mod({ id: "library", path: "/mods/library.jar", name: "Library" }),
    ]);
    assert.equal(graph.roots.length, 1);
    assert.equal(graph.roots[0].name, "App");
    assert.equal(graph.roots[0].children.length, 1);
    assert.equal(graph.roots[0].children[0].name, "Library");
});

test("missing dependencies become flagged leaves", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "app", path: "/mods/app.jar", name: "App", dependencies: [{ modId: "ghost", type: "required" }] }),
    ]);
    assert.equal(graph.missing.length, 1);
    assert.equal(graph.missing[0].isMissing, true);
    assert.equal(graph.roots[0].children[0].isMissing, true);
});

test("jij ids satisfy dependencies and ignore pseudo deps", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "app", path: "/mods/app.jar", name: "App", dependencies: [{ modId: "minecraft", type: "required" }, { modId: "bundled", type: "required" }] }),
        mod({ id: "outer", path: "/mods/outer.jar", name: "Outer", jijMods: [{ id: "bundled", name: "Bundled" }] }),
    ]);
    assert.equal(graph.missing.length, 0);
    const app = graph.roots.find((node) => node.name === "App")!;
    assert.equal(app.children.length, 1);
    assert.equal(app.children[0].name, "Outer");
});

test("flatten honors expanded keys and depth", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "a", path: "/a.jar", name: "A", dependencies: [{ modId: "b", type: "required" }] }),
        mod({ id: "b", path: "/b.jar", name: "B", dependencies: [{ modId: "c", type: "required" }] }),
        mod({ id: "c", path: "/c.jar", name: "C" }),
    ]);
    assert.equal(flattenModDependencyTree(graph.roots, new Set()).length, 1);
    const expanded = flattenModDependencyTree(graph.roots, new Set(["/a.jar", "/b.jar"]));
    assert.deepEqual(expanded.map((row) => row.node.name), ["A", "B", "C"]);
});

test("dependency cycles still produce a root and a bounded tree", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "a", path: "/a.jar", name: "A", dependencies: [{ modId: "b", type: "required" }] }),
        mod({ id: "b", path: "/b.jar", name: "B", dependencies: [{ modId: "a", type: "required" }] }),
    ]);
    assert.equal(graph.roots.length, 1);
    const rows = flattenModDependencyTree(graph.roots, new Set(["/a.jar", "/b.jar"]));
    assert.deepEqual(rows.map((row) => row.node.name), ["A", "B", "A"]);
});

test("cycle members reachable from an acyclic root stay nested", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "top", path: "/top.jar", name: "Top", dependencies: [{ modId: "a", type: "required" }] }),
        mod({ id: "a", path: "/a.jar", name: "A", dependencies: [{ modId: "b", type: "required" }] }),
        mod({ id: "b", path: "/b.jar", name: "B", dependencies: [{ modId: "a", type: "required" }] }),
    ]);
    assert.deepEqual(graph.roots.map((node) => node.name), ["Top"]);
});

test("matching keys keep ancestors and hide unrelated branches", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "app", path: "/app.jar", name: "App", dependencies: [{ modId: "lib", type: "required" }] }),
        mod({ id: "lib", path: "/lib.jar", name: "Library" }),
        mod({ id: "other", path: "/other.jar", name: "Other" }),
    ]);
    const visible = collectMatchingTreeKeys(graph.roots, (node) => node.name === "Library");
    assert.deepEqual([...visible].sort(), ["/app.jar", "/lib.jar"]);
    const rows = flattenModDependencyTree(graph.roots, new Set(), { visibleKeys: visible, forceExpand: true });
    assert.deepEqual(rows.map((row) => row.node.name), ["App", "Library"]);
});

test("flatten keeps synthetic missing leaves next to a visible match", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "app", path: "/app.jar", name: "App", dependencies: [{ modId: "ghost", type: "required" }] }),
    ]);
    const visible = collectMatchingTreeKeys(graph.roots, () => true);
    assert.equal(visible.has("missing:ghost"), true);
    const rows = flattenModDependencyTree(graph.roots, new Set(["/app.jar"]), { visibleKeys: visible });
    assert.deepEqual(rows.map((row) => row.node.name), ["App", "ghost"]);
});

test("optional dependencies connect to local providers without becoming missing", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "app", path: "/app.jar", name: "App", dependencies: [{ modId: "lib", type: "optional" }] }),
        mod({ id: "lib", path: "/lib.jar", name: "Library" }),
    ]);
    const app = graph.roots.find((node) => node.name === "App")!;
    assert.equal(app.children.length, 1);
    assert.equal(app.children[0].name, "Library");
    assert.equal(app.children[0].isOptional, true);
    assert.equal(graph.missing.length, 0);

    const withoutProvider = buildModDependencyGraph([
        mod({ id: "app", path: "/app.jar", name: "App", dependencies: [{ modId: "lib", type: "optional" }] }),
    ]);
    assert.equal(withoutProvider.missing.length, 0);
    assert.equal(withoutProvider.roots[0].children.length, 0);
});

test("a required edge dominates an optional edge for node status", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "optional-user", path: "/optional.jar", name: "Optional User", dependencies: [{ modId: "lib", type: "optional" }] }),
        mod({ id: "required-user", path: "/required.jar", name: "Required User", dependencies: [{ modId: "lib", type: "required" }] }),
        mod({ id: "lib", path: "/lib.jar", name: "Library" }),
    ]);
    const lib = graph.roots.find((node) => node.name === "Optional User")!.children.find((node) => node.name === "Library")!;
    assert.equal(lib.isOptional, false);
});

test("jij sub-mods are synthetic children of their host", () => {
    const graph = buildModDependencyGraph([
        mod({
            id: "outer",
            path: "/outer.jar",
            name: "Outer",
            jijMods: [{ id: "bundled", name: "Bundled Library" }],
        }),
    ]);
    const outer = graph.roots.find((node) => node.name === "Outer")!;
    const jij = outer.children.find((node) => node.name === "Bundled Library")!;
    assert.equal(jij.isJij, true);
    assert.equal(jij.isMissing, false);
    assert.deepEqual(jij.parents, ["/outer.jar"]);
});

test("related keys cover both dependency directions", () => {
    const graph = buildModDependencyGraph([
        mod({ id: "a", path: "/a.jar", name: "A", dependencies: [{ modId: "b", type: "required" }] }),
        mod({ id: "b", path: "/b.jar", name: "B", dependencies: [{ modId: "c", type: "required" }] }),
        mod({ id: "c", path: "/c.jar", name: "C" }),
        mod({ id: "d", path: "/d.jar", name: "D", dependencies: [{ modId: "b", type: "required" }] }),
    ]);
    const relations = collectRelatedTreeKeys(graph.roots, "/b.jar");
    assert.deepEqual([...relations.dependencies], ["/c.jar"]);
    assert.deepEqual([...relations.dependents].sort(), ["/a.jar", "/d.jar"]);
    assert.deepEqual([...collectRelatedTreeKeys(graph.roots, "").dependencies], []);
});

test("reverse dependents apply required-only, enabled, and pseudo filtering", () => {
    const matches = findReverseDependents([
        mod({ id: "app", path: "/app.jar", name: "App", dependencies: [{ modId: "lib", version: ">=1.0", type: "required" }] }),
        mod({ id: "disabled", path: "/disabled.jar", name: "Disabled", enabled: false, dependencies: [{ modId: "lib", type: "required" }] }),
        mod({ id: "optional", path: "/optional.jar", name: "Optional", dependencies: [{ modId: "lib", type: "optional" }] }),
        mod({ id: "pseudo", path: "/pseudo.jar", name: "Pseudo", dependencies: [{ modId: "minecraft", type: "required" }] }),
        mod({ id: "excluded", path: "/excluded.jar", name: "Excluded", dependencies: [{ modId: "lib", type: "required" }] }),
    ], ["lib", "minecraft"], { excludePaths: new Set(["/excluded.jar"]) });

    assert.deepEqual(matches.map((match) => match.name), ["App"]);
    assert.deepEqual(matches[0].requirements, [{ modId: "lib", version: ">=1.0" }]);
});
