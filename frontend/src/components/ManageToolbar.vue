<template>
    <div class="manage-header md-stagger">
        <div class="manage-header-copy">
            <h1 class="text-h5 font-weight-medium">{{ $t("manage.title") }}</h1>
            <div class="text-body-2 text-medium-emphasis">
                {{ title }}
            </div>
        </div>
        <v-btn class="md-btn-press md-hover-scale" color="primary" prepend-icon="mdi-refresh" :loading="isRefreshing" @click="$emit('refresh')">
            {{ $t("manage.refresh") }}
        </v-btn>
        <v-btn-toggle v-model="viewMode" class="manage-view-toggle" density="comfortable" variant="outlined"
            divided mandatory color="primary">
            <v-btn value="list" size="small" prepend-icon="mdi-format-list-bulleted">
                {{ $t("manage.viewMode.list") }}
            </v-btn>
            <v-btn value="tree" size="small" prepend-icon="mdi-file-tree">
                {{ $t("manage.viewMode.tree") }}
            </v-btn>
        </v-btn-toggle>
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

.manage-view-toggle {
    flex: 0 0 auto;
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
