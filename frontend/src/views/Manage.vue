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

        <div v-else-if="mods.length === 0" class="empty-state md-animate-fade-up">
            <v-icon class="md-animate-float" icon="mdi-package-variant" size="48"></v-icon>
            <div class="text-body-1 mt-3">{{ $t("manage.noMods") }}</div>
        </div>

        <v-table v-else class="mod-table" density="comfortable" fixed-header>
            <thead>
                <tr>
                    <th>{{ $t("manage.columns.mod") }}</th>
                    <th>{{ $t("manage.columns.version") }}</th>
                    <th>{{ $t("manage.columns.file") }}</th>
                    <th>{{ $t("manage.columns.state") }}</th>
                </tr>
            </thead>
            <tbody>
                <tr v-for="(mod, index) in mods" :key="modRowKey(mod)" class="md-animate-fade-up" :style="rowEnterStyle(index)">
                    <td>
                        <div class="font-weight-medium">{{ mod.name || mod.id }}</div>
                        <div class="text-caption text-medium-emphasis">{{ mod.id }}</div>
                    </td>
                    <td>{{ mod.version || "-" }}</td>
                    <td class="file-cell">{{ mod.fileName || mod.path || "-" }}</td>
                    <td>
                        <v-chip :color="mod.enabled ? 'success' : 'warning'" size="small" variant="tonal">
                            {{ mod.enabled ? $t("manage.enabled") : $t("manage.disabled") }}
                        </v-chip>
                    </td>
                </tr>
            </tbody>
        </v-table>
    </v-container>
</template>

<script setup>
import { computed } from "vue";
import { storeToRefs } from "pinia";

import { useMinecraftStore } from "../stores/minecraft";

const minecraftStore = useMinecraftStore();
const { isRefreshing } = storeToRefs(minecraftStore);

const mods = computed(() => minecraftStore.mods);

const hasSelectedInstance = computed(() => {
    return minecraftStore.hasSelectedInstance;
});

const selectedInstanceLabel = computed(() => {
    return minecraftStore.selectedInstanceLabel;
});

const modRowKey = (mod) => {
    return [mod.id, mod.sha1, mod.path, mod.fileName].filter(Boolean).join("|");
};

const rowEnterStyle = (index) => ({
    animationDelay: `${Math.min(index, 15) * 30}ms`,
});

const refreshMods = async () => {
    await minecraftStore.refreshSelectedMods();
};
</script>

<style scoped>
.manage-page {
    max-width: 1080px;
    min-height: calc(100vh - 32px);
}

.manage-header {
    align-items: center;
    display: flex;
    gap: 16px;
    justify-content: space-between;
    margin-bottom: 24px;
}

.empty-state {
    align-items: center;
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 320px;
}

.mod-table {
    max-height: calc(100vh - 176px);
}

.file-cell {
    max-width: 360px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}
</style>
