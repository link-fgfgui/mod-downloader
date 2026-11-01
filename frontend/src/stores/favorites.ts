import { create } from "zustand";

import { api } from "@/services/app";
import { currentLocale } from "@/i18n";
import type {
  ExportFavoritePackwizResult,
  FavoriteBulkOperationResult,
  FavoriteCrossVersionCopyApplyResult,
  FavoriteCrossVersionCopyPreview,
  FavoriteList,
  FavoriteListContents,
  FavoriteMod,
  FavoriteModEntry,
  FavoriteOverview,
} from "@/types";

export type FavoriteModDraft = {
  platform: string;
  modId: string;
  versionId?: string;
  minecraftVersion?: string;
  modLoader?: string;
  title?: string;
  slug?: string;
  iconUrl?: string;
  description?: string;
  categories?: string[];
};

export type FavoriteIconMode = "mdi" | "project";

export type FavoriteCrossVersionCopyTarget = {
  sourceListId: string;
  minecraftVersion: string;
  modLoader: string;
  ignoreConflicts?: boolean;
};

export const normalizeKeyPart = (value?: string) => (value || "").trim().toLowerCase();
export const normalizeMinecraftVersion = (value?: string) => (value || "").trim();

const favoriteKey = (
  mod: Pick<FavoriteMod, "platform" | "modId" | "minecraftVersion" | "modLoader">,
) =>
  [
    normalizeKeyPart(mod.platform),
    normalizeKeyPart(mod.modId),
    normalizeMinecraftVersion(mod.minecraftVersion),
    normalizeKeyPart(mod.modLoader),
  ].join("|");

const hasFavoriteScope = (minecraftVersion: string, modLoader: string) =>
  Boolean(minecraftVersion && modLoader);

const emptyOverview = (): FavoriteOverview => ({ pinned: [], ungrouped: [] });

// Core returns both sections already scope-filtered and ordered.
const flattenOverview = (overview: FavoriteOverview) => [
  ...(overview.pinned ?? []),
  ...(overview.ungrouped ?? []),
];

interface FavoritesState {
  lists: FavoriteList[];
  overview: FavoriteOverview;
  selectedListId: string;
  displayMinecraftVersion: string;
  displayModLoader: string;
  contents: FavoriteListContents | null;
  items: FavoriteModEntry[];
  loadingLists: boolean;
  loadingItems: boolean;
  exportingPackwiz: boolean;
  saving: boolean;
  error: string;
  setDisplayScope: (minecraftVersion: string, modLoader: string) => boolean;
  closeSelectedList: () => void;
  refreshOverview: () => Promise<FavoriteList[]>;
  loadLists: () => Promise<void>;
  loadItems: (listId?: string) => Promise<void>;
  selectList: (listId: string) => Promise<void>;
  createList: (name: string, minecraftVersion?: string, modLoader?: string) => Promise<FavoriteList | null>;
  renameList: (listId: string, name: string) => Promise<FavoriteList | null>;
  deleteList: (listId: string) => Promise<boolean>;
  updateListMetadata: (list: FavoriteList, patch: Partial<FavoriteList>) => Promise<FavoriteList | null>;
  updateListIcon: (
    list: FavoriteList,
    mode: FavoriteIconMode,
    value: string,
    platform?: string,
  ) => Promise<FavoriteList | null>;
  clearListIcon: (list: FavoriteList) => Promise<FavoriteList | null>;
  setListPinned: (list: FavoriteList, pinned: boolean) => Promise<FavoriteList | null>;
  reorderLists: (ids: string[]) => Promise<boolean>;
  addDrafts: (listId: string, drafts: FavoriteModDraft[]) => Promise<FavoriteMod[]>;
  remove: (mod: FavoriteMod) => Promise<boolean>;
  removeMany: (mods: FavoriteMod[]) => Promise<void>;
  exportPackwiz: (listId?: string) => Promise<ExportFavoritePackwizResult | null>;
  copySelectedMods: (
    targetListIds: string[],
    mods: FavoriteMod[],
  ) => Promise<FavoriteBulkOperationResult | null>;
  copyList: (sourceListId: string, targetListId: string) => Promise<FavoriteBulkOperationResult | null>;
  addListReference: (parentListId: string, childListId: string) => Promise<unknown>;
  previewCrossVersionCopy: (
    target: FavoriteCrossVersionCopyTarget,
  ) => Promise<FavoriteCrossVersionCopyPreview>;
  applyCrossVersionCopy: (
    target: FavoriteCrossVersionCopyTarget,
  ) => Promise<FavoriteCrossVersionCopyApplyResult>;
}

