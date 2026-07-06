<template>
    <v-dialog v-model="isOpen" max-width="460">
        <v-card>
            <v-card-title class="text-h6">{{ $t("favorites.addDialog.title") }}</v-card-title>
            <v-card-text>
                <v-select
                    v-model="selectedListId"
                    :items="favoritesStore.lists"
                    item-title="name"
                    item-value="id"
                    :label="$t('favorites.addDialog.list')"
                    density="comfortable"
                    variant="outlined"
                    :loading="favoritesStore.isLoadingLists"
                    hide-details
                ></v-select>

                <div class="add-list-row mt-4">
                    <v-text-field
                        v-model="newListName"
                        :label="$t('favorites.addDialog.newList')"
                        density="compact"
                        variant="outlined"
                        hide-details
                        @keyup.enter.prevent="createList"
                    ></v-text-field>
                    <v-btn
                        icon="mdi-plus"
                        variant="tonal"
                        :disabled="!newListName.trim()"
                        @click="createList"
                    ></v-btn>
                </div>

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
                    :disabled="!selectedListId || drafts.length === 0"
                    @click="add"
                >
                    {{ $t("favorites.actions.add") }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useFavoritesStore, type FavoriteModDraft } from "../stores/favorites";

const emit = defineEmits(["added"]);

const favoritesStore = useFavoritesStore();
const isOpen = ref(false);
const drafts = ref<FavoriteModDraft[]>([]);
const selectedListId = ref("");
const newListName = ref("");
const isSaving = ref(false);

const open = async (items: FavoriteModDraft[]) => {
    drafts.value = items;
    await favoritesStore.loadLists();
    selectedListId.value = favoritesStore.selectedListId || favoritesStore.lists[0]?.id || "";
    isOpen.value = true;
};

const createList = async () => {
    const name = newListName.value.trim();
    if (!name) return;
    const list = await favoritesStore.createList(name);
    if (list?.id) {
        selectedListId.value = list.id;
        newListName.value = "";
    }
};

const add = async () => {
    if (!selectedListId.value || drafts.value.length === 0) return;
    isSaving.value = true;
    try {
        const saved = await favoritesStore.addDrafts(selectedListId.value, drafts.value);
        emit("added", saved);
        isOpen.value = false;
    } finally {
        isSaving.value = false;
    }
};

defineExpose({ open });
</script>

<style scoped>
.add-list-row {
    align-items: center;
    display: grid;
    gap: 8px;
    grid-template-columns: minmax(0, 1fr) 40px;
}
</style>
