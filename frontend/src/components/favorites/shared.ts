import type { FavoriteModEntry } from "@/types";

export type CopyMode = "selected" | "list" | "reference";

export const displayName = (item: FavoriteModEntry) => item.title || item.slug || item.modId;
