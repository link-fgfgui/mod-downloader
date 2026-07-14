<template>
    <v-container class="favorites-page pa-6 md-page" fluid>
        <aside class="favorites-rail">
            <div class="favorites-rail-header">
                <h1 class="text-h6 font-weight-medium">{{ $t("favorites.title") }}</h1>
                <div class="rail-actions">
                    <v-btn icon="mdi-plus" size="small" variant="tonal" @click="openListEdit()"></v-btn>
                </div>
            </div>

            <v-progress-linear v-if="favoritesStore.isLoadingLists" indeterminate class="mb-2"></v-progress-linear>

            <div v-if="favoritesStore.lists.length" class="favorites-list">
                <section v-if="favoritesStore.pinnedLists.length" class="rail-section">
                    <div class="rail-section-title">
                        <v-icon icon="mdi-pin" size="14"></v-icon>
                        {{ $t("favorites.sections.pinned") }}
                    </div>
                    <v-list density="compact" nav>
                        <v-list-item
                            v-for="pinnedList in favoritesStore.pinnedLists"
                            :key="pinnedList.id"
                            :class="['favorite-list-row', { 'favorite-list-row--drop-before': dropTargetId === pinnedList.id }]"
                            :active="pinnedList.id === favoritesStore.selectedListId"
                            :title="pinnedList.name"
                            @click="favoritesStore.selectList(pinnedList.id)"
                            @contextmenu.prevent.stop="openListMenu(pinnedList, $event)"
                            @dragover.prevent="onListDragOver(pinnedList.id, $event)"
                            @dragleave="onListDragLeave(pinnedList.id)"
                            @drop="onListDrop(pinnedList.id, favoritesStore.pinnedLists, null, true, $event)"
                        >
                            <template #prepend>
                                <v-icon class="favorite-drag-handle me-2" draggable="true" icon="mdi-drag"
                                    size="16" @dragstart.stop="onListDragStart(pinnedList, $event)" @dragend="onListDragEnd"></v-icon>
                                <v-avatar class="list-avatar" rounded="lg" size="24">
                                    <v-img v-if="listIconUrl(pinnedList)" :src="listIconUrl(pinnedList)" :alt="pinnedList.name"></v-img>
                                    <v-icon v-else :icon="listIconName(pinnedList)" size="20"></v-icon>
                                </v-avatar>
                            </template>
                            <template #append>
                                <div class="row-actions">
                                    <v-icon icon="mdi-pin" size="14"></v-icon>
                                    <v-btn icon="mdi-dots-vertical" size="x-small" variant="text" @click.stop="openListMenu(pinnedList, $event)"></v-btn>
                                </div>
                            </template>
                        </v-list-item>
                    </v-list>
                </section>

                <section class="rail-section" @dragover.prevent="setMoveDropEffect" @drop="onListSectionDrop('', $event)">
                    <div class="rail-section-title">
                        <v-icon icon="mdi-playlist-star" size="14"></v-icon>
                        {{ $t("favorites.sections.lists") }}
                    </div>
                    <v-list density="compact" nav>
                        <v-list-item
                            v-for="looseList in favoritesStore.ungroupedLists"
                            :key="looseList.id"
                            :class="['favorite-list-row', { 'favorite-list-row--drop-before': dropTargetId === looseList.id }]"
                            :active="looseList.id === favoritesStore.selectedListId"
                            :title="looseList.name"
                            @click="favoritesStore.selectList(looseList.id)"
                            @contextmenu.prevent.stop="openListMenu(looseList, $event)"
                            @dragover.prevent="onListDragOver(looseList.id, $event)"
                            @dragleave="onListDragLeave(looseList.id)"
                            @drop.stop="onListDrop(looseList.id, favoritesStore.ungroupedLists, '', false, $event)"
                        >
                            <template #prepend>
                                <v-icon class="favorite-drag-handle me-2" draggable="true" icon="mdi-drag"
                                    size="16" @dragstart.stop="onListDragStart(looseList, $event)" @dragend="onListDragEnd"></v-icon>
                                <v-avatar class="list-avatar" rounded="lg" size="24">
                                    <v-img v-if="listIconUrl(looseList)" :src="listIconUrl(looseList)" :alt="looseList.name"></v-img>
                                    <v-icon v-else :icon="listIconName(looseList)" size="20"></v-icon>
                                </v-avatar>
                            </template>
                            <template #append>
                                <div class="row-actions">
                                    <v-btn icon="mdi-dots-vertical" size="x-small" variant="text" @click.stop="openListMenu(looseList, $event)"></v-btn>
                                </div>
                            </template>
                        </v-list-item>
                    </v-list>
                </section>

            </div>

            <v-menu v-model="listMenu.show" :target="listMenu.target">
                <v-list v-if="listMenu.list" density="compact">
                    <v-list-item prepend-icon="mdi-pencil" :title="$t('favorites.actions.rename')" @click="openListEdit(listMenu.list)"></v-list-item>
                    <v-list-item :prepend-icon="listMenu.list.pinned ? 'mdi-pin-off' : 'mdi-pin'"
                        :title="$t(listMenu.list.pinned ? 'favorites.actions.unpin' : 'favorites.actions.pin')"
                        @click="favoritesStore.setListPinned(listMenu.list, !listMenu.list.pinned)"></v-list-item>
                    <v-list-item prepend-icon="mdi-palette" :title="$t('favorites.actions.customize')" @click="openMetadata(listMenu.list)"></v-list-item>
                    <v-list-item prepend-icon="mdi-content-copy" :title="$t('favorites.actions.copyList')" @click="openListCopy(listMenu.list)"></v-list-item>
                    <v-list-item prepend-icon="mdi-view-list" :title="$t('favorites.actions.asSearchResults')" @click="openListAsSearchResults(listMenu.list)"></v-list-item>
                    <v-list-item prepend-icon="mdi-link-variant" :title="$t('favorites.actions.referenceList')" @click="openReference(listMenu.list)"></v-list-item>
                    <v-list-item prepend-icon="mdi-swap-horizontal" :title="$t('favorites.actions.migrate')" @click="openMigration(listMenu.list)"></v-list-item>
                    <v-list-item prepend-icon="mdi-delete" :title="$t('favorites.actions.delete')" @click="openDelete(listMenu.list)"></v-list-item>
                </v-list>
            </v-menu>

            <div v-if="!favoritesStore.lists.length" class="empty-rail text-body-2 text-medium-emphasis">
                {{ $t("favorites.empty.noLists") }}
            </div>
        </aside>

        <main class="favorites-main">
            <div v-if="selectedList" class="favorites-main-header">
                <div>
                    <div class="title-line">
                        <v-avatar class="list-avatar" rounded="lg" size="32">
                            <v-img v-if="listIconUrl(selectedList)" :src="listIconUrl(selectedList)" :alt="selectedList.name"></v-img>
                            <v-icon v-else :icon="listIconName(selectedList)" size="26"></v-icon>
                        </v-avatar>
                        <h2 class="text-h5 font-weight-medium">{{ selectedList.name }}</h2>
                    </div>
                    <div class="text-body-2 text-medium-emphasis">
                        {{ $t("favorites.itemCount", { n: favoritesStore.items.length }) }}
                        <span v-if="favoritesStore.contents?.refs?.length"> · {{ $t("favorites.references.count", { n: favoritesStore.contents.refs.length }) }}</span>
                    </div>
                </div>
                <div class="header-actions">
                    <v-btn
                        prepend-icon="mdi-archive-arrow-down"
                        variant="tonal"
                        :disabled="favoritesStore.items.length === 0"
                        :loading="favoritesStore.isExportingPackwiz"
                        @click="exportPackwiz"
                    >
                        {{ $t("favorites.actions.exportPackwiz") }}
                    </v-btn>
                    <v-btn prepend-icon="mdi-refresh" variant="tonal" :loading="favoritesStore.isLoadingItems" @click="favoritesStore.loadItems()">
                        {{ $t("favorites.actions.refresh") }}
                    </v-btn>
                    <v-btn prepend-icon="mdi-swap-horizontal" variant="tonal" @click="openMigration(selectedList)">
                        {{ $t("favorites.actions.migrate") }}
                    </v-btn>
                </div>
            </div>

            <div v-if="!selectedList" class="empty-state">
                <v-icon icon="mdi-playlist-plus" size="48"></v-icon>
                <div class="text-body-1 mt-3">{{ $t("favorites.empty.selectOrCreate") }}</div>
            </div>

            <div v-else-if="!favoritesStore.isLoadingItems && favoritesStore.items.length === 0" class="empty-state">
                <v-icon icon="mdi-star-outline" size="48"></v-icon>
                <div class="text-body-1 mt-3">{{ $t("favorites.empty.noItems") }}</div>
            </div>

            <VirtualList v-else class="favorites-items" :items="favoritesStore.items" :item-height="82" :item-key="itemKey">
                <template #item="{ item, selected, onClick }">
                    <v-list-item
                        class="favorite-mod-row mb-2 border-b md-hover-lift"
                        :class="{ 'favorite-item-selected': selected, 'favorite-item-referenced': item.referenced }"
                        :bg-color="selected ? undefined : 'surface'"
                        rounded="lg"
                        elevation="1"
                        lines="two"
                        @click="onClick"
                    >
                        <template #prepend>
                            <v-avatar :class="['me-3', { 'favorite-mod-icon--actionable': canPinItem(item) }]"
                                color="surface-container-high" rounded="lg" size="48" @click.stop="openPinDialog(item)">
                                <v-img v-if="item.iconUrl" :src="item.iconUrl" :alt="displayName(item)"></v-img>
                                <v-icon v-else icon="mdi-package-variant" color="on-surface-variant"></v-icon>
                            </v-avatar>
                        </template>

                        <v-list-item-title class="font-weight-medium">
                            {{ displayName(item) }}
                            <v-chip v-if="item.referenced" class="ms-2" size="x-small" variant="tonal">{{ item.sourceListName }}</v-chip>
                        </v-list-item-title>
                        <v-list-item-subtitle class="text-caption text-medium-emphasis">
                            {{ item.platform }} / {{ item.modId }}
                        </v-list-item-subtitle>

                        <template #append>
                            <v-btn
                                v-if="!item.referenced"
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
                    <v-tooltip :text="$t('favorites.actions.copySelected')" location="top">
                        <template #activator="{ props: tip }">
                            <v-btn v-bind="tip" :aria-label="$t('favorites.actions.copySelected')" icon="mdi-content-copy"
                                size="small" variant="tonal" @click="openSelectedCopy(selectedItems, clearSelection)"></v-btn>
                        </template>
                    </v-tooltip>
                    <v-tooltip :text="$t('favorites.actions.removeSelected')" location="top">
                        <template #activator="{ props: tip }">
                            <v-btn v-bind="tip" :aria-label="$t('favorites.actions.removeSelected')" icon="mdi-playlist-remove"
                                size="small" variant="tonal" color="error" @click="removeSelected(selectedItems, clearSelection)"></v-btn>
                        </template>
                    </v-tooltip>
                    <v-tooltip :text="$t('download.selection.deselectAll')" location="top">
                        <template #activator="{ props: tip }">
                            <v-btn v-bind="tip" :aria-label="$t('download.selection.deselectAll')" icon="mdi-selection-off"
                                size="small" variant="tonal" @click="clearSelection()"></v-btn>
                        </template>
                    </v-tooltip>
                </template>
            </VirtualList>
        </main>

        <v-dialog v-model="listEdit.show" max-width="420">
            <v-card>
                <v-card-title>{{ listEdit.id ? $t("favorites.dialog.renameTitle") : $t("favorites.dialog.createTitle") }}</v-card-title>
                <v-card-text>
                    <v-text-field v-model="listEdit.name" :label="$t('favorites.dialog.name')" autofocus @keyup.enter.prevent="saveList"></v-text-field>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="listEdit.show = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                    <v-btn color="primary" variant="flat" :disabled="!listEdit.name.trim()" @click="saveList">{{ $t("favorites.actions.save") }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="metadataDialog.show" max-width="520">
            <v-card>
                <v-card-title>{{ $t("favorites.dialog.customizeTitle") }}</v-card-title>
                <v-card-text>
                    <v-switch v-model="metadataDialog.pinned" color="primary" inset density="comfortable" hide-details :label="$t('favorites.actions.pin')"></v-switch>
                    <v-select v-model="metadataDialog.iconKind" :items="iconModeOptions" item-title="title" item-value="value" :label="$t('favorites.dialog.iconMode')"></v-select>
                    <v-text-field v-model="metadataDialog.iconValue" :label="metadataDialog.iconKind === 'mdi' ? $t('favorites.dialog.mdiIcon') : $t('favorites.dialog.projectSlug')"></v-text-field>
                    <v-select v-if="metadataDialog.iconKind === 'project'" v-model="metadataDialog.iconPlatform" :items="platformOptions" :label="$t('favorites.dialog.platform')"></v-select>
                </v-card-text>
                <v-card-actions>
                    <v-btn variant="text" @click="clearIcon">{{ $t("favorites.actions.clearIcon") }}</v-btn>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="metadataDialog.show = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                    <v-btn color="primary" variant="flat" @click="saveMetadata">{{ $t("favorites.actions.save") }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="copyDialog.show" max-width="520">
            <v-card>
                <v-card-title>{{ copyDialogTitle }}</v-card-title>
                <v-card-text>
                    <v-select
                        v-if="copyDialog.mode === 'selected'"
                        v-model="copyDialog.targetListIds"
                        :items="targetListOptions"
                        item-title="title"
                        item-value="value"
                        chips
                        multiple
                        :label="$t('favorites.dialog.targetLists')"
                    ></v-select>
                    <v-select
                        v-else
                        v-model="copyDialog.targetListId"
                        :items="targetListOptions"
                        item-title="title"
                        item-value="value"
                        :label="$t('favorites.dialog.targetList')"
                    ></v-select>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="copyDialog.show = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                    <v-btn color="primary" variant="flat" :disabled="!copyDialogReady" @click="applyCopyDialog">{{ $t("favorites.actions.apply") }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-dialog v-model="migrationDialog.show" max-width="760">
            <v-card>
                <v-card-title>{{ $t("favorites.dialog.migrationTitle") }}</v-card-title>
                <v-card-text>
                    <div class="migration-grid">
                        <MinecraftTargetFields
                            v-model:minecraft-version="migrationDialog.minecraftVersion"
                            v-model:mod-loader="migrationDialog.modLoader"
                            :versions="minecraftStore.releaseVersions"
                            :mod-loaders="minecraftStore.modLoaderList"
                            :minecraft-version-label="$t('favorites.dialog.minecraftVersion')"
                            :mod-loader-label="$t('favorites.dialog.modLoader')"
                        ></MinecraftTargetFields>
                    </div>
                    <v-checkbox v-model="migrationDialog.ignoreConflicts" :label="$t('favorites.dialog.ignoreConflicts')"></v-checkbox>
                    <v-btn variant="tonal" prepend-icon="mdi-eye" :disabled="!migrationReady" @click="previewMigration">{{ $t("favorites.actions.preview") }}</v-btn>

                    <div v-if="migrationDialog.preview" class="migration-preview mt-4">
                        <v-chip size="small" color="success" variant="tonal">{{ $t("favorites.migration.matched", { n: migrationDialog.preview.matched?.length || 0 }) }}</v-chip>
                        <v-chip size="small" color="warning" variant="tonal">{{ $t("favorites.migration.conflicts", { n: migrationDialog.preview.conflicts?.length || 0 }) }}</v-chip>
                        <v-list density="compact" class="mt-2">
                            <v-list-item v-for="match in migrationDialog.preview.matched || []" :key="`m-${match.source.id}`" prepend-icon="mdi-check" :title="displayName(match.source)" :subtitle="match.version.versionId || match.version.id"></v-list-item>
                            <v-list-item v-for="conflict in migrationDialog.preview.conflicts || []" :key="`c-${conflict.source.id}`" prepend-icon="mdi-alert" :title="displayName(conflict.source)" :subtitle="conflict.reason"></v-list-item>
                        </v-list>
                    </div>
                </v-card-text>
                <v-card-actions>
                    <v-spacer></v-spacer>
                    <v-btn variant="text" @click="migrationDialog.show = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                    <v-btn color="primary" variant="flat" :disabled="!migrationApplyReady" @click="applyMigration">{{ $t("favorites.actions.apply") }}</v-btn>
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

        <v-dialog v-model="pinDialog.show" max-width="680" @after-leave="clearClosedPinDialog">
            <v-card class="version-dialog">
                <v-toolbar density="compact" color="surface">
                    <v-toolbar-title>{{ displayName(pinDialog.item || {}) }}</v-toolbar-title>
                    <v-btn icon="mdi-close" variant="text" @click="pinDialog.show = false"></v-btn>
                </v-toolbar>
                <v-divider></v-divider>
                <ModVersionList :versions="pinDialog.versions" :loading="pinDialog.loading" action="pin"
                    :pinned-version-id="pinDialog.pinnedVersionId" :busy-version-id="pinDialog.busyVersionId"
                    @select="pinFavoriteVersion"></ModVersionList>
            </v-card>
        </v-dialog>

        <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="3500">
            {{ snackbar.message }}
        </v-snackbar>
    </v-container>
