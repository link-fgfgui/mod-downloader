<template>
    <v-container class="settings-page" fluid>
        <h1 class="text-h5 mb-4">{{ $t('settings.title') }}</h1>

        <v-progress-linear v-if="settingsStore.isLoading" indeterminate class="mb-4" />

        <v-row>
            <v-col cols="12" md="6">
                <v-card class="mb-4">
                    <v-card-title>{{ $t('settings.theme.label') }}</v-card-title>
                    <v-card-text>
                        <v-radio-group v-model="settingsStore.draftTheme" @update:model-value="onThemeChange">
                            <v-radio :label="$t('settings.theme.dark')" value="dark" />
                            <v-radio :label="$t('settings.theme.light')" value="light" />
                            <v-radio :label="$t('settings.theme.system')" value="system" />
                        </v-radio-group>
                    </v-card-text>
                </v-card>

                <v-card class="mb-4">
                    <v-card-title>{{ $t('settings.animations.label') }}</v-card-title>
                    <v-card-text>
                        <div class="text-caption text-medium-emphasis mb-2">
                            {{ $t('settings.animations.mode') }}
                        </div>
                        <v-btn-toggle v-model="settingsStore.draftAnimationMode" color="primary" density="comfortable"
                            divided mandatory variant="outlined" class="animation-mode-toggle mb-3">
                            <v-btn :value="animationModeOff" size="small">
                                {{ $t('settings.animations.modes.off') }}
                            </v-btn>
                            <v-btn :value="animationModeVuetify" size="small">
                                {{ $t('settings.animations.modes.vuetify') }}
                            </v-btn>
                            <v-btn :value="animationModeGsap" size="small">
                                {{ $t('settings.animations.modes.gsap') }}
                            </v-btn>
                        </v-btn-toggle>
                        <div class="d-flex align-center gap-2 mb-3">
                            <v-slider v-model="settingsStore.draftAnimationDurationMultiplier"
                                :disabled="animationsDisabled" :min="minAnimationDurationMultiplier"
                                :max="maxAnimationDurationMultiplier" :step="0.25" density="compact" hide-details />
                            <v-text-field v-model.number="settingsStore.draftAnimationDurationMultiplier"
                                :disabled="animationsDisabled" type="number"
                                :min="minAnimationDurationMultiplier" :max="maxAnimationDurationMultiplier"
                                step="0.25" suffix="x" density="compact" hide-details class="multiplier-input" />
                        </div>
                        <v-btn :loading="settingsStore.isSavingAnimations" variant="outlined"
                            prepend-icon="mdi-content-save" @click="saveAnimations">
                            {{ $t('settings.animations.save') }}
                        </v-btn>
                    </v-card-text>
                </v-card>

                <v-card class="mb-4">
                    <v-card-title>{{ $t('settings.cleanup.label') }}</v-card-title>
                    <v-card-text>
                        <v-switch
                            v-model="settingsStore.draftAutoScanUnusedDependencies"
                            color="primary"
                            density="comfortable"
                            hide-details
                            :label="$t('settings.cleanup.autoScan')"
                        />
                        <v-btn
                            :loading="settingsStore.isSavingUnusedDependencyCleanup"
                            variant="outlined"
                            prepend-icon="mdi-content-save"
                            class="mt-3"
                            @click="saveCleanup"
                        >
                            {{ $t('settings.cleanup.save') }}
                        </v-btn>
                    </v-card-text>
                </v-card>

                <v-card class="mb-4">
                    <v-card-title>{{ $t('settings.minecraftDir.label') }}</v-card-title>
                    <v-card-text>
                        <v-text-field :model-value="settingsStore.view?.minecraftDir" readonly density="compact"
                            hide-details class="mb-2" />
                        <div class="settings-action-row settings-action-row--center">
                            <v-btn :loading="settingsStore.isChoosingDir" variant="outlined"
                                prepend-icon="mdi-folder-open" @click="chooseDir">
                                {{ $t('settings.minecraftDir.choose') }}
                            </v-btn>
                            <v-btn :loading="settingsStore.isValidatingDir" variant="text" prepend-icon="mdi-check-circle"
                                @click="settingsStore.validateDir()">
                                {{ $t('settings.minecraftDir.valid') }}
                            </v-btn>
                            <v-icon v-if="settingsStore.dirValid === true" color="success">mdi-check</v-icon>
                            <v-icon v-else-if="settingsStore.dirValid === false" color="error">mdi-close</v-icon>
                        </div>
                    </v-card-text>
                </v-card>

                <v-card class="mb-4">
                    <v-card-title>{{ $t('settings.cacheDir.label') }}</v-card-title>
                    <v-card-text>
                        <v-text-field :model-value="settingsStore.view?.cacheDir || $t('settings.cacheDir.default')"
                            readonly density="compact" hide-details class="mb-2" />
                        <v-text-field :model-value="settingsStore.view?.cachePath"
                            :label="$t('settings.cacheDir.path')" readonly density="compact" hide-details
                            class="mb-2" />
                        <div class="settings-action-row">
                            <v-btn :loading="settingsStore.isChoosingCacheDir" variant="outlined"
                                prepend-icon="mdi-folder-open" @click="chooseCacheDir">
                                {{ $t('settings.cacheDir.choose') }}
                            </v-btn>
                            <v-btn :disabled="!settingsStore.view?.cacheDir"
                                :loading="settingsStore.isSavingCacheDir" variant="text" prepend-icon="mdi-restore"
                                @click="resetCacheDir">
                                {{ $t('settings.cacheDir.reset') }}
                            </v-btn>
                        </div>
                    </v-card-text>
                </v-card>
            </v-col>

            <v-col cols="12" md="6">
                <v-card class="mb-4">
                    <v-card-title>{{ $t('settings.mcim.label') }}</v-card-title>
                    <v-card-subtitle>{{ $t('settings.mcim.hint') }}</v-card-subtitle>
                    <v-card-text>
                        <v-switch v-model="settingsStore.draftMCIMEnabled" color="primary" density="comfortable"
                            hide-details :label="$t('settings.mcim.useMirror')" />
                        <v-btn :loading="settingsStore.isSavingMCIM" variant="outlined"
                            prepend-icon="mdi-content-save" class="mt-3" @click="saveMCIM">
                            {{ $t('settings.mcim.save') }}
                        </v-btn>
                    </v-card-text>
                </v-card>

                <v-card class="mb-4">
                    <v-card-title>{{ $t('settings.apiKeys.curseforge.label') }}</v-card-title>
                    <v-card-subtitle>{{ $t('settings.apiKeys.curseforge.hint') }}</v-card-subtitle>
                    <v-card-text>
                        <div class="mb-2">
                            <v-chip v-if="settingsStore.hasCurseforgeKey" color="success" size="small">
                                {{ $t('settings.apiKeys.curseforge.status') }}: {{ settingsStore.view?.curseforgeKeyMask }}
                            </v-chip>
                            <v-chip v-else color="default" size="small">
                                {{ $t('settings.apiKeys.curseforge.statusEmpty') }}
                            </v-chip>
                        </div>
                        <v-text-field v-model="settingsStore.draftCurseforgeKey"
                            :label="$t('settings.apiKeys.curseforge.placeholder')" type="password" density="compact"
                            hide-details class="mb-2" />
                        <div class="settings-action-row">
                            <v-btn :loading="settingsStore.isSavingKeys" variant="outlined"
                                prepend-icon="mdi-content-save" @click="saveKeys">
                                {{ $t('settings.apiKeys.curseforge.save') }}
                            </v-btn>
                            <v-btn :disabled="!settingsStore.hasCurseforgeKey" :loading="settingsStore.isSavingKeys"
                                variant="text" color="error" prepend-icon="mdi-delete" @click="clearCurseforge">
                                {{ $t('settings.apiKeys.curseforge.clear') }}
                            </v-btn>
                        </div>
                    </v-card-text>
                </v-card>

                <v-card>
                    <v-card-title>{{ $t('settings.apiKeys.modrinth.label') }}</v-card-title>
                    <v-card-subtitle>{{ $t('settings.apiKeys.modrinth.hint') }}</v-card-subtitle>
                    <v-card-text>
                        <div class="mb-2">
                            <v-chip v-if="settingsStore.hasModrinthKey" color="success" size="small">
                                {{ $t('settings.apiKeys.modrinth.status') }}: {{ settingsStore.view?.modrinthKeyMask }}
                            </v-chip>
                            <v-chip v-else color="default" size="small">
                                {{ $t('settings.apiKeys.modrinth.statusEmpty') }}
                            </v-chip>
                        </div>
                        <v-text-field v-model="settingsStore.draftModrinthKey"
                            :label="$t('settings.apiKeys.modrinth.placeholder')" type="password" density="compact"
                            hide-details class="mb-2" />
                        <div class="settings-action-row">
                            <v-btn :loading="settingsStore.isSavingKeys" variant="outlined"
                                prepend-icon="mdi-content-save" @click="saveKeys">
                                {{ $t('settings.apiKeys.modrinth.save') }}
                            </v-btn>
                            <v-btn :disabled="!settingsStore.hasModrinthKey" :loading="settingsStore.isSavingKeys"
                                variant="text" color="error" prepend-icon="mdi-delete" @click="clearModrinth">
                                {{ $t('settings.apiKeys.modrinth.clear') }}
                            </v-btn>
                        </div>
                    </v-card-text>
                </v-card>
            </v-col>
        </v-row>

        <v-snackbar v-model="snackbar.show" :color="snackbar.color" timeout="2000">
            {{ snackbar.message }}
        </v-snackbar>
    </v-container>
