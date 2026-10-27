<template>
    <v-container class="manage-page pa-6" fluid>
        <ManageToolbar
            v-model:search-input="searchInput"
            v-model:enabled-filter="enabledFilter"
            v-model:sort-mode="sortMode"
            v-model:view-mode="viewMode"
            :title="selectedInstanceLabel"
            :is-refreshing="isRefreshing"
            :has-selected-instance="hasSelectedInstance"
            :enabled-filter-options="enabledFilterOptions"
            :sort-options="sortOptions"
            :is-scanning-unused-dependencies="isScanningUnusedDependencies"
            @refresh="refreshMods"
        ></ManageToolbar>

        <Transition name="manage-view" mode="out-in">
            <v-alert v-if="!hasSelectedInstance" key="alert" type="info" variant="tonal">
                {{ $t("manage.noInstance") }}
            </v-alert>

            <ManageTreeView v-else-if="viewMode === 'tree'" key="tree"
                :mods="minecraftStore.mods"
                :groups="allGroupedMods"
                :applied-search="appliedSearch"
                :enabled-filter="enabledFilter"
                :busy="isBatchBusy"
                :has-selected-instance="hasSelectedInstance"
                @toggle="onTreeToggle"
                @uninstall="onTreeUninstall"
                @search-version="onTreeSearchVersion"
                @error="onTreeError"
            ></ManageTreeView>

            <div v-else-if="groupedMods.length === 0" key="empty" class="empty-state">
                <v-icon icon="mdi-package-variant" size="48"></v-icon>
                <div class="text-body-1 mt-3">
                    {{ hasActiveFilter ? $t("manage.search.empty") : $t("manage.noMods") }}
                </div>
            </div>

            <ManageModList v-else key="list"
                :items="managedRows"
                :item-key="modRowKey"
                :selectable="isManagedRowSelectable"
                :busy="isBatchBusy"
                :pending-toggle-path="pendingTogglePath"
                :incompatible-expanded="incompatibleExpanded"
                :incompatible-count="incompatibleGroupedMods.length"
                :display-name="displayModName"
                :has-grouped-details="hasGroupedDetails"
                :group-tooltip="groupTooltip"
                :strong-mod-ids="strongModIds"
                :version-tooltip="versionTooltip"
                :can-replace-version="canReplaceVersion"
                :categories="modCategories"
                :category-tooltip="categoryTooltip"
                :can-favorite-group="canFavoriteGroup"
                :can-favorite-selection="canFavoriteSelection"
                :unused-candidate-for="unusedCandidateFor"
                :unused-tooltip="unusedTooltip"
                :selection-actions="manageSelectionActions"
                @toggle-incompatible="incompatibleExpanded = !incompatibleExpanded"
                @open-versions="openVersionDialog"
                @open-favorites="openAddFavorites"
                @toggle-group="toggleGroup"
            ></ManageModList>
        </Transition>

        <ManageModDialogs
            v-model:delete-dialog="deleteDialog"
            v-model:disable-impact-dialog="disableImpactDialog"
            :pending-delete-count="pendingDeleteCount"
            :busy="isBatchBusy"
            :batch-operation="batchOperation"
            :disable-impacts="disableImpacts"
            :restoring-dependency-id="restoringDependencyId"
            :restored-dependencies="restoredDependencies"
            @after-leave-delete="clearClosedDeleteDialog"
            @after-leave-disable="clearClosedDisableImpactDialog"
            @confirm-delete="confirmDelete"
            @confirm-disable="confirmDisableImpact"
            @restore="restoreDependency"
        ></ManageModDialogs>

        <ManageVersionDialog ref="manageVersionDialog" @queued="onVersionQueued" @error="onVersionError" @notify="showSnackbar"></ManageVersionDialog>

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
import { useRouter } from "vue-router";

