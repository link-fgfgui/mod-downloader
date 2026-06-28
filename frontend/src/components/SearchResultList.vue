<template>
    <div class="search-result-wrapper" @keydown="onKeydown">
        <v-virtual-scroll ref="scrollRef" :items="virtualItems" class="search-result-scroll px-2" item-height="88"
            tabindex="0">
            <template #default="{ item }">
                <v-list-item v-if="item.type === 'result'" :key="item.key" :title="item.result.title"
                    :subtitle="item.result.description"
                    :class="['mb-2 border-b md-animate-fade-y md-hover-lift', { 'search-result-selected': selectedIndices.has(item.index) }]"
                    :bg-color="selectedIndices.has(item.index) ? undefined : 'surface'"
                    rounded="xl" elevation="1"
                    lines="two"
                    :style="itemEnterStyle(item.index)"
                    @click="onItemClick(item.index, $event)">
                    <template #prepend>
                        <div class="align-self-start pt-1 me-3">
                            <v-avatar class="cursor-pointer" color="surface-container-high" rounded="lg" size="48"
                                @click.stop="emit('show-versions', item.result)">
                                <v-img v-if="item.result.iconUrl" :src="item.result.iconUrl" :alt="item.result.title"></v-img>
                                <v-icon v-else :icon="item.result.icon" color="on-surface-variant"></v-icon>
                            </v-avatar>
                        </div>
                    </template>

                    <template #append>
                        <div class="d-flex align-center g-2">
                            <v-chip size="small" variant="flat" color="surface-container-highest" class="text-caption">
                                {{ item.result.platform }}
                            </v-chip>

                            <v-btn class="md-btn-press md-hover-scale transition-btn" icon variant="tonal" rounded="xl"
                                size="small" :color="colorFor(item.index)" :loading="loadingFor(item.index)"
                                :disabled="disabledFor(item.index)" @click.stop="onInstall(item.index, true)"
                                @contextmenu.prevent.stop="onInstall(item.index, false)">
                                <Transition name="icon-fade" mode="out-in">
                                    <v-icon :key="iconFor(item.index)" :icon="iconFor(item.index)"></v-icon>
                                </Transition>
                            </v-btn>
                        </div>
                    </template>
                </v-list-item>

                <div v-else :ref="setLoadMoreTarget" class="load-more-target py-4 text-center">
                    <v-progress-circular v-if="loadingMore" color="primary" indeterminate size="24"></v-progress-circular>
                    <span v-else-if="results.length && !hasMore" class="text-caption text-medium-emphasis">{{ $t('search.noMoreResults') }}</span>
                </div>
            </template>
        </v-virtual-scroll>

        <Transition name="action-bar">
            <div v-if="selectedIndices.size > 0" class="floating-action-bar">
                <v-chip size="small" variant="tonal" color="primary" class="me-2">
                    {{ $t('download.selection.count', { n: selectedIndices.size }) }}
                </v-chip>

                <v-btn size="small" variant="tonal" color="primary" class="me-1"
                    prepend-icon="mdi-download-multiple"
                    @click="onBatchDownload">
                    {{ $t('download.selection.downloadAll') }}
                </v-btn>

                <v-btn size="small" variant="tonal" color="secondary" class="me-1"
                    prepend-icon="mdi-pin-off"
                    @click="onBatchUnpin">
                    {{ $t('download.selection.unpin') }}
                </v-btn>

                <v-btn size="small" variant="tonal" class="me-1"
                    prepend-icon="mdi-content-copy"
                    @click="onCopyNames">
                    {{ $t('download.selection.copyNames') }}
                </v-btn>

                <v-btn size="small" variant="tonal" color="error"
                    prepend-icon="mdi-selection-off"
                    @click="clearSelection">
                    {{ $t('download.selection.deselectAll') }}
                </v-btn>
            </div>
        </Transition>
    </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from "vue";

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

const emit = defineEmits(["install", "load-more", "show-versions", "batch-download", "batch-unpin"]);

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
    const items = props.results.map((result, index) => ({
        type: "result",
        key: result.id || `${result.platform}-${result.slug}-${index}`,
        result,
        index,
    }));
    if (props.results.length > 0 || props.loadingMore) {
        items.push({
            type: "footer",
            key: "load-more-footer",
        });
    }
    return items;
});

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
    if (!state) {
        return true;
    }
    return Boolean(props.downloadingKeys[state.key]);
};

const disabledFor = (index) => Boolean(stateFor(index)?.disabled);

const itemEnterStyle = (index) => ({
    animationDelay: `${Math.min(index, 5) * 40}ms`,
});

const clearSelection = () => {
    selectedIndices.clear();
    lastClickedIndex = null;
};

const selectAll = () => {
    for (let i = 0; i < props.results.length; i++) {
        selectedIndices.add(i);
    }
};

const onItemClick = (index, event) => {
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
};

const onKeydown = (event) => {
    if (event.key === "a" && (event.ctrlKey || event.metaKey)) {
        event.preventDefault();
        selectAll();
    } else if (event.key === "Escape") {
        clearSelection();
    }
};

const selectedResults = () => {
    return [...selectedIndices].sort((a, b) => a - b).map((i) => props.results[i]).filter(Boolean);
};

const onBatchDownload = () => {
    emit("batch-download", selectedResults());
};

const onBatchUnpin = () => {
    emit("batch-unpin", selectedResults());
};

const onCopyNames = async () => {
    const names = selectedResults().map((r) => r.title).join("\n");
    try {
        await navigator.clipboard.writeText(names);
    } catch {
        // fallback: silently fail
    }
};

const setLoadMoreTarget = (element) => {
    if (loadMoreTarget.value === element) {
        return;
    }
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
        const resultCount = props.results.length;
        if (entries.some((entry) => entry.isIntersecting) && resultCount > 2 && resultCount !== lastLoadMoreResultCount && props.hasMore && !props.loadingMore) {
            lastLoadMoreResultCount = resultCount;
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

let previousResultsRef = null;

watch(() => props.results, (next, previous) => {
    if (next.length < (previous?.length || 0)) {
        lastLoadMoreResultCount = 0;
    }
    if (next !== previousResultsRef && next.length <= (previous?.length || 0)) {
        clearSelection();
    }
    previousResultsRef = next;
});
</script>

<style scoped>
.g-2 {
    gap: 8px;
}

.search-result-wrapper {
    position: relative;
    display: flex;
    flex-direction: column;
    flex: 1 1 auto;
    min-height: 0;
}

.search-result-scroll {
    flex: 1 1 auto;
    min-height: 0;
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

.load-more-target {
    min-height: 56px;
}

.cursor-pointer {
    cursor: pointer;
}

.search-result-selected {
    background-color: rgba(var(--v-theme-primary), 0.12) !important;
    transition: background-color var(--md-transition-fast) ease;
}

.floating-action-bar {
    position: absolute;
    bottom: 16px;
    left: 50%;
    transform: translateX(-50%);
    display: flex;
    align-items: center;
    padding: 8px 16px;
    border-radius: 16px;
    background-color: rgba(var(--v-theme-surface), 0.92);
    backdrop-filter: blur(8px);
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.3);
    z-index: 10;
    white-space: nowrap;
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
