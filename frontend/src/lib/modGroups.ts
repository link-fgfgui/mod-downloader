import type { JijModInfo, ModInfo } from "@/types";
import { buildModSearchText } from "@/utils/modSearchText";

export interface ModGroup {
  primary: ModInfo;
  strong: ModInfo[];
  jij: JijModInfo[];
}

export function groupMods(mods: ModInfo[]): ModGroup[] {
  const groups = new Map<string, ModGroup>();
  for (const mod of mods || []) {
    const key = mod.fileName || mod.path || mod.id;
    if (!key) continue;
    if (!groups.has(key)) groups.set(key, { primary: mod, strong: [], jij: [] });
    const group = groups.get(key)!;
    group.strong.push(mod);
    addJijMods(group, mod.jijMods || []);
    if (!group.primary) group.primary = mod;
  }
  return [...groups.values()].filter((group) => group.primary);
}

function addJijMods(group: ModGroup, mods: JijModInfo[]) {
  for (const mod of mods) {
    const id = (mod.id || "").trim();
    if (!id || group.jij.some((existing) => (existing.id || "").toLowerCase() === id.toLowerCase())) {
      continue;
    }
    group.jij.push(mod);
  }
}

export function groupRowKey(group: ModGroup) {
  const mod = group.primary;
  return [mod.id, mod.sha1, mod.path, mod.fileName].filter(Boolean).join("|");
}

export function displayModName(group: ModGroup | null | undefined) {
  const mod = group?.primary;
  return mod?.onlineName || mod?.name || mod?.id || "";
}

export function modCategories(group: ModGroup) {
  const categories = group.primary?.categories || [];
  const seen = new Set<string>();
  return categories
    .map((category) => (category || "").trim())
    .filter((category) => {
      const key = category.toLowerCase();
      if (!key || seen.has(key)) return false;
      seen.add(key);
      return true;
    });
}

export function groupLoaders(group: ModGroup) {
  const loaders = new Set<string>();
  for (const mod of group.strong || []) {
    for (const loader of mod.loaders || []) {
      const normalized = (loader || "").trim().toLowerCase();
      if (normalized) loaders.add(normalized);
    }
  }
  return loaders;
}

export function isGroupIncompatible(group: ModGroup, selectedLoader: string) {
  const loaders = groupLoaders(group);
  const normalized = (selectedLoader || "").trim().toLowerCase();
  return loaders.size > 0 && Boolean(normalized) && !loaders.has(normalized);
}

export function searchableGroupText(group: ModGroup) {
  return buildModSearchText(group.strong || []);
}

export function strongModIds(group: ModGroup) {
  return group.strong.map((mod) => mod.id).filter(Boolean);
}

export function strongModNames(group: ModGroup) {
  return group.strong.map((mod) => mod.name || mod.id).filter(Boolean);
}

export function hasGroupedDetails(group: ModGroup) {
  return group.strong.length > 1 || group.jij.length > 0;
}

export function groupTooltip(group: ModGroup) {
  const parts: string[] = [];
  if (group.strong.length > 1) {
    parts.push(`Declared mods: ${strongModNames(group).join(", ")}`);
  }
  if (group.jij.length) {
    parts.push(group.jij.map((mod) => mod.name || mod.id).filter(Boolean).join(", "));
  }
  return parts.join("\n");
}
