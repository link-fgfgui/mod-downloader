<template>
    <v-divider></v-divider>
    <div class="pa-2">
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
        </transition>
        <span
            v-show="!isExpanded"
            style="font-size: 32px; writing-mode: vertical-rl"
        >
            {{ selectedVersionName }}
        </span>
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
const { minecraftDir: downloadFolder, isRefreshing } = storeToRefs(minecraftStore);

const versionList = computed(() =>
    minecraftStore.versions
        .map((version) => typeof version === "string" ? version : version.name || version.id)
        .filter(Boolean),
);

const selectedVersionName = computed({
    get: () => {
        const selected = minecraftStore.selectedVersion;
        return selected?.name || selected?.id || "";
    },
    set: (version: string) => {
        void selectVersion(version);
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
<style scoped></style>
