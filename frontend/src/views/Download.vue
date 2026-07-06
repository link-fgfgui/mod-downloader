<template>
    <v-container class="download-page pb-6 px-6 pt-0 md-page" fluid>
        <div
            :class="[
                'search-spacer',
                { 'search-spacer--expanded': isEmptySearch },
            ]"
        ></div>

        <div class="search-controls">
            <v-row class="align-center md-stagger" dense>
                <v-col cols="12" md="10">
                    <v-text-field
                        v-model="searchText"
                        placeholder="emi"
                        label="Search mods"
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
                        Search
                    </v-btn>
                </v-col>
            </v-row>

            <v-row class="mt-4 md-stagger" dense>
                <v-col cols="12" md="6">
                    <v-select
                        v-model="selectedVersion"
                        :items="versionList"
                        label="Minecraft Version"
                        variant="outlined"
                        density="comfortable"
                        hide-details
                    ></v-select>
                </v-col>
                <v-col cols="12" md="6">
                    <v-select
                        v-model="selectedModLoader"
                        :items="modLoaderList"
                        label="Mod Loader"
                        variant="outlined"
                        density="comfortable"
                        hide-details
                    ></v-select>
                </v-col>
            </v-row>
        </div>

        <div
            :class="[
                'search-spacer',
                { 'search-spacer--expanded': isEmptySearch },
            ]"
        ></div>

        <SearchResultList
            :class="[
                'search-result-list',
                { 'search-result-list--empty': isEmptySearch },
            ]"
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

        <v-overlay
            v-model="showDirOverlay"
            transition="scale-fade"
            contained
            persistent
            class="align-center justify-center"
            scrim="surface"
        >
            <v-card
                class="pa-6 text-center md-animate-scale"
                max-width="420"
                variant="elevated"
            >
                <v-icon
                    class="mb-3 md-animate-pulse"
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
                        selectedMod?.title || "Mod Versions"
                    }}</v-toolbar-title>
                    <v-btn
                        icon="mdi-close"
                        variant="text"
                        @click="showVersionsOverlay = false"
                    ></v-btn>
                </v-toolbar>

                <v-divider></v-divider>

                <div v-if="isLoadingVersions" class="pa-8 text-center">
                    <v-progress-circular
                        color="primary"
                        indeterminate
                    ></v-progress-circular>
                </div>
                <v-list
                    v-else-if="matchingVersions.length"
                    class="version-list py-0"
                    density="comfortable"
                    lines="one"
                >
                    <v-list-item
                        v-for="version in matchingVersions"
                        :key="version.id"
                        :title="versionFileName(version)"
                    >
                        <template #append>
                            <v-btn
                                :color="
                                    isPinnedVersion(version)
                                        ? 'primary'
                                        : 'surface-variant'
                                "
                                :icon="
                                    isPinnedVersion(version)
                                        ? 'mdi-pin'
                                        : 'mdi-pin-outline'
                                "
                                variant="tonal"
                                size="small"
                                :loading="pinningVersionID === version.id"
                                :disabled="isPinningAnotherVersion(version)"
                                @click="pinVersion(version)"
                            ></v-btn>
                        </template>
                    </v-list-item>
                </v-list>
                <div v-else class="pa-8 text-center text-medium-emphasis">
                    {{ $t("download.noMatchingVersions") }}
                </div>
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
            transition="scale-fade"
            max-width="420"
        >
            <v-card class="md-animate-scale">
                <v-card-title class="text-h6">
                    {{
                        confirmDialog.status === "update"
                            ? $t("download.confirmReplace.updateTitle")
                            : $t("download.confirmReplace.conflictTitle")
                    }}
                </v-card-title>
                <v-card-text>
                    {{
                        confirmDialog.status === "update"
                            ? $t("download.confirmReplace.updateBody")
                            : $t("download.confirmReplace.conflictBody")
                    }}
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="confirmDialog.show = false">{{
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

        <AddToFavoriteDialog ref="addFavoriteDialog" @added="onFavoritesAdded"></AddToFavoriteDialog>
    </v-container>
</template>

<script setup lang="ts">
import { computed, onActivated, onDeactivated, ref, watch } from "vue";
import { storeToRefs } from "pinia";

import SearchResultList from "../components/SearchResultList.vue";
import AddToFavoriteDialog from "../components/AddToFavoriteDialog.vue";
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
    selectedVersion,
    selectedModLoader,
    showDirOverlay,
    isSearching,
    isLoadingMore,
    hasMoreResults,
    showVersionsOverlay,
    isLoadingVersions,
    versionList,
    modLoaderList,
    searchResults,
    selectedMod,
    matchingVersions,
    pinningVersionID,
    downloadStates,
    downloadingKeys,
    snackbar,
    confirmDialog,
} = storeToRefs(downloadStore);

const isEmptySearch = computed(
    () => !searchResults.value.length && !isSearching.value,
);

const searchMods = () => downloadStore.runSearch();
const loadMoreSearchResults = () => downloadStore.loadMoreSearchResults();
const installMod = (payload: {
    result?: models.ModProject;
    key?: string;
    status?: string;
    confirm?: boolean;
}) => downloadStore.installMod(payload);
const confirmInstall = () => downloadStore.confirmInstall();
const openVersionsOverlay = (result: models.ModProject) =>
    downloadStore.openVersionsOverlay(result);
const pinVersion = (version: models.ModVersion) =>
    downloadStore.pinVersion(version);
const isPinnedVersion = (version: models.ModVersion) =>
    downloadStore.isPinnedVersion(version);
const isPinningAnotherVersion = (version: models.ModVersion) =>
    downloadStore.isPinningAnotherVersion(version);
const versionFileName = (version: models.ModVersion) =>
    downloadStore.versionFileName(version);

const batchDownload = (results: models.ModProject[]) => {
    for (const result of results) {
        const index = searchResults.value.indexOf(result);
        if (index === -1) continue;
        const state = downloadStates.value[index];
        installMod({ result, key: state?.key, status: state?.status, confirm: false });
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
    minecraftVersion: selectedVersion.value,
    modLoader: selectedModLoader.value,
    title: result.title,
    slug: result.slug,
    iconUrl: result.iconUrl,
    description: result.description,
    categories: result.categories || [],
});

const openAddFavorites = (results: models.ModProject[]) => {
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
    downloadStore.setVersionList(minecraftStore.releaseVersions);
    downloadStore.applySelectedInstance(minecraftStore.selectedVersion);
};

onActivated(syncDownloadPageState);

onDeactivated(() => {
    downloadStore.stop();
});

watch(
    () => minecraftStore.releaseVersions,
    (versions) => {
        downloadStore.setVersionList(versions);
    },
);

watch(
    () => minecraftStore.selectedVersion,
    (version) => {
        downloadStore.applySelectedInstance(version);
    },
);

watch(
    [selectedVersion, selectedModLoader],
    () => {
        downloadStore.clearDownloadStates();
    },
    { deep: false },
);
</script>

<style scoped>
.download-page {
    display: flex;
    flex-direction: column;
    max-width: 960px;
    min-height: calc(100vh - 32px);
}

.search-spacer {
    flex: 0 0 0;
    transition: flex 0.4s var(--md-ease-out);
}

.search-spacer--expanded {
    flex: 1 1 0;
}

.search-controls {
    flex: 0 0 auto;
}

.search-result-list {
    flex: 0 0 auto;
    max-height: calc(100vh - 170px);
    transition:
        flex 0.4s var(--md-ease-out),
        max-height 0.4s var(--md-ease-out);
}

.search-result-list--empty {
    flex: 0 0 0 !important;
    max-height: 0 !important;
}

.version-overlay {
    overflow: hidden;
}

.version-list {
    max-height: calc(100vh - 128px);
    overflow-y: auto;
}
</style>
