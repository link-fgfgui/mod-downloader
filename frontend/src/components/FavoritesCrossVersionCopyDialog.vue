<template>
    <v-dialog
        v-model="dialog.show"
        max-width="760"
        :persistent="dialog.isApplying"
        @after-leave="clearClosedDialog"
    >
        <v-card>
            <v-card-title>{{ $t("favorites.dialog.crossVersionCopyTitle") }}</v-card-title>
            <v-card-text>
                <div class="cross-version-copy-grid">
                    <MinecraftTargetFields
                        v-model:minecraft-version="dialog.minecraftVersion"
                        v-model:mod-loader="dialog.modLoader"
                        :versions="minecraftStore.releaseVersions"
                        :mod-loaders="minecraftStore.modLoaderList"
                        :minecraft-version-label="$t('favorites.dialog.minecraftVersion')"
                        :mod-loader-label="$t('favorites.dialog.modLoader')"
                        :disabled="dialog.isApplying"
                    ></MinecraftTargetFields>
                </div>
                <v-alert class="mt-3" type="info" variant="tonal" density="compact">
                    {{ $t("favorites.crossVersionCopy.target", {
                        name: dialog.sourceListName,
                        minecraftVersion: dialog.minecraftVersion,
                        modLoader: dialog.modLoader,
                    }) }}
                </v-alert>
                <v-checkbox
                    v-model="dialog.ignoreConflicts"
                    :label="$t('favorites.dialog.ignoreConflicts')"
                    :disabled="dialog.isApplying"
                ></v-checkbox>
                <v-btn
                    variant="tonal"
                    prepend-icon="mdi-eye"
                    :disabled="!copyReady || dialog.isApplying"
                    :loading="dialog.isPreviewing"
                    @click="previewCopy"
                >{{ $t("favorites.actions.preview") }}</v-btn>

                <div v-if="dialog.preview" class="cross-version-copy-preview mt-4">
                    <v-alert v-if="dialog.preview.nameConflict" class="mb-3" type="error" variant="tonal" density="compact">
                        {{ $t("favorites.crossVersionCopy.nameConflict") }}
                    </v-alert>
                    <v-alert
                        v-else-if="dialog.preview.errors?.length"
                        class="mb-3"
                        type="error"
                        variant="tonal"
                        density="compact"
                    >
                        <div v-for="message in dialog.preview.errors" :key="message">{{ message }}</div>
                    </v-alert>
                    <v-chip size="small" color="success" variant="tonal">{{ $t("favorites.crossVersionCopy.matched", { n: dialog.preview.matched?.length || 0 }) }}</v-chip>
                    <v-chip size="small" color="warning" variant="tonal">{{ $t("favorites.crossVersionCopy.conflicts", { n: dialog.preview.conflicts?.length || 0 }) }}</v-chip>
                    <v-list density="compact" class="mt-2">
                        <v-list-item v-for="match in dialog.preview.matched || []" :key="`m-${match.source.id}`" prepend-icon="mdi-check" :title="displayName(match.source)" :subtitle="match.version.versionId || match.version.id"></v-list-item>
                        <v-list-item v-for="conflict in dialog.preview.conflicts || []" :key="`c-${conflict.source.id}`" prepend-icon="mdi-alert" :title="displayName(conflict.source)" :subtitle="conflict.reason"></v-list-item>
                    </v-list>
                </div>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <v-btn variant="text" :disabled="dialog.isApplying" @click="dialog.show = false">{{ $t("favorites.actions.cancel") }}</v-btn>
                <v-btn
                    color="primary"
                    variant="flat"
                    :disabled="!applyReady"
                    :loading="dialog.isApplying"
                    @click="applyCopy"
                >{{ $t("favorites.actions.apply") }}</v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import MinecraftTargetFields from "./MinecraftTargetFields.vue";
import { useFavoritesStore } from "../stores/favorites";
import { useMinecraftStore } from "../stores/minecraft";

const emit = defineEmits(["message"]);

const { t } = useI18n();
const favoritesStore = useFavoritesStore();
const minecraftStore = useMinecraftStore();
const operation = ref(0);
const dialog = reactive({
    show: false,
    sourceListId: "",
    sourceListName: "",
    sourceMinecraftVersion: "",
    sourceModLoader: "",
    minecraftVersion: "",
    modLoader: "",
    ignoreConflicts: false,
    preview: null,
    isPreviewing: false,
    isApplying: false,
});

const displayName = (item) => item.title || item.slug || item.modId;

const copyReady = computed(() => {
    const minecraftVersion = dialog.minecraftVersion.trim();
    const modLoader = dialog.modLoader.trim().toLocaleLowerCase();
    const sourceMinecraftVersion = dialog.sourceMinecraftVersion.trim();
    const sourceModLoader = dialog.sourceModLoader.trim().toLocaleLowerCase();
    return Boolean(
        dialog.sourceListId &&
        minecraftVersion &&
        modLoader &&
        (minecraftVersion !== sourceMinecraftVersion || modLoader !== sourceModLoader)
    );
});

