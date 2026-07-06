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
        <template #item="{ item, index, selected, onClick, enterStyle }">
            <v-list-item :key="itemKey(item, index)" :title="item.title"
                :subtitle="item.description"
                :class="['mb-2 border-b md-animate-fade-y md-hover-lift', { 'search-result-selected': selected }]"
                :bg-color="selected ? undefined : 'surface'"
                rounded="xl" elevation="1"
                lines="two"
                :style="enterStyle"
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
                    <div class="d-flex align-center g-2">
                        <v-chip size="small" variant="flat" color="surface-container-highest" class="text-caption">
                            {{ item.platform }}
                        </v-chip>

                        <v-btn class="md-btn-press md-hover-scale transition-btn" icon="mdi-playlist-plus"
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
            <v-btn size="small" variant="tonal" color="primary" class="me-1"
                prepend-icon="mdi-download-multiple"
                @click="emit('batch-download', selectedItems)">
                {{ $t('download.selection.downloadAll') }}
            </v-btn>

            <v-btn size="small" variant="tonal" color="secondary" class="me-1"
                prepend-icon="mdi-playlist-plus"
                @click="emit('add-favorite', selectedItems)">
                {{ $t('favorites.addToFavorites') }}
            </v-btn>

            <v-btn size="small" variant="tonal" color="secondary" class="me-1"
                prepend-icon="mdi-pin-off"
                @click="emit('batch-unpin', selectedItems)">
                {{ $t('download.selection.unpin') }}
            </v-btn>

            <v-btn size="small" variant="tonal" class="me-1"
                prepend-icon="mdi-content-copy"
                @click="onCopyNames(selectedItems)">
                {{ $t('download.selection.copyNames') }}
            </v-btn>

            <v-btn size="small" variant="tonal" color="error"
                prepend-icon="mdi-selection-off"
                @click="clearSelection()">
                {{ $t('download.selection.deselectAll') }}
            </v-btn>
        </template>
    </VirtualList>
</template>

<script setup>
import VirtualList from "./VirtualList.vue";

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
        confirm: allowConfirm,
    });
};

const iconFor = (index) => stateFor(index)?.icon || "mdi-download";

const colorFor = (index) => stateFor(index)?.color || "primary";

const loadingFor = (index) => {
    const state = stateFor(index);
    if (!state) return true;
    return Boolean(state.loading || props.downloadingKeys[state.key]);
};

const disabledFor = (index) => Boolean(stateFor(index)?.disabled);

const onCopyNames = async (selectedItems) => {
    const names = selectedItems.map((r) => r.title).join("\n");
    try {
        await navigator.clipboard.writeText(names);
    } catch {
        // fallback: silently fail
    }
};
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
</style>
