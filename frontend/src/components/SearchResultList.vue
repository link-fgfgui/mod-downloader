<template>
    <v-list lines="two" bg-color="transparent" class="px-2">
        <v-list-item v-for="(result, index) in results" :key="result.id" :title="result.title" :subtitle="result.description"
            class="mb-2 border-b" bg-color="surface" rounded="xl" elevation="1">
            <template #prepend>
                <div class="align-self-start pt-1 me-3">
                    <v-avatar class="cursor-pointer" color="surface-container-high" rounded="lg" size="48"
                        @click="emit('show-versions', result)">
                        <v-img v-if="result.iconUrl" :src="result.iconUrl" :alt="result.title"></v-img>
                        <v-icon v-else :icon="result.icon" color="on-surface-variant"></v-icon>
                    </v-avatar>
                </div>
            </template>

            <template #append>
                <div class="d-flex align-center g-2">
                    <v-chip size="small" variant="flat" color="surface-container-highest" class="text-caption">
                        {{ result.platform }}
                    </v-chip>

                    <v-btn variant="tonal" rounded="xl" size="small"
                        :color="colorFor(index)"
                        :icon="iconFor(index)"
                        :loading="loadingFor(index)"
                        :disabled="disabledFor(index)"
                        @click="onInstall(index, true)"
                        @contextmenu.prevent="onInstall(index, false)"/>
                </div>
            </template>
        </v-list-item>

        <div ref="loadMoreTarget" class="load-more-target py-4 text-center">
            <v-progress-circular v-if="loadingMore" color="primary" indeterminate size="24"></v-progress-circular>
            <span v-else-if="results.length && !hasMore" class="text-caption text-medium-emphasis">{{ $t('search.noMoreResults') }}</span>
        </div>
    </v-list>
</template>

<script setup>
import { onMounted, onUnmounted, ref, watch } from "vue";

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

.load-more-target {
    min-height: 56px;
}

.cursor-pointer {
    cursor: pointer;
}
</style>
