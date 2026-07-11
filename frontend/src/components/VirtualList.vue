<template>
    <div class="virtual-list-wrapper" @keydown="onKeydown" @mousemove="emit('pointer-move')">
        <v-virtual-scroll ref="virtualScroll" :items="virtualItems" class="virtual-list-scroll px-2" :item-height="itemHeight"
            :style="{ '--virtual-list-item-height': `${itemHeight}px` }"
            tabindex="0"
            @scroll.passive="emit('scroll')">
            <template #default="{ item }">
                <template v-if="item.type === 'item'">
                    <slot name="item" :item="item.raw" :index="item.index" :selected="selectedIndices.has(item.index)"
                        :on-click="(event) => onItemClick(item.index, event)">
                    </slot>
                </template>

                <div v-else :ref="setLoadMoreTarget" class="load-more-target py-4 text-center">
                    <v-progress-circular v-if="loadingMore" color="primary" indeterminate size="24"></v-progress-circular>
                    <span v-else-if="items.length && !hasMore" class="text-caption text-medium-emphasis">{{ noMoreText }}</span>
                </div>
            </template>
        </v-virtual-scroll>

        <Transition name="action-bar" @after-leave="clearActionBarSnapshot">
            <div v-if="selectable && selectedIndices.size > 0" class="floating-action-bar">
                <v-chip size="small" variant="tonal" color="primary" class="me-2">
                    {{ visibleSelectionCount }}
                </v-chip>

                <slot name="actions" :selected-items="visibleSelectedItemsList" :selected-indices="visibleSelectedIndices"
                    :clear-selection="clearSelection">
                </slot>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { computed, nextTick, onActivated, onDeactivated, onMounted, onUnmounted, reactive, ref, watch } from "vue";

let activeListKeyHandler = null;
let globalKeyListenerInstalled = false;

const isEditableTarget = (target) => {
    if (!(target instanceof HTMLElement)) return false;
    return target.isContentEditable || Boolean(target.closest("input, textarea, select, [contenteditable='true']"));
};

const hasActiveDialog = () => Boolean(document.querySelector(
    ".v-overlay--active:not(.v-overlay--contained):not(.v-snackbar):not(.v-tooltip)",
));

const forwardActiveListKeydown = (event) => {
    if (event.defaultPrevented || isEditableTarget(event.target) || hasActiveDialog()) return;
    activeListKeyHandler?.(event);
};

const installGlobalKeyListener = () => {
    if (globalKeyListenerInstalled) return;
    document.addEventListener("keydown", forwardActiveListKeydown);
    globalKeyListenerInstalled = true;
};

const props = defineProps({
    items: {
        type: Array,
        default: () => [],
    },
    itemHeight: {
        type: Number,
        default: 88,
    },
    itemKey: {
        type: Function,
        default: (item, index) => index,
    },
    hasMore: {
        type: Boolean,
        default: false,
    },
    loadingMore: {
        type: Boolean,
        default: false,
    },
    selectable: {
        type: Boolean,
        default: true,
    },
    noMoreText: {
        type: String,
        default: "",
    },
});

const emit = defineEmits(["load-more", "item-click", "scroll", "pointer-move"]);

const virtualScroll = ref(null);
const loadMoreTarget = ref(null);
let observer = null;
let lastLoadMoreResultCount = 0;

const selectedIndices = reactive(new Set());
const actionBarSnapshot = ref(null);
const latestSelectedItems = ref([]);
let lastClickedIndex = null;

const virtualItems = computed(() => {
    const wrapped = props.items.map((raw, index) => ({
        type: "item",
        key: props.itemKey(raw, index),
        raw,
        index,
    }));
    if (props.loadingMore || props.hasMore || (props.items.length > 0 && props.noMoreText)) {
        wrapped.push({
            type: "footer",
            key: "load-more-footer",
        });
    }
    return wrapped;
});

const selectedItemsList = computed(() => {
    return [...selectedIndices].sort((a, b) => a - b).map((i) => props.items[i]).filter(Boolean);
});

const refreshLatestSelectedItems = () => {
    latestSelectedItems.value = selectedItemsList.value;
};

const snapshotActionBarSelection = () => {
    if (selectedIndices.size === 0) return;
    actionBarSnapshot.value = {
        indices: new Set(selectedIndices),
        items: latestSelectedItems.value.length ? latestSelectedItems.value : selectedItemsList.value,
    };
};

const clearSelection = () => {
    snapshotActionBarSelection();
    selectedIndices.clear();
    lastClickedIndex = null;
};

const selectAll = () => {
    actionBarSnapshot.value = null;
    for (let i = 0; i < props.items.length; i++) {
        selectedIndices.add(i);
    }
    refreshLatestSelectedItems();
};

