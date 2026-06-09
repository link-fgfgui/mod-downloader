<template>
    <v-container class="download-page pa-6" fluid>
        <v-row class="align-center" dense>
            <v-col cols="12" md="10">
                <v-text-field v-model="searchText" placeholder="emi" label="Search mods" prepend-inner-icon="mdi-magnify"
                    variant="outlined" density="comfortable" hide-details clearable
                    @keyup.enter.prevent="searchMods"></v-text-field>
            </v-col>
            <v-col cols="12" md="2">
                <v-btn block color="primary" height="48" prepend-icon="mdi-magnify" :loading="isSearching"
                    @click="searchMods">
                    Search
                </v-btn>
            </v-col>
        </v-row>

        <v-row class="mt-4" dense>
            <v-col cols="12" md="6">
                <v-select v-model="selectedVersion" :items="versionList" label="Minecraft Version" variant="outlined"
                    density="comfortable" hide-details></v-select>
            </v-col>
            <v-col cols="12" md="6">
                <v-select v-model="selectedModLoader" :items="modLoaderList" label="Mod Loader" variant="outlined"
                    density="comfortable" hide-details></v-select>
            </v-col>
        </v-row>

        <SearchResultList :results="searchResults" :loading-more="isLoadingMore" :has-more="hasMoreResults"
            :states="downloadStates" :downloading-keys="downloadingKeys" @install="installMod"
            @load-more="loadMoreSearchResults" @show-versions="openVersionsOverlay"></SearchResultList>

        <v-overlay v-model="showDirOverlay" contained persistent class="align-center justify-center" scrim="surface">
            <v-card class="pa-6 text-center" max-width="420" variant="elevated">
                <v-icon class="mb-3" color="warning" icon="mdi-folder-alert" size="48"></v-icon>
                <v-card-title class="pa-0 text-h6">{{ $t('download.selectDirTitle') }}</v-card-title>
                <v-card-text class="px-0 pb-0 pt-3">
                    {{ $t('download.selectDirDesc') }}
                </v-card-text>
            </v-card>
        </v-overlay>

        <v-overlay v-model="showVersionsOverlay" location="center" scrim="surface">
            <v-card class="version-overlay" width="680" max-width="calc(100vw - 32px)" max-height="calc(100vh - 64px)">
                <v-toolbar density="compact" color="surface">
                    <v-toolbar-title>{{ selectedMod?.title || "Mod Versions" }}</v-toolbar-title>
                    <v-btn icon="mdi-close" variant="text" @click="showVersionsOverlay = false"></v-btn>
                </v-toolbar>

                <v-divider></v-divider>

                <div v-if="isLoadingVersions" class="pa-8 text-center">
                    <v-progress-circular color="primary" indeterminate></v-progress-circular>
                </div>
                <v-list v-else-if="matchingVersions.length" class="version-list py-0" density="comfortable" lines="one">
                    <v-list-item v-for="version in matchingVersions" :key="version.id" :title="versionFileName(version)">
                        <template #append>
                            <v-btn :color="isPinnedVersion(version) ? 'primary' : 'surface-variant'"
                                :icon="isPinnedVersion(version) ? 'mdi-pin' : 'mdi-pin-outline'" variant="tonal"
                                size="small" :loading="pinningVersionID === version.id"
                                :disabled="isPinningAnotherVersion(version)" @click="pinVersion(version)"></v-btn>
                        </template>
                    </v-list-item>
                </v-list>
                <div v-else class="pa-8 text-center text-medium-emphasis">
                    {{ $t('download.noMatchingVersions') }}
                </div>
            </v-card>
        </v-overlay>

        <v-snackbar v-model="snackbar.show" :color="snackbar.color" location="bottom" timeout="3000">
            {{ snackbar.text }}
        </v-snackbar>

        <v-dialog v-model="confirmDialog.show" max-width="420">
            <v-card>
                <v-card-title class="text-h6">
                    {{ confirmDialog.status === 'update' ? $t('download.confirmReplace.updateTitle') : $t('download.confirmReplace.conflictTitle') }}
                </v-card-title>
                <v-card-text>
                    {{ confirmDialog.status === 'update' ? $t('download.confirmReplace.updateBody') : $t('download.confirmReplace.conflictBody') }}
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="confirmDialog.show = false">{{ $t('download.confirmReplace.cancel') }}</v-btn>
                    <v-btn color="warning" variant="flat" @click="confirmInstall">{{ $t('download.confirmReplace.confirm') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-container>
</template>

<script setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import SearchResultList from "../components/SearchResultList.vue";
import {
    GetMinecraftReleaseVersions,
    GetDownloadStates,
    GetPinnedModVersion,
    GetSelectedVersion,
    ListMatchingProjectVersions,
    PinModVersion,
    QueueModDownload,
    SearchMods,
    ValidateMinecraftDir,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const { t } = useI18n();

const minecraftDirChangedEvent = "minecraft-dir-changed";
const selectedVersionChangedEvent = "selected-version-changed";
const searchModsUpdatedEvent = "search-mods-updated";
const downloadQueueUpdatedEvent = "download-queue-updated";
const downloadFailedEvent = "download-failed";

// Go QueueModDownload 跳过原因 → i18n key 映射
const downloadErrorKeys = {
    "invalid request": "invalidRequest",
    "no matching version": "noMatchingVersion",
    "missing download url": "missingDownloadUrl",
    "no selected version": "noSelectedVersion",
};

const searchText = ref("");
const selectedVersion = ref("");
const selectedModLoader = ref("Fabric");
const showDirOverlay = ref(false);
const isSearching = ref(false);
const isLoadingMore = ref(false);
const hasMoreResults = ref(true);
const showVersionsOverlay = ref(false);
const isLoadingVersions = ref(false);

const versionList = ref([]);
const modLoaderList = ["Fabric", "Forge", "NeoForge"];
const searchResults = ref([]);
const activeSearchRequestID = ref("");
const searchPageSize = 10;
const nextSearchOffset = ref(0);
const activeSearchAppend = ref(false);
const appendBaseResults = ref([]);
const selectedMod = ref(null);
const matchingVersions = ref([]);
const pinnedVersion = ref(null);
const pinningVersionID = ref("");
const downloadStates = ref([]);
const downloadingKeys = ref({});
let activeDownloadStateRequestID = "";

const snackbar = ref({ show: false, text: "", color: "success" });
const showSnackbar = (text, color = "success") => {
    snackbar.value = { show: true, text, color };
};

const checkMinecraftDir = async () => {
    showDirOverlay.value = !(await ValidateMinecraftDir());
};

const loadVersions = async () => {
    versionList.value = await GetMinecraftReleaseVersions();
    if (!selectedVersion.value && versionList.value.length > 0) {
        selectedVersion.value = versionList.value[0];
    }
};

const runSearch = async ({ append = false } = {}) => {
    if (append && (isSearching.value || isLoadingMore.value || !hasMoreResults.value)) {
        return;
    }

    const requestID = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
    activeSearchRequestID.value = requestID;
    activeDownloadStateRequestID = "";
    activeSearchAppend.value = append;
    const offset = append ? nextSearchOffset.value : 0;
    if (append) {
        appendBaseResults.value = [...searchResults.value];
        isLoadingMore.value = true;
    } else {
        searchResults.value = [];
        downloadStates.value = [];
        appendBaseResults.value = [];
        nextSearchOffset.value = 0;
        hasMoreResults.value = true;
        isSearching.value = true;
    }

    try {
        await SearchMods({
            requestId: requestID,
            query: searchText.value,
            version: selectedVersion.value,
            modLoader: selectedModLoader.value,
            offset,
            limit: searchPageSize,
        });
        // 结果只由 search-mods-updated 事件写入；此处仅在请求仍有效时推进分页 offset。
        if (activeSearchRequestID.value === requestID) {
            nextSearchOffset.value = offset + searchPageSize;
        }
    } finally {
        if (activeSearchRequestID.value === requestID) {
            if (append) {
                isLoadingMore.value = false;
            } else {
                isSearching.value = false;
            }
        }
    }
};

const searchMods = async () => {
    await runSearch();
};

const loadMoreSearchResults = async () => {
    await runSearch({ append: true });
};

const refreshDownloadStates = async () => {
    if (isSearching.value || isLoadingMore.value) {
        return;
    }
    const requestID = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
    activeDownloadStateRequestID = requestID;
    const results = searchResults.value || [];

    const states = await GetDownloadStates({
        results,
        minecraftVersion: selectedVersion.value,
        modLoader: selectedModLoader.value,
    });
    if (activeDownloadStateRequestID !== requestID) {
        return;
    }
    downloadStates.value = states || [];
};

// 需要二次确认的状态（黄色按钮：替换旧 jar 属不可逆操作）。
const confirmStatuses = new Set(["update", "conflict"]);
const confirmDialog = ref({ show: false, status: "", result: null, key: "" });

// installMod 由按钮触发。confirm=true 表示来自左键且属黄色状态 → 先弹确认框；
// confirm=false（右键直达 / 普通下载）→ 直接执行。
const installMod = (payload) => {
    const { result, key, status, confirm } = payload || {};
    if (!key || downloadingKeys.value[key]) {
        return;
    }
    if (confirm && confirmStatuses.has(status)) {
        confirmDialog.value = { show: true, status, result, key };
        return;
    }
    void doInstall({ result, key });
};

const confirmInstall = () => {
    const { result, key } = confirmDialog.value;
    confirmDialog.value = { show: false, status: "", result: null, key: "" };
    void doInstall({ result, key });
};

const doInstall = async ({ result, key }) => {
    if (!key || downloadingKeys.value[key]) {
        return;
    }

    downloadingKeys.value = { ...downloadingKeys.value, [key]: true };
    try {
        const res = await QueueModDownload({
            projectId: key,
            result,
            minecraftVersion: selectedVersion.value,
            modLoader: selectedModLoader.value,
        });
        if (res?.skipped) {
            const errorKey = downloadErrorKeys[res.reason];
            showSnackbar(errorKey ? t(`download.errors.${errorKey}`) : t("download.errors.generic"), "error");
        } else {
            showSnackbar(t("download.queued"), "success");
            await refreshDownloadStates();
        }
    } finally {
        const next = { ...downloadingKeys.value };
        delete next[key];
        downloadingKeys.value = next;
    }
};

const modIDFromResult = (result) => {
    const id = result?.id || "";
    return id.includes(":") ? id.split(":").slice(1).join(":") : id;
};

const openVersionsOverlay = async (result) => {
    selectedMod.value = result;
    matchingVersions.value = [];
    pinnedVersion.value = null;
    pinningVersionID.value = "";
    showVersionsOverlay.value = true;
    isLoadingVersions.value = true;

    try {
        const [versions, pin] = await Promise.all([
            ListMatchingProjectVersions(result, selectedVersion.value, selectedModLoader.value),
            GetPinnedModVersion(result.platform, modIDFromResult(result), selectedVersion.value, selectedModLoader.value),
        ]);
        matchingVersions.value = versions || [];
        pinnedVersion.value = pin?.versionId ? pin : null;
    } finally {
        isLoadingVersions.value = false;
    }
};

const pinVersion = async (version) => {
    if (!selectedMod.value || pinningVersionID.value) {
        return;
    }

    pinningVersionID.value = version.id;
    try {
        const pin = await PinModVersion({
            platform: selectedMod.value.platform,
            modId: modIDFromResult(selectedMod.value),
            versionId: version.id,
            minecraftVersion: selectedVersion.value,
            modLoader: selectedModLoader.value,
        });
        pinnedVersion.value = pin?.versionId ? pin : null;
    } finally {
        pinningVersionID.value = "";
    }
};

const isPinnedVersion = (version) => pinnedVersion.value?.versionId === version.id;

const isPinningAnotherVersion = (version) => Boolean(pinningVersionID.value && pinningVersionID.value !== version.id);

const versionFileName = (version) => version.fileName || version.name || version.version || version.id;

const applySelectedVersion = (version) => {
    // 只认显式的 Minecraft 版本字段；实例显示名(name)绝不能进版本选择器,
    // 否则会变成无效的 MC 版本导致搜索为空。未知实例(无 minecraftVersion)则保持当前选择不变。
    const minecraftVersion = version?.minecraftVersion || version?.MinecraftVersion;
    if (minecraftVersion) {
        if (!versionList.value.includes(minecraftVersion)) {
            versionList.value = [minecraftVersion, ...versionList.value];
        }
        selectedVersion.value = minecraftVersion;
    }

    const modLoader = version?.modLoader || version?.ModLoader;
    if (modLoader) {
        const normalizedModLoader = modLoader.toLowerCase();
        const matchingModLoader = modLoaderList.find((item) => item.toLowerCase() === normalizedModLoader);
        if (matchingModLoader) {
            selectedModLoader.value = matchingModLoader;
        }
    }
};

let stopListeningMinecraftDirChanged = null;
let stopListeningSelectedVersionChanged = null;
let stopListeningSearchModsUpdated = null;
let stopListeningDownloadQueueUpdated = null;
let stopListeningDownloadFailed = null;

onMounted(() => {
    // 先同步注册监听，避免在 await 期间错过事件
    stopListeningMinecraftDirChanged = EventsOn(minecraftDirChangedEvent, () => {
        checkMinecraftDir();
        loadVersions();
    });
    stopListeningSelectedVersionChanged = EventsOn(selectedVersionChangedEvent, applySelectedVersion);
    stopListeningDownloadQueueUpdated = EventsOn(downloadQueueUpdatedEvent, () => {
        refreshDownloadStates();
    });
    stopListeningDownloadFailed = EventsOn(downloadFailedEvent, (event) => {
        const fileName = event?.fileName || event?.FileName || t("download.errors.unknownFile");
        const reason = event?.reason || event?.Reason || t("download.errors.generic");
        showSnackbar(t("download.errors.failedWithReason", { fileName, reason }), "error");
        refreshDownloadStates();
    });
    stopListeningSearchModsUpdated = EventsOn(searchModsUpdatedEvent, (update) => {
        const requestID = update?.requestId || update?.RequestID;
        if (requestID !== activeSearchRequestID.value) {
            return;
        }

        const nextResults = update.results || update.Results || [];
        const append = Boolean(update.append ?? update.Append ?? activeSearchAppend.value);
        searchResults.value = append ? [...appendBaseResults.value, ...nextResults] : nextResults;
        hasMoreResults.value = nextResults.length >= searchPageSize;
        const loading = Boolean(update.loading ?? update.Loading);
        if (append) {
            isLoadingMore.value = loading;
        } else {
            isSearching.value = loading;
        }
        if (!loading) {
            void refreshDownloadStates();
        }
    });

    // 初始化：拉一次当前选中实例，使下拉框与实际选中实例一致
    // （否则从别的页面切回时会错过 selected-version-changed 事件）。
    void (async () => {
        checkMinecraftDir();
        await loadVersions();
        const selected = await GetSelectedVersion();
        if (selected && (selected.minecraftVersion || selected.modLoader)) {
            applySelectedVersion(selected);
        }
    })();
});

onUnmounted(() => {
    stopListeningMinecraftDirChanged?.();
    stopListeningSelectedVersionChanged?.();
    stopListeningSearchModsUpdated?.();
    stopListeningDownloadQueueUpdated?.();
    stopListeningDownloadFailed?.();
});

watch([selectedVersion, selectedModLoader], () => {
    downloadStates.value = [];
    refreshDownloadStates();
}, { deep: false });
</script>

<style scoped>
.download-page {
    max-width: 960px;
    min-height: calc(100vh - 32px);
}

.version-overlay {
    overflow: hidden;
}

.version-list {
    max-height: calc(100vh - 128px);
    overflow-y: auto;
}
</style>
