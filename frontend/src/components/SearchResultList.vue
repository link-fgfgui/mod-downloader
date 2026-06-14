<template>
    <v-virtual-scroll :items="virtualItems" class="search-result-scroll px-2" item-height="88">
        <template #default="{ item }">
            <v-list-item v-if="item.type === 'result'" :key="item.key" :title="item.result.title"
                :subtitle="item.result.description" class="mb-2 border-b" bg-color="surface" rounded="xl" elevation="1"
                lines="two">
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

                        <v-btn variant="tonal" rounded="xl" size="small"
                            :color="colorFor(item.index)"
                            :icon="iconFor(item.index)"
                            :loading="loadingFor(item.index)"
                            :disabled="disabledFor(item.index)"
                            @click="onInstall(item.index, true)"
                            @contextmenu.prevent="onInstall(item.index, false)"/>
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
});

onUnmounted(() => {
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
    height: calc(100vh - 236px);
    min-height: 320px;
}

.load-more-target {
    min-height: 56px;
}

.cursor-pointer {
    cursor: pointer;
}
</style>
