<template>
    <v-dialog v-model="isOpen" max-width="460">
        <v-card>
            <v-card-title class="text-h6">{{ $t("favorites.addDialog.title") }}</v-card-title>
            <v-card-text>
                <v-combobox
                    v-model="selectedListName"
                    :items="favoriteListNames"
                    :label="$t('favorites.addDialog.list')"
                    density="comfortable"
                    variant="outlined"
                    :loading="favoritesStore.isLoadingLists"
                    hide-details
                ></v-combobox>

                <div class="text-caption text-medium-emphasis mt-3">
                    {{ $t("favorites.addDialog.count", { n: drafts.length }) }}
                </div>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <v-btn variant="text" @click="isOpen = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                <v-btn
                    color="primary"
                    variant="flat"
                    :loading="isSaving"
                    :disabled="!selectedListName.trim() || drafts.length === 0"
                    @click="add"
                >
                    {{ $t("favorites.actions.add") }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useFavoritesStore, type FavoriteModDraft } from "../stores/favorites";

const emit = defineEmits(["added"]);

const favoritesStore = useFavoritesStore();
const isOpen = ref(false);
const drafts = ref<FavoriteModDraft[]>([]);
const selectedListName = ref("");
const isSaving = ref(false);
const favoriteListNames = computed(() => favoritesStore.lists.map((list) => list.name));

const open = async (items: FavoriteModDraft[]) => {
    drafts.value = items;
    await favoritesStore.loadLists();
    selectedListName.value = favoritesStore.selectedList?.name || favoritesStore.lists[0]?.name || "";
    isOpen.value = true;
};

const resolveListId = async () => {
    const value = selectedListName.value.trim();
    if (!value) return "";
    const namedList = favoritesStore.lists.find((list) => list.name.trim().toLocaleLowerCase() === value.toLocaleLowerCase());
    if (namedList) return namedList.id;
    return (await favoritesStore.createList(value))?.id || "";
};

const add = async () => {
    if (!selectedListName.value.trim() || drafts.value.length === 0) return;
    isSaving.value = true;
    try {
        const listId = await resolveListId();
        if (!listId) return;
        const saved = await favoritesStore.addDrafts(listId, drafts.value);
        emit("added", saved);
        isOpen.value = false;
    } finally {
        isSaving.value = false;
    }
};

defineExpose({ open });
</script>