const pendingKeys = new Set<string>();
let itemsRequestId = 0;

export const useFavoritesStore = create<FavoritesState>((set, get) => ({
  lists: [],
  overview: emptyOverview(),
  selectedListId: "",
  displayMinecraftVersion: "",
  displayModLoader: "",
  contents: null,
  items: [],
  loadingLists: false,
  loadingItems: false,
  exportingPackwiz: false,
  saving: false,
  error: "",

  setDisplayScope(minecraftVersion, modLoader) {
    const nextMinecraftVersion = normalizeMinecraftVersion(minecraftVersion);
    const nextModLoader = normalizeKeyPart(modLoader);
    if (
      get().displayMinecraftVersion === nextMinecraftVersion &&
      get().displayModLoader === nextModLoader
    ) {
      return false;
    }
    set({ displayMinecraftVersion: nextMinecraftVersion, displayModLoader: nextModLoader });
    // loadLists() re-checks the selection against the new scope.
    return true;
  },

  closeSelectedList() {
    set({ selectedListId: "", contents: null, items: [] });
  },

  async refreshOverview() {
    const overview =
      (await api.ListFavoriteOverview(get().displayMinecraftVersion, get().displayModLoader)) ??
      emptyOverview();
    const lists = flattenOverview(overview);
    set({ lists, overview });
    return lists;
  },

  async loadLists() {
    set({ loadingLists: true, error: "" });
    try {
      const lists = await get().refreshOverview();
      const selected = lists.find((list) => list.id === get().selectedListId);
      if (!selected) get().closeSelectedList();
      else await get().loadItems(selected.id);
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ loadingLists: false });
    }
  },

  async loadItems(listId) {
    const targetListId = listId || get().selectedListId;
    if (!targetListId) {
      set({ items: [], contents: null });
      return;
    }
    const requestId = ++itemsRequestId;
    set({ loadingItems: true, error: "" });
    try {
      const contents = await api.ListFavoriteContentsForScope(
        targetListId,
        get().displayMinecraftVersion,
        get().displayModLoader,
      );
      if (requestId !== itemsRequestId) return;
      set({ contents, items: contents?.mods ?? [] });
    } catch (error) {
      if (requestId !== itemsRequestId) return;
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      if (requestId === itemsRequestId) set({ loadingItems: false });
    }
  },

  async selectList(listId) {
    set({ selectedListId: listId });
    await get().loadItems(listId);
  },

  async createList(name, minecraftVersion, modLoader) {
    const scopedMinecraftVersion = normalizeMinecraftVersion(
      minecraftVersion ?? get().displayMinecraftVersion,
    );
    const scopedModLoader = normalizeKeyPart(modLoader ?? get().displayModLoader);
    if (!hasFavoriteScope(scopedMinecraftVersion, scopedModLoader)) return null;
    const list = await api.CreateFavoriteList(name, scopedMinecraftVersion, scopedModLoader);
    if (!list?.id) return null;
    set({ selectedListId: list.id, contents: null, items: [] });
    await get().refreshOverview();
    return list;
  },

  async renameList(listId, name) {
    const list = await api.RenameFavoriteList(listId, name);
    if (!list?.id) return null;
    await get().refreshOverview();
    return list;
  },

  async deleteList(listId) {
    const ok = await api.DeleteFavoriteList(listId);
    if (!ok) return false;
    if (get().selectedListId === listId) get().closeSelectedList();
    await get().refreshOverview();
    return true;
  },

  async updateListMetadata(list, patch) {
    const updated = await api.UpdateFavoriteListMetadata({ ...list, ...patch } as FavoriteList);
    if (!updated?.id) return null;
    await get().refreshOverview();
    return updated;
  },

  async updateListIcon(list, mode, value, platform = "modrinth") {
    const iconValue = value.trim();
    let iconUrl = "";
    if (mode === "project" && iconValue) {
      try {
        const project = await api.LookupProjectBySlug(platform, iconValue, "", "");
        iconUrl = project?.iconUrl || "";
      } catch {
        iconUrl = "";
      }
    }
    return get().updateListMetadata(list, { iconKind: mode, iconValue, iconUrl });
  },

  async clearListIcon(list) {
    return get().updateListMetadata(list, { iconKind: "", iconValue: "", iconUrl: "" });
  },

  async setListPinned(list, pinned) {
    return get().updateListMetadata(list, { pinned });
  },

  async reorderLists(ids) {
    const ok = await api.ReorderFavoriteLists(ids);
    if (ok) await get().loadLists();
    return ok;
  },

  async addDrafts(listId, drafts) {
    if (!listId || drafts.length === 0) return [];
    const saved: FavoriteMod[] = [];
    for (const draft of drafts) {
      const key = favoriteKey({
        platform: draft.platform,
        modId: draft.modId,
        minecraftVersion: draft.minecraftVersion,
        modLoader: draft.modLoader,
      } as FavoriteMod);
      if (pendingKeys.has(key)) continue;
      pendingKeys.add(key);
      try {
        const mod = await api.AddFavoriteMod({
          listId,
          platform: draft.platform,
          modId: draft.modId,
          versionId: draft.versionId || "",
          minecraftVersion: draft.minecraftVersion || "",
          modLoader: draft.modLoader || "",
          title: draft.title || "",
          slug: draft.slug || "",
          iconUrl: draft.iconUrl || "",
          description: draft.description || "",
          categories: draft.categories || [],
        } as FavoriteMod);
        if (mod?.id) saved.push(mod);
      } finally {
        pendingKeys.delete(key);
      }
    }
    if (get().selectedListId === listId) await get().loadItems(listId);
    return saved;
  },

  async remove(mod) {
    const ok = await api.RemoveFavoriteMod(
      mod.listId,
      mod.platform,
      mod.modId,
      mod.minecraftVersion || "",
      mod.modLoader || "",
    );
    if (ok) {
      set((state) => ({ items: state.items.filter((item) => favoriteKey(item) !== favoriteKey(mod)) }));
    }
    return ok;
  },

  async removeMany(mods) {
    for (const mod of mods) await get().remove(mod);
  },

  async exportPackwiz(listId) {
    const targetListId = listId || get().selectedListId;
    if (!targetListId || get().exportingPackwiz) return null;
    set({ exportingPackwiz: true });
    try {
      return await api.ExportFavoriteListPackwizZip(
        targetListId,
        get().displayMinecraftVersion,
        get().displayModLoader,
        currentLocale(),
      );
    } finally {
      set({ exportingPackwiz: false });
    }
  },

  async copySelectedMods(targetListIds, mods) {
    if (!targetListIds.length || !mods.length) return null;
    const result = await api.AddFavoriteModsToLists({ targetListIds, mods });
    await get().loadLists();
    if (targetListIds.includes(get().selectedListId)) await get().loadItems(get().selectedListId);
    return result;
  },

  async copyList(sourceListId, targetListId) {
    const result = await api.CopyFavoriteListToList({ sourceListId, targetListId });
    if (targetListId === get().selectedListId) await get().loadItems(targetListId);
    return result;
  },

  async addListReference(parentListId, childListId) {
    const ref = await api.AddFavoriteListReference(parentListId, childListId);
    if (parentListId === get().selectedListId) await get().loadItems(parentListId);
    return ref;
  },

  async previewCrossVersionCopy(target) {
    return api.PreviewFavoriteListCrossVersionCopy({
      sourceListId: target.sourceListId,
      minecraftVersion: target.minecraftVersion,
      modLoader: target.modLoader,
      ignoreConflicts: Boolean(target.ignoreConflicts),
    });
  },

  async applyCrossVersionCopy(target) {
    const result = await api.ApplyFavoriteListCrossVersionCopy({
      sourceListId: target.sourceListId,
      minecraftVersion: target.minecraftVersion,
      modLoader: target.modLoader,
      ignoreConflicts: Boolean(target.ignoreConflicts),
    });
    if (result?.applied) await get().loadLists();
    return result;
  },
}));

export const favoriteModKey = favoriteKey;
export { hasFavoriteScope };
