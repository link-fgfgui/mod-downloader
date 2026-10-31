// Builds a local dependency forest from the scanned instance mods.
//
// Edges are derived from each mod's declared required dependencies (the parser
// already strips minecraft/java/loader pseudo-deps). Optional dependencies only
// become edges when a local provider exists, so an uninstalled optional library
// never turns into a false "missing" gap. JIJ sub-mods become synthetic children
// of their host. A dependency that no local mod provides becomes a synthetic
// "missing" leaf so gaps are visible. Roots are the source components of the
// graph, i.e. mods no other local mod depends on. Strongly connected components
// are condensed first so a dependency cycle still yields a usable tree instead
// of an empty root list.

import { buildModSearchText } from "@/utils/modSearchText";

export interface ModTreeNode {
  key: string;
  name: string;
  version: string;
  enabled: boolean;
  isRoot: boolean;
  isJij: boolean;
  isMissing: boolean;
  isOptional: boolean;
  path: string;
  iconUrl: string;
  onlinePlatform: string;
  onlineProjectId: string;
  searchText: string;
  children: ModTreeNode[];
  parents: string[];
}

interface BuildableNode extends ModTreeNode {
  raw: LocalModLike;
  requiredDependencyIds: string[];
  optionalDependencyIds: string[];
  requiredIncoming: number;
  optionalIncoming: number;
}

interface LocalModLike {
  id?: string;
  name?: string;
  onlineName?: string;
  version?: string;
  fileName?: string;
  path?: string;
  enabled?: boolean;
  iconUrl?: string;
  onlinePlatform?: string;
  onlineProjectId?: string;
  dependencies?: Array<{ modId?: string; type?: string; version?: string }> | null;
  jijMods?: Array<{ id?: string; name?: string }> | null;
}

export interface ModDependencyGraph {
  roots: ModTreeNode[];
  missing: ModTreeNode[];
}

export interface ModRequirement {
  modId: string;
  version: string;
}

export interface ReverseDependencyMatch {
  key: string;
  name: string;
  version: string;
  requirements: ModRequirement[];
}

