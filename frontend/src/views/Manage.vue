<template>
    <v-container class="manage-page pa-6 md-page" fluid>
        <div class="manage-header md-stagger">
            <div>
                <h1 class="text-h5 font-weight-medium">{{ $t("manage.title") }}</h1>
                <div class="text-body-2 text-medium-emphasis">
                    {{ selectedInstanceLabel }}
                </div>
            </div>
            <v-btn class="md-btn-press md-hover-scale" color="primary" prepend-icon="mdi-refresh" :loading="isRefreshing" @click="refreshMods">
                {{ $t("manage.refresh") }}
            </v-btn>
        </div>

        <v-alert v-if="!hasSelectedInstance" type="info" variant="tonal">
            {{ $t("manage.noInstance") }}
        </v-alert>

        <div v-else-if="groupedMods.length === 0" class="empty-state md-animate-fade-up">
            <v-icon class="md-animate-float" icon="mdi-package-variant" size="48"></v-icon>
            <div class="text-body-1 mt-3">{{ $t("manage.noMods") }}</div>
        </div>

        <VirtualList v-else :items="groupedMods" :item-height="72" :item-key="modRowKey"
            class="manage-list">
            <template #item="{ item: group, selected, onClick, enterStyle }">
                <v-list-item class="mb-2 border-b md-animate-fade-y md-hover-lift"
                    :class="{ 'manage-item-selected': selected }"
                    :bg-color="selected ? undefined : 'surface'"
                    rounded="xl" elevation="1" lines="two"
                    :style="enterStyle"
                    @click="onClick">
                    <template #prepend>
                        <div class="align-self-start pt-1 me-3">
                            <v-avatar color="surface-container-high" rounded="lg" size="48">
                                <v-img v-if="group.primary.iconUrl" :src="group.primary.iconUrl" :alt="displayModName(group)"></v-img>
                                <v-icon v-else icon="mdi-package-variant" color="on-surface-variant"></v-icon>
                            </v-avatar>
                        </div>
                    </template>

                    <v-list-item-title class="font-weight-medium">
                            <v-tooltip v-if="hasGroupedDetails(group)" :text="groupTooltip(group)" location="top">
                                <template #activator="{ props: tip }">
                                    <span v-bind="tip">{{ displayModName(group) }}
                                        <v-icon icon="mdi-package-variant-closed-plus" size="14" class="ms-1 text-medium-emphasis"></v-icon>
                                    </span>
                                </template>
                        </v-tooltip>
                        <span v-else>{{ displayModName(group) }}</span>
                    </v-list-item-title>
                    <v-list-item-subtitle class="manage-subtitle text-caption text-medium-emphasis">
                        <span class="manage-subtitle-scroll">
                            <span class="manage-subtitle-details">
                                {{ strongModIds(group).join(", ") }}
                                <span v-if="group.primary.version"> · {{ group.primary.version }}</span>
                                <span v-if="group.primary.fileName || group.primary.path"> · {{ group.primary.fileName || group.primary.path }}</span>
                            </span>
                            <v-tooltip v-if="modCategories(group).length" :text="categoryTooltip(group)" location="top">
                                <template #activator="{ props: tagTip }">
                                    <span v-bind="tagTip" class="manage-category-strip">
                                        <v-chip
                                            v-for="category in modCategories(group)"
                                            :key="category"
                                            class="manage-category-chip"
                                            size="x-small"
                                            variant="tonal"
                                        >
                                            {{ category }}
                                        </v-chip>
                                    </span>
                                </template>
                            </v-tooltip>
                        </span>
                    </v-list-item-subtitle>

                    <template #append>
                        <div class="manage-actions">
                            <v-btn
                                icon="mdi-playlist-plus"
                                variant="tonal"
                                color="secondary"
                                size="small"
                                :disabled="!canFavoriteGroup(group)"
                                @click.stop="openAddFavorites([group])"
                            ></v-btn>
                            <v-chip :color="group.primary.enabled ? 'success' : 'warning'" size="small" variant="tonal">
                                {{ group.primary.enabled ? $t("manage.enabled") : $t("manage.disabled") }}
                            </v-chip>
                        </div>
                    </template>
                </v-list-item>
            </template>

            <template #actions="{ selectedItems, clearSelection }">
                <v-tooltip :disabled="canFavoriteSelection(selectedItems)" :text="$t('favorites.invalidSelection')" location="top">
                    <template #activator="{ props: tip }">
                        <span v-bind="tip" class="d-inline-flex me-1">
                            <v-btn size="small" variant="tonal" color="secondary"
                                prepend-icon="mdi-playlist-plus"
                                :disabled="!canFavoriteSelection(selectedItems)"
                                @click="openAddFavorites(selectedItems)">
                                {{ $t('favorites.addToFavorites') }}
                            </v-btn>
                        </span>
                    </template>
                </v-tooltip>

                <v-btn size="small" variant="tonal" class="me-1"
                    prepend-icon="mdi-content-copy"
                    :disabled="isBatchBusy"
                    @click="copyModNames(selectedItems)">
                    {{ $t('manage.copyNames') }}
                </v-btn>

                <v-btn size="small" variant="tonal" class="me-1"
                    prepend-icon="mdi-identifier"
                    :disabled="isBatchBusy"
                    @click="copyModIds(selectedItems)">
                    {{ $t('manage.copyIds') }}
                </v-btn>

                <v-btn size="small" variant="tonal" color="primary" class="me-1"
                    :prepend-icon="primaryBatchAction(selectedItems) === 'disable' ? 'mdi-toggle-switch-off-outline' : 'mdi-toggle-switch-outline'"
                    :loading="batchOperation === primaryBatchAction(selectedItems)"
                    :disabled="isBatchBusy"
                    @click="applyBatchOperation(selectedItems, primaryBatchAction(selectedItems), clearSelection)"
                    @contextmenu.prevent="applyBatchOperation(selectedItems, 'invert', clearSelection)">
                    {{ primaryBatchAction(selectedItems) === 'disable' ? $t('manage.disableSelected') : $t('manage.enableSelected') }}
                </v-btn>

                <v-btn size="small" variant="tonal" color="warning" class="me-1"
                    prepend-icon="mdi-swap-horizontal"
                    :loading="batchOperation === 'invert'"
                    :disabled="isBatchBusy"
                    @click="applyBatchOperation(selectedItems, 'invert', clearSelection)">
                    {{ $t('manage.invertSelected') }}
                </v-btn>

                <v-btn size="small" variant="tonal" color="error" class="me-1"
                    prepend-icon="mdi-delete"
                    :disabled="isBatchBusy"
                    @click="openDeleteDialog(selectedItems, clearSelection)">
                    {{ $t('manage.deleteSelected') }}
                </v-btn>

                <v-btn size="small" variant="tonal" color="error"
                    prepend-icon="mdi-selection-off"
                    :disabled="isBatchBusy"
                    @click="clearSelection()">
                    {{ $t('download.selection.deselectAll') }}
                </v-btn>
            </template>
        </VirtualList>

        <v-dialog v-model="deleteDialog" max-width="420">
            <v-card>
                <v-card-title>{{ $t("manage.confirmDelete.title") }}</v-card-title>
                <v-card-text>
                    {{ $t("manage.confirmDelete.body", { n: pendingDeleteCount }) }}
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" :disabled="isBatchBusy" @click="deleteDialog = false">
                        {{ $t("manage.confirmDelete.cancel") }}
                    </v-btn>
                    <v-btn color="error" variant="tonal" :loading="batchOperation === 'delete'" @click="confirmDelete">
                        {{ $t("manage.confirmDelete.confirm") }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <AddToFavoriteDialog ref="addFavoriteDialog" @added="onFavoritesAdded"></AddToFavoriteDialog>

        <v-snackbar v-model="showOperationError" color="error" timeout="5000">
            {{ $t("manage.operationFailed") }}<span v-if="operationError">: {{ operationError }}</span>
        </v-snackbar>

        <v-snackbar v-model="snackbar.show" color="success" timeout="2000">
            {{ $t(snackbar.key) }}
        </v-snackbar>
    </v-container>
</template>

<script setup>
import { computed, onActivated, ref } from "vue";
import { storeToRefs } from "pinia";

import VirtualList from "../components/VirtualList.vue";
import AddToFavoriteDialog from "../components/AddToFavoriteDialog.vue";
import { useMinecraftStore } from "../stores/minecraft";
import { ApplyLocalModBatchOperation } from "../../wailsjs/go/main/App";

const minecraftStore = useMinecraftStore();
const { isRefreshing } = storeToRefs(minecraftStore);
const batchOperation = ref("");
const deleteDialog = ref(false);
const pendingDeleteGroups = ref([]);
const pendingDeleteCount = ref(0);
const operationError = ref("");
const showOperationError = ref(false);
const addFavoriteDialog = ref(null);
const snackbar = ref({ show: false, key: "" });
let pendingDeleteClearSelection = null;

const isBatchBusy = computed(() => batchOperation.value !== "");

const groupedMods = computed(() => {
    const raw = minecraftStore.mods;
    const groups = new Map();
    for (const mod of raw) {
        const key = mod.fileName || mod.path || mod.id;
        if (!groups.has(key)) {
            groups.set(key, { primary: null, strong: [], jij: [] });
        }
        const group = groups.get(key);
        group.strong.push(mod);
        addJijMods(group, mod.jijMods || []);
        if (!group.primary) {
            group.primary = mod;
        }
    }
    return [...groups.values()].filter((group) => group.primary);
});

const hasSelectedInstance = computed(() => {
    return minecraftStore.hasSelectedInstance;
});

const selectedInstanceLabel = computed(() => {
    return minecraftStore.selectedInstanceLabel;
});

const modRowKey = (group) => {
    const mod = group.primary;
    return [mod.id, mod.sha1, mod.path, mod.fileName].filter(Boolean).join("|");
};

const displayModName = (group) => {
    const mod = group.primary || {};
    return mod.onlineName || mod.name || mod.id;
};

const modCategories = (group) => {
    const categories = group.primary?.categories || [];
    const seen = new Set();
    return categories
        .map((category) => (category || "").trim())
        .filter((category) => {
            const key = category.toLowerCase();
            if (!key || seen.has(key)) {
                return false;
            }
            seen.add(key);
            return true;
        });
};

const categoryTooltip = (group) => modCategories(group).join(", ");

const canFavoriteGroup = (group) => {
    return Boolean(group?.primary?.onlinePlatform && group?.primary?.onlineProjectId);
};

const canFavoriteSelection = (groups) => {
    return groups.length > 0 && groups.every(canFavoriteGroup);
};

const favoriteDraftFromGroup = (group) => ({
    platform: group.primary.onlinePlatform,
    modId: group.primary.onlineProjectId,
    minecraftVersion: group.primary.minecraftVersion || minecraftStore.selectedVersion?.minecraftVersion || minecraftStore.selectedVersion?.MinecraftVersion || "",
    modLoader: group.primary.modLoader || minecraftStore.selectedVersion?.modLoader || minecraftStore.selectedVersion?.ModLoader || "",
    title: displayModName(group),
    slug: group.primary.onlineSlug || "",
    iconUrl: group.primary.iconUrl || "",
    description: group.primary.description || "",
    categories: modCategories(group),
});

const openAddFavorites = (groups) => {
    if (!canFavoriteSelection(groups)) return;
    addFavoriteDialog.value?.open(groups.map(favoriteDraftFromGroup));
};

const onFavoritesAdded = () => {
    snackbar.value = { show: true, key: "favorites.added" };
};

const strongModIds = (group) => {
    return group.strong.map((m) => m.id).filter(Boolean);
};

const strongModNames = (group) => {
    return group.strong.map((m) => m.name || m.id).filter(Boolean);
};

const addJijMods = (group, mods) => {
    for (const mod of mods) {
        const id = (mod.id || "").trim();
        if (!id || group.jij.some((existing) => (existing.id || "").toLowerCase() === id.toLowerCase())) {
            continue;
        }
        group.jij.push(mod);
    }
};

const hasGroupedDetails = (group) => {
    return group.strong.length > 1 || group.jij.length > 0;
};

const groupTooltip = (group) => {
    const parts = [];
    if (group.strong.length > 1) {
        parts.push(`Declared mods: ${strongModNames(group).join(", ")}`);
    }
    if (group.jij.length) {
        parts.push(`Bundled JiJ: ${group.jij.map((m) => m.name || m.id).filter(Boolean).join(", ")}`);
    }
    return parts.join("\n");
};

const refreshMods = async () => {
    await minecraftStore.refreshSelectedMods();
};

const copyModNames = async (groups) => {
    const names = groups.flatMap(strongModNames).join("\n");
    try { await navigator.clipboard.writeText(names); } catch {}
};

const copyModIds = async (groups) => {
    const ids = groups.flatMap(strongModIds).join("\n");
    try { await navigator.clipboard.writeText(ids); } catch {}
};

const selectedGroupPaths = (groups) => {
    const seen = new Set();
    const paths = [];
    for (const group of groups) {
        const path = (group.primary?.path || "").trim();
        if (!path || seen.has(path)) {
            continue;
        }
        seen.add(path);
        paths.push(path);
    }
    return paths;
};

const primaryBatchAction = (groups) => {
    return groups.some((group) => group.primary?.enabled) ? "disable" : "enable";
};

const applyBatchOperation = async (groups, action, clearSelection) => {
    const paths = selectedGroupPaths(groups);
    if (!paths.length || batchOperation.value) {
        return;
    }
    batchOperation.value = action;
    operationError.value = "";
    showOperationError.value = false;
    try {
        const version = await ApplyLocalModBatchOperation({ paths, action });
        minecraftStore.applySelectedVersion(version);
        clearSelection?.();
    } catch (error) {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
        await minecraftStore.refreshSelectedMods();
    } finally {
        batchOperation.value = "";
    }
};

const openDeleteDialog = (groups, clearSelection) => {
    pendingDeleteGroups.value = [...groups];
    pendingDeleteCount.value = selectedGroupPaths(groups).length;
    pendingDeleteClearSelection = clearSelection;
    deleteDialog.value = true;
};

const confirmDelete = async () => {
    const groups = pendingDeleteGroups.value;
    const clearSelection = pendingDeleteClearSelection;
    await applyBatchOperation(groups, "delete", clearSelection);
    if (!showOperationError.value) {
        deleteDialog.value = false;
        pendingDeleteGroups.value = [];
        pendingDeleteCount.value = 0;
        pendingDeleteClearSelection = null;
    }
};

const errorMessage = (error) => {
    if (!error) {
        return "";
    }
    if (typeof error === "string") {
        return error;
    }
    return error.message || String(error);
};

onActivated(async () => {
    await minecraftStore.start();
    await minecraftStore.refreshSelectedMods();
});
</script>

<style scoped>
.manage-page {
    max-width: 1080px;
    min-height: calc(100vh - 32px);
    display: flex;
    flex-direction: column;
}

.manage-header {
    align-items: center;
    display: flex;
    gap: 16px;
    justify-content: space-between;
    margin-bottom: 24px;
    flex: 0 0 auto;
}

.empty-state {
    align-items: center;
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 320px;
}

.manage-list {
    flex: 1 1 auto;
    max-height: calc(100vh - 176px);
}

.manage-item-selected {
    background-color: rgba(var(--v-theme-primary), 0.12) !important;
    transition: background-color var(--md-transition-fast) ease;
}

.manage-subtitle {
    overflow: visible;
}

.manage-subtitle-scroll {
    align-items: center;
    display: flex;
    gap: 6px;
    min-width: 0;
    overflow-x: auto;
    overflow-y: hidden;
    scrollbar-width: thin;
    white-space: nowrap;
}

.manage-subtitle-details {
    flex: 0 0 auto;
}

.manage-category-strip {
    display: inline-flex;
    flex: 1 1 auto;
    flex-wrap: nowrap;
    gap: 4px;
    min-width: min(72px, 100%);
    overflow: hidden;
}

.manage-category-chip {
    flex: 0 0 auto;
    max-width: 120px;
}

.manage-category-chip :deep(.v-chip__content) {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.manage-actions {
    align-items: center;
    display: flex;
    gap: 8px;
}
</style>
