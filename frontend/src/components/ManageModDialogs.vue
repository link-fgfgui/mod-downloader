<template>
    <v-dialog :model-value="deleteDialog" max-width="420"
        @update:model-value="$emit('update:deleteDialog', $event)"
        @after-leave="$emit('after-leave-delete')">
        <v-card>
            <v-card-title>{{ $t("manage.confirmDelete.title") }}</v-card-title>
            <v-card-text>
                {{ $t("manage.confirmDelete.body", { n: pendingDeleteCount }) }}
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <v-btn variant="text" :disabled="busy" @click="$emit('update:deleteDialog', false)">
                    {{ $t("manage.confirmDelete.cancel") }}
                </v-btn>
                <v-btn color="error" variant="tonal" :loading="batchOperation === 'delete'" @click="$emit('confirm-delete')">
                    {{ $t("manage.confirmDelete.confirm") }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>

    <v-dialog :model-value="disableImpactDialog" max-width="680"
        @update:model-value="$emit('update:disableImpactDialog', $event)"
        @after-leave="$emit('after-leave-disable')">
        <v-card>
            <v-card-title>{{ $t("manage.disableImpact.title") }}</v-card-title>
            <v-card-text>
                <div class="text-body-2 mb-3">{{ $t("manage.disableImpact.body") }}</div>
                <v-list density="compact" class="disable-impact-list">
                    <v-list-item
                        v-for="impact in disableImpacts"
                        :key="impact.modId"
                        :title="$t('manage.disableImpact.missing', { id: impact.modId })"
                        :subtitle="disableImpactSubtitle(impact)"
                    >
                        <template #prepend>
                            <v-icon icon="mdi-alert-circle-outline" color="warning"></v-icon>
                        </template>
                        <template #append>
                            <v-tooltip v-if="impact.candidates?.length === 1" :text="$t('manage.disableImpact.restore')" location="top">
                                <template #activator="{ props: tip }">
                                    <v-btn
                                        v-bind="tip"
                                        icon="mdi-download"
                                        size="small"
                                        variant="tonal"
                                        color="primary"
                                        :loading="restoringDependencyId === impact.modId"
                                        :disabled="Boolean(restoringDependencyId) || restoredDependencies[impact.modId]"
                                        @click="$emit('restore', impact, impact.candidates[0])"
                                    ></v-btn>
                                </template>
                            </v-tooltip>
                            <v-menu v-else-if="impact.candidates?.length > 1">
                                <template #activator="{ props: menu }">
                                    <v-btn
                                        v-bind="menu"
                                        icon="mdi-download-multiple"
                                        size="small"
                                        variant="tonal"
                                        color="primary"
                                        :title="$t('manage.disableImpact.chooseSource')"
                                        :aria-label="$t('manage.disableImpact.chooseSource')"
                                        :loading="restoringDependencyId === impact.modId"
                                        :disabled="Boolean(restoringDependencyId) || restoredDependencies[impact.modId]"
                                    ></v-btn>
                                </template>
                                <v-list density="compact">
                                    <v-list-item
                                        v-for="candidate in impact.candidates"
                                        :key="`${candidate.platform}:${candidate.projectId}:${candidate.versionId}`"
                                        :title="candidate.title || candidate.projectId"
                                        :subtitle="candidate.platform"
                                        prepend-icon="mdi-download"
                                        @click="$emit('restore', impact, candidate)"
                                    ></v-list-item>
                                </v-list>
                            </v-menu>
                            <v-tooltip v-else :text="$t('manage.disableImpact.cacheMiss')" location="top">
                                <template #activator="{ props: tip }">
                                    <v-icon v-bind="tip" icon="mdi-cloud-off-outline" class="text-medium-emphasis"></v-icon>
                                </template>
                            </v-tooltip>
                        </template>
                    </v-list-item>
                </v-list>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <v-btn variant="text" :disabled="busy || Boolean(restoringDependencyId)" @click="$emit('update:disableImpactDialog', false)">
                    {{ $t("manage.disableImpact.cancel") }}
                </v-btn>
                <v-btn color="warning" variant="tonal" :loading="Boolean(batchOperation)" :disabled="Boolean(restoringDependencyId)" @click="$emit('confirm-disable')">
                    {{ $t("manage.disableImpact.continue") }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup>
import { useI18n } from "vue-i18n";

defineProps({
    deleteDialog: { type: Boolean, default: false },
    pendingDeleteCount: { type: Number, default: 0 },
    busy: { type: Boolean, default: false },
    batchOperation: { type: String, default: "" },
    disableImpactDialog: { type: Boolean, default: false },
    disableImpacts: { type: Array, default: () => [] },
    restoringDependencyId: { type: String, default: "" },
    restoredDependencies: { type: Object, default: () => ({}) },
});

defineEmits([
    "update:deleteDialog",
    "update:disableImpactDialog",
    "after-leave-delete",
    "after-leave-disable",
    "confirm-delete",
    "confirm-disable",
    "restore",
]);

const { t } = useI18n();

const disableImpactSubtitle = (impact) => {
    const names = (impact.affectedMods || []).map((mod) => mod.name || mod.fileName || mod.path).filter(Boolean);
    return t("manage.disableImpact.affected", { mods: names.join(", ") });
};
</script>

<style scoped>
.disable-impact-list {
    max-height: min(48vh, 420px);
    overflow-y: auto;
    padding-inline-end: 16px;
}
</style>