</template>

<script setup>
import { computed, onActivated, reactive, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import MinecraftTargetFields from "../components/MinecraftTargetFields.vue";
import ModVersionList from "../components/ModVersionList.vue";
import VirtualList from "../components/VirtualList.vue";
import { useFavoritesStore } from "../stores/favorites";
import { useDownloadSearchStore } from "../stores/downloadSearch";
import { useMinecraftStore } from "../stores/minecraft";
import { GetPinnedModVersion, ListMatchingProjectVersions, PinModVersion } from "../../wailsjs/go/main/App";

const favoritesStore = useFavoritesStore();
const downloadStore = useDownloadSearchStore();
const minecraftStore = useMinecraftStore();
const router = useRouter();
const { t } = useI18n();
const selectedList = computed(() => favoritesStore.selectedList);
const draggedListId = ref("");
const dropTargetId = ref("");
const pendingClearSelection = ref(null);

const listEdit = reactive({ show: false, id: "", name: "" });
const listMenu = reactive({ show: false, list: null, target: null });
const deleteDialog = reactive({ show: false, list: null });
const metadataDialog = reactive({ show: false, list: null, pinned: false, iconKind: "mdi", iconValue: "", iconPlatform: "modrinth" });
const copyDialog = reactive({ show: false, mode: "selected", sourceListId: "", targetListId: "", targetListIds: [], mods: [] });
const migrationDialog = reactive({ show: false, sourceListId: "", targetListId: "", minecraftVersion: "", modLoader: "", ignoreConflicts: false, preview: null });
const pinDialog = reactive({ show: false, item: null, versions: [], loading: false, pinnedVersionId: "", busyVersionId: "" });

const targetListOptions = computed(() => favoritesStore.lists.map((list) => ({ title: list.name, value: list.id })));
const iconModeOptions = computed(() => [
    { title: "MDI", value: "mdi" },
    { title: t("favorites.dialog.projectIcon"), value: "project" },
]);
const platformOptions = ["modrinth", "curseforge"];
const copyDialogTitle = computed(() => {
    if (copyDialog.mode === "reference") return t("favorites.dialog.referenceTitle");
    if (copyDialog.mode === "list") return t("favorites.dialog.copyListTitle");
    return t("favorites.dialog.copySelectedTitle");
});
const copyDialogReady = computed(() => copyDialog.mode === "selected" ? copyDialog.targetListIds.length > 0 : Boolean(copyDialog.targetListId));
const migrationReady = computed(() => {
    const minecraftVersion = migrationDialog.minecraftVersion.trim();
    const modLoader = migrationDialog.modLoader.trim().toLocaleLowerCase();
    const currentMinecraftVersion = minecraftStore.selectedMinecraftVersion.trim();
    const currentModLoader = minecraftStore.selectedModLoader.trim().toLocaleLowerCase();
    return Boolean(
        migrationDialog.sourceListId &&
        migrationDialog.targetListId &&
        minecraftVersion &&
        modLoader &&
        (minecraftVersion !== currentMinecraftVersion || modLoader !== currentModLoader)
    );
});
const migrationApplyReady = computed(() => {
    const preview = migrationDialog.preview;
    if (!migrationReady.value || !preview) return false;
    return migrationDialog.ignoreConflicts || (preview.conflicts || []).length === 0;
});
const snackbar = ref({ show: false, message: "", color: "success" });

const itemKey = (item) => favoritesStore.itemKey(item);
const displayName = (item) => item.title || item.slug || item.modId;
const canPinItem = (item) => Boolean(item?.platform && item?.modId);
const projectFromFavorite = (item) => ({
    id: `${item.platform}:${item.modId}`,
    platform: item.platform,
    projectId: item.modId,
    slug: item.slug || "",
    title: displayName(item),
    icon: "mdi-package-variant",
    iconUrl: item.iconUrl || "",
    description: item.description || "",
    downloads: 0,
    categories: item.categories || [],
    updatedAt: 0,
    cachedAt: 0,
});
const listIconUrl = (list) => (list?.iconKind === "project" ? list.iconUrl || "" : "");
const listIconName = (list) => (list?.iconKind === "mdi" && list.iconValue ? list.iconValue : "mdi-playlist-star");
const showMessage = (message, color = "success") => {
    snackbar.value = { show: true, message, color };
};
const openListMenu = (list, event) => {
    listMenu.show = false;
    listMenu.list = list;
    listMenu.target = event.type === "contextmenu" ? [event.clientX, event.clientY] : event.currentTarget;
    listMenu.show = true;
};
const openListAsSearchResults = async (list) => {
    if (!list?.id) return;
    await favoritesStore.selectList(list.id);
    await downloadStore.showResults((favoritesStore.items || []).map(projectFromFavorite));
    listMenu.show = false;
    await router.push({ name: "Download" });
};
const syncFavoriteDisplayScope = () => {
    favoritesStore.setDisplayScope(
        minecraftStore.selectedMinecraftVersion,
        minecraftStore.selectedModLoader,
    );
};

const openListEdit = (list = null) => {
    listEdit.show = true;
    listEdit.id = list?.id || "";
    listEdit.name = list?.name || "";
};
const saveList = async () => {
    const name = listEdit.name.trim();
    if (!name) return;
    if (listEdit.id) await favoritesStore.renameList(listEdit.id, name);
    else await favoritesStore.createList(name);
    listEdit.show = false;
};
const openDelete = (list) => {
    deleteDialog.show = true;
    deleteDialog.list = list;
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
const openPinDialog = async (item) => {
    if (!canPinItem(item)) return;
    pinDialog.show = true;
    pinDialog.item = item;
    pinDialog.versions = [];
    pinDialog.pinnedVersionId = "";
    pinDialog.loading = true;
    try {
        const [versions, pin] = await Promise.all([
            ListMatchingProjectVersions(projectFromFavorite(item), favoritesStore.displayMinecraftVersion, favoritesStore.displayModLoader),
            GetPinnedModVersion(item.platform, item.modId, favoritesStore.displayMinecraftVersion, favoritesStore.displayModLoader),
        ]);
        pinDialog.versions = versions || [];
        pinDialog.pinnedVersionId = pin?.versionId || "";
    } finally {
        pinDialog.loading = false;
    }
};
const pinFavoriteVersion = async (version) => {
    const item = pinDialog.item;
    if (!item || pinDialog.busyVersionId) return;
    pinDialog.busyVersionId = version.id;
    try {
        const pin = await PinModVersion({
            platform: item.platform,
            modId: item.modId,
            versionId: version.id,
            minecraftVersion: favoritesStore.displayMinecraftVersion,
            modLoader: favoritesStore.displayModLoader,
        });
        pinDialog.pinnedVersionId = pin?.versionId || "";
    } finally {
        pinDialog.busyVersionId = "";
    }
};
const clearClosedPinDialog = () => {
    if (pinDialog.show) return;
    pinDialog.item = null;
    pinDialog.versions = [];
    pinDialog.pinnedVersionId = "";
};
const openMetadata = (list) => {
    metadataDialog.show = true;
    metadataDialog.list = list;
    metadataDialog.pinned = Boolean(list.pinned);
    metadataDialog.iconKind = list.iconKind || "mdi";
    metadataDialog.iconValue = list.iconValue || "";
    metadataDialog.iconPlatform = "modrinth";
};
const saveMetadata = async () => {
    if (!metadataDialog.list) return;
    const list = await favoritesStore.updateListMetadata(metadataDialog.list, {
        pinned: metadataDialog.pinned,
    });
    if (list) await favoritesStore.updateListIcon(list, metadataDialog.iconKind, metadataDialog.iconValue, metadataDialog.iconPlatform);
    metadataDialog.show = false;
};
const clearIcon = async () => {
    if (!metadataDialog.list) return;
    await favoritesStore.clearListIcon(metadataDialog.list);
    metadataDialog.iconValue = "";
};
const openSelectedCopy = (items, clearSelection) => {
    copyDialog.show = true;
    copyDialog.mode = "selected";
    copyDialog.mods = items.filter((item) => !item.referenced);
    copyDialog.targetListIds = [];
    copyDialog.targetListId = "";
    pendingClearSelection.value = clearSelection;
};
const openListCopy = (list) => {
    copyDialog.show = true;
    copyDialog.mode = "list";
    copyDialog.sourceListId = list.id;
    copyDialog.targetListId = "";
    copyDialog.targetListIds = [];
    copyDialog.mods = [];
};
const openReference = (list) => {
    copyDialog.show = true;
    copyDialog.mode = "reference";
    copyDialog.sourceListId = list.id;
    copyDialog.targetListId = "";
    copyDialog.targetListIds = [];
    copyDialog.mods = [];
};
const applyCopyDialog = async () => {
    if (copyDialog.mode === "selected") {
        const result = await favoritesStore.copySelectedMods(copyDialog.targetListIds, copyDialog.mods);
        pendingClearSelection.value?.();
        showMessage(result ? t("favorites.resultSummary", { added: result.added, updated: result.updated, skipped: result.skipped }) : "");
    } else if (copyDialog.mode === "list") {
        const result = await favoritesStore.copyList(copyDialog.sourceListId, copyDialog.targetListId);
        showMessage(result ? t("favorites.resultSummary", { added: result.added, updated: result.updated, skipped: result.skipped }) : "");
    } else {
        const ref = await favoritesStore.addListReference(copyDialog.targetListId, copyDialog.sourceListId);
        showMessage(ref?.id ? t("favorites.references.added") : t("favorites.references.skipped"));
    }
    copyDialog.show = false;
};
const openMigration = (list) => {
    migrationDialog.show = true;
    migrationDialog.sourceListId = list.id;
    migrationDialog.targetListId = list.id;
    migrationDialog.minecraftVersion = minecraftStore.selectedMinecraftVersion;
    migrationDialog.modLoader = minecraftStore.selectedModLoader;
    migrationDialog.ignoreConflicts = false;
    migrationDialog.preview = null;
};
const previewMigration = async () => {
    migrationDialog.preview = await favoritesStore.previewMigration(migrationDialog);
};
const applyMigration = async () => {
    const result = await favoritesStore.applyMigration(migrationDialog);
    migrationDialog.preview = result?.preview || migrationDialog.preview;
    if (result?.applied) migrationDialog.show = false;
    showMessage(result?.applied ? t("favorites.migration.applied") : t("favorites.migration.notApplied"));
};
const removeSelected = async (items, clearSelection) => {
    await favoritesStore.removeMany(items.filter((item) => !item.referenced));
    clearSelection();
};
const onListDragStart = (list, event) => {
    draggedListId.value = list.id;
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", list.id);
    const preview = document.createElement("div");
    preview.className = "favorite-drag-preview";
    preview.textContent = list.name;
    document.body.appendChild(preview);
    event.dataTransfer.setDragImage(preview, 16, 16);
    requestAnimationFrame(() => preview.remove());
};
const onListDragEnd = () => {
    draggedListId.value = "";
    dropTargetId.value = "";
};
const setMoveDropEffect = (event) => {
    if (event.dataTransfer) event.dataTransfer.dropEffect = "move";
};
const onListDragOver = (listId, event) => {
    setMoveDropEffect(event);
    if (draggedListId.value && draggedListId.value !== listId) dropTargetId.value = listId;
};
const onListDragLeave = (listId) => {
    if (dropTargetId.value === listId) dropTargetId.value = "";
};
const moveListToSection = async (groupId, pinned = false) => {
    const sourceListId = draggedListId.value;
    draggedListId.value = "";
    const sourceList = favoritesStore.lists.find((list) => list.id === sourceListId);
    if (!sourceList) return null;
    const targetGroupId = groupId === null ? sourceList.groupId || "" : groupId;
    if (Boolean(sourceList.pinned) === pinned && (sourceList.groupId || "") === targetGroupId) return sourceList;
    return favoritesStore.updateListMetadata(sourceList, { groupId: targetGroupId, pinned });
};
const onListSectionDrop = async (groupId, event) => {
    setMoveDropEffect(event);
    await moveListToSection(groupId);
    dropTargetId.value = "";
};
const onListDrop = async (targetListId, sectionLists, groupId, pinned = false, event) => {
    setMoveDropEffect(event);
    dropTargetId.value = "";
    const sourceListId = draggedListId.value;
    if (!sourceListId || sourceListId === targetListId) {
        draggedListId.value = "";
        return;
    }
    const sourceList = await moveListToSection(groupId, pinned);
    if (!sourceList) return;
    const ids = sectionLists.map((list) => list.id).filter((id) => id !== sourceListId);
    const targetIndex = ids.indexOf(targetListId);
    if (targetIndex < 0) return;
    ids.splice(targetIndex, 0, sourceListId);
    await favoritesStore.reorderLists(ids);
};

const exportPackwiz = async () => {
    if (!selectedList.value || favoritesStore.items.length === 0) return;
    try {
        const result = await favoritesStore.exportPackwiz(selectedList.value.id);
        if (!result || result.canceled) return;
        snackbar.value = { show: true, message: t("favorites.export.success"), color: "success" };
    } catch (error) {
        snackbar.value = {
            show: true,
            message: t("favorites.export.failed", { reason: errorMessage(error) }),
            color: "error",
        };
    }
};

const errorMessage = (error) => {
    if (!error) return t("favorites.export.unknownError");
    if (typeof error === "string") return error;
    return error.message || String(error);
};

watch(
    () => [migrationDialog.minecraftVersion, migrationDialog.modLoader],
    () => {
        migrationDialog.preview = null;
    },
);

watch(
    () => [minecraftStore.selectedMinecraftVersion, minecraftStore.selectedModLoader],
    ([minecraftVersion, modLoader]) => {
        favoritesStore.setDisplayScope(String(minecraftVersion || ""), String(modLoader || ""));
    },
);

onActivated(() => {
    syncFavoriteDisplayScope();
    void favoritesStore.loadLists();
});
</script>

<style scoped>
.favorites-page {
    display: grid;
    gap: 20px;
    grid-template-columns: minmax(240px, 310px) minmax(0, 1fr);
    height: calc(100vh - 32px);
    min-height: 0;
    overflow: hidden;
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
.favorites-main-header,
.title-line,
.header-actions,
.rail-actions,
.rail-section-title,
.row-actions {
    align-items: center;
    display: flex;
    gap: 8px;
    min-width: 0;
}

.favorites-rail-header,
.favorites-main-header {
    justify-content: space-between;
    margin-bottom: 16px;
}

.favorites-main-header {
    flex-wrap: wrap;
}

.favorites-main-header > div:first-child {
    flex: 1 1 260px;
    min-width: 0;
}

.favorites-main-header h2,
.rail-section-title span,
.favorites-list :deep(.v-list-item-title),
.favorites-items :deep(.v-list-item-title),
.favorites-items :deep(.v-list-item-subtitle) {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.favorites-list {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
}

.rail-section {
    margin-bottom: 10px;
}

.rail-section-title {
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
    font-size: 0.76rem;
    font-weight: 700;
    justify-content: space-between;
    min-height: 28px;
    padding-inline: 6px 2px;
    text-transform: uppercase;
}

.favorite-list-row {
    border-radius: 8px !important;
    position: relative;
}

.favorite-list-row--drop-before::before {
    background: rgb(var(--v-theme-primary));
    border-radius: 2px;
    content: "";
    height: 3px;
    inset: -2px 8px auto;
    position: absolute;
    z-index: 2;
}

.favorite-drag-handle {
    cursor: grab;
}

.favorite-drag-handle:active {
    cursor: grabbing;
}

:global(.favorite-drag-preview) {
    background: rgb(var(--v-theme-surface));
    border: 1px solid rgb(var(--v-theme-primary));
    border-radius: 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.24);
    color: rgb(var(--v-theme-on-surface));
    max-width: 240px;
    overflow: hidden;
    padding: 8px 12px;
    pointer-events: none;
    position: fixed;
    text-overflow: ellipsis;
    white-space: nowrap;
    z-index: 9999;
}

.favorites-main {
    display: flex;
    flex-direction: column;
}

.favorites-items {
    flex: 1 1 auto;
    min-height: 0;
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

.favorite-item-referenced {
    border-inline-start: 3px solid rgb(var(--v-theme-tertiary));
}

.favorite-mod-icon--actionable {
    cursor: pointer;
}

.migration-grid {
    min-width: 0;
}

@media (max-width: 900px) {
    .favorites-page {
        grid-template-columns: 1fr;
    }

    .favorites-rail {
        border-inline-end: 0;
        border-bottom: 1px solid rgba(var(--v-theme-outline), 0.24);
        max-height: 320px;
        padding-block-end: 16px;
        padding-inline-end: 0;
    }
}

@media (max-width: 599.98px) {
    .favorites-page {
        gap: 16px;
        padding: 16px !important;
    }

    .header-actions,
    .migration-grid {
        width: 100%;
    }

    .header-actions {
        flex-wrap: wrap;
    }

    .migration-grid {
        grid-template-columns: 1fr;
    }

    .favorites-items :deep(.v-list-item__prepend) {
        display: none;
    }
}
</style>
