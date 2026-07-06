<template>
    <v-container fluid>
        <v-row class="align-center mb-4">
            <v-col cols="12" md="auto">
                <h1 class="text-h5">{{ $t('unpin.title') }}</h1>
            </v-col>
            <v-col cols="12" md="auto" class="unpin-filter-row">
                <v-text-field v-model="pinnedModsStore.filterPlatform" :label="$t('unpin.columns.platform')"
                    class="unpin-filter-field" density="compact" hide-details clearable />
                <v-text-field v-model="pinnedModsStore.filterMinecraftVersion" :label="$t('unpin.columns.minecraftVersion')"
                    class="unpin-filter-field unpin-filter-field-wide" density="compact" hide-details clearable />
                <v-text-field v-model="pinnedModsStore.filterModLoader" :label="$t('unpin.columns.modLoader')"
                    class="unpin-filter-field" density="compact" hide-details clearable />
            </v-col>
            <v-col cols="12" md="auto" class="unpin-action-row ml-auto">
                <v-btn :loading="pinnedModsStore.isLoading" variant="outlined" prepend-icon="mdi-refresh"
                    @click="pinnedModsStore.load()">
                    {{ $t('unpin.refresh') }}
                </v-btn>
                <v-btn :disabled="!pinnedModsStore.filteredPins.length" color="error" prepend-icon="mdi-pin-off"
                    @click="confirmAll = true">
                    {{ $t('unpin.unpinAll') }}
                </v-btn>
            </v-col>
        </v-row>

        <v-progress-linear v-if="pinnedModsStore.isLoading" indeterminate class="mb-4" />

        <v-alert v-if="!pinnedModsStore.isLoading && !pinnedModsStore.hasPins" type="info" variant="tonal">
            {{ $t('unpin.empty') }}
        </v-alert>

        <div v-else class="unpin-table-wrap">
            <v-data-table :items="pinnedModsStore.filteredPins" :headers="headers" density="compact"
                class="unpin-table elevation-1" hide-default-footer :items-per-page="-1">
                <template #[actionsSlotName]="{ item }">
                    <v-btn :loading="pinnedModsStore.pendingUnpinKeys.has(pinnedModsStore.pinKey(item))"
                        variant="outlined" size="small" prepend-icon="mdi-pin-off" @click="unpin(item as database.PinnedMod)">
                        {{ $t('unpin.unpin') }}
                    </v-btn>
                </template>
            </v-data-table>
        </div>

        <v-dialog v-model="confirmAll" max-width="420">
            <v-card>
                <v-card-title>{{ $t('unpin.confirmAll.title') }}</v-card-title>
                <v-card-text>{{ $t('unpin.confirmAll.body') }}</v-card-text>
                <v-card-actions>
                    <v-spacer />
                    <v-btn variant="text" @click="confirmAll = false">{{ $t('unpin.confirmAll.cancel') }}</v-btn>
                    <v-btn color="error" variant="elevated" @click="unpinAll">{{ $t('unpin.confirmAll.confirm') }}</v-btn>
                </v-card-actions>
            </v-card>
        </v-dialog>

        <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="2000">
            {{ snackbar.message }}
        </v-snackbar>
    </v-container>
</template>

<script setup lang="ts">
import { onActivated, ref } from "vue";
import { useI18n } from "vue-i18n";
import { usePinnedModsStore } from "../stores/pinnedMods";
import type { database } from "../../wailsjs/go/models";

const { t } = useI18n();
const pinnedModsStore = usePinnedModsStore();
const confirmAll = ref(false);
const snackbar = ref({ show: false, message: "", color: "success" });
const actionsSlotName = "item.actions";

const headers = [
    { title: t('unpin.columns.platform'), key: 'platform' },
    { title: t('unpin.columns.modId'), key: 'modId' },
    { title: t('unpin.columns.versionId'), key: 'versionId' },
    { title: t('unpin.columns.minecraftVersion'), key: 'minecraftVersion' },
    { title: t('unpin.columns.modLoader'), key: 'modLoader' },
    { title: t('unpin.unpin'), key: 'actions', sortable: false, align: 'end' as const },
];

onActivated(() => {
    void pinnedModsStore.load();
});

async function unpin(pin: database.PinnedMod) {
    try {
        const ok = await pinnedModsStore.unpin(pin);
        if (ok) {
            snackbar.value = { show: true, message: t('unpin.removed'), color: "success" };
            return;
        }
        snackbar.value = { show: true, message: t('unpin.removeFailed'), color: "warning" };
    } catch {
        await pinnedModsStore.load();
        snackbar.value = { show: true, message: t('unpin.removeFailed'), color: "error" };
    }
}

async function unpinAll() {
    confirmAll.value = false;
    await pinnedModsStore.unpinAllFiltered();
    await pinnedModsStore.load();
}
</script>

<style scoped>
.unpin-filter-row,
.unpin-action-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.unpin-filter-field {
    flex: 1 1 140px;
    min-width: 0;
}

.unpin-filter-field-wide {
    flex-basis: 160px;
}

.unpin-action-row {
    justify-content: flex-end;
}

.unpin-table-wrap {
    max-width: 100%;
    overflow-x: auto;
}

.unpin-table {
    min-width: max-content;
}

@media (max-width: 599.98px) {
    .unpin-action-row {
        justify-content: stretch;
    }

    .unpin-action-row :deep(.v-btn) {
        flex: 1 1 auto;
    }
}
</style>