const onItemClick = (index, event) => {
    if (!props.selectable) return;

    if (event.shiftKey && lastClickedIndex !== null) {
        event.preventDefault();
        const from = Math.min(lastClickedIndex, index);
        const to = Math.max(lastClickedIndex, index);
        if (!event.ctrlKey && !event.metaKey) {
            actionBarSnapshot.value = null;
            selectedIndices.clear();
        }
        for (let i = from; i <= to; i++) {
            selectedIndices.add(i);
        }
        refreshLatestSelectedItems();
    } else if (event.ctrlKey || event.metaKey) {
        if (selectedIndices.has(index)) {
            if (selectedIndices.size === 1) {
                snapshotActionBarSelection();
            }
            selectedIndices.delete(index);
            if (selectedIndices.size > 0) {
                refreshLatestSelectedItems();
            }
        } else {
            actionBarSnapshot.value = null;
            selectedIndices.add(index);
            refreshLatestSelectedItems();
        }
    } else {
        if (selectedIndices.has(index) && selectedIndices.size === 1) {
            snapshotActionBarSelection();
            selectedIndices.clear();
        } else {
            actionBarSnapshot.value = null;
            selectedIndices.clear();
            selectedIndices.add(index);
            refreshLatestSelectedItems();
        }
    }
    lastClickedIndex = index;
    emit("item-click", index, event);
};

const onKeydown = (event) => {
    if (!props.selectable) return;
    if (event.key === "a" && (event.ctrlKey || event.metaKey)) {
        event.preventDefault();
        selectAll();
    } else if (event.key === "Escape") {
        if (selectedIndices.size > 0) {
            clearSelection();
            event.stopPropagation();
        }
    }
};

let measureFrame = 0;
const scheduleMeasure = async () => {
    await nextTick();
    cancelAnimationFrame(measureFrame);
    measureFrame = requestAnimationFrame(() => {
        virtualScroll.value?.calculateVisibleItems?.();
    });
};

const activateList = () => {
    activeListKeyHandler = onKeydown;
    installGlobalKeyListener();
    void scheduleMeasure();
};

const deactivateList = () => {
    if (activeListKeyHandler === onKeydown) {
        activeListKeyHandler = null;
    }
};

const visibleSelectedIndices = computed(() => (
    selectedIndices.size > 0 ? selectedIndices : (actionBarSnapshot.value?.indices || selectedIndices)
));

const visibleSelectedItemsList = computed(() => (
    selectedIndices.size > 0 ? selectedItemsList.value : (actionBarSnapshot.value?.items || [])
));

const visibleSelectionCount = computed(() => (
    selectedIndices.size > 0 ? selectedIndices.size : (actionBarSnapshot.value?.indices?.size || 0)
));

const clearActionBarSnapshot = () => {
    if (selectedIndices.size === 0) {
        actionBarSnapshot.value = null;
        latestSelectedItems.value = [];
    }
};

const setLoadMoreTarget = (element) => {
    if (loadMoreTarget.value === element) return;
    if (loadMoreTarget.value) {
        observer?.unobserve(loadMoreTarget.value);
    }
    loadMoreTarget.value = element;
    if (element) {
        observer?.observe(element);
    }
};

onMounted(() => {
    activateList();
    observer = new IntersectionObserver((entries) => {
        const count = props.items.length;
        if (entries.some((e) => e.isIntersecting) && count > 2 && count !== lastLoadMoreResultCount && props.hasMore && !props.loadingMore) {
            lastLoadMoreResultCount = count;
            emit("load-more");
        }
    }, { rootMargin: "160px" });

    if (loadMoreTarget.value) {
        observer.observe(loadMoreTarget.value);
    }

});

onActivated(activateList);
onDeactivated(deactivateList);

onUnmounted(() => {
    deactivateList();
    cancelAnimationFrame(measureFrame);
    observer?.disconnect();
    observer = null;
});

let previousItemsRef = null;

watch(() => props.items, (next, previous) => {
    if (next.length < (previous?.length || 0)) {
        lastLoadMoreResultCount = 0;
    }
    if (next !== previousItemsRef && next.length <= (previous?.length || 0)) {
        clearSelection();
    }
    previousItemsRef = next;
    void scheduleMeasure();
});

watch(() => props.itemHeight, scheduleMeasure);
</script>

<style scoped>
.virtual-list-wrapper {
    position: relative;
    display: flex;
    flex-direction: column;
    flex: 1 1 auto;
    min-height: 0;
}

.virtual-list-scroll {
    flex: 1 1 auto;
    min-height: 0;
    scrollbar-gutter: stable;
}

.virtual-list-scroll :deep(.v-virtual-scroll__item) {
    box-sizing: border-box;
    contain: layout size style;
    height: var(--virtual-list-item-height);
    max-height: var(--virtual-list-item-height);
    min-height: var(--virtual-list-item-height);
}

.load-more-target {
    min-height: 56px;
}

.floating-action-bar {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: 4px;
    justify-content: center;
    max-width: calc(100% - 32px);
    padding: 8px 16px;
    border-radius: 16px;
    background-color: rgba(var(--v-theme-surface), 0.92);
    backdrop-filter: blur(8px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
    z-index: 10;
}

.action-bar-enter-active {
    transition: opacity var(--md-transition-normal) var(--md-ease-out),
                transform var(--md-transition-normal) var(--md-ease-spring);
}

.action-bar-leave-active {
    transition: opacity var(--md-transition-fast) ease,
                transform var(--md-transition-fast) ease;
}

.action-bar-enter-from {
    opacity: 0;
    transform: translateX(-50%) translateY(16px) scale(0.94);
}

.action-bar-leave-to {
    opacity: 0;
    transform: translateX(-50%) translateY(8px) scale(0.96);
}
</style>
