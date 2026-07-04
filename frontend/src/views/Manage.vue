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
                                <v-img v-if="group.primary.iconUrl" :src="group.primary.iconUrl" :alt="group.primary.name"></v-img>
                                <v-icon v-else icon="mdi-package-variant" color="on-surface-variant"></v-icon>
                            </v-avatar>
                        </div>
                    </template>

                    <v-list-item-title class="font-weight-medium">
                            <v-tooltip v-if="hasGroupedDetails(group)" :text="groupTooltip(group)" location="top">
                                <template #activator="{ props: tip }">
                                    <span v-bind="tip">{{ group.primary.name || group.primary.id }}
                                        <v-icon icon="mdi-package-variant-closed-plus" size="14" class="ms-1 text-medium-emphasis"></v-icon>
                                    </span>
                                </template>
                        </v-tooltip>
                        <span v-else>{{ group.primary.name || group.primary.id }}</span>
                    </v-list-item-title>
                    <v-list-item-subtitle class="text-caption text-medium-emphasis">
                        {{ strongModIds(group).join(", ") }}
                        <span v-if="group.primary.version"> · {{ group.primary.version }}</span>
                        <span v-if="group.primary.fileName || group.primary.path"> · {{ group.primary.fileName || group.primary.path }}</span>
                    </v-list-item-subtitle>

                    <template #append>
                        <v-chip :color="group.primary.enabled ? 'success' : 'warning'" size="small" variant="tonal">
                            {{ group.primary.enabled ? $t("manage.enabled") : $t("manage.disabled") }}
                        </v-chip>
                    </template>
                </v-list-item>
            </template>

            <template #actions="{ selectedItems, clearSelection }">
                <v-btn size="small" variant="tonal" class="me-1"
                    prepend-icon="mdi-content-copy"
                    @click="copyModNames(selectedItems)">
                    {{ $t('manage.copyNames') }}
                </v-btn>

                <v-btn size="small" variant="tonal" class="me-1"
                    prepend-icon="mdi-identifier"
                    @click="copyModIds(selectedItems)">
                    {{ $t('manage.copyIds') }}
                </v-btn>

                <v-btn size="small" variant="tonal" color="error"
                    prepend-icon="mdi-selection-off"
                    @click="clearSelection()">
                    {{ $t('download.selection.deselectAll') }}
                </v-btn>
            </template>
        </VirtualList>
    </v-container>
</template>

<script setup>
import { computed, onActivated } from "vue";
import { storeToRefs } from "pinia";

import VirtualList from "../components/VirtualList.vue";
import { useMinecraftStore } from "../stores/minecraft";

const minecraftStore = useMinecraftStore();
const { isRefreshing } = storeToRefs(minecraftStore);

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

onActivated(() => {
    minecraftStore.refreshSelectedMods();
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
</style>
