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

                <div v-if="scopeLabel" class="d-flex align-center ga-1 mt-2 text-caption text-medium-emphasis">
                    <v-icon icon="mdi-tag-outline" size="14"></v-icon>
                    <span>{{ $t("favorites.addDialog.scopeLabel", { scope: scopeLabel }) }}</span>
                </div>

                <v-alert v-if="scopeError" class="mt-3" type="warning" variant="tonal" density="compact">
                    {{ $t("favorites.addDialog.scopeConflict") }}
                </v-alert>

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
import {
    listMatchesScope,
    normalizeKeyPart,
    normalizeMinecraftVersion,
    useFavoritesStore,
    type FavoriteModDraft,
} from "../stores/favorites";

const emit = defineEmits(["added"]);

const favoritesStore = useFavoritesStore();
const isOpen = ref(false);
const drafts = ref<FavoriteModDraft[]>([]);
const selectedListName = ref("");
const isListMenuOpen = ref(false);
const isSaving = ref(false);
const scopeError = ref(false);
const listMenuProps = {
    location: "top" as const,
    maxHeight: 200,
    offset: 4,
    transition: "md-menu-pop",
};

// All drafts share the same target scope (they come from one selection), so the
// first draft is authoritative for filtering and for the scope label.
const draftMinecraftVersion = computed(() => normalizeMinecraftVersion(drafts.value[0]?.minecraftVersion));
const draftModLoader = computed(() => normalizeKeyPart(drafts.value[0]?.modLoader));

const favoriteListNames = computed(() => favoritesStore.lists
    .filter((list) => listMatchesScope(list, draftMinecraftVersion.value, draftModLoader.value))
    .map((list) => list.name));

const scopeLabel = computed(() => {
    if (!draftMinecraftVersion.value || !draftModLoader.value) return "";
    return `${draftMinecraftVersion.value} / ${draftModLoader.value.toUpperCase()}`;
});

const canCreateList = computed(() => drafts.value.every((draft) =>
    Boolean(draft.minecraftVersion?.trim() && draft.modLoader?.trim())
));

const open = async (items: FavoriteModDraft[]) => {
    drafts.value = items;
    scopeError.value = false;
    await favoritesStore.loadLists();
    const inScope = favoritesStore.lists.filter((list) =>
        listMatchesScope(list, normalizeMinecraftVersion(items[0]?.minecraftVersion), normalizeKeyPart(items[0]?.modLoader)),
    );
    selectedListName.value = inScope.find((list) => list.id === favoritesStore.selectedListId)?.name
        || inScope[0]?.name
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
    scopeError.value = false;
};

const resolveListId = async () => {
    const name = selectedListName.value.trim();
    if (!name) return "";
    const existing = favoritesStore.lists.find((list) => list.name.trim().toLocaleLowerCase() === name.toLocaleLowerCase());
    if (existing) {
        // Never reuse a list that belongs to a different Minecraft/loader scope:
        // that would silently pollute the wrong environment.
        if (!listMatchesScope(existing, draftMinecraftVersion.value, draftModLoader.value)) {
            scopeError.value = true;
            return "";
        }
        return existing.id;
    }

    const draft = drafts.value[0];
    return (await favoritesStore.createList(name, draft?.minecraftVersion, draft?.modLoader))?.id || "";
};

const add = async () => {
    if (!canCreateList.value || !selectedListName.value.trim() || drafts.value.length === 0 || isSaving.value) return;
    isSaving.value = true;
    scopeError.value = false;
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
