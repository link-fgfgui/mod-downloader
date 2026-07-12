<template>
    <v-container class="manage-page pa-6 md-page" fluid>
        <div class="manage-header md-stagger">
            <div class="manage-header-copy">
                <h1 class="text-h5 font-weight-medium">{{ $t("manage.title") }}</h1>
                <div class="text-body-2 text-medium-emphasis">
                    {{ selectedInstanceLabel }}
                </div>
            </div>
            <v-btn class="md-btn-press md-hover-scale" color="primary" prepend-icon="mdi-refresh" :loading="isRefreshing" @click="refreshMods">
                {{ $t("manage.refresh") }}
            </v-btn>
            <v-btn
                class="md-btn-press md-hover-scale"
                color="secondary"
                variant="tonal"
                prepend-icon="mdi-broom"
                :loading="isScanningUnusedDependencies"
                :disabled="isBatchBusy || isRefreshing"
                @click="scanUnusedDependencies"
            >
                {{ $t("manage.cleanup.scan") }}
            </v-btn>
        </div>

        <v-text-field v-if="hasSelectedInstance" v-model="searchInput" class="manage-search"
            prepend-inner-icon="mdi-magnify" :label="$t('manage.search.label')" clearable
            density="comfortable" hide-details />

        <v-alert v-if="!hasSelectedInstance" type="info" variant="tonal">
            {{ $t("manage.noInstance") }}
        </v-alert>

        <div v-else-if="groupedMods.length === 0" class="empty-state md-animate-fade-up">
            <v-icon class="md-animate-float" icon="mdi-package-variant" size="48"></v-icon>
            <div class="text-body-1 mt-3">
                {{ appliedSearch ? $t("manage.search.empty") : $t("manage.noMods") }}
            </div>
        </div>

        <VirtualList v-else :items="groupedMods" :item-height="72" :item-key="modRowKey"
            class="manage-list" @scroll="onListScroll" @pointer-move="restoreListTooltips">
            <template #item="{ item: group, selected, onClick }">
                <v-list-item class="mb-2 border-b md-hover-lift"
                    :class="{ 'manage-item-selected': selected }"
                    :bg-color="selected ? undefined : 'surface'"
                    rounded="xl" elevation="1" lines="two"
                    @click="onClick">
                    <template #prepend>
                        <div class="align-self-start pt-1 me-3">
                            <v-avatar color="surface-container-high" rounded="lg" size="48">
                                <v-progress-circular
                                    v-if="group.primary.onlineMetadataLoading"
                                    color="primary"
                                    indeterminate
                                    size="24"
                                    width="2"
                                ></v-progress-circular>
                                <v-img v-else-if="group.primary.iconUrl" :src="group.primary.iconUrl" :alt="displayModName(group)"></v-img>
                                <v-icon v-else icon="mdi-package-variant" color="on-surface-variant"></v-icon>
                            </v-avatar>
                        </div>
                    </template>

                    <v-list-item-title class="font-weight-medium">
                            <v-tooltip v-if="hasGroupedDetails(group)" :disabled="listTooltipsPaused" :text="groupTooltip(group)" location="top">
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
                                <v-tooltip v-if="group.primary.version" :disabled="listTooltipsPaused" :text="versionTooltip(group)" location="top">
                                    <template #activator="{ props: versionTip }">
                                        <button
                                            v-bind="versionTip"
                                            class="manage-version-button"
                                            type="button"
                                            :disabled="!canReplaceVersion(group)"
                                            @click.stop="openVersionDialog(group)"
                                        >
                                            · {{ group.primary.version }}
                                        </button>
                                    </template>
                                </v-tooltip>
                                <span v-if="group.primary.fileName || group.primary.path"> · {{ group.primary.fileName || group.primary.path }}</span>
                            </span>
                            <v-tooltip v-if="modCategories(group).length" :disabled="listTooltipsPaused" :text="categoryTooltip(group)" location="top">
                                <template #activator="{ props: tagTip }">
                                    <span v-bind="tagTip" class="manage-category-strip">
                                        <v-chip class="manage-category-chip" size="x-small" variant="tonal">
                                            {{ modCategories(group)[0] }}
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
                            <v-btn
                                class="manage-status-toggle"
                                :color="group.primary.enabled ? 'success' : 'warning'"
                                :prepend-icon="group.primary.enabled ? 'mdi-toggle-switch' : 'mdi-toggle-switch-off-outline'"
                                size="small"
                                variant="tonal"
                                :loading="pendingTogglePath === group.primary.path"
                                :disabled="isBatchBusy"
                                :aria-label="group.primary.enabled ? $t('manage.disableSelected') : $t('manage.enableSelected')"
                                :title="group.primary.enabled ? $t('manage.disableSelected') : $t('manage.enableSelected')"
                                @click.stop="toggleGroup(group)"
                            >
                                {{ group.primary.enabled ? $t("manage.enabled") : $t("manage.disabled") }}
                            </v-btn>
                        </div>
                    </template>
                </v-list-item>
            </template>

            <template #actions="{ selectedItems, clearSelection }">
                <v-tooltip :text="canFavoriteSelection(selectedItems) ? $t('favorites.addToFavorites') : $t('favorites.invalidSelection')" location="top">
                    <template #activator="{ props: tip }">
                        <span v-bind="tip" class="d-inline-flex">
                            <v-btn :aria-label="$t('favorites.addToFavorites')" icon="mdi-playlist-plus" size="small" variant="tonal" color="secondary"
                                :disabled="!canFavoriteSelection(selectedItems)"
                                @click="openAddFavorites(selectedItems)"></v-btn>
                        </span>
                    </template>
                </v-tooltip>

                <v-tooltip v-for="action in manageSelectionActions(selectedItems, clearSelection)" :key="action.label"
                    :text="action.label" location="top">
                    <template #activator="{ props: tip }">
                        <span v-bind="tip" class="d-inline-flex">
                            <v-btn :aria-label="action.label" :icon="action.icon" size="small" variant="tonal"
                                :color="action.color" :loading="action.loading" :disabled="isBatchBusy"
                                @click="action.run"></v-btn>
                        </span>
                    </template>
                </v-tooltip>
            </template>
        </VirtualList>

        <v-dialog v-model="deleteDialog" max-width="420" @after-leave="clearClosedDeleteDialog">
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
                    <v-btn color="error" variant="tonal" :loading="isPreparingDelete || batchOperation === 'delete'" @click="confirmDelete">
                        {{ $t("manage.confirmDelete.confirm") }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="cleanupDialog" max-width="640">
            <v-card>
                <v-card-title>{{ $t("manage.cleanup.dialogTitle") }}</v-card-title>
                <v-card-text>
                    <v-list density="compact" class="cleanup-candidate-list">
                        <v-list-item
                            v-for="candidate in cleanupCandidates"
                            :key="candidate.path"
                            :title="cleanupCandidateName(candidate)"
                            :subtitle="cleanupCandidateSubtitle(candidate)"
                        >
                            <template #prepend>
                                <v-icon icon="mdi-package-variant-remove" color="warning"></v-icon>
                            </template>
                        </v-list-item>
                    </v-list>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" :disabled="cleanupOperation" @click="cleanupDialog = false">
                        {{ $t("manage.cleanup.cancel") }}
                    </v-btn>
                    <v-btn color="error" variant="tonal" :loading="cleanupOperation" @click="confirmCleanupDelete">
                        {{ $t("manage.cleanup.deleteCandidates", { n: cleanupCandidates.length }) }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="versionDialog" max-width="680" @after-leave="clearClosedVersionDialog">
            <v-card class="version-dialog">
                <v-toolbar density="compact" color="surface">
                    <v-toolbar-title>{{ versionDialogTitle }}</v-toolbar-title>
                    <v-btn icon="mdi-close" variant="text" @click="versionDialog = false"></v-btn>
                </v-toolbar>
                <v-divider></v-divider>
                <ModVersionList
                    :versions="matchingVersions"
                    :loading="isLoadingVersions"
                    action="replace"
                    :installed-version-id="selectedVersionGroup?.primary?.onlineVersionId || ''"
                    :installed-sha1="selectedVersionGroup?.primary?.sha1 || ''"
                    :busy-version-id="replacingVersionId"
                    :pinned-version-id="pinnedVersionId"
                    :pin-busy-version-id="pinningVersionId"
                    show-pin-action
                    disable-installed-action
                    @select="replaceWithVersion"
                    @pin="pinManageVersion"
                ></ModVersionList>
            </v-card>
        </v-dialog>

        <AddToFavoriteDialog ref="addFavoriteDialog" @added="onFavoritesAdded"></AddToFavoriteDialog>

        <v-snackbar v-model="showOperationError" color="error" timeout="5000">
            {{ $t("manage.operationFailed") }}<span v-if="operationError">: {{ operationError }}</span>
        </v-snackbar>

        <v-snackbar v-model="snackbar.show" :color="snackbar.color || 'success'" timeout="3000">
            {{ $t(snackbar.key, snackbar.params || {}) }}
        </v-snackbar>
    </v-container>
</template>

<script setup>
import { computed, onActivated, onDeactivated, ref, watch } from "vue";
import { storeToRefs } from "pinia";
import { useI18n } from "vue-i18n";

import VirtualList from "../components/VirtualList.vue";
import AddToFavoriteDialog from "../components/AddToFavoriteDialog.vue";
import ModVersionList from "../components/ModVersionList.vue";
import { useMinecraftStore } from "../stores/minecraft";
import { useSettingsStore } from "../stores/settings";
import { ApplyLocalModBatchOperation, GetPinnedModVersion, ListMatchingProjectVersions, PinModVersion, QueueModDownload, ScanUnusedDependencies } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const minecraftStore = useMinecraftStore();
const settingsStore = useSettingsStore();
const { t } = useI18n();
const { isRefreshing } = storeToRefs(minecraftStore);
const batchOperation = ref("");
const pendingTogglePath = ref("");
const isScanningUnusedDependencies = ref(false);
const isPreparingDelete = ref(false);
const cleanupOperation = ref(false);
const deleteDialog = ref(false);
const cleanupDialog = ref(false);
const cleanupCandidates = ref([]);
const pendingDeleteGroups = ref([]);
const pendingDeleteCount = ref(0);
const operationError = ref("");
const showOperationError = ref(false);
const addFavoriteDialog = ref(null);
const snackbar = ref({ show: false, key: "", color: "success", params: {} });
const listTooltipsPaused = ref(false);
const searchInput = ref("");
const appliedSearch = ref("");
const versionDialog = ref(false);
const selectedVersionGroup = ref(null);
const matchingVersions = ref([]);
const isLoadingVersions = ref(false);
const replacingVersionId = ref("");
const pinnedVersionId = ref("");
const pinningVersionId = ref("");
let replacementQueued = false;
let stopListeningDownloadQueue = null;
let pendingDeleteClearSelection = null;
let listScrollTimer = 0;

const restoreListTooltips = () => {
    window.clearTimeout(listScrollTimer);
    listTooltipsPaused.value = false;
};

const onListScroll = () => {
    listTooltipsPaused.value = true;
    window.clearTimeout(listScrollTimer);
    listScrollTimer = window.setTimeout(() => {
        listTooltipsPaused.value = false;
    }, 1000);
};

const isBatchBusy = computed(() => batchOperation.value !== "");

const allGroupedMods = computed(() => {
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

const groupedMods = computed(() => {
    if (!appliedSearch.value) {
        return allGroupedMods.value;
    }
    return allGroupedMods.value.filter((group) => searchableGroupText(group).includes(appliedSearch.value));
});

const searchableGroupText = (group) => {
    const values = [];
    const seen = new Set();
    const collect = (value) => {
        if (value === null || value === undefined || typeof value === "boolean") return;
        if (typeof value === "string" || typeof value === "number") {
            values.push(String(value));
            return;
        }
        if (typeof value !== "object" || seen.has(value)) return;
        seen.add(value);
        if (Array.isArray(value)) {
            value.forEach(collect);
            return;
        }
        Object.values(value).forEach(collect);
    };
    collect(group);
    return values.join("\n").toLocaleLowerCase();
};

watch(searchInput, (value, _previous, onCleanup) => {
    const timer = window.setTimeout(() => {
        appliedSearch.value = (value || "").trim().toLocaleLowerCase();
    }, 1000);
    onCleanup(() => window.clearTimeout(timer));
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

const canReplaceVersion = (group) => Boolean(
    group?.primary?.onlinePlatform && group?.primary?.onlineProjectId && hasSelectedInstance.value,
);

const versionTooltip = (group) => {
    const mod = group.primary || {};
    const onlineVersion = mod.onlineVersion || t("manage.version.onlineUnknown");
    const details = [t("manage.version.online", { version: onlineVersion }), mod.onlineFileName].filter(Boolean);
    if (canReplaceVersion(group)) details.push(t("manage.version.choose"));
    return details.join("\n");
};

const onlineProjectFromGroup = (group) => {
    const mod = group.primary;
    const platform = mod.onlinePlatform || "";
    const projectId = mod.onlineProjectId || "";
    return {
        id: `${platform.toLowerCase()}:${projectId}`,
        platform,
        projectId,
        slug: mod.onlineSlug || "",
        title: displayModName(group),
        icon: "mdi-package-variant",
        iconUrl: mod.iconUrl || "",
        description: mod.description || "",
        downloads: 0,
        categories: modCategories(group),
        updatedAt: 0,
        cachedAt: 0,
    };
};

const versionDialogTitle = computed(() =>
    selectedVersionGroup.value ? t("manage.version.title", { name: displayModName(selectedVersionGroup.value) }) : "",
);

const openVersionDialog = async (group) => {
    if (!canReplaceVersion(group)) return;
    selectedVersionGroup.value = group;
    matchingVersions.value = [];
    replacingVersionId.value = "";
    pinnedVersionId.value = "";
    pinningVersionId.value = "";
    versionDialog.value = true;
    isLoadingVersions.value = true;
    try {
        const project = onlineProjectFromGroup(group);
        const [versions, pin] = await Promise.all([
            ListMatchingProjectVersions(project, minecraftStore.selectedMinecraftVersion, minecraftStore.selectedModLoader),
            GetPinnedModVersion(project.platform, project.projectId, minecraftStore.selectedMinecraftVersion, minecraftStore.selectedModLoader),
        ]);
        matchingVersions.value = versions || [];
        pinnedVersionId.value = pin?.versionId || "";
    } catch (error) {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
    } finally {
        isLoadingVersions.value = false;
    }
};

const pinManageVersion = async (version) => {
    const group = selectedVersionGroup.value;
    if (!group || pinningVersionId.value) return;
    const project = onlineProjectFromGroup(group);
    pinningVersionId.value = version.id;
    try {
        const pin = await PinModVersion({
            platform: project.platform,
            modId: project.projectId,
            versionId: version.id,
            minecraftVersion: minecraftStore.selectedMinecraftVersion,
            modLoader: minecraftStore.selectedModLoader,
        });
        pinnedVersionId.value = pin?.versionId || "";
    } finally {
        pinningVersionId.value = "";
    }
};

const replaceWithVersion = async (version) => {
    const group = selectedVersionGroup.value;
    if (!group || replacingVersionId.value) return;
    replacingVersionId.value = version.id;
    try {
        const project = onlineProjectFromGroup(group);
        const result = await QueueModDownload({
            projectId: group.primary.onlineProjectId,
            result: project,
            minecraftVersion: minecraftStore.selectedMinecraftVersion,
            modLoader: minecraftStore.selectedModLoader,
            versionId: version.id,
        });
        if (result?.skipped) {
            showSnackbar("download.errors.generic", "error");
            return;
        }
        replacementQueued = true;
        versionDialog.value = false;
        showSnackbar("manage.version.queued", "success");
    } catch (error) {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
    } finally {
        replacingVersionId.value = "";
    }
};

const clearClosedVersionDialog = () => {
    if (versionDialog.value) return;
    selectedVersionGroup.value = null;
    matchingVersions.value = [];
    pinnedVersionId.value = "";
};

const canFavoriteGroup = (group) => {
    return Boolean(group?.primary?.onlinePlatform && group?.primary?.onlineProjectId);
};

const canFavoriteSelection = (groups) => {
    return groups.length > 0 && groups.every(canFavoriteGroup);
};

const favoriteDraftFromGroup = (group) => ({
    platform: group.primary.onlinePlatform,
    modId: group.primary.onlineProjectId,
    minecraftVersion: minecraftStore.selectedMinecraftVersion || group.primary.minecraftVersion || minecraftStore.selectedVersion?.minecraftVersion || minecraftStore.selectedVersion?.MinecraftVersion || "",
    modLoader: minecraftStore.selectedModLoader || group.primary.modLoader || minecraftStore.selectedVersion?.modLoader || minecraftStore.selectedVersion?.ModLoader || "",
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
    snackbar.value = { show: true, key: "favorites.added", color: "success", params: {} };
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
        parts.push(group.jij.map((m) => m.name || m.id).filter(Boolean).join(", "));
    }
    return parts.join("\n");
};

const refreshMods = async () => {
    await minecraftStore.refreshSelectedMods();
};

const showSnackbar = (key, color = "success", params = {}) => {
    snackbar.value = { show: true, key, color, params };
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

const manageSelectionActions = (groups, clearSelection) => {
    const primaryAction = primaryBatchAction(groups);
    return [
        { label: t("manage.copyNames"), icon: "mdi-content-copy", run: () => copyModNames(groups) },
        { label: t("manage.copyIds"), icon: "mdi-identifier", run: () => copyModIds(groups) },
        {
            label: t(primaryAction === "disable" ? "manage.disableSelected" : "manage.enableSelected"),
            icon: primaryAction === "disable" ? "mdi-toggle-switch-off-outline" : "mdi-toggle-switch-outline",
            color: "primary",
            loading: batchOperation.value === primaryAction,
            run: () => applyBatchOperation(groups, primaryAction, clearSelection),
        },
        { label: t("manage.invertSelected"), icon: "mdi-swap-horizontal", color: "warning", loading: batchOperation.value === "invert", run: () => applyBatchOperation(groups, "invert", clearSelection) },
        { label: t("manage.deleteSelected"), icon: "mdi-delete", color: "error", run: () => openDeleteDialog(groups, clearSelection) },
        { label: t("download.selection.deselectAll"), icon: "mdi-selection-off", color: "error", run: clearSelection },
    ];
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
        return true;
    } catch (error) {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
        await minecraftStore.refreshSelectedMods();
        return false;
    } finally {
        batchOperation.value = "";
    }
};

const toggleGroup = async (group) => {
    const path = (group.primary?.path || "").trim();
    if (!path || isBatchBusy.value) return;
    pendingTogglePath.value = path;
    try {
        await applyBatchOperation([group], group.primary.enabled ? "disable" : "enable");
    } finally {
        pendingTogglePath.value = "";
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
    const paths = selectedGroupPaths(groups);
    let cleanupResult = null;
    let cleanupError = null;
    isPreparingDelete.value = true;
    try {
        if (settingsStore.view?.autoScanUnusedDependencies !== false) {
            try {
                cleanupResult = await scanUnusedDependencyCandidates(paths, false);
            } catch (error) {
                cleanupError = errorMessage(error);
            }
        }
    } finally {
        isPreparingDelete.value = false;
    }
    const ok = await applyBatchOperation(groups, "delete", clearSelection);
    if (ok) {
        deleteDialog.value = false;
        if (cleanupError) {
            showSnackbar("manage.cleanup.scanFailed", "warning");
        } else if (cleanupResult) {
            showCleanupResult(cleanupResult, true);
        } else {
            showSnackbar("manage.cleanup.deleteComplete", "success");
        }
    }
};

const scanUnusedDependencyCandidates = async (excludedPaths = [], refreshFirst = true) => {
    if (refreshFirst) {
        await minecraftStore.refreshSelectedMods();
    }
    return await ScanUnusedDependencies({ excludedPaths });
};

const scanUnusedDependencies = async () => {
    if (!hasSelectedInstance.value || isScanningUnusedDependencies.value) {
        return;
    }
    isScanningUnusedDependencies.value = true;
    try {
        const result = await scanUnusedDependencyCandidates();
        showCleanupResult(result, false);
    } catch (error) {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
    } finally {
        isScanningUnusedDependencies.value = false;
    }
};

const showCleanupResult = (result, afterDelete) => {
    const candidates = result?.candidates || [];
    if (!candidates.length) {
        showSnackbar(afterDelete ? "manage.cleanup.noneAfterDelete" : "manage.cleanup.none", "success");
        return;
    }
    cleanupCandidates.value = candidates;
    cleanupDialog.value = true;
    showSnackbar("manage.cleanup.found", "warning", { n: candidates.length });
};

const cleanupCandidateName = (candidate) => {
    return candidate.onlineName || candidate.name || candidate.fileName || candidate.path;
};

const cleanupEvidenceText = (candidate) => {
    const evidence = candidate.evidence || [];
    if (evidence.includes("online-library") && evidence.includes("required-by-excluded")) {
        return t("manage.cleanup.evidence.onlineAndDeleted");
    }
    if (evidence.includes("required-by-excluded")) {
        return t("manage.cleanup.evidence.deleted");
    }
    return t("manage.cleanup.evidence.online");
};

const cleanupCandidateSubtitle = (candidate) => {
    const ids = (candidate.modIds || []).join(", ");
    const detail = [ids, candidate.path].filter(Boolean).join(" · ");
    return `${detail} · ${cleanupEvidenceText(candidate)}`;
};

const confirmCleanupDelete = async () => {
    const paths = cleanupCandidates.value.map((candidate) => candidate.path).filter(Boolean);
    if (!paths.length || cleanupOperation.value) {
        return;
    }
    cleanupOperation.value = true;
    operationError.value = "";
    showOperationError.value = false;
    try {
        const version = await ApplyLocalModBatchOperation({ paths, action: "delete" });
        minecraftStore.applySelectedVersion(version);
        cleanupDialog.value = false;
        cleanupCandidates.value = [];
        showSnackbar("manage.cleanup.deleted", "success", { n: paths.length });
    } catch (error) {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
        await minecraftStore.refreshSelectedMods();
    } finally {
        cleanupOperation.value = false;
    }
};

const clearClosedDeleteDialog = () => {
    if (deleteDialog.value) return;
    pendingDeleteGroups.value = [];
    pendingDeleteCount.value = 0;
    pendingDeleteClearSelection = null;
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
    if (!stopListeningDownloadQueue) {
        stopListeningDownloadQueue = EventsOn("download-queue-updated", (state) => {
            if (!replacementQueued || state?.active || state?.Active) return;
            replacementQueued = false;
            void minecraftStore.refreshSelectedMods();
        });
    }
    await minecraftStore.start();
    await settingsStore.load();
    await minecraftStore.ensureSelectedModsLoaded();
});

onDeactivated(() => {
    window.clearTimeout(listScrollTimer);
    listTooltipsPaused.value = false;
    stopListeningDownloadQueue?.();
    stopListeningDownloadQueue = null;
});
</script>

<style scoped>
.manage-page {
    height: calc(100vh - 32px);
    max-width: 1080px;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.manage-header {
    align-items: center;
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    justify-content: space-between;
    margin-bottom: 24px;
    flex: 0 0 auto;
}

.manage-header-copy {
    flex: 1 1 220px;
    min-width: 0;
}

.manage-search {
    flex: 0 0 auto;
    margin-bottom: 16px;
    width: 100%;
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
    min-height: 0;
}

.cleanup-candidate-list {
    max-height: 360px;
    overflow-y: auto;
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
    overflow-x: hidden;
    overflow-y: hidden;
    white-space: nowrap;
}

.manage-subtitle-details {
    flex: 1 1 auto;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
}

.manage-version-button {
    background: transparent;
    border: 0;
    color: inherit;
    cursor: pointer;
    font: inherit;
    letter-spacing: 0;
    padding: 0;
}

.manage-version-button:disabled {
    cursor: default;
}

.manage-category-strip {
    display: inline-flex;
    flex: 0 0 auto;
    flex-wrap: nowrap;
    gap: 4px;
    min-width: 0;
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

.manage-status-toggle {
    min-width: 96px;
}

@media (max-width: 599.98px) {
    .manage-page {
        padding: 16px !important;
    }

    .manage-header {
        align-items: flex-start;
    }

    .manage-header h1 {
        font-size: 1.25rem !important;
        line-height: 1.75rem;
    }

    .manage-header-copy {
        flex-basis: 100%;
    }

    .manage-actions {
        gap: 4px;
    }

    .manage-status-toggle {
        min-width: 36px;
        width: 36px;
    }

    .manage-status-toggle :deep(.v-btn__content) {
        display: none;
    }

    .manage-list :deep(.v-list-item__prepend) {
        display: none;
    }
}
</style>
