<template>
    <v-container class="favorites-page pa-6 md-page" fluid>
        <aside class="favorites-rail">
            <div class="favorites-rail-header">
                <h1 class="text-h6 font-weight-medium">{{ $t("favorites.title") }}</h1>
                <v-btn icon="mdi-plus" size="small" variant="tonal" @click="openCreate"></v-btn>
            </div>

            <v-progress-linear v-if="favoritesStore.isLoadingLists" indeterminate class="mb-2"></v-progress-linear>

            <v-list v-if="favoritesStore.lists.length" class="favorites-list" density="compact" nav>
                <v-list-item
                    v-for="list in favoritesStore.lists"
                    :key="list.id"
                    :active="list.id === favoritesStore.selectedListId"
                    prepend-icon="mdi-playlist-star"
                    :title="list.name"
                    @click="favoritesStore.selectList(list.id)"
                >
                    <template #append>
                        <v-menu>
                            <template #activator="{ props }">
                                <v-btn
                                    v-bind="props"
                                    icon="mdi-dots-vertical"
                                    size="x-small"
                                    variant="text"
                                    @click.stop
                                ></v-btn>
                            </template>
                            <v-list density="compact">
                                <v-list-item prepend-icon="mdi-pencil" :title="$t('favorites.actions.rename')" @click="openRename(list)"></v-list-item>
                                <v-list-item prepend-icon="mdi-delete" :title="$t('favorites.actions.delete')" @click="openDelete(list)"></v-list-item>
                            </v-list>
                        </v-menu>
                    </template>
                </v-list-item>
            </v-list>

            <div v-else class="empty-rail text-body-2 text-medium-emphasis">
                {{ $t("favorites.empty.noLists") }}
            </div>
        </aside>

        <main class="favorites-main">
            <div v-if="selectedList" class="favorites-main-header">
                <div>
                    <h2 class="text-h5 font-weight-medium">{{ selectedList.name }}</h2>
                    <div class="text-body-2 text-medium-emphasis">
                        {{ $t("favorites.itemCount", { n: favoritesStore.items.length }) }}
                    </div>
                </div>
                <v-btn prepend-icon="mdi-refresh" variant="tonal" :loading="favoritesStore.isLoadingItems" @click="favoritesStore.loadItems()">
                    {{ $t("favorites.actions.refresh") }}
                </v-btn>
            </div>

            <div v-if="!selectedList" class="empty-state">
                <v-icon icon="mdi-playlist-plus" size="48"></v-icon>
                <div class="text-body-1 mt-3">{{ $t("favorites.empty.selectOrCreate") }}</div>
            </div>

            <div v-else-if="!favoritesStore.isLoadingItems && favoritesStore.items.length === 0" class="empty-state">
                <v-icon icon="mdi-star-outline" size="48"></v-icon>
                <div class="text-body-1 mt-3">{{ $t("favorites.empty.noItems") }}</div>
            </div>

            <VirtualList
                v-else
                class="favorites-items"
                :items="favoritesStore.items"
                :item-height="76"
                :item-key="itemKey"
            >
                <template #item="{ item, selected, onClick, enterStyle }">
                    <v-list-item
                        class="mb-2 border-b md-animate-fade-y md-hover-lift"
                        :class="{ 'favorite-item-selected': selected }"
                        :bg-color="selected ? undefined : 'surface'"
                        rounded="xl"
                        elevation="1"
                        lines="two"
                        :style="enterStyle"
                        @click="onClick"
                    >
                        <template #prepend>
                            <v-avatar color="surface-container-high" rounded="lg" size="48" class="me-3">
                                <v-img v-if="item.iconUrl" :src="item.iconUrl" :alt="displayName(item)"></v-img>
                                <v-icon v-else icon="mdi-package-variant" color="on-surface-variant"></v-icon>
                            </v-avatar>
                        </template>

                        <v-list-item-title class="font-weight-medium">{{ displayName(item) }}</v-list-item-title>
                        <v-list-item-subtitle class="text-caption text-medium-emphasis">
                            {{ item.platform }} / {{ item.modId }}
                            <span v-if="item.minecraftVersion"> · {{ item.minecraftVersion }}</span>
                            <span v-if="item.modLoader"> · {{ item.modLoader }}</span>
                        </v-list-item-subtitle>

                        <template #append>
                            <v-btn
                                icon="mdi-playlist-remove"
                                variant="tonal"
                                color="error"
                                size="small"
                                @click.stop="favoritesStore.remove(item)"
                            ></v-btn>
                        </template>
                    </v-list-item>
                </template>

                <template #actions="{ selectedItems, clearSelection }">
                    <v-btn
                        size="small"
                        variant="tonal"
                        color="error"
                        prepend-icon="mdi-playlist-remove"
                        @click="removeSelected(selectedItems, clearSelection)"
                    >
                        {{ $t("favorites.actions.removeSelected") }}
                    </v-btn>
                    <v-btn
                        size="small"
                        variant="tonal"
                        class="ms-1"
                        prepend-icon="mdi-selection-off"
                        @click="clearSelection()"
                    >
                        {{ $t("download.selection.deselectAll") }}
                    </v-btn>
                </template>
            </VirtualList>
        </main>

        <v-dialog v-model="editDialog.show" max-width="420">
            <v-card>
                <v-card-title>{{ editDialog.mode === "create" ? $t("favorites.dialog.createTitle") : $t("favorites.dialog.renameTitle") }}</v-card-title>
                <v-card-text>
                    <v-text-field v-model="editDialog.name" :label="$t('favorites.dialog.name')" autofocus @keyup.enter.prevent="saveList"></v-text-field>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="editDialog.show = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                    <v-btn color="primary" variant="flat" :disabled="!editDialog.name.trim()" @click="saveList">{{ $t("favorites.actions.save") }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="deleteDialog.show" max-width="420" @after-leave="clearClosedDeleteDialog">
            <v-card>
                <v-card-title>{{ $t("favorites.dialog.deleteTitle") }}</v-card-title>
                <v-card-text>{{ $t("favorites.dialog.deleteBody", { name: deleteDialog.list?.name || "" }) }}</v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="deleteDialog.show = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                    <v-btn color="error" variant="flat" @click="deleteList">{{ $t("favorites.actions.delete") }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>
    </v-container>
