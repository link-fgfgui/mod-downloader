<template>
    <v-container class="download-page pb-6 px-6 pt-0" fluid>
        <div class="search-controls">
            <v-row class="align-center" density="comfortable">
                <v-col cols="12" md="10">
                    <v-text-field
                        v-model="searchText"
                        placeholder="emi"
                        :label="$t('search.label')"
                        prepend-inner-icon="mdi-magnify"
                        variant="outlined"
                        density="comfortable"
                        hide-details
                        clearable
                        @keyup.enter.prevent="searchMods"
                    ></v-text-field>
                </v-col>
                <v-col cols="12" md="2">
                    <v-btn
                        class="md-btn-press"
                        block
                        color="primary"
                        height="48"
                        prepend-icon="mdi-magnify"
                        :loading="isSearching"
                        @click="searchMods"
                    >
                        {{ $t("search.action") }}
                    </v-btn>
                </v-col>
            </v-row>
        </div>

        <Transition name="md-fade-slide" mode="out-in">
            <div v-if="isEmptySearch" key="empty" class="search-empty">
                <v-icon
                    :icon="hasSearchQuery ? 'mdi-file-search-outline' : 'mdi-compass-outline'"
                    color="primary"
                    size="56"
                ></v-icon>
                <div class="text-h6 mt-4">
                    {{ hasSearchQuery ? $t("download.searchEmpty.title") : $t("download.searchIdle.title") }}
                </div>
                <div class="text-body-2 text-medium-emphasis mt-2">
                    {{ hasSearchQuery ? $t("download.searchEmpty.hint") : $t("download.searchIdle.hint") }}
                </div>
            </div>

            <SearchResultList
                v-else
                key="results"
                class="search-result-list"
                :results="searchResults"
                :loading-more="isLoadingMore"
                :has-more="hasMoreResults"
                :states="downloadStates"
                :downloading-keys="downloadingKeys"
                @install="installMod"
                @load-more="loadMoreSearchResults"
                @show-versions="openVersionsOverlay"
                @batch-download="batchDownload"
                @batch-unpin="batchUnpin"
                @add-favorite="openAddFavorites"
            ></SearchResultList>
        </Transition>

        <v-overlay
            v-model="showDirOverlay"
            transition="scale-fade"
            contained
            persistent
            class="align-center justify-center"
            scrim="surface"
        >
            <v-card
                class="pa-6 text-center"
                max-width="420"
                variant="elevated"
            >
                <v-icon
                    class="mb-3"
                    color="warning"
                    icon="mdi-folder-alert"
                    size="48"
                ></v-icon>
                <v-card-title class="pa-0 text-h6">{{
                    $t("download.selectDirTitle")
                }}</v-card-title>
                <v-card-text class="px-0 pb-0 pt-3">
                    {{ $t("download.selectDirDesc") }}
                </v-card-text>
            </v-card>
        </v-overlay>

        <v-overlay
            v-model="showVersionsOverlay"
            transition="scale-fade"
            location="center"
            scrim="surface"
        >
            <v-card
                class="version-overlay"
                width="680"
                max-width="calc(100vw - 32px)"
                max-height="calc(100vh - 64px)"
            >
                <v-toolbar density="compact" color="surface">
                    <v-toolbar-title>{{
                        selectedMod?.title || $t("download.versionsTitle")
                    }}</v-toolbar-title>
                    <v-btn
                        icon="mdi-close"
                        variant="text"
                        @click="showVersionsOverlay = false"
                    ></v-btn>
                </v-toolbar>

                <v-divider></v-divider>

                <ModVersionList
                    :versions="matchingVersions"
                    :loading="isLoadingVersions"
                    action="pin"
                    :pinned-version-id="downloadStore.pinnedVersion?.versionId || ''"
                    :busy-version-id="pinningVersionID"
                    @select="pinVersion"
                ></ModVersionList>
            </v-card>
        </v-overlay>

        <v-snackbar
            v-model="snackbar.show"
            :color="snackbar.color"
            location="bottom"
            timeout="3000"
        >
            {{ $t(snackbar.key, snackbar.params) }}
        </v-snackbar>

        <v-dialog
            v-model="confirmDialog.show"
            max-width="420"
            @after-leave="clearClosedConfirmDialog"
        >
            <v-card>
                <v-card-title class="text-h6">
                    {{ $t(confirmDialogTitleKey) }}
                </v-card-title>
                <v-card-text>
                    <div v-if="confirmDialog.status === 'conflict' && confirmDialog.conflictFileName" class="font-weight-medium mb-2">
                        {{ $t("download.confirmReplace.conflictFile", { file: confirmDialog.conflictFileName }) }}
                    </div>
                    <div>{{ $t(confirmDialogBodyKey) }}</div>
                    <div v-if="confirmDialog.status === 'conflict'" class="text-caption text-medium-emphasis mt-3">
                        {{ $t("download.confirmReplace.conflictHint") }}
                    </div>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="closeConfirmDialog">{{
                        $t("download.confirmReplace.cancel")
                    }}</v-btn>
                    <v-btn
                        color="warning"
                        variant="flat"
                        @click="confirmInstall"
                        >{{ $t("download.confirmReplace.confirm") }}</v-btn
                    >
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog
            :model-value="preflightDialog.show"
            max-width="560"
            @update:model-value="onPreflightDialogModel"
            @after-leave="clearClosedPreflightDialog"
        >
            <v-card>
                <v-card-title class="text-h6">
                    {{ $t("download.preflight.title") }}
                </v-card-title>
                <v-card-text>
                    <div class="mb-3">
                        {{ $t("download.preflight.summary", { missing: preflightDialog.missing, conflicts: preflightDialog.conflicts }) }}
                    </div>
                    <v-list density="compact" class="download-preflight-list">
                        <v-list-item
                            v-for="(entry, index) in preflightDialog.entries"
                            :key="`${entry.projectId}-${index}`"
                            :title="entry.title || entry.projectId"
                            :subtitle="entry.localName
                                ? $t('download.preflight.localVersion', { name: entry.localName, version: entry.localVersion || $t('download.preflight.unknownVersion') })
                                : entry.projectId"
                        >
                            <template #prepend>
                                <v-icon :color="preflightStatusMeta(entry).color" :icon="preflightStatusMeta(entry).icon"></v-icon>
                            </template>
                            <template #append>
                                <v-chip :color="preflightStatusMeta(entry).color" size="x-small" variant="tonal">
                                    {{ $t(preflightStatusMeta(entry).label) }}
                                </v-chip>
                            </template>
                        </v-list-item>
                    </v-list>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="closePreflightDialog">
                        {{ $t("download.preflight.cancel") }}
                    </v-btn>
                    <v-btn color="primary" variant="flat" @click="confirmPreflightInstall">
                        {{ $t("download.preflight.confirm") }}
                    </v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog
            :model-value="batchConfirmDialog.show"
            max-width="560"
            @update:model-value="onBatchDialogModel"
        >
            <v-card>
                <v-card-title class="text-h6">
                    {{ $t("download.confirmReplace.batchIncompatibleTitle") }}
                </v-card-title>
                <v-card-text>
                    <div class="mb-3">{{ $t("download.confirmReplace.batchIncompatibleBody") }}</div>
                    <div class="batch-conflict-list">
                        <div v-for="entry in batchConfirmDialog.conflicts" :key="entry.key" class="batch-conflict-entry">
                            <div class="font-weight-bold">{{ entry.title || entry.key }}</div>
                            <div
                                v-for="conflict in entry.conflicts"
                                :key="`${entry.key}-${conflict.incompatibleProjectKey}`"
                                class="batch-conflict-detail"
                            >
                                <div>{{ conflict.incompatibleTitle || conflict.incompatibleProjectKey }}</div>
                                <div
                                    v-for="path in conflict.paths"
                                    :key="path.path"
                                    class="text-medium-emphasis text-caption"
                                >
                                    {{ path.fileName || path.path }}
                                </div>
                            </div>
                        </div>
                    </div>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="cancelBatchInstall">{{
                        $t("download.confirmReplace.cancel")
                    }}</v-btn>
                    <v-btn variant="tonal" color="warning" @click="skipConflictedBatchInstall">{{
                        $t("download.confirmReplace.skipConflicted")
                    }}</v-btn>
                    <v-btn color="warning" variant="flat" @click="confirmBatchInstall">{{
                        $t("download.confirmReplace.confirm")
                    }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <AddToFavoriteDialog ref="addFavoriteDialog" @added="onFavoritesAdded"></AddToFavoriteDialog>
    </v-container>
</template>

<script setup lang="ts">
import { computed, onActivated, onDeactivated, ref, watch } from "vue";
import { storeToRefs } from "pinia";

import SearchResultList from "../components/SearchResultList.vue";
import AddToFavoriteDialog from "../components/AddToFavoriteDialog.vue";
import ModVersionList from "../components/ModVersionList.vue";
import { ValidateMinecraftDir } from "../../wailsjs/go/main/App";
import { useDownloadSearchStore } from "../stores/downloadSearch";
import { useMinecraftStore } from "../stores/minecraft";
import type { FavoriteModDraft } from "../stores/favorites";
import type { structs, models } from "../../wailsjs/go/models";

const downloadStore = useDownloadSearchStore();
const minecraftStore = useMinecraftStore();
const addFavoriteDialog = ref<InstanceType<typeof AddToFavoriteDialog> | null>(null);

const {
    searchText,
    showDirOverlay,
    isSearching,
    isLoadingMore,
    hasMoreResults,
    showVersionsOverlay,
    isLoadingVersions,
    searchResults,
    selectedMod,
    matchingVersions,
    pinningVersionID,
    downloadStates,
    downloadingKeys,
    snackbar,
    confirmDialog,
    preflightDialog,
    batchConfirmDialog,
} = storeToRefs(downloadStore);

const isEmptySearch = computed(
    () => !searchResults.value.length && !isSearching.value,
);
const hasSearchQuery = computed(() => searchText.value.trim().length > 0);

const searchMods = () => downloadStore.runSearch();
const loadMoreSearchResults = () => downloadStore.loadMoreSearchResults();
const installMod = (payload: {
    result?: models.ModProject;
    key?: string;
    status?: string;
    conflictFileName?: string;
    confirm?: boolean;
}) => downloadStore.installMod(payload);
const confirmInstall = () => downloadStore.confirmInstall();
const confirmBatchInstall = () => downloadStore.confirmBatchInstall();
const skipConflictedBatchInstall = () => downloadStore.skipConflictedBatchInstall();
const cancelBatchInstall = () => downloadStore.cancelBatchInstall();
const closeConfirmDialog = () => downloadStore.closeConfirmDialog();
const clearClosedConfirmDialog = () => downloadStore.clearClosedConfirmDialog();
const confirmPreflightInstall = () => downloadStore.confirmPreflightInstall();
const closePreflightDialog = () => downloadStore.closePreflightDialog();
const clearClosedPreflightDialog = () => downloadStore.clearClosedPreflightDialog();

const preflightStatusMeta = (entry: structs.DependencyPreflightEntry) => {
    switch (entry.status) {
        case "missing":
            return { icon: "mdi-download", color: "info", label: "download.preflight.status.missing" };
        case "conflict":
            return { icon: "mdi-alert-octagon-outline", color: "error", label: "download.preflight.status.conflict" };
        default:
            return { icon: "mdi-check-circle-outline", color: "success", label: "download.preflight.status.fulfilled" };
    }
};
const onPreflightDialogModel = (value: boolean) => {
    if (!value && preflightDialog.value.show) {
        closePreflightDialog();
    }
};
const openVersionsOverlay = (result: models.ModProject) =>
    downloadStore.openVersionsOverlay(result);
const pinVersion = (version: models.ModVersion) =>
    downloadStore.pinVersion(version);

const batchDownload = (results: models.ModProject[]) => {
    void downloadStore.batchInstall(results);
};

const confirmDialogTitleKey = computed(() => {
    switch (confirmDialog.value.status) {
        case "update":
            return "download.confirmReplace.updateTitle";
        case "incompatible":
            return "download.confirmReplace.incompatibleTitle";
        default:
            return "download.confirmReplace.conflictTitle";
    }
});

const confirmDialogBodyKey = computed(() => {
    switch (confirmDialog.value.status) {
        case "update":
            return "download.confirmReplace.updateBody";
        case "incompatible":
            return "download.confirmReplace.incompatibleBody";
        default:
            return "download.confirmReplace.conflictBody";
    }
});

const onBatchDialogModel = (value: boolean) => {
    if (!value && batchConfirmDialog.value.show) {
        cancelBatchInstall();
    }
};

const batchUnpin = (results: models.ModProject[]) => {
    downloadStore.batchUnpin(results);
};

const modIDFromResult = (result: models.ModProject) => {
    const id = result?.id || "";
    return id.includes(":") ? id.split(":").slice(1).join(":") : (result.projectId || id);
};

const favoriteDraftFromResult = (result: models.ModProject): FavoriteModDraft => ({
    platform: result.platform,
    modId: modIDFromResult(result),
    minecraftVersion: minecraftStore.selectedMinecraftVersion,
    modLoader: minecraftStore.selectedModLoader,
    title: result.title,
    slug: result.slug,
    iconUrl: result.iconUrl,
    description: result.description,
    categories: result.categories || [],
});

const openAddFavorites = (results: models.ModProject[]) => {
    if (!minecraftStore.selectedMinecraftVersion.trim() || !minecraftStore.selectedModLoader.trim()) return;
    const drafts = results.map(favoriteDraftFromResult).filter((draft) => draft.platform && draft.modId);
    if (drafts.length) {
        addFavoriteDialog.value?.open(drafts);
    }
};

const onFavoritesAdded = () => {
    downloadStore.showSnackbar("favorites.added", "success");
};

const checkMinecraftDir = async () => {
    showDirOverlay.value = !(await ValidateMinecraftDir());
};

const syncDownloadPageState = async () => {
    await downloadStore.start();
    await checkMinecraftDir();
    downloadStore.setTargetTuple(
        minecraftStore.selectedMinecraftVersion,
        minecraftStore.selectedModLoader,
        minecraftStore.hasSelectedInstance,
    );
    await downloadStore.refreshDownloadStates();
};

onActivated(syncDownloadPageState);

onDeactivated(() => {
    downloadStore.stop();
});

watch(
    () => [
        minecraftStore.selectedMinecraftVersion,
        minecraftStore.selectedModLoader,
        minecraftStore.hasSelectedInstance,
    ],
    ([minecraftVersion, modLoader, hasSelectedInstance]) => {
        downloadStore.setTargetTuple(
            String(minecraftVersion || ""),
            String(modLoader || ""),
            Boolean(hasSelectedInstance),
        );
    },
);
</script>

<style scoped>
.download-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    max-width: 960px;
    min-height: 0;
    overflow: hidden;
}

.search-controls {
    flex: 0 0 auto;
    padding-top: 12px;
}

.search-empty {
    align-items: center;
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    justify-content: center;
    min-height: 0;
    padding-bottom: 96px;
    text-align: center;
}

.search-result-list {
    flex: 1 1 auto;
    min-height: 0;
}

.version-overlay {
    overflow: hidden;
}

.batch-conflict-list {
    display: grid;
    gap: 10px;
    max-height: min(320px, 52vh);
    overflow-y: auto;
    padding-inline-end: 16px;
}

.download-preflight-list {
    max-height: min(320px, 52vh);
    overflow-y: auto;
    padding-inline-end: 16px;
}

.batch-conflict-entry {
    border: 1px solid rgba(var(--v-border-color), 0.3);
    border-radius: 6px;
    padding: 10px;
}

.batch-conflict-detail {
    margin-top: 6px;
}

@media (max-width: 599.98px) {
    .download-page {
        padding-inline: 16px !important;
    }
}
</style>
