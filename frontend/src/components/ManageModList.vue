<template>
    <VirtualList :items="items" :item-height="72" :item-key="itemKey"
        :item-selectable="selectable"
        class="manage-list" @scroll="onListScroll" @pointer-move="restoreListTooltips">
        <template #item="{ item: group, selected, onClick }">
            <v-list-item
                v-if="group.kind === 'incompatible-toggle'"
                class="incompatible-toggle mb-2 border-b"
                rounded="lg"
                lines="two"
                @click="emit('toggle-incompatible')"
            >
                <template #prepend>
                    <v-icon icon="mdi-package-variant-remove" color="warning"></v-icon>
                </template>
                <v-list-item-title>
                    {{ $t("manage.incompatible.title", { n: incompatibleCount }) }}
                </v-list-item-title>
                <v-list-item-subtitle>
                    {{ $t(incompatibleExpanded ? "manage.incompatible.expanded" : "manage.incompatible.collapsed") }}
                </v-list-item-subtitle>
                <template #append>
                    <v-icon :icon="incompatibleExpanded ? 'mdi-chevron-up' : 'mdi-chevron-down'"></v-icon>
                </template>
            </v-list-item>
            <v-list-item v-else class="mb-2 border-b md-hover-lift"
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
                            <v-img v-else-if="group.primary.iconUrl" :src="group.primary.iconUrl" :alt="displayName(group)"></v-img>
                            <v-icon v-else icon="mdi-package-variant" color="on-surface-variant"></v-icon>
                        </v-avatar>
                    </div>
                </template>

                <v-list-item-title class="font-weight-medium">
                        <v-tooltip v-if="hasGroupedDetails(group)" :disabled="listTooltipsPaused" :text="groupTooltip(group)" location="top">
                            <template #activator="{ props: tip }">
                                <span v-bind="tip">{{ displayName(group) }}
                                    <v-icon icon="mdi-package-variant-closed-plus" size="14" class="ms-1 text-medium-emphasis"></v-icon>
                                </span>
                            </template>
                    </v-tooltip>
                    <span v-else>{{ displayName(group) }}</span>
                    <v-chip v-if="unusedCandidateFor(group)" class="ms-2" color="warning" size="x-small" variant="tonal"
                        prepend-icon="mdi-broom" :title="unusedTooltip(group)">
                        {{ $t("manage.unused.chip") }}
                    </v-chip>
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
                                        @click.stop="emit('open-versions', group)"
                                    >
                                        · {{ group.primary.version }}
                                    </button>
                                </template>
                            </v-tooltip>
                            <span v-if="group.primary.fileName || group.primary.path"> · {{ group.primary.fileName || group.primary.path }}</span>
                        </span>
                        <v-tooltip v-if="categories(group).length" :disabled="listTooltipsPaused" :text="categoryTooltip(group)" location="top">
                            <template #activator="{ props: tagTip }">
                                <span v-bind="tagTip" class="manage-category-strip">
                                    <v-chip class="manage-category-chip" size="x-small" variant="tonal">
                                        {{ categories(group)[0] }}
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
                            @click.stop="emit('open-favorites', [group])"
                        ></v-btn>
                        <v-btn
                            class="manage-status-toggle"
                            :color="group.primary.enabled ? 'success' : 'warning'"
                            :prepend-icon="group.primary.enabled ? 'mdi-toggle-switch' : 'mdi-toggle-switch-off-outline'"
                            size="small"
                            variant="tonal"
                            :loading="pendingTogglePath === group.primary.path"
                            :disabled="busy"
                            :aria-label="group.primary.enabled ? $t('manage.disableSelected') : $t('manage.enableSelected')"
                            :title="group.primary.enabled ? $t('manage.disableSelected') : $t('manage.enableSelected')"
                            @click.stop="emit('toggle-group', group)"
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
                            @click="emit('open-favorites', selectedItems)"></v-btn>
                    </span>
                </template>
            </v-tooltip>

            <v-tooltip v-for="action in selectionActions(selectedItems, clearSelection)" :key="action.label"
                :text="action.label" location="top">
                <template #activator="{ props: tip }">
                    <span v-bind="tip" class="d-inline-flex">
                        <v-btn :aria-label="action.label" :icon="action.icon" size="small" variant="tonal"
                            :color="action.color" :loading="action.loading" :disabled="busy"
                            @click="action.run"></v-btn>
                    </span>
                </template>
            </v-tooltip>
        </template>
    </VirtualList>
</template>

<script setup>
import { onBeforeUnmount, ref } from "vue";

import VirtualList from "./VirtualList.vue";

defineProps({
    items: { type: Array, default: () => [] },
    itemKey: { type: Function, required: true },
    selectable: { type: Function, required: true },
    busy: { type: Boolean, default: false },
    pendingTogglePath: { type: String, default: "" },
    incompatibleExpanded: { type: Boolean, default: false },
    incompatibleCount: { type: Number, default: 0 },
    displayName: { type: Function, required: true },
    hasGroupedDetails: { type: Function, required: true },
    groupTooltip: { type: Function, required: true },
    strongModIds: { type: Function, required: true },
    versionTooltip: { type: Function, required: true },
    canReplaceVersion: { type: Function, required: true },
    categories: { type: Function, required: true },
    categoryTooltip: { type: Function, required: true },
    canFavoriteGroup: { type: Function, required: true },
    canFavoriteSelection: { type: Function, required: true },
    unusedCandidateFor: { type: Function, required: true },
    unusedTooltip: { type: Function, required: true },
    selectionActions: { type: Function, required: true },
});

const emit = defineEmits(["toggle-incompatible", "open-versions", "open-favorites", "toggle-group"]);

const listTooltipsPaused = ref(false);
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

onBeforeUnmount(() => {
    window.clearTimeout(listScrollTimer);
});
</script>

<style scoped>
.manage-list {
    flex: 1 1 auto;
    min-height: 0;
}

.incompatible-toggle {
    background: rgba(var(--v-theme-warning), 0.08);
    cursor: pointer;
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