export interface ReverseDependencyOptions {
  excludePaths?: Set<string>;
  requireEnabled?: boolean;
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

const PSEUDO_DEPENDENCIES = new Set([
  "minecraft",
  "java",
  "fabricloader",
  "quilt_loader",
  "forge",
  "neoforge",
  "modloader",
]);

const normalizeId = (value?: string) => (value || "").trim().toLowerCase();

const displayName = (mod: LocalModLike) =>
  mod?.onlineName || mod?.name || mod?.id || mod?.fileName || "";

const requiredDependenciesOf = (mod: LocalModLike): ModRequirement[] => {
  const seen = new Set<string>();
  const requirements: ModRequirement[] = [];
  for (const dep of mod?.dependencies || []) {
    const type = normalizeId(dep?.type);
    if (type && type !== "required") continue;
    const id = normalizeId(dep?.modId);
    if (!id || PSEUDO_DEPENDENCIES.has(id) || seen.has(id)) continue;
    seen.add(id);
    requirements.push({ modId: id, version: (dep?.version || "").trim() });
  }
  return requirements;
};

const optionalDependencyIdsOf = (mod: LocalModLike): string[] => {
  const ids = new Set<string>();
  for (const dep of mod?.dependencies || []) {
    if (normalizeId(dep?.type) !== "optional") continue;
    const id = normalizeId(dep?.modId);
    if (id && !PSEUDO_DEPENDENCIES.has(id)) ids.add(id);
  }
  return [...ids];
};

export const providedIdsOf = (mod: LocalModLike): string[] => {
  const ids = new Set<string>();
  const id = normalizeId(mod?.id);
  if (id) ids.add(id);
  for (const jij of mod?.jijMods || []) {
    const jijId = normalizeId(jij?.id);
    if (jijId) ids.add(jijId);
  }
  return [...ids];
};

const byName = (a: ModTreeNode, b: ModTreeNode) =>
  a.name.toLowerCase().localeCompare(b.name.toLowerCase()) || a.key.localeCompare(b.key);

const stronglyConnectedComponents = (nodes: Map<string, BuildableNode>) => {
  const index = new Map<string, number>();
  const lowLink = new Map<string, number>();
  const onStack = new Set<string>();
  const stack: string[] = [];
  const component = new Map<string, number>();
  let nextIndex = 0;
  let componentCount = 0;

  const visit = (key: string) => {
    index.set(key, nextIndex);
    lowLink.set(key, nextIndex);
    nextIndex += 1;
    stack.push(key);
    onStack.add(key);
    const node = nodes.get(key)!;
    for (const child of node.children) {
      if (!nodes.has(child.key)) continue;
      if (!index.has(child.key)) {
        visit(child.key);
        lowLink.set(key, Math.min(lowLink.get(key)!, lowLink.get(child.key)!));
      } else if (onStack.has(child.key)) {
        lowLink.set(key, Math.min(lowLink.get(key)!, index.get(child.key)!));
      }
    }
    if (lowLink.get(key) === index.get(key)) {
      while (stack.length > 0) {
        const member = stack.pop()!;
        onStack.delete(member);
        component.set(member, componentCount);
        if (member === key) break;
      }
      componentCount += 1;
    }
  };

  for (const key of nodes.keys()) {
    if (!index.has(key)) visit(key);
  }
  return { component, componentCount };
};

const sourceRootKeys = (nodes: Map<string, BuildableNode>): string[] => {
  const { component, componentCount } = stronglyConnectedComponents(nodes);
  const componentInDegree = new Array<number>(componentCount).fill(0);
  for (const node of nodes.values()) {
    const from = component.get(node.key)!;
    for (const child of node.children) {
      const to = component.get(child.key);
      if (to === undefined || to === from) continue;
      componentInDegree[to] += 1;
    }
  }
  const representatives = new Map<number, BuildableNode>();
  for (const node of nodes.values()) {
    const id = component.get(node.key)!;
    if (componentInDegree[id] !== 0) continue;
    const current = representatives.get(id);
    if (!current || byName(node, current) < 0) representatives.set(id, node);
  }
  return [...representatives.values()].map((node) => node.key);
};

const jijNodeKey = (hostKey: string, jijId: string) => `jij:${hostKey}:${jijId}`;

export function buildModDependencyGraph(mods: LocalModLike[]): ModDependencyGraph {
  const nodes = new Map<string, BuildableNode>();
  const providerIndex = new Map<string, string[]>();

  for (const mod of mods || []) {
    const key = mod?.path || mod?.fileName || mod?.id;
    if (!key || nodes.has(key)) continue;
    const providedIds = providedIdsOf(mod);
    const node: BuildableNode = {
      key,
      name: displayName(mod),
      version: mod?.version || "",
      enabled: Boolean(mod?.enabled),
      isRoot: false,
      isJij: false,
      isMissing: false,
      isOptional: false,
      path: mod?.path || "",
      iconUrl: mod?.iconUrl || "",
      onlinePlatform: mod?.onlinePlatform || "",
      onlineProjectId: mod?.onlineProjectId || "",
      searchText: buildModSearchText([mod]),
      children: [],
      parents: [],
      raw: mod,
      requiredDependencyIds: requiredDependenciesOf(mod).map((dep) => dep.modId),
      optionalDependencyIds: optionalDependencyIdsOf(mod),
      requiredIncoming: 0,
      optionalIncoming: 0,
    };
    nodes.set(key, node);
    for (const id of providedIds) {
      if (!providerIndex.has(id)) providerIndex.set(id, []);
      providerIndex.get(id)!.push(key);
    }
  }

  const missingNodes = new Map<string, ModTreeNode>();

  const linkProviders = (node: BuildableNode, dependencyId: string, optional: boolean) => {
    const providers = providerIndex.get(dependencyId) || [];
    if (!providers.length) return false;
    for (const providerKey of providers) {
      if (providerKey === node.key) continue;
      const provider = nodes.get(providerKey)!;
      if (node.children.some((child) => child.key === providerKey)) continue;
      node.children.push(provider);
      provider.parents.push(node.key);
      if (optional) provider.optionalIncoming += 1;
      else provider.requiredIncoming += 1;
    }
    return true;
  };

  for (const node of nodes.values()) {
    for (const depId of node.requiredDependencyIds) {
      if (linkProviders(node, depId, false)) continue;
      const missingKey = `missing:${depId}`;
      let missing = missingNodes.get(missingKey);
      if (!missing) {
        missing = {
          key: missingKey,
          name: depId,
          version: "",
          enabled: false,
          isRoot: false,
          isJij: false,
          isMissing: true,
          isOptional: false,
          path: "",
          iconUrl: "",
          onlinePlatform: "",
          onlineProjectId: "",
          searchText: depId,
          children: [],
          parents: [],
        };
        missingNodes.set(missingKey, missing);
      }
      if (!node.children.some((child) => child.key === missingKey)) node.children.push(missing);
    }
    for (const depId of node.optionalDependencyIds) {
      linkProviders(node, depId, true);
    }
  }

  for (const node of nodes.values()) {
    node.isOptional = node.requiredIncoming === 0 && node.optionalIncoming > 0;
  }

  const jijNodes = new Map<string, ModTreeNode>();
  for (const node of nodes.values()) {
    for (const jij of node.raw?.jijMods || []) {
      const jijId = normalizeId(jij?.id);
      if (!jijId) continue;
      const key = jijNodeKey(node.key, jijId);
      if (jijNodes.has(key)) continue;
      const jijNode: ModTreeNode = {
        key,
        name: jij?.name || jij?.id || jijId,
        version: "",
        enabled: node.enabled,
        isRoot: false,
        isJij: true,
        isMissing: false,
        isOptional: false,
        path: "",
        iconUrl: "",
        onlinePlatform: "",
        onlineProjectId: "",
        searchText: buildModSearchText([{ id: jij?.id, name: jij?.name }]),
        children: [],
        parents: [node.key],
      };
      jijNodes.set(key, jijNode);
      node.children.push(jijNode);
    }
  }

  const roots: ModTreeNode[] = [];
  for (const key of sourceRootKeys(nodes)) {
    const node = nodes.get(key)!;
    node.isRoot = true;
    roots.push(node);
  }

  roots.sort((a, b) => Number(b.enabled) - Number(a.enabled) || byName(a, b));
  for (const node of nodes.values()) node.children.sort(byName);

  return { roots, missing: [...missingNodes.values()].sort(byName) };
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

export function collectRelatedTreeKeys(roots: ModTreeNode[], selectedKey: string): TreeRelationKeys {
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

export function findReverseDependents(
  mods: LocalModLike[],
  providedIds: string[],
  options: ReverseDependencyOptions = {},
): ReverseDependencyMatch[] {
  const provided = new Set((providedIds || []).map(normalizeId).filter(Boolean));
  if (!provided.size) return [];
  const excludedPaths = options.excludePaths || new Set<string>();
  const requireEnabled = options.requireEnabled !== false;
  const matches = new Map<string, ReverseDependencyMatch>();

  for (const mod of mods || []) {
    if (!mod) continue;
    if (requireEnabled && !mod.enabled) continue;
    const path = (mod.path || "").trim();
    if (path && excludedPaths.has(path)) continue;
    const requirements = requiredDependenciesOf(mod).filter((dep) => provided.has(dep.modId));
    if (!requirements.length) continue;
    const key = path || mod.fileName || mod.id || "";
    if (!key) continue;
    matches.set(key, {
      key,
      name: displayName(mod),
      version: mod.version || "",
      requirements,
    });
  }

  return [...matches.values()].sort(
    (a, b) => a.name.toLowerCase().localeCompare(b.name.toLowerCase()) || a.key.localeCompare(b.key),
  );
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