import ManageModDialogs from "../components/ManageModDialogs.vue";
import ManageModList from "../components/ManageModList.vue";
import ManageTreeView from "../components/ManageTreeView.vue";
import ManageVersionDialog from "../components/ManageVersionDialog.vue";
import ManageToolbar from "../components/ManageToolbar.vue";
import AddToFavoriteDialog from "../components/AddToFavoriteDialog.vue";
import { matchesSearchExpression } from "../utils/searchExpression";
import { buildModSearchText } from "../utils/modSearchText";
import { downloadQueueUpdatedEvent } from "../utils/events";
import { useDownloadSearchStore } from "../stores/downloadSearch";
import { useUnusedDependencies } from "../composables/useUnusedDependencies";
import { useLocalModSelection } from "../composables/useLocalModSelection";
import { useMinecraftStore } from "../stores/minecraft";
import { useSettingsStore } from "../stores/settings";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const router = useRouter();
const minecraftStore = useMinecraftStore();
const settingsStore = useSettingsStore();
const downloadStore = useDownloadSearchStore();
const { t } = useI18n();
const { isRefreshing } = storeToRefs(minecraftStore);
const sortMode = ref("default");
const operationError = ref("");
const showOperationError = ref(false);
const addFavoriteDialog = ref(null);
const snackbar = ref({ show: false, key: "", color: "success", params: {} });
const searchInput = ref("");
const appliedSearch = ref("");
const enabledFilter = ref("all");
const incompatibleExpanded = ref(false);
const viewMode = ref("list");
let replacementQueued = false;
let stopListeningDownloadQueue = null;

const {
    batchOperation,
    isBatchBusy,
    pendingTogglePath,
    deleteDialog,
    pendingDeleteCount,
    disableImpactDialog,
    disableImpacts,
    restoringDependencyId,
    restoredDependencies,
    manageSelectionActions,
    restoreDependency,
    confirmDisableImpact,
    clearClosedDisableImpactDialog,
    toggleGroup,
    openDeleteDialog,
    confirmDelete,
    clearClosedDeleteDialog,
    applyBatchOperation,
} = useLocalModSelection({
    minecraftStore,
    t,
    onError: (error) => {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
    },
    onClearError: () => {
        operationError.value = "";
        showOperationError.value = false;
    },
    onNotify: (key, color, params) => showSnackbar(key, color, params),
    onReplacementQueued: () => {
        replacementQueued = true;
    },
});

const enabledFilterOptions = computed(() => [
    { title: t("manage.enabled"), value: "enabled" },
    { title: t("manage.disabled"), value: "disabled" },
    { title: t("manage.filter.all"), value: "all" },
]);
const hasActiveFilter = computed(() => appliedSearch.value !== "" || enabledFilter.value !== "all");

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
    return allGroupedMods.value.filter((group) => {
        const matchesSearch = matchesSearchExpression(searchableGroupText(group), appliedSearch.value);
        const matchesEnabledState = enabledFilter.value === "all"
            || group.primary.enabled === (enabledFilter.value === "enabled");
        return matchesSearch && matchesEnabledState;
    });
});

const groupLoaders = (group) => {
    const loaders = new Set();
    for (const mod of group.strong || []) {
        for (const loader of mod.loaders || []) {
            const normalized = (loader || "").trim().toLowerCase();
            if (normalized) loaders.add(normalized);
        }
    }
    return loaders;
};

const isGroupIncompatible = (group) => {
    const loaders = groupLoaders(group);
    const selectedLoader = (minecraftStore.selectedModLoader || "").trim().toLowerCase();
    return loaders.size > 0 && Boolean(selectedLoader) && !loaders.has(selectedLoader);
};

const compareGroupNames = (a, b) =>
    displayModName(a).toLowerCase().localeCompare(displayModName(b).toLowerCase());

const sortedGroupedMods = computed(() => {
    const groups = groupedMods.value;
    if (sortMode.value === "default") return groups;
    const sorted = [...groups];
    if (sortMode.value === "name") {
        sorted.sort(compareGroupNames);
    } else {
        // Unused dependencies first, then by name; the scan is a list sort
        // condition, not a separate destructive workflow.
        sorted.sort((a, b) => Number(Boolean(unusedCandidateFor(b))) - Number(Boolean(unusedCandidateFor(a))) || compareGroupNames(a, b));
    }
    return sorted;
});

