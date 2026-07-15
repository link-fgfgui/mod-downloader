import { defineStore } from "pinia";
import {
    AddFavoriteListReference,
    AddFavoriteMod,
    AddFavoriteModsToLists,
    ApplyFavoriteListCrossVersionCopy,
    CopyFavoriteListToList,
    CreateFavoriteList,
    DeleteFavoriteList,
    ExportFavoriteListPackwizZip,
    ListFavoriteContents,
    ListFavoriteLists,
    LookupProjectBySlug,
    PreviewFavoriteListCrossVersionCopy,
    RemoveFavoriteMod,
    RenameFavoriteList,
    ReorderFavoriteLists,
    UpdateFavoriteListMetadata,
} from "../../wailsjs/go/main/App";
import type { appcore, main, storage } from "../../wailsjs/go/models";
import { currentLocale } from "../plugins/i18n";

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

const normalizeKeyPart = (value?: string) => (value || "").trim().toLowerCase();
const normalizeMinecraftVersion = (value?: string) => (value || "").trim();
const favoriteKey = (mod: Pick<storage.FavoriteMod, "platform" | "modId" | "minecraftVersion" | "modLoader">) =>
    [normalizeKeyPart(mod.platform), normalizeKeyPart(mod.modId), normalizeMinecraftVersion(mod.minecraftVersion), normalizeKeyPart(mod.modLoader)].join("|");

const hasFavoriteScope = (minecraftVersion: string, modLoader: string) => Boolean(minecraftVersion && modLoader);

const favoriteMatchesScope = (
    mod: Pick<storage.FavoriteMod, "minecraftVersion" | "modLoader">,
    minecraftVersion: string,
    modLoader: string,
) =>
    hasFavoriteScope(minecraftVersion, modLoader) &&
    normalizeMinecraftVersion(mod.minecraftVersion) === minecraftVersion &&
    normalizeKeyPart(mod.modLoader) === modLoader;

const favoriteContentsForScope = (
    contents: storage.FavoriteListContents | null,
    minecraftVersion: string,
    modLoader: string,
): storage.FavoriteListContents | null => {
    if (!contents) return null;
    return {
        ...contents,
        mods: (contents.mods || []).filter((mod) => favoriteMatchesScope(mod, minecraftVersion, modLoader)),
    } as storage.FavoriteListContents;
};

const listOrder = (a: storage.FavoriteList, b: storage.FavoriteList) =>
    Number(Boolean(b.pinned)) - Number(Boolean(a.pinned)) || a.sortOrder - b.sortOrder || a.name.localeCompare(b.name);

const listMatchesScope = (list: storage.FavoriteList, minecraftVersion: string, modLoader: string) => {
    const listVersion = normalizeMinecraftVersion(list.minecraftVersion);
    const listLoader = normalizeKeyPart(list.modLoader);
    if (!listVersion && !listLoader) return true;
    return hasFavoriteScope(minecraftVersion, modLoader) && listVersion === minecraftVersion && listLoader === modLoader;
};

