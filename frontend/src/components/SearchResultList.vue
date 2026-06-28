<template>
    <v-virtual-scroll ref="scrollRef" :items="virtualItems" class="search-result-scroll px-2" item-height="88">
        <template #default="{ item }">
            <v-list-item v-if="item.type === 'result'" :key="item.key" :title="item.result.title"
                :subtitle="item.result.description" class="mb-2 border-b md-animate-fade-y md-hover-lift" bg-color="surface" rounded="xl" elevation="1"
                lines="two"
                :style="itemEnterStyle(item.index)">
                <template #prepend>
                    <div class="align-self-start pt-1 me-3">
                        <v-avatar class="cursor-pointer" color="surface-container-high" rounded="lg" size="48"
                            @click="emit('show-versions', item.result)">
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
                            :disabled="disabledFor(item.index)" @click="onInstall(item.index, true)"
                            @contextmenu.prevent="onInstall(item.index, false)">
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
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";

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

const emit = defineEmits(["install", "load-more", "show-versions"]);

const loadMoreTarget = ref(null);
let observer = null;
let lastLoadMoreResultCount = 0;

// Virtual scroll enter-animation direction.
// Scrolling down → items enter from below (translateY positive).
// Scrolling up → items enter from above (translateY negative).
// Updated directly on the DOM (non-reactive) to avoid triggering
// Vue re-renders of the v-virtual-scroll slot on direction change.
const scrollRef = ref(null);
let lastScrollTop = 0;
let currentDirection = "down";

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

// allowConfirm=true（左键）→ 黄色状态交由父组件弹确认框；
// allowConfirm=false（右键）→ 高级用户直达，跳过确认。
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

watch(() => props.results.length, (next, previous) => {
    if (next < previous) {
        lastLoadMoreResultCount = 0;
    }
});
</script>

<style scoped>
.g-2 {
    gap: 8px;
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
</style>
