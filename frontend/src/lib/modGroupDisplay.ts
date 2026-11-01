// Display-only projections over core's LocalModGroup.
//
// Grouping, Jij de-duplication, categories, the loader compatibility verdict
// and the canonical search text are decided by core (appcore.BuildLocalModGroups).
// This module only shapes those values for rendering.

import type { LocalModGroup } from "@/types";
import { pinyinInitialsOf } from "@/utils/modSearchText";

export function groupRowKey(group: LocalModGroup) {
  return group.rowKey;
}

export function strongModIds(group: LocalModGroup) {
  return (group.strong || []).map((mod) => mod.id).filter(Boolean);
}

export function strongModNames(group: LocalModGroup) {
  return (group.strong || []).map((mod) => mod.name || mod.id).filter(Boolean);
}

export function hasGroupedDetails(group: LocalModGroup) {
  return (group.strong?.length || 0) > 1 || (group.jij?.length || 0) > 0;
}

export function groupTooltip(group: LocalModGroup) {
  const parts: string[] = [];
  if ((group.strong?.length || 0) > 1) {
    parts.push(`Declared mods: ${strongModNames(group).join(", ")}`);
  }
  if (group.jij?.length) {
    parts.push(group.jij.map((mod) => mod.name || mod.id).filter(Boolean).join(", "));
  }
  return parts.join("\n");
}

// Core indexes the canonical human-facing fields; the GUI adds pinyin initials
// on top so CJK names stay searchable by their initials ("nlz" -> "钠离子").
export function groupSearchText(group: LocalModGroup) {
  const initials = [
    pinyinInitialsOf(group.displayName),
    ...(group.strong || []).flatMap((mod) => [pinyinInitialsOf(mod.name), pinyinInitialsOf(mod.onlineName || "")]),
    ...(group.categories || []).map((category) => pinyinInitialsOf(category)),
    ...(group.jij || []).map((mod) => pinyinInitialsOf(mod.name)),
  ].filter(Boolean);
  return [group.searchText, ...initials].join("\n");
}
