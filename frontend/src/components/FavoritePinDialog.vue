<template>
    <v-dialog v-model="show" max-width="680" @after-leave="clearClosed">
        <v-card>
            <v-toolbar density="compact" color="surface">
                <v-toolbar-title>{{ displayName(item || {}) }}</v-toolbar-title>
                <v-btn icon="mdi-close" variant="text" @click="show = false"></v-btn>
            </v-toolbar>
            <v-divider></v-divider>
            <ModVersionList :versions="versions" :loading="loading" action="pin"
                :pinned-version-id="pinnedVersionId" :busy-version-id="busyVersionId"
                @select="pinVersion"></ModVersionList>
        </v-card>
    </v-dialog>
</template>

<script setup>
import { ref } from "vue";

import ModVersionList from "./ModVersionList.vue";
import { useFavoritesStore } from "../stores/favorites";
import { GetPinnedModVersion, ListMatchingProjectVersions, PinModVersion } from "../../wailsjs/go/main/App";

const favoritesStore = useFavoritesStore();
const show = ref(false);
const item = ref(null);
const versions = ref([]);
const loading = ref(false);
const pinnedVersionId = ref("");
const busyVersionId = ref("");

const displayName = (value) => value.title || value.slug || value.modId || "";

const projectFromFavorite = (favorite) => ({
    id: `${favorite.platform}:${favorite.modId}`,
    platform: favorite.platform,
    projectId: favorite.modId,
    slug: favorite.slug || "",
    title: displayName(favorite),
    icon: "mdi-package-variant",
    iconUrl: favorite.iconUrl || "",
    description: favorite.description || "",
    downloads: 0,
    categories: favorite.categories || [],
    updatedAt: 0,
    cachedAt: 0,
});

const open = async (favorite) => {
    if (!favorite?.platform || !favorite?.modId) return;
    show.value = true;
    item.value = favorite;
    versions.value = [];
    pinnedVersionId.value = "";
    loading.value = true;
    try {
        const [matched, pin] = await Promise.all([
            ListMatchingProjectVersions(projectFromFavorite(favorite), favoritesStore.displayMinecraftVersion, favoritesStore.displayModLoader),
            GetPinnedModVersion(favorite.platform, favorite.modId, favoritesStore.displayMinecraftVersion, favoritesStore.displayModLoader),
        ]);
        versions.value = matched || [];
        pinnedVersionId.value = pin?.versionId || "";
    } finally {
        loading.value = false;
    }
};

const pinVersion = async (version) => {
    if (!item.value || busyVersionId.value) return;
    busyVersionId.value = version.id;
    try {
        const pin = await PinModVersion({
            platform: item.value.platform,
            modId: item.value.modId,
            versionId: version.id,
            minecraftVersion: favoritesStore.displayMinecraftVersion,
            modLoader: favoritesStore.displayModLoader,
        });
        pinnedVersionId.value = pin?.versionId || "";
    } finally {
        busyVersionId.value = "";
    }
};

const clearClosed = () => {
    if (show.value) return;
    item.value = null;
    versions.value = [];
    pinnedVersionId.value = "";
    busyVersionId.value = "";
};

defineExpose({ open });
</script>
