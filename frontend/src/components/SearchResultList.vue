<template>
    <VirtualList
        :items="results"
        :item-height="88"
        :item-key="itemKey"
        :has-more="hasMore"
        :loading-more="loadingMore"
        :no-more-text="$t('search.noMoreResults')"
        @load-more="emit('load-more')"
    >
        <template #item="{ item, index, selected, onClick }">
            <v-list-item :key="itemKey(item, index)" :title="item.title"
                :subtitle="item.description"
                :class="['mb-2 border-b md-hover-lift', { 'search-result-selected': selected }]"
                :bg-color="selected ? undefined : 'surface'"
                rounded="xl" elevation="1"
                lines="two"
                @click="onClick">
                <template #prepend>
                    <div class="align-self-start pt-1 me-3">
                        <v-avatar class="cursor-pointer" color="surface-container-high" rounded="lg" size="48"
                            @click.stop="emit('show-versions', item)">
                            <v-img v-if="item.iconUrl" :src="item.iconUrl" :alt="item.title"></v-img>
                            <v-icon v-else :icon="item.icon" color="on-surface-variant"></v-icon>
                        </v-avatar>
                    </div>
                </template>

                <template #append>
                    <div class="search-result-actions d-flex align-center g-2">
                        <v-chip size="small" variant="flat" color="surface-container-highest"
                            class="search-result-platform text-caption">
                            {{ item.platform }}
                        </v-chip>

                        <v-btn class="search-result-favorite-btn md-btn-press md-hover-scale transition-btn" icon="mdi-playlist-plus"
                            variant="tonal" rounded="xl" size="small" color="secondary"
                            @click.stop="emit('add-favorite', [item])">
                        </v-btn>

                        <v-btn class="md-btn-press md-hover-scale transition-btn" icon variant="tonal" rounded="xl"
                            size="small" :color="colorFor(index)" :loading="loadingFor(index)"
                            :disabled="disabledFor(index)" @click.stop="onInstall(index, true)"
                            @contextmenu.prevent.stop="onInstall(index, false)">
                            <Transition name="icon-fade" mode="out-in">
                                <v-icon :key="iconFor(index)" :icon="iconFor(index)"></v-icon>
                            </Transition>
                        </v-btn>
                    </div>
                </template>
            </v-list-item>
        </template>

        <template #actions="{ selectedItems, clearSelection }">
            <v-tooltip v-for="action in selectionActions(selectedItems, clearSelection)" :key="action.label"
                :text="action.label" location="top">
                <template #activator="{ props: tip }">
                    <v-btn v-bind="tip" :aria-label="action.label" :icon="action.icon" size="small"
                        variant="tonal" :color="action.color" @click="action.run"></v-btn>
                </template>
            </v-tooltip>
        </template>
    </VirtualList>
</template>

<script setup>
import { useI18n } from "vue-i18n";
import VirtualList from "./VirtualList.vue";

const { t: $t } = useI18n();

const props = defineProps({
    results: {
        type: Array,
        default: () => [],
    },
    loadingMore: {
        type: Boolean,
        default: false,
    },
    hasMore: {
        type: Boolean,
        default: true,
    },
    states: {
        type: Array,
        default: () => [],
    },
    downloadingKeys: {
        type: Object,
        default: () => ({}),
    },
});

const emit = defineEmits(["install", "load-more", "show-versions", "batch-download", "batch-unpin", "add-favorite"]);

const itemKey = (item, index) => item.id || `${item.platform}-${item.slug}-${index}`;

const stateFor = (index) => props.states[index];

const onInstall = (index, allowConfirm) => {
    const state = stateFor(index);
    emit("install", {
        result: props.results[index],
        key: state?.key,
        status: state?.status,
        conflictFileName: state?.conflictFileName,
        confirm: allowConfirm,
    });
};

const iconFor = (index) => stateFor(index)?.icon || "mdi-download";

const colorFor = (index) => stateFor(index)?.color || "surface-variant";

const loadingFor = (index) => {
    const state = stateFor(index);
    if (!state) return false;
    return Boolean(state.loading || props.downloadingKeys[state.key]);
};

const disabledFor = (index) => {
    const state = stateFor(index);
    return !state || Boolean(state.disabled);
};

const onCopyNames = async (selectedItems) => {
    const names = selectedItems.map((r) => r.title).join("\n");
    try {
        await navigator.clipboard.writeText(names);
    } catch {
        // fallback: silently fail
    }
};

const selectionActions = (selectedItems, clearSelection) => [
    { label: $t("download.selection.downloadAll"), icon: "mdi-download-multiple", color: "primary", run: () => emit("batch-download", selectedItems) },
    { label: $t("favorites.addToFavorites"), icon: "mdi-playlist-plus", color: "secondary", run: () => emit("add-favorite", selectedItems) },
    { label: $t("download.selection.unpin"), icon: "mdi-pin-off", color: "secondary", run: () => emit("batch-unpin", selectedItems) },
    { label: $t("download.selection.copyNames"), icon: "mdi-content-copy", run: () => onCopyNames(selectedItems) },
    { label: $t("download.selection.deselectAll"), icon: "mdi-selection-off", color: "error", run: clearSelection },
];
</script>

<style scoped>
.g-2 {
    gap: 8px;
}

.transition-btn {
    transition: transform 0.2s ease, color 0.2s ease, background-color 0.2s ease;
}

.icon-fade-enter-active,
.icon-fade-leave-active {
    transition: opacity 0.2s ease, transform 0.2s ease;
}

.icon-fade-enter-from,
.icon-fade-leave-to {
    opacity: 0;
    transform: scale(0.8);
}

.cursor-pointer {
    cursor: pointer;
}

.search-result-selected {
    background-color: rgba(var(--v-theme-primary), 0.12) !important;
    transition: background-color var(--md-transition-fast) ease;
}

@media (max-width: 599.98px) {
    :deep(.v-list-item) {
        padding-inline: 8px;
    }

    .cursor-pointer {
        height: 40px !important;
        width: 40px !important;
    }

    :deep(.v-list-item__append) {
        margin-inline-start: 8px;
    }

    .search-result-actions {
        gap: 4px;
    }

    .search-result-actions :deep(.v-btn) {
        height: 36px;
        width: 36px;
    }

    .search-result-favorite-btn {
        display: none;
    }

    .search-result-platform {
        display: none;
    }
}
</style>
