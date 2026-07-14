<template>
    <v-dialog
        :model-value="isOpen"
        :persistent="isSaving"
        max-width="460"
        @update:model-value="onDialogModelValue"
        @after-leave="clearClosedDialog"
    >
        <v-card>
            <v-card-title class="text-h6">{{ $t("favorites.addDialog.title") }}</v-card-title>
            <v-card-text>
                <v-combobox
                    v-model="selectedChoice"
                    v-model:menu="isListMenuOpen"
                    :items="favoritesStore.lists"
                    :menu-props="listMenuProps"
                    item-title="name"
                    item-value="id"
                    return-object
                    :label="$t('favorites.addDialog.list')"
                    density="comfortable"
                    variant="outlined"
                    :loading="favoritesStore.isLoadingLists"
                    :disabled="isSaving"
                    hide-details
                ></v-combobox>

                <div class="text-caption text-medium-emphasis mt-3">
                    {{ $t("favorites.addDialog.count", { n: drafts.length }) }}
                </div>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <v-btn variant="text" :disabled="isSaving" @click="close">{{ $t("favorites.actions.cancel") }}</v-btn>
                <v-btn
                    color="primary"
                    variant="flat"
                    :loading="isSaving"
                    :disabled="isSaving || !selectedChoice || drafts.length === 0"
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
import type { storage } from "../../wailsjs/go/models";

const emit = defineEmits(["added"]);

const favoritesStore = useFavoritesStore();
const isOpen = ref(false);
const drafts = ref<FavoriteModDraft[]>([]);
const selectedChoice = ref<storage.FavoriteList | string | null>(null);
const isListMenuOpen = ref(false);
const isSaving = ref(false);
const listMenuProps = {
    location: "top" as const,
    maxHeight: 200,
    offset: 4,
};

const open = async (items: FavoriteModDraft[]) => {
    drafts.value = items;
    await favoritesStore.loadLists();
    selectedChoice.value = favoritesStore.lists.find((list) => list.id === favoritesStore.selectedListId)
        || favoritesStore.lists[0]
        || null;
    isOpen.value = true;
};

const close = () => {
    if (isSaving.value) return;
    isListMenuOpen.value = false;
    isOpen.value = false;
};

const onDialogModelValue = (value: boolean) => {
    if (!value) close();
};

const clearClosedDialog = () => {
    if (isOpen.value) return;
    drafts.value = [];
    selectedChoice.value = null;
    isListMenuOpen.value = false;
};

const resolveListId = async () => {
    const choice = selectedChoice.value;
    if (!choice) return "";
    if (typeof choice !== "string") return choice.id;

    const name = choice.trim();
    if (!name) return "";
    const existing = favoritesStore.lists.find((list) => list.name.trim().toLocaleLowerCase() === name.toLocaleLowerCase());
    if (existing) return existing.id;

    return (await favoritesStore.createList(name))?.id || "";
};

const add = async () => {
    if (!selectedChoice.value || drafts.value.length === 0 || isSaving.value) return;
    isSaving.value = true;
    try {
        const listId = await resolveListId();
        if (!listId) return;
        const saved = await favoritesStore.addDrafts(listId, drafts.value);
        emit("added", saved);
        isListMenuOpen.value = false;
        isOpen.value = false;
    } finally {
        isSaving.value = false;
    }
};

defineExpose({ open });
</script>