const compatibleGroupedMods = computed(() => sortedGroupedMods.value.filter((group) => !isGroupIncompatible(group)));
const incompatibleGroupedMods = computed(() => groupedMods.value.filter(isGroupIncompatible));
const managedRows = computed(() => {
    if (!incompatibleGroupedMods.value.length) {
        return compatibleGroupedMods.value;
    }
    const rows = [...compatibleGroupedMods.value, { kind: "incompatible-toggle" }];
    if (incompatibleExpanded.value) {
        rows.push(...incompatibleGroupedMods.value);
    }
    return rows;
});

const onTreeToggle = async (group, action) => {
    await applyBatchOperation([group], action);
};

const onTreeUninstall = (group) => {
    openDeleteDialog([group], null);
};

const onTreeSearchVersion = (group) => {
    downloadStore.searchText = displayModName(group);
    void router.push({ name: "Download" });
};

const onTreeError = (error) => {
    operationError.value = errorMessage(error);
    showOperationError.value = true;
};

const selectedCompatibilityKey = computed(() => {
    const selected = minecraftStore.selectedVersion;
    const instance = selected?.id || selected?.name || "";
    return `${instance}\u0000${minecraftStore.selectedModLoader || ""}`;
});

watch(selectedCompatibilityKey, () => {
    incompatibleExpanded.value = false;
    unusedCandidates.value = [];
});

// Explicit field whitelist (see utils/modSearchText.ts). Never reflect over the
// whole group: `path` embeds the absolute install path (which contains the
// Minecraft version) and `dependencies` declares a `minecraft` requirement, so
// a reflective walk made every mod match a version-name query.
const searchableGroupText = (group) => buildModSearchText(group.strong || []);

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
    if (group.kind === "incompatible-toggle") return "incompatible-toggle";
    const mod = group.primary;
    return [mod.id, mod.sha1, mod.path, mod.fileName].filter(Boolean).join("|");
};

const isManagedRowSelectable = (group) => group?.kind !== "incompatible-toggle";

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

const manageVersionDialog = ref(null);

const openVersionDialog = (group) => {
    manageVersionDialog.value?.open(group);
};

const onVersionQueued = () => {
    replacementQueued = true;
};

const onVersionError = (error) => {
    operationError.value = errorMessage(error);
    showOperationError.value = true;
};

const canFavoriteGroup = (group) => {
    return Boolean(
        minecraftStore.selectedMinecraftVersion.trim() &&
        minecraftStore.selectedModLoader.trim() &&
        group?.primary?.onlinePlatform &&
        group?.primary?.onlineProjectId,
    );
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

const {
    isScanningUnusedDependencies,
    unusedCandidates,
    unusedCandidateFor,
    unusedTooltip,
    jijDetectionEnabled,
} = useUnusedDependencies({
    mods: computed(() => minecraftStore.mods),
    hasSelectedInstance,
    sortMode,
    onError: (error) => {
        operationError.value = errorMessage(error);
        showOperationError.value = true;
    },
});

const sortOptions = computed(() => [
    { title: t("manage.sort.default"), value: "default" },
    { title: t("manage.sort.name"), value: "name" },
    {
        title: t("manage.sort.unused"),
        value: "unused",
        props: jijDetectionEnabled.value ? {} : { disabled: true, subtitle: t("manage.sort.unusedDisabled") },
    },
]);

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
        stopListeningDownloadQueue = EventsOn(downloadQueueUpdatedEvent, (state) => {
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
    stopListeningDownloadQueue?.();
    stopListeningDownloadQueue = null;
});
</script>

<style scoped>
.manage-page {
    height: 100%;
    max-width: 1080px;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.manage-view-enter-active {
    transition:
        opacity var(--md-transition-normal) var(--md-ease-out),
        transform var(--md-transition-normal) var(--md-ease-out);
}

.manage-view-leave-active {
    transition: opacity var(--md-transition-fast) var(--md-ease-in);
}

.manage-view-enter-from {
    opacity: 0;
    transform: translateY(6px);
}

.manage-view-leave-to {
    opacity: 0;
}

.empty-state {
    align-items: center;
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 320px;
}

@media (max-width: 599.98px) {
    .manage-page {
        padding: 16px !important;
    }
}
</style>