const previewIsCurrent = computed(() => {
    const preview = dialog.preview;
    if (!preview) return false;
    return preview.sourceListId === dialog.sourceListId &&
        preview.minecraftVersion === dialog.minecraftVersion.trim() &&
        preview.modLoader === dialog.modLoader.trim().toLocaleLowerCase();
});

const applyReady = computed(() => {
    const preview = dialog.preview;
    if (!copyReady.value || !previewIsCurrent.value || !preview) return false;
    if (dialog.isPreviewing || dialog.isApplying) return false;
    if (preview.nameConflict || (preview.errors || []).length > 0 || (preview.matched || []).length === 0) return false;
    return dialog.ignoreConflicts || (preview.conflicts || []).length === 0;
});

const errorMessage = (error) => {
    if (!error) return t("favorites.export.unknownError");
    if (typeof error === "string") return error;
    return error.message || String(error);
};

const request = () => ({
    sourceListId: dialog.sourceListId,
    minecraftVersion: dialog.minecraftVersion.trim(),
    modLoader: dialog.modLoader.trim().toLocaleLowerCase(),
    ignoreConflicts: dialog.ignoreConflicts,
});

const requestIsCurrent = (value) =>
    dialog.show &&
    value.sourceListId === dialog.sourceListId &&
    value.minecraftVersion === dialog.minecraftVersion.trim() &&
    value.modLoader === dialog.modLoader.trim().toLocaleLowerCase();

const previewCopy = async () => {
    if (!copyReady.value || dialog.isPreviewing) return;
    const current = request();
    const currentOperation = operation.value;
    dialog.isPreviewing = true;
    try {
        const preview = await favoritesStore.previewCrossVersionCopy(current);
        if (currentOperation === operation.value && requestIsCurrent(current)) {
            dialog.preview = preview;
        }
    } catch (error) {
        if (currentOperation === operation.value && requestIsCurrent(current)) {
            emit("message", t("favorites.crossVersionCopy.notApplied", { reason: errorMessage(error) }), "error");
        }
    } finally {
        if (currentOperation === operation.value) {
            dialog.isPreviewing = false;
        }
    }
};

const applyCopy = async () => {
    if (!applyReady.value) return;
    const current = request();
    dialog.isApplying = true;
    try {
        const result = await favoritesStore.applyCrossVersionCopy(current);
        dialog.preview = result?.preview || dialog.preview;
        if (result?.applied && result.targetList?.id) {
            dialog.show = false;
            emit("message", t("favorites.crossVersionCopy.success", {
                name: result.targetList.name,
                added: result.result?.added || 0,
                skipped: result.result?.skipped || 0,
            }));
            return;
        }
        const reason = result?.result?.errors?.[0] ||
            result?.preview?.errors?.[0] ||
            t("favorites.crossVersionCopy.unknownError");
        emit("message", t("favorites.crossVersionCopy.notApplied", { reason }), "error");
    } catch (error) {
        emit("message", t("favorites.crossVersionCopy.notApplied", { reason: errorMessage(error) }), "error");
    } finally {
        dialog.isApplying = false;
    }
};

const clearClosedDialog = () => {
    if (dialog.show) return;
    operation.value += 1;
    dialog.sourceListId = "";
    dialog.sourceListName = "";
    dialog.sourceMinecraftVersion = "";
    dialog.sourceModLoader = "";
    dialog.minecraftVersion = "";
    dialog.modLoader = "";
    dialog.ignoreConflicts = false;
    dialog.preview = null;
    dialog.isPreviewing = false;
    dialog.isApplying = false;
};

const open = (list) => {
    operation.value += 1;
    dialog.show = true;
    dialog.sourceListId = list.id;
    dialog.sourceListName = list.name;
    dialog.sourceMinecraftVersion = list.minecraftVersion || "";
    dialog.sourceModLoader = list.modLoader || "";
    dialog.minecraftVersion = minecraftStore.selectedMinecraftVersion;
    dialog.modLoader = minecraftStore.selectedModLoader;
    dialog.ignoreConflicts = false;
    dialog.preview = null;
    dialog.isPreviewing = false;
    dialog.isApplying = false;
};

watch(
    () => [dialog.minecraftVersion, dialog.modLoader],
    () => {
        dialog.preview = null;
    },
);

defineExpose({ open });
</script>

<style scoped>
.cross-version-copy-grid {
    min-width: 0;
}

@media (max-width: 599.98px) {
    .cross-version-copy-grid {
        grid-template-columns: 1fr;
        width: 100%;
    }
}
</style>