export const useFavoritesStore = defineStore("favorites", {
    state: () => ({
        lists: [] as storage.FavoriteList[],
        selectedListId: "",
        displayMinecraftVersion: "",
        displayModLoader: "",
        contents: null as storage.FavoriteListContents | null,
        items: [] as storage.FavoriteModEntry[],
        isLoadingLists: false,
        isLoadingItems: false,
        isExportingPackwiz: false,
        isSaving: false,
        pendingKeys: new Set<string>(),
    }),
    getters: {
        selectedList(state): storage.FavoriteList | null {
            return state.lists.find((list) => list.id === state.selectedListId && listMatchesScope(list, state.displayMinecraftVersion, state.displayModLoader)) || null;
        },
        itemKey: () => favoriteKey,
        pinnedLists(state): storage.FavoriteList[] {
            return state.lists.filter((list) => list.pinned && listMatchesScope(list, state.displayMinecraftVersion, state.displayModLoader)).sort(listOrder);
        },
        ungroupedLists(state): storage.FavoriteList[] {
            return state.lists.filter((list) => !list.pinned && listMatchesScope(list, state.displayMinecraftVersion, state.displayModLoader)).sort(listOrder);
        },
    },
    actions: {
        setDisplayScope(minecraftVersion: string, modLoader: string) {
            const nextMinecraftVersion = normalizeMinecraftVersion(minecraftVersion);
            const nextModLoader = normalizeKeyPart(modLoader);
            if (this.displayMinecraftVersion === nextMinecraftVersion && this.displayModLoader === nextModLoader) {
                return false;
            }
            this.displayMinecraftVersion = nextMinecraftVersion;
            this.displayModLoader = nextModLoader;
            if (!this.selectedList) this.closeSelectedList();
            return true;
        },
        closeSelectedList() {
            this.selectedListId = "";
            this.contents = null;
            this.items = [];
        },
        async loadLists() {
            this.isLoadingLists = true;
            try {
                this.lists = (await ListFavoriteLists() || []).sort(listOrder);
                if (this.selectedListId && !this.selectedList) {
                    this.closeSelectedList();
                }
                if (this.selectedListId) {
                    await this.loadItems(this.selectedListId);
                } else {
                    this.items = [];
                    this.contents = null;
                }
            } finally {
                this.isLoadingLists = false;
            }
        },
        async loadItems(listId?: string) {
            const targetListId = listId || this.selectedListId;
            if (!targetListId) {
                this.items = [];
                this.contents = null;
                return;
            }
            this.isLoadingItems = true;
            try {
                this.contents = favoriteContentsForScope(
                    (await ListFavoriteContents(targetListId)) || null,
                    this.displayMinecraftVersion,
                    this.displayModLoader,
                );
                this.items = this.contents?.mods || [];
            } finally {
                this.isLoadingItems = false;
            }
        },
        async selectList(listId: string) {
            this.selectedListId = listId;
            await this.loadItems(listId);
        },
        async createList(name: string) {
            const list = await CreateFavoriteList(name, this.displayMinecraftVersion, this.displayModLoader);
            if (!list?.id) return null;
            this.lists = [...this.lists, list].sort(listOrder);
            this.selectedListId = list.id;
            this.contents = null;
            this.items = [];
            return list;
        },
        async renameList(listId: string, name: string) {
            const list = await RenameFavoriteList(listId, name);
            if (!list?.id) return null;
            this.lists = this.lists.map((item) => (item.id === list.id ? list : item)).sort(listOrder);
            return list;
        },
        async deleteList(listId: string) {
            const ok = await DeleteFavoriteList(listId);
            if (!ok) return false;
            this.lists = this.lists.filter((list) => list.id !== listId);
            if (this.selectedListId === listId) {
                this.closeSelectedList();
            }
            return true;
        },
        async updateListMetadata(list: storage.FavoriteList, patch: Partial<storage.FavoriteList>) {
            const updated = await UpdateFavoriteListMetadata({
                ...list,
                ...patch,
            } as storage.FavoriteList);
            if (!updated?.id) return null;
            this.lists = this.lists.map((item) => (item.id === updated.id ? updated : item)).sort(listOrder);
            return updated;
        },
        async updateListIcon(list: storage.FavoriteList, mode: FavoriteIconMode, value: string, platform = "modrinth") {
            const iconValue = value.trim();
            let iconUrl = "";
            if (mode === "project" && iconValue) {
                const project = await LookupProjectBySlug(platform, iconValue, "", "");
                iconUrl = project?.iconUrl || "";
            }
            return this.updateListMetadata(list, {
                iconKind: mode,
                iconValue,
                iconUrl,
            });
        },
        async clearListIcon(list: storage.FavoriteList) {
            return this.updateListMetadata(list, {
                iconKind: "",
                iconValue: "",
                iconUrl: "",
            });
        },
        async setListPinned(list: storage.FavoriteList, pinned: boolean) {
            return this.updateListMetadata(list, { pinned });
        },
        async reorderLists(ids: string[]) {
            const ok = await ReorderFavoriteLists(ids);
            if (ok) {
                await this.loadLists();
            }
            return ok;
        },
        async addDrafts(listId: string, drafts: FavoriteModDraft[]) {
            if (!listId || drafts.length === 0) return [];
            const saved: storage.FavoriteMod[] = [];
            for (const draft of drafts) {
                const key = favoriteKey({
                    platform: draft.platform,
                    modId: draft.modId,
                    minecraftVersion: draft.minecraftVersion,
                    modLoader: draft.modLoader,
                } as storage.FavoriteMod);
                if (this.pendingKeys.has(key)) continue;
                this.pendingKeys.add(key);
                try {
                    const mod = await AddFavoriteMod({
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
                    } as storage.FavoriteMod);
                    if (mod?.id) {
                        saved.push(mod);
                    }
                } finally {
                    this.pendingKeys.delete(key);
                }
            }
            if (this.selectedListId === listId) {
                await this.loadItems(listId);
            }
            return saved;
        },
        async remove(mod: storage.FavoriteMod) {
            const ok = await RemoveFavoriteMod(
                mod.listId,
                mod.platform,
                mod.modId,
                mod.minecraftVersion || "",
                mod.modLoader || "",
            );
            if (ok) {
                this.items = this.items.filter((item) => favoriteKey(item) !== favoriteKey(mod));
            }
            return ok;
        },
        async removeMany(mods: storage.FavoriteMod[]) {
            for (const mod of mods) {
                await this.remove(mod);
            }
        },
        async exportPackwiz(listId?: string): Promise<main.ExportFavoritePackwizResult | null> {
            const targetListId = listId || this.selectedListId;
            if (!targetListId || this.isExportingPackwiz) return null;
            this.isExportingPackwiz = true;
            try {
                return await ExportFavoriteListPackwizZip(
                    targetListId,
                    this.displayMinecraftVersion,
                    this.displayModLoader,
                    currentLocale(),
                );
            } finally {
                this.isExportingPackwiz = false;
            }
        },
        async copySelectedMods(targetListIds: string[], mods: storage.FavoriteMod[]) {
            if (!targetListIds.length || !mods.length) return null;
            const result = await AddFavoriteModsToLists({
                targetListIds,
                mods,
            } as appcore.FavoriteBulkAddRequest);
            await this.loadLists();
            if (targetListIds.includes(this.selectedListId)) {
                await this.loadItems(this.selectedListId);
            }
            return result;
        },
        async copyList(sourceListId: string, targetListId: string) {
            const result = await CopyFavoriteListToList({
                sourceListId,
                targetListId,
            } as appcore.FavoriteListCopyRequest);
            if (targetListId === this.selectedListId) {
                await this.loadItems(targetListId);
            }
            return result;
        },
        async addListReference(parentListId: string, childListId: string) {
            const ref = await AddFavoriteListReference(parentListId, childListId);
            if (parentListId === this.selectedListId) {
                await this.loadItems(parentListId);
            }
            return ref;
        },
        async previewCrossVersionCopy(target: FavoriteCrossVersionCopyTarget) {
            return PreviewFavoriteListCrossVersionCopy({
                sourceListId: target.sourceListId,
                minecraftVersion: target.minecraftVersion,
                modLoader: target.modLoader,
                ignoreConflicts: Boolean(target.ignoreConflicts),
            } as appcore.FavoriteCrossVersionCopyRequest);
        },
        async applyCrossVersionCopy(target: FavoriteCrossVersionCopyTarget) {
            const result = await ApplyFavoriteListCrossVersionCopy({
                sourceListId: target.sourceListId,
                minecraftVersion: target.minecraftVersion,
                modLoader: target.modLoader,
                ignoreConflicts: Boolean(target.ignoreConflicts),
            } as appcore.FavoriteCrossVersionCopyRequest);
            if (result?.applied) {
                await this.loadLists();
            }
            return result;
        },
    },
});
