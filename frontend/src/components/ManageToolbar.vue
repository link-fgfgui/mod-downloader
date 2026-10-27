<template>
    <div class="manage-header">
        <div class="manage-header-copy">
            <h1 class="text-h5 font-weight-medium">{{ $t("manage.title") }}</h1>
            <div class="text-body-2 text-medium-emphasis">
                {{ title }}
            </div>
        </div>
        <v-btn class="md-btn-press md-hover-scale" color="primary" prepend-icon="mdi-refresh" :loading="isRefreshing" @click="$emit('refresh')">
            {{ $t("manage.refresh") }}
        </v-btn>
        <div ref="switchRef" class="manage-view-switch">
            <span class="manage-view-indicator" :style="indicatorStyle" aria-hidden="true"></span>
            <v-btn-toggle v-model="viewMode" class="manage-view-toggle" density="comfortable" variant="text"
                mandatory>
                <v-btn value="list" size="small" prepend-icon="mdi-format-list-bulleted">
                    {{ $t("manage.viewMode.list") }}
                </v-btn>
                <v-btn value="tree" size="small" prepend-icon="mdi-file-tree">
                    {{ $t("manage.viewMode.tree") }}
                </v-btn>
            </v-btn-toggle>
        </div>
    </div>

    <div v-if="hasSelectedInstance" class="manage-filters">
        <v-text-field v-model="searchInput" class="manage-search"
            prepend-inner-icon="mdi-magnify" :label="$t('manage.search.label')" clearable
            density="comfortable" hide-details />
        <v-tooltip :text="$t('manage.search.syntaxTip')" location="bottom" max-width="280">
            <template #activator="{ props: syntaxTip }">
                <v-btn
                    v-bind="syntaxTip"
                    class="manage-search-help"
                    icon="mdi-help-circle-outline"
                    variant="text"
                    size="small"
                    :aria-label="$t('manage.search.syntaxTip')"
                />
            </template>
        </v-tooltip>
        <v-select v-model="enabledFilter" class="manage-enabled-filter"
            :items="enabledFilterOptions" item-title="title" item-value="value"
            :label="$t('manage.filter.label')" density="comfortable" hide-details />
        <v-select v-if="viewMode === 'list'" v-model="sortMode" class="manage-sort" :items="sortOptions"
            item-title="title" item-value="value" item-props :loading="isScanningUnusedDependencies"
            :label="$t('manage.sort.label')" density="comfortable" hide-details />
    </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, onUpdated, ref } from "vue";

defineProps({
    title: { type: String, default: "" },
    isRefreshing: { type: Boolean, default: false },
    hasSelectedInstance: { type: Boolean, default: false },
    enabledFilterOptions: { type: Array, default: () => [] },
    sortOptions: { type: Array, default: () => [] },
    isScanningUnusedDependencies: { type: Boolean, default: false },
});

defineEmits(["refresh"]);

const searchInput = defineModel("searchInput", { type: String, default: "" });
const enabledFilter = defineModel("enabledFilter", { type: String, default: "all" });
const sortMode = defineModel("sortMode", { type: String, default: "default" });
const viewMode = defineModel("viewMode", { type: String, default: "list" });

const switchRef = ref(null);
const indicator = ref({ x: 0, width: 0, ready: false });
let resizeObserver = null;

const indicatorStyle = computed(() => ({
    opacity: indicator.value.ready ? 1 : 0,
    transform: `translate3d(${indicator.value.x}px, 0, 0)`,
    width: `${indicator.value.width}px`,
}));

const measureIndicator = () => {
    const buttons = switchRef.value?.querySelectorAll(".v-btn");
    if (!buttons || buttons.length < 2) return;
    const activeButton = buttons[viewMode.value === "tree" ? 1 : 0];
    const switchRect = switchRef.value.getBoundingClientRect();
    const buttonRect = activeButton.getBoundingClientRect();
    const next = {
        x: buttonRect.left - switchRect.left,
        width: buttonRect.width,
        ready: true,
    };
    const current = indicator.value;
    if (current.ready
        && Math.abs(current.x - next.x) < 0.5
        && Math.abs(current.width - next.width) < 0.5) {
        return;
    }
    indicator.value = next;
};

onMounted(() => {
    void nextTick(measureIndicator);
    resizeObserver = new ResizeObserver(() => measureIndicator());
    if (switchRef.value) {
        resizeObserver.observe(switchRef.value);
        switchRef.value.querySelectorAll(".v-btn").forEach((button) => resizeObserver.observe(button));
    }
});

onUpdated(measureIndicator);

onBeforeUnmount(() => {
    resizeObserver?.disconnect();
    resizeObserver = null;
});
</script>

<style scoped>
.manage-header {
    align-items: center;
    display: flex;
    flex-wrap: wrap;
    gap: 16px;
    justify-content: space-between;
    margin-bottom: 24px;
    flex: 0 0 auto;
}

.manage-header-copy {
    flex: 1 1 220px;
    min-width: 0;
}

.manage-filters {
    display: flex;
    flex: 0 0 auto;
    gap: 12px;
    margin-bottom: 16px;
}

.manage-search {
    flex: 1 1 320px;
    min-width: 0;
}

.manage-enabled-filter {
    flex: 0 0 180px;
}

.manage-sort {
    flex: 0 0 200px;
}

.manage-search-help {
    align-self: center;
    flex: 0 0 auto;
}

.manage-view-switch {
    align-items: stretch;
    background: rgba(var(--v-theme-surface-variant), 0.28);
    border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
    border-radius: 10px;
    display: inline-flex;
    flex: 0 0 auto;
    isolation: isolate;
    padding: 2px;
    position: relative;
}

.manage-view-indicator {
    background: rgb(var(--v-theme-primary));
    border-radius: 8px;
    bottom: 2px;
    left: 0;
    position: absolute;
    top: 2px;
    transition:
        opacity var(--md-transition-fast) var(--md-ease-out),
        transform var(--md-transition-slow) var(--md-ease-spring),
        width var(--md-transition-slow) var(--md-ease-spring);
    z-index: 0;
}

.manage-view-toggle {
    background: transparent;
    flex: 0 0 auto;
    position: relative;
    z-index: 1;
}

.manage-view-toggle :deep(.v-btn) {
    background: transparent !important;
    color: rgba(var(--v-theme-on-surface), 0.72);
    transition: color var(--md-transition-fast) var(--md-ease-out);
}

.manage-view-toggle :deep(.v-btn:not(.v-btn--active):hover) {
    background: rgba(var(--v-theme-on-surface), 0.06) !important;
    color: rgb(var(--v-theme-on-surface));
}

.manage-view-toggle :deep(.v-btn--active) {
    color: rgb(var(--v-theme-on-primary)) !important;
}

.manage-view-toggle :deep(.v-btn__overlay) {
    display: none;
}

@media (max-width: 600px) {
    .manage-filters {
        flex-wrap: wrap;
    }

    .manage-search,
    .manage-enabled-filter,
    .manage-sort {
        flex: 1 1 100%;
    }
}

@media (max-width: 599.98px) {
    .manage-header {
        align-items: flex-start;
    }

    .manage-header h1 {
        font-size: 1.25rem !important;
        line-height: 1.75rem;
    }

    .manage-header-copy {
        flex-basis: 100%;
    }
}
</style>
