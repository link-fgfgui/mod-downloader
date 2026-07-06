<template>
    <div class="virtual-list-wrapper" @keydown="onKeydown">
        <v-virtual-scroll ref="scrollRef" :items="virtualItems" class="virtual-list-scroll px-2" :item-height="itemHeight"
            tabindex="0">
            <template #default="{ item }">
                <template v-if="item.type === 'item'">
                    <slot name="item" :item="item.raw" :index="item.index" :selected="selectedIndices.has(item.index)"
                        :on-click="(event) => onItemClick(item.index, event)"
                        :enter-style="itemEnterStyle(item.index)">
                    </slot>
                </template>

                <div v-else :ref="setLoadMoreTarget" class="load-more-target py-4 text-center">
                    <v-progress-circular v-if="loadingMore" color="primary" indeterminate size="24"></v-progress-circular>
                    <span v-else-if="items.length && !hasMore" class="text-caption text-medium-emphasis">{{ noMoreText }}</span>
                </div>
            </template>
        </v-virtual-scroll>

        <Transition name="action-bar">
            <div v-if="selectable && selectedIndices.size > 0" class="floating-action-bar">
                <v-chip size="small" variant="tonal" color="primary" class="me-2">
                    {{ selectedIndices.size }}
                </v-chip>

                <slot name="actions" :selected-items="selectedItemsList" :selected-indices="selectedIndices"
                    :clear-selection="clearSelection">
                </slot>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue";

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

const emit = defineEmits(["load-more", "item-click"]);

const loadMoreTarget = ref(null);
let observer = null;
let lastLoadMoreResultCount = 0;

const scrollRef = ref(null);
let lastScrollTop = 0;
let currentDirection = "down";

const selectedIndices = reactive(new Set());
let lastClickedIndex = null;

const handleScroll = (e) => {
    const el = e.currentTarget;
    const currentScrollTop = el.scrollTop;
    if (currentScrollTop > lastScrollTop && currentDirection !== "down") {
        currentDirection = "down";
        el.style.setProperty("--md-fade-y-offset", "18px");
    } else if (currentScrollTop < lastScrollTop && currentDirection !== "up") {
        currentDirection = "up";
        el.style.setProperty("--md-fade-y-offset", "-18px");
    }
    lastScrollTop = currentScrollTop;
};

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

const itemEnterStyle = (index) => ({
    animationDelay: `${Math.min(index, 5) * 40}ms`,
});

const clearSelection = () => {
    selectedIndices.clear();
    lastClickedIndex = null;
};

const selectAll = () => {
    for (let i = 0; i < props.items.length; i++) {
        selectedIndices.add(i);
    }
};

const onItemClick = (index, event) => {
    if (!props.selectable) return;

    if (event.shiftKey && lastClickedIndex !== null) {
        const from = Math.min(lastClickedIndex, index);
        const to = Math.max(lastClickedIndex, index);
        if (!event.ctrlKey && !event.metaKey) {
            selectedIndices.clear();
        }
        for (let i = from; i <= to; i++) {
            selectedIndices.add(i);
        }
    } else if (event.ctrlKey || event.metaKey) {
        if (selectedIndices.has(index)) {
            selectedIndices.delete(index);
        } else {
            selectedIndices.add(index);
        }
    } else {
        if (selectedIndices.has(index) && selectedIndices.size === 1) {
            selectedIndices.clear();
        } else {
            selectedIndices.clear();
            selectedIndices.add(index);
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

const selectedItemsList = computed(() => {
    return [...selectedIndices].sort((a, b) => a - b).map((i) => props.items[i]).filter(Boolean);
});

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

    const scrollEl = scrollRef.value?.$el;
    if (scrollEl) {
        lastScrollTop = scrollEl.scrollTop;
        scrollEl.style.setProperty("--md-fade-y-offset", "18px");
        scrollEl.addEventListener("scroll", handleScroll, { passive: true });
    }
});

onUnmounted(() => {
    const scrollEl = scrollRef.value?.$el;
    if (scrollEl) {
        scrollEl.removeEventListener("scroll", handleScroll);
    }
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
});
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
