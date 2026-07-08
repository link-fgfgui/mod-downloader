import { defineStore } from "pinia";
import {
    AddFavoriteListReference,
    AddFavoriteMod,
    AddFavoriteModsToLists,
    ApplyFavoriteListMigration,
    CopyFavoriteListToList,
    CreateFavoriteGroup,
    CreateFavoriteList,
    DeleteFavoriteGroup,
    DeleteFavoriteList,
    ExportFavoriteListPackwizZip,
    ListFavoriteContents,
    ListFavoriteGroups,
    ListFavoriteLists,
    LookupProjectBySlug,
    PreviewFavoriteListMigration,
    RemoveFavoriteMod,
    RenameFavoriteGroup,
    RenameFavoriteList,
    ReorderFavoriteGroups,
    ReorderFavoriteLists,
    UpdateFavoriteListMetadata,
} from "../../wailsjs/go/main/App";
import type { appcore, database, main } from "../../wailsjs/go/models";

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

export type FavoriteMigrationTarget = {
    sourceListId: string;
    targetListId: string;
    minecraftVersion: string;
    modLoader: string;
    ignoreConflicts?: boolean;
};

const normalizeKeyPart = (value?: string) => (value || "").trim().toLowerCase();
const favoriteKey = (mod: Pick<database.FavoriteMod, "platform" | "modId" | "minecraftVersion" | "modLoader">) =>
    [normalizeKeyPart(mod.platform), normalizeKeyPart(mod.modId), mod.minecraftVersion || "", normalizeKeyPart(mod.modLoader)].join("|");

const listOrder = (a: database.FavoriteList, b: database.FavoriteList) =>
    Number(Boolean(b.pinned)) - Number(Boolean(a.pinned)) || a.sortOrder - b.sortOrder || a.name.localeCompare(b.name);

const groupOrder = (a: database.FavoriteGroup, b: database.FavoriteGroup) =>
    a.sortOrder - b.sortOrder || a.name.localeCompare(b.name);

export const useFavoritesStore = defineStore("favorites", {
    state: () => ({
        groups: [] as database.FavoriteGroup[],
        lists: [] as database.FavoriteList[],
        selectedListId: "",
        contents: null as database.FavoriteListContents | null,
        items: [] as database.FavoriteModEntry[],
        isLoadingLists: false,
        isLoadingItems: false,
        isExportingPackwiz: false,
        isSaving: false,
        pendingKeys: new Set<string>(),
    }),
    getters: {
        selectedList(state): database.FavoriteList | null {
            return state.lists.find((list) => list.id === state.selectedListId) || state.lists[0] || null;
        },
        itemKey: () => favoriteKey,
        pinnedLists(state): database.FavoriteList[] {
            return state.lists.filter((list) => list.pinned).sort(listOrder);
        },
        ungroupedLists(state): database.FavoriteList[] {
            return state.lists.filter((list) => !list.pinned && !list.groupId).sort(listOrder);
        },
        groupedLists(state): Record<string, database.FavoriteList[]> {
            return state.lists.reduce<Record<string, database.FavoriteList[]>>((acc, list) => {
                if (!list.groupId || list.pinned) return acc;
                acc[list.groupId] = acc[list.groupId] || [];
                acc[list.groupId].push(list);
                acc[list.groupId].sort(listOrder);
                return acc;
            }, {});
        },
        sortedGroups(state): database.FavoriteGroup[] {
            return [...state.groups].sort(groupOrder);
        },
    },
    actions: {
        async loadLists() {
            this.isLoadingLists = true;
            try {
                const [groups, lists] = await Promise.all([ListFavoriteGroups(), ListFavoriteLists()]);
                this.groups = (groups || []).sort(groupOrder);
                this.lists = (lists || []).sort(listOrder);
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
                this.contents = null;
                return;
            }
            this.isLoadingItems = true;
            try {
                this.contents = (await ListFavoriteContents(targetListId)) || null;
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
            const list = await CreateFavoriteList(name);
            if (!list?.id) return null;
            this.lists = [...this.lists, list].sort(listOrder);
            this.selectedListId = list.id;
            await this.loadLists();
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
                this.selectedListId = this.lists[0]?.id || "";
                await this.loadItems(this.selectedListId);
            }
            return true;
        },
        async createGroup(name: string) {
            const group = await CreateFavoriteGroup(name);
            if (!group?.id) return null;
            this.groups = [...this.groups, group].sort(groupOrder);
            return group;
        },
        async renameGroup(groupId: string, name: string) {
            const group = await RenameFavoriteGroup(groupId, name);
            if (!group?.id) return null;
            this.groups = this.groups.map((item) => (item.id === group.id ? group : item)).sort(groupOrder);
            return group;
        },
        async deleteGroup(groupId: string) {
            const ok = await DeleteFavoriteGroup(groupId);
            if (!ok) return false;
            await this.loadLists();
            return true;
        },
        async updateListMetadata(list: database.FavoriteList, patch: Partial<database.FavoriteList>) {
            const updated = await UpdateFavoriteListMetadata({
                ...list,
                ...patch,
            } as database.FavoriteList);
            if (!updated?.id) return null;
            this.lists = this.lists.map((item) => (item.id === updated.id ? updated : item)).sort(listOrder);
            return updated;
        },
        async updateListIcon(list: database.FavoriteList, mode: FavoriteIconMode, value: string, platform = "modrinth") {
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
        async clearListIcon(list: database.FavoriteList) {
            return this.updateListMetadata(list, {
                iconKind: "",
                iconValue: "",
                iconUrl: "",
            });
        },
        async setListGroup(list: database.FavoriteList, groupId: string) {
            return this.updateListMetadata(list, { groupId });
        },
        async setListPinned(list: database.FavoriteList, pinned: boolean) {
            return this.updateListMetadata(list, { pinned });
        },
        async reorderLists(ids: string[]) {
            const ok = await ReorderFavoriteLists(ids);
            if (ok) {
                await this.loadLists();
            }
            return ok;
        },
        async reorderGroups(ids: string[]) {
            const ok = await ReorderFavoriteGroups(ids);
            if (ok) {
                await this.loadLists();
            }
            return ok;
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
        async exportPackwiz(listId?: string): Promise<main.ExportFavoritePackwizResult | null> {
            const targetListId = listId || this.selectedListId;
            if (!targetListId || this.isExportingPackwiz) return null;
            this.isExportingPackwiz = true;
            try {
                return await ExportFavoriteListPackwizZip(targetListId);
            } finally {
                this.isExportingPackwiz = false;
            }
        },
        async copySelectedMods(targetListIds: string[], mods: database.FavoriteMod[]) {
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
        async previewMigration(target: FavoriteMigrationTarget) {
            return PreviewFavoriteListMigration({
                sourceListId: target.sourceListId,
                targetListId: target.targetListId,
                minecraftVersion: target.minecraftVersion,
                modLoader: target.modLoader,
                ignoreConflicts: Boolean(target.ignoreConflicts),
            } as appcore.FavoriteMigrationRequest);
        },
        async applyMigration(target: FavoriteMigrationTarget) {
            const result = await ApplyFavoriteListMigration({
                sourceListId: target.sourceListId,
                targetListId: target.targetListId,
                minecraftVersion: target.minecraftVersion,
                modLoader: target.modLoader,
                ignoreConflicts: Boolean(target.ignoreConflicts),
            } as appcore.FavoriteMigrationRequest);
            if (target.targetListId === this.selectedListId) {
                await this.loadItems(target.targetListId);
            }
            return result;
        },
    },
});