</template>

<script setup lang="ts">
import { computed, onActivated, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useSettingsStore } from "../stores/settings";
import { applyVuetifyTheme } from "../composables/useTheme";
import {
    animationModeGsap,
    animationModeOff,
    animationModeVuetify,
    applyAnimationSettings,
    maxAnimationDurationMultiplier,
    minAnimationDurationMultiplier,
} from "../composables/useAnimationSettings";

const { t } = useI18n();
const settingsStore = useSettingsStore();
const snackbar = ref({ show: false, message: "", color: "success" });
const animationsDisabled = computed(() => settingsStore.draftAnimationMode === animationModeOff);

onActivated(() => {
    void settingsStore.load();
});

async function onThemeChange() {
    const next = await settingsStore.saveTheme();
    applyVuetifyTheme(next);
    snackbar.value = { show: true, message: t('settings.theme.saved'), color: "success" };
}

async function saveAnimations() {
    const next = await settingsStore.saveAnimationSettings();
    applyAnimationSettings(next);
    snackbar.value = { show: true, message: t('settings.animations.saved'), color: "success" };
}

async function saveCleanup() {
    await settingsStore.saveUnusedDependencyCleanupSettings();
    snackbar.value = { show: true, message: t('settings.cleanup.saved'), color: "success" };
}

