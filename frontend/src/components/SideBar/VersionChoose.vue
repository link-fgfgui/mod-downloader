<template>
    <v-divider></v-divider>
    <div class="pa-2">
        <div v-show="isExpanded">
            <div class="d-flex align-center" style="gap: 8px">
                <v-select
                    v-model="selectedVersion"
                    :items="versionList"
                    label="Version"
                    density="compact"
                    hide-details
                    variant="outlined"
                    class="flex-grow-1"
                    @update:model-value="selectVersion"
                ></v-select>
                <v-btn
                    icon="mdi-refresh"
                    variant="text"
                    :disabled="isRefreshing"
                    density="compact"
                    size="small"
                    @click="refreshSelectedMods"
                ></v-btn>
            </div>
            <v-text-field
                v-model="downloadFolder"
                label=".minecraft folder"
                density="compact"
                hide-details
                variant="outlined"
                class="mt-2"
                prepend-inner-icon="mdi-folder-open"
                readonly
                @click="selectFolder"
            ></v-text-field>
        </div>
        <span
            v-show="!isExpanded"
            style="font-size: 32px; writing-mode: vertical-rl"
            >{{ selectedVersion }}</span
        >
    </div>
</template>
<script setup>
import { onMounted, ref } from "vue";

import {
    ChooseMinecraftDir,
    GetMinecraftDir,
    GetVersions,
    RefreshSelectedVersionMods,
    RefreshVersions,
    SelectVersion,
} from "../../../wailsjs/go/main/App";
// Bottom area data
const selectedVersion = ref("");
const versionList = ref([]);
const isRefreshing = ref(false);
const downloadFolder = ref("");
const props = defineProps({
    isExpanded: {
        type: Boolean,
        default: false,
    },
});
const refreshVersions = async (force = false) => {
    isRefreshing.value = true;
    try {
        const versions =
            (force ? await RefreshVersions() : await GetVersions()) || [];
        versionList.value = versions
            .map(
                (version) =>
                    version.name || version.Name || version.id || version.ID,
            )
            .filter(Boolean);
        if (!selectedVersion.value && versionList.value.length > 0) {
            selectedVersion.value = versionList.value[0];
            await selectVersion(selectedVersion.value);
        }
    } finally {
        isRefreshing.value = false;
    }
};

const refreshSelectedMods = async () => {
    isRefreshing.value = true;
    try {
        await RefreshSelectedVersionMods();
    } finally {
        isRefreshing.value = false;
    }
};

const selectVersion = async (version) => {
    if (version) {
        await SelectVersion(version);
    }
};

const selectFolder = async () => {
    const result = await ChooseMinecraftDir();
    if (result) {
        downloadFolder.value = result;
        await refreshVersions(true);
    }
};

onMounted(async () => {
    downloadFolder.value = await GetMinecraftDir();
    await refreshVersions();
});
</script>
<style scoped></style>
