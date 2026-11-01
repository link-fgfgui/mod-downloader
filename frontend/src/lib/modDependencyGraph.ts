// Shapes core's flat dependency forest into the nested tree the Manage view
// renders.
//
// Graph construction (provider matching, JIJ children, missing leaves, cycle
// condensation) lives in core/modgraph. This module only hydrates key
// references into child objects and keeps the traversal/selection helpers the
// React tree needs.

import { pinyinInitialsOf } from "@/utils/modSearchText";
import type { ModDependencyGraph, ModGraphNode } from "@/types";

export interface ModTreeNode extends ModGraphNode {
  children: ModTreeNode[];
  parents: string[];
  /** Core search text plus GUI-only pinyin initials. */
  searchText: string;
}

export interface HydratedModDependencyGraph {
  roots: ModTreeNode[];
  missing: ModTreeNode[];
}

export interface TreeRelationKeys {
  dependencies: Set<string>;
  dependents: Set<string>;
}

export interface FlattenedTreeRow {
  key: string;
  node: ModTreeNode;
  depth: number;
  hasChildren: boolean;
  expanded: boolean;
}

export interface FlattenTreeOptions {
  visibleKeys?: Set<string> | null;
  forceExpand?: boolean;
}

const EMPTY_GRAPH: ModDependencyGraph = { nodes: [], roots: [], missing: [] };

const nodeSearchText = (node: ModGraphNode) =>
  [node.searchText, ...(node.searchPinyinSeeds ?? []).map(pinyinInitialsOf)]
    .filter(Boolean)
    .join("\n");

export function hydrateModDependencyGraph(
  graph: ModDependencyGraph | null | undefined,
): HydratedModDependencyGraph {
  const source = graph ?? EMPTY_GRAPH;
  const byKey = new Map<string, ModTreeNode>();
  for (const node of source.nodes ?? []) {
    byKey.set(node.key, {
      ...node,
      children: [],
      parents: [...(node.parentKeys ?? [])],
      searchText: nodeSearchText(node),
    });
  }
  for (const node of byKey.values()) {
    node.children = (node.childKeys ?? [])
      .map((key) => byKey.get(key))
      .filter((child): child is ModTreeNode => Boolean(child));
  }
  const resolve = (keys: string[] | null | undefined) =>
    (keys ?? []).map((key) => byKey.get(key)).filter((node): node is ModTreeNode => Boolean(node));
  return {
    roots: resolve(source.roots),
    missing: resolve(source.missing),
  };
}

export function collectMatchingTreeKeys(
  roots: ModTreeNode[],
  matches: (node: ModTreeNode) => boolean,
): Set<string> {
  const visible = new Set<string>();
  const memo = new Map<string, boolean>();
  const visiting = new Set<string>();

  const visit = (node: ModTreeNode): boolean => {
    const cached = memo.get(node.key);
    if (cached !== undefined) return cached;
    if (visiting.has(node.key)) return false;
    visiting.add(node.key);

    let included = !node.isMissing && matches(node);
    for (const child of node.children) {
      if (visit(child)) included = true;
    }
    visiting.delete(node.key);
    memo.set(node.key, included);
    if (included) {
      visible.add(node.key);
      for (const child of node.children) {
        if (child.isMissing) visible.add(child.key);
      }
    }
    return included;
  };

  for (const root of roots) visit(root);
  return visible;
}

const indexTreeNodes = (roots: ModTreeNode[]): Map<string, ModTreeNode> => {
  const index = new Map<string, ModTreeNode>();
  const walk = (node: ModTreeNode) => {
    if (index.has(node.key)) return;
    index.set(node.key, node);
    for (const child of node.children) walk(child);
  };
  for (const root of roots) walk(root);
  return index;
};

export function collectRelatedTreeKeys(
  roots: ModTreeNode[],
  selectedKey: string,
): TreeRelationKeys {
  const dependencies = new Set<string>();
  const dependents = new Set<string>();
  if (!selectedKey) return { dependencies, dependents };
  const index = indexTreeNodes(roots);
  const selected = index.get(selectedKey);
  if (!selected) return { dependencies, dependents };

  const visited = new Set([selected.key]);
  const walkDown = (node: ModTreeNode) => {
    for (const child of node.children) {
      if (visited.has(child.key)) continue;
      visited.add(child.key);
      dependencies.add(child.key);
      walkDown(child);
    }
  };
  walkDown(selected);

  const walkUp = (node: ModTreeNode, seen: Set<string>) => {
    for (const parentKey of node.parents) {
      if (seen.has(parentKey)) continue;
      seen.add(parentKey);
      dependents.add(parentKey);
      const parent = index.get(parentKey);
      if (parent) walkUp(parent, seen);
    }
  };
  walkUp(selected, new Set([selected.key]));
  return { dependencies, dependents };
}

const MAX_VISIBLE_DEPTH = 12;

const isVisible = (node: ModTreeNode, options: FlattenTreeOptions) =>
  !options.visibleKeys || options.visibleKeys.has(node.key);

export function flattenModDependencyTree(
  roots: ModTreeNode[],
  expandedKeys: Set<string>,
  options: FlattenTreeOptions = {},
  depth = 0,
  ancestors: Set<string> = new Set(),
  parentRowKey = "",
): FlattenedTreeRow[] {
  const rows: FlattenedTreeRow[] = [];
  for (const node of roots) {
    if (!isVisible(node, options)) continue;
    const visibleChildren = node.children.filter((child) => isVisible(child, options));
    const hasChildren = visibleChildren.length > 0 && depth < MAX_VISIBLE_DEPTH;
    const expanded = hasChildren && (options.forceExpand || expandedKeys.has(node.key));
    const key = parentRowKey ? `${parentRowKey}\u0000${node.key}` : node.key;
    rows.push({ key, node, depth, hasChildren, expanded });
    if (expanded && !ancestors.has(node.key)) {
      const nextAncestors = new Set(ancestors).add(node.key);
      rows.push(
        ...flattenModDependencyTree(visibleChildren, expandedKeys, options, depth + 1, nextAncestors, key),
      );
    }
  }
  return rows;
}
