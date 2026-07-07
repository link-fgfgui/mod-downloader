<template>
    <v-divider></v-divider>
    <div class="pa-2 position-relative">
        <transition name="md-expand">
            <div v-show="isExpanded" class="md-animate-fade-up">
                <div class="d-flex align-center" style="gap: 8px">
                    <v-select
                        v-model="selectedVersionName"
                        :items="versionList"
                        label="Version"
                        density="compact"
                        hide-details
                        variant="outlined"
                        class="flex-grow-1"
                    ></v-select>
                    <v-btn
                        class="md-btn-press md-hover-scale"
                        icon="mdi-refresh"
                        variant="text"
                        :disabled="isRefreshing || isLoading"
                        density="compact"
                        size="small"
                        @click="refreshSelectedMods"
                    ></v-btn>
                </div>
                <v-combobox
                    v-model="selectedMinecraftVersion"
                    :items="releaseVersionList"
                    label="Minecraft Version"
                    density="compact"
                    hide-details
                    variant="outlined"
                    class="mt-2"
                    clearable
                ></v-combobox>
                <v-select
                    v-model="selectedModLoader"
                    :items="modLoaderList"
                    label="Mod Loader"
                    density="compact"
                    hide-details
                    variant="outlined"
                    class="mt-2"
                ></v-select>
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
        </transition>
        <div
            v-show="!isExpanded"
            class="version-rail-summary"
            :title="selectedVersionName || 'Version'"
            :aria-label="selectedVersionName || 'Version'"
        >
            <v-icon icon="mdi-cube-outline" size="24"></v-icon>
        </div>
        <v-overlay
            v-model="isLoading"
            contained
            persistent
            class="align-center justify-center"
            scrim="rgba(var(--v-theme-surface), 0.7)"
        >
            <v-progress-circular indeterminate color="primary" />
        </v-overlay>
    </div>
</template>
<script setup lang="ts">
import { computed } from "vue";
import { storeToRefs } from "pinia";

import { useMinecraftStore } from "../../stores/minecraft";

const props = defineProps({
    isExpanded: {
        type: Boolean,
        default: false,
    },
});

void props;

const minecraftStore = useMinecraftStore();
const {
    minecraftDir: downloadFolder,
    isRefreshing,
    isLoading,
    selectedMinecraftVersion: storeMinecraftVersion,
    selectedModLoader: storeModLoader,
} = storeToRefs(minecraftStore);

const versionList = computed(() =>
    minecraftStore.versions
        .map((version) => typeof version === "string" ? version : version.name || version.id)
        .filter(Boolean),
);

const releaseVersionList = computed(() => minecraftStore.releaseVersions);
const modLoaderList = computed(() => minecraftStore.modLoaderList);

const selectedVersionName = computed({
    get: () => {
        const selected = minecraftStore.selectedVersion;
        return selected?.name || selected?.id || "";
    },
    set: (version: string) => {
        void selectVersion(version);
    },
});

const selectedMinecraftVersion = computed({
    get: () => storeMinecraftVersion.value,
    set: (version: string | null) => {
        minecraftStore.setSelectedMinecraftVersion(version || "");
    },
});

const selectedModLoader = computed({
    get: () => storeModLoader.value,
    set: (modLoader: string) => {
        minecraftStore.setSelectedModLoader(modLoader);
    },
});

const refreshSelectedMods = async () => {
    await minecraftStore.refreshSelectedMods();
};

const selectVersion = async (version: string) => {
    if (version) {
        await minecraftStore.selectVersion(version);
    }
};

const selectFolder = async () => {
    await minecraftStore.chooseMinecraftDir();
};
</script>
<style scoped>
.version-rail-summary {
    align-items: center;
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
    display: flex;
    justify-content: center;
    min-height: 48px;
}
</style>
