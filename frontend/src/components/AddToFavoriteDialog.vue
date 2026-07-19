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
                    v-model="selectedListName"
                    v-model:menu="isListMenuOpen"
                    :items="favoriteListNames"
                    :menu-props="listMenuProps"
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
                    :disabled="isSaving || !canCreateList || !selectedListName.trim() || drafts.length === 0"
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
const isListMenuOpen = ref(false);
const isSaving = ref(false);
const listMenuProps = {
    location: "top" as const,
    maxHeight: 200,
    offset: 4,
};
const favoriteListNames = computed(() => favoritesStore.lists.map((list) => list.name));
const canCreateList = computed(() => drafts.value.every((draft) =>
    Boolean(draft.minecraftVersion?.trim() && draft.modLoader?.trim())
));

const open = async (items: FavoriteModDraft[]) => {
    drafts.value = items;
    await favoritesStore.loadLists();
    selectedListName.value = favoritesStore.lists.find((list) => list.id === favoritesStore.selectedListId)?.name
        || favoritesStore.lists[0]?.name
        || "";
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
    selectedListName.value = "";
    isListMenuOpen.value = false;
};

const resolveListId = async () => {
    const name = selectedListName.value.trim();
    if (!name) return "";
    const existing = favoritesStore.lists.find((list) => list.name.trim().toLocaleLowerCase() === name.toLocaleLowerCase());
    if (existing) return existing.id;

    const draft = drafts.value[0];
    return (await favoritesStore.createList(name, draft?.minecraftVersion, draft?.modLoader))?.id || "";
};

const add = async () => {
    if (!canCreateList.value || !selectedListName.value.trim() || drafts.value.length === 0 || isSaving.value) return;
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