</template>

<script setup>
import { computed, onActivated, reactive } from "vue";
import VirtualList from "../components/VirtualList.vue";
import { useFavoritesStore } from "../stores/favorites";

const favoritesStore = useFavoritesStore();

const selectedList = computed(() => favoritesStore.selectedList);

const editDialog = reactive({
    show: false,
    mode: "create",
    id: "",
    name: "",
});
const deleteDialog = reactive({
    show: false,
    list: null,
});

const itemKey = (item) => favoritesStore.itemKey(item);
const displayName = (item) => item.title || item.slug || item.modId;

const openCreate = () => {
    editDialog.show = true;
    editDialog.mode = "create";
    editDialog.id = "";
    editDialog.name = "";
};

const openRename = (list) => {
    editDialog.show = true;
    editDialog.mode = "rename";
    editDialog.id = list.id;
    editDialog.name = list.name;
};

const openDelete = (list) => {
    deleteDialog.show = true;
    deleteDialog.list = list;
};

const saveList = async () => {
    const name = editDialog.name.trim();
    if (!name) return;
    if (editDialog.mode === "create") {
        await favoritesStore.createList(name);
    } else {
        await favoritesStore.renameList(editDialog.id, name);
    }
    editDialog.show = false;
};

const deleteList = async () => {
    if (!deleteDialog.list) return;
    await favoritesStore.deleteList(deleteDialog.list.id);
    deleteDialog.show = false;
};

const clearClosedDeleteDialog = () => {
    if (deleteDialog.show) return;
    deleteDialog.list = null;
};

const removeSelected = async (items, clearSelection) => {
    await favoritesStore.removeMany(items);
    clearSelection();
};

onActivated(() => {
    void favoritesStore.loadLists();
});
</script>

<style scoped>
.favorites-page {
    display: grid;
    gap: 20px;
    grid-template-columns: minmax(220px, 280px) minmax(0, 1fr);
    min-height: calc(100vh - 32px);
}

.favorites-rail,
.favorites-main {
    min-height: 0;
    min-width: 0;
}

.favorites-rail {
    border-inline-end: 1px solid rgba(var(--v-theme-outline), 0.24);
    display: flex;
    flex-direction: column;
    padding-inline-end: 16px;
}

.favorites-rail-header,
.favorites-main-header {
    align-items: center;
    display: flex;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 16px;
    min-width: 0;
}

.favorites-main-header {
    flex-wrap: wrap;
}

.favorites-main-header > div {
    flex: 1 1 220px;
    min-width: 0;
}

.favorites-main-header h2 {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.favorites-list :deep(.v-list-item-title),
.favorites-items :deep(.v-list-item-title),
.favorites-items :deep(.v-list-item-subtitle) {
    overflow: hidden;
    text-overflow: ellipsis;
}

.favorites-list {
    min-height: 0;
    overflow-y: auto;
}

.favorites-main {
    display: flex;
    flex-direction: column;
}

.favorites-items {
    flex: 1 1 auto;
    max-height: calc(100vh - 176px);
}

.empty-state,
.empty-rail {
    align-items: center;
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
    display: flex;
    flex-direction: column;
    justify-content: center;
}

.empty-state {
    flex: 1 1 auto;
    min-height: 320px;
}

.empty-rail {
    min-height: 220px;
}

.favorite-item-selected {
    background-color: rgba(var(--v-theme-primary), 0.12) !important;
}

@media (max-width: 820px) {
    .favorites-page {
        grid-template-columns: 1fr;
    }

    .favorites-rail {
        border-inline-end: 0;
        border-bottom: 1px solid rgba(var(--v-theme-outline), 0.24);
        max-height: 260px;
        padding-block-end: 16px;
        padding-inline-end: 0;
    }
}

@media (max-width: 599.98px) {
    .favorites-page {
        gap: 16px;
        padding: 16px !important;
    }

    .favorites-items :deep(.v-list-item__prepend) {
        display: none;
    }

    .favorites-items :deep(.v-list-item__append) {
        margin-inline-start: 8px;
    }
}
</style>