async function saveMCIM() {
    await settingsStore.saveMCIMSettings();
    snackbar.value = { show: true, message: t('settings.mcim.saved'), color: "success" };
}

async function chooseDir() {
    await settingsStore.chooseMinecraftDir();
}

async function chooseCacheDir() {
    const changed = await settingsStore.chooseCacheDir();
    if (changed) {
        snackbar.value = { show: true, message: t('settings.cacheDir.saved'), color: "success" };
    }
}

async function resetCacheDir() {
    await settingsStore.resetCacheDir();
    snackbar.value = { show: true, message: t('settings.cacheDir.saved'), color: "success" };
}

async function saveKeys() {
    const hadKey = settingsStore.hasCurseforgeKey || settingsStore.hasModrinthKey;
    const clearing = settingsStore.clearCurseforgeKey || settingsStore.clearModrinthKey;
    await settingsStore.saveApiKeys();
    snackbar.value = {
        show: true,
        message: (hadKey && clearing) ? t('settings.apiKeys.cleared') : t('settings.apiKeys.saved'),
        color: "success",
    };
}

function clearCurseforge() {
    settingsStore.clearCurseforgeKey = true;
    void saveKeys();
}

function clearModrinth() {
    settingsStore.clearModrinthKey = true;
    void saveKeys();
}
</script>

<style scoped>
.gap-2 {
    gap: 8px;
}

.settings-action-row {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
}

.settings-action-row--center {
    align-items: center;
}

.settings-page :deep(.v-card-title) {
    line-height: 1.35;
    overflow: visible;
    overflow-wrap: anywhere;
    white-space: normal;
}

.settings-page :deep(.v-card-subtitle) {
    line-height: 1.4;
    overflow: visible;
    overflow-wrap: anywhere;
    white-space: normal;
}

.multiplier-input {
    max-width: 112px;
}

.animation-mode-toggle {
    width: 100%;
}

.animation-mode-toggle :deep(.v-btn) {
    flex: 1 1 0;
    min-width: 0;
}
</style>
