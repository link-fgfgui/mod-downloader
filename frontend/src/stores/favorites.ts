import { defineStore } from "pinia";
import {
    AddFavoriteMod,
    CreateFavoriteList,
    DeleteFavoriteList,
    ListFavoriteLists,
    ListFavoriteMods,
    RemoveFavoriteMod,
    RenameFavoriteList,
} from "../../wailsjs/go/main/App";
import type { database } from "../../wailsjs/go/models";

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

const favoriteKey = (mod: Pick<database.FavoriteMod, "platform" | "modId" | "minecraftVersion" | "modLoader">) =>
    [mod.platform, mod.modId, mod.minecraftVersion || "", mod.modLoader || ""].join("|");

export const useFavoritesStore = defineStore("favorites", {
    state: () => ({
        lists: [] as database.FavoriteList[],
        selectedListId: "",
        items: [] as database.FavoriteMod[],
        isLoadingLists: false,
        isLoadingItems: false,
        pendingKeys: new Set<string>(),
    }),
    getters: {
        selectedList(state): database.FavoriteList | null {
            return state.lists.find((list) => list.id === state.selectedListId) || state.lists[0] || null;
        },
        itemKey: () => favoriteKey,
    },
    actions: {
        async loadLists() {
            this.isLoadingLists = true;
            try {
                this.lists = (await ListFavoriteLists()) || [];
                if (!this.selectedListId || !this.lists.some((list) => list.id === this.selectedListId)) {
                    this.selectedListId = this.lists[0]?.id || "";
                }
                if (this.selectedListId) {
                    await this.loadItems(this.selectedListId);
                } else {
                    this.items = [];
                }
            } finally {
                this.isLoadingLists = false;
            }
        },
        async loadItems(listId?: string) {
            const targetListId = listId || this.selectedListId;
            if (!targetListId) {
                this.items = [];
                return;
            }
            this.isLoadingItems = true;
            try {
                this.items = (await ListFavoriteMods(targetListId)) || [];
            } finally {
                this.isLoadingItems = false;
            }
        },
        async selectList(listId: string) {
            this.selectedListId = listId;
            await this.loadItems(listId);
        },
        async createList(name: string) {
            const list = await CreateFavoriteList(name);
            if (!list?.id) return null;
            this.lists = [...this.lists, list].sort((a, b) => a.sortOrder - b.sortOrder || a.name.localeCompare(b.name));
            this.selectedListId = list.id;
            this.items = [];
            return list;
        },
        async renameList(listId: string, name: string) {
            const list = await RenameFavoriteList(listId, name);
            if (!list?.id) return null;
            this.lists = this.lists.map((item) => (item.id === list.id ? list : item));
            return list;
        },
        async deleteList(listId: string) {
            const ok = await DeleteFavoriteList(listId);
            if (!ok) return false;
            this.lists = this.lists.filter((list) => list.id !== listId);
            if (this.selectedListId === listId) {
                this.selectedListId = this.lists[0]?.id || "";
                await this.loadItems(this.selectedListId);
            }
            return true;
        },
        async addDrafts(listId: string, drafts: FavoriteModDraft[]) {
            if (!listId || drafts.length === 0) return [];
            const saved: database.FavoriteMod[] = [];
            for (const draft of drafts) {
                const key = favoriteKey({
                    platform: draft.platform,
                    modId: draft.modId,
                    minecraftVersion: draft.minecraftVersion,
                    modLoader: draft.modLoader,
                } as database.FavoriteMod);
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
                    } as database.FavoriteMod);
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
        async remove(mod: database.FavoriteMod) {
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
        async removeMany(mods: database.FavoriteMod[]) {
            for (const mod of mods) {
                await this.remove(mod);
            }
        },
    },
});
