import { defineStore } from "pinia";
import {
    GetSettings,
    SaveLanguage,
    SaveTheme,
    SaveAnimationSettings,
    SaveApiKeys,
    SaveUnusedDependencyCleanupSettings,
    SaveMCIMSettings,
    SaveNetworkSettings,
    SaveCacheDirPreference,
    ChooseCacheDir,
} from "../../wailsjs/go/main/App";
import type { main } from "../../wailsjs/go/models";
import {
    animationModeEnabled,
    defaultAnimationMode,
    defaultAnimationDurationMultiplier,
    normalizeAnimationMode,
    normalizeAnimationDurationMultiplier,
} from "../composables/useAnimationSettings";
import {
    applyLanguagePreference,
    currentLocale,
    languageSystem,
    normalizeLanguagePreference,
} from "../plugins/i18n";

const themeDark = "dark";
const themeLight = "light";
const themeSystem = "system";
const apiKeyKeepSentinel = "<keep>";
const autoSaveTimers = new Map<string, number>();

export const useSettingsStore = defineStore("settings", {
    state: () => ({
        view: null as main.SettingsView | null,
        isLoading: false,
        isSavingTheme: false,
        isSavingLanguage: false,
        isSavingAnimations: false,
        isSavingUnusedDependencyCleanup: false,
        isSavingMCIM: false,
        isSavingKeys: false,
        isSavingCacheDir: false,
        isChoosingCacheDir: false,
        draftTheme: "",
        draftLanguage: languageSystem,
        draftAnimationMode: defaultAnimationMode,
        draftAnimationDurationMultiplier: defaultAnimationDurationMultiplier,
        draftAutoScanUnusedDependencies: false,
        draftMCIMEnabled: false,
        draftFileConcurrency: 4,
        draftConcurrentDownloads: 1,
        draftAdaptiveFileConcurrency: false,
        draftTargetDownloadRateMiB: 1,
        draftVerifySHA1: false,
        draftRequestsPerSecond: 0,
        isSavingNetwork: false,
        draftCurseforgeKey: "",
        draftModrinthKey: "",
        clearCurseforgeKey: false,
        clearModrinthKey: false,
        autoSaveError: "",
    }),
    getters: {
        hasCurseforgeKey: (s) => Boolean(s.view?.hasCurseforgeKey),
        hasModrinthKey: (s) => Boolean(s.view?.hasModrinthKey),
    },
    actions: {
        scheduleAutoSave(kind: "animations" | "cleanup" | "mcim" | "network" | "keys", delay = 600) {
            const previous = autoSaveTimers.get(kind);
            if (previous) window.clearTimeout(previous);
            autoSaveTimers.set(kind, window.setTimeout(async () => {
                autoSaveTimers.delete(kind);
                this.autoSaveError = "";
                try {
                    if (kind === "animations") await this.saveAnimationSettings();
                    else if (kind === "cleanup") await this.saveUnusedDependencyCleanupSettings();
                    else if (kind === "mcim") await this.saveMCIMSettings();
                    else if (kind === "network") await this.saveNetworkSettings();
                    else await this.saveApiKeys();
                } catch (error) {
                    this.autoSaveError = error instanceof Error ? error.message : String(error);
                    await this.load();
                }
            }, delay));
        },
        async load() {
            this.isLoading = true;
            try {
                this.view = await GetSettings();
                this.draftTheme = this.view?.theme || themeDark;
                this.draftLanguage = normalizeLanguagePreference(this.view?.language);
                this.draftAnimationMode = normalizeAnimationMode(
                    this.view?.animationMode,
                    this.view?.animationEnabled
                );
                this.draftAnimationDurationMultiplier = normalizeAnimationDurationMultiplier(
                    this.view?.animationDurationMultiplier ?? defaultAnimationDurationMultiplier
                );
                this.draftAutoScanUnusedDependencies = this.view?.autoScanUnusedDependencies ?? false;
                this.draftMCIMEnabled = this.view?.mcimEnabled ?? false;
                this.draftFileConcurrency = this.view?.fileConcurrency ?? 4;
                this.draftConcurrentDownloads = this.view?.concurrentDownloads ?? 1;
                this.draftAdaptiveFileConcurrency = this.view?.adaptiveFileConcurrency ?? false;
                this.draftTargetDownloadRateMiB = this.view?.targetDownloadRateMiB ?? 1;
                this.draftVerifySHA1 = this.view?.verifySha1 ?? false;
                this.draftRequestsPerSecond = this.view?.requestsPerSecond ?? 0;
                this.draftCurseforgeKey = "";
                this.draftModrinthKey = "";
                this.clearCurseforgeKey = false;
                this.clearModrinthKey = false;
            } finally {
                this.isLoading = false;
            }
        },
        async saveTheme() {
            this.isSavingTheme = true;
            try {
                const next = await SaveTheme(this.draftTheme);
                if (this.view) this.view.theme = next;
                this.draftTheme = next;
                return next;
            } finally {
                this.isSavingTheme = false;
            }
        },
        async saveLanguage() {
            this.isSavingLanguage = true;
            try {
                const next = normalizeLanguagePreference(await SaveLanguage(this.draftLanguage));
                if (this.view) this.view.language = next;
                this.draftLanguage = next;
                applyLanguagePreference(next);
                return next;
            } finally {
                this.isSavingLanguage = false;
            }
        },
        async saveAnimationSettings() {
            this.isSavingAnimations = true;
            try {
                const animationMode = normalizeAnimationMode(this.draftAnimationMode);
                const req = {
                    animationMode,
                    animationEnabled: animationModeEnabled(animationMode),
                    animationDurationMultiplier: normalizeAnimationDurationMultiplier(
                        this.draftAnimationDurationMultiplier
                    ),
                };
                this.view = await SaveAnimationSettings(req);
                this.draftAnimationMode = normalizeAnimationMode(this.view.animationMode, this.view.animationEnabled);
                this.draftAnimationDurationMultiplier = normalizeAnimationDurationMultiplier(
                    this.view.animationDurationMultiplier
                );
                return {
                    animationMode: this.draftAnimationMode,
                    animationEnabled: animationModeEnabled(this.draftAnimationMode),
                    animationDurationMultiplier: this.draftAnimationDurationMultiplier,
                };
            } finally {
                this.isSavingAnimations = false;
            }
        },
        async saveUnusedDependencyCleanupSettings() {
            this.isSavingUnusedDependencyCleanup = true;
            try {
                this.view = await SaveUnusedDependencyCleanupSettings({
                    autoScanUnusedDependencies: this.draftAutoScanUnusedDependencies,
                });
                this.draftAutoScanUnusedDependencies = this.view.autoScanUnusedDependencies;
                return this.view;
            } finally {
                this.isSavingUnusedDependencyCleanup = false;
            }
        },
        async saveMCIMSettings() {
            this.isSavingMCIM = true;
            try {
                this.view = await SaveMCIMSettings({ mcimEnabled: this.draftMCIMEnabled });
                this.draftMCIMEnabled = this.view.mcimEnabled;
                return this.view;
            } finally {
                this.isSavingMCIM = false;
            }
        },
        async saveNetworkSettings() {
            this.isSavingNetwork = true;
            try {
                this.view = await SaveNetworkSettings({
                    fileConcurrency: this.draftFileConcurrency,
                    concurrentDownloads: this.draftConcurrentDownloads,
                    adaptiveFileConcurrency: this.draftAdaptiveFileConcurrency,
                    targetDownloadRateMiB: this.draftTargetDownloadRateMiB,
                    verifySha1: this.draftVerifySHA1,
                    requestsPerSecond: this.draftRequestsPerSecond,
                });
                this.draftFileConcurrency = this.view.fileConcurrency;
                this.draftConcurrentDownloads = this.view.concurrentDownloads;
                this.draftAdaptiveFileConcurrency = this.view.adaptiveFileConcurrency;
                this.draftTargetDownloadRateMiB = this.view.targetDownloadRateMiB;
                this.draftVerifySHA1 = this.view.verifySha1;
                this.draftRequestsPerSecond = this.view.requestsPerSecond;
                return this.view;
            } finally {
                this.isSavingNetwork = false;
            }
        },
        async saveApiKeys() {
            this.isSavingKeys = true;
            try {
                const req = {
                    curseforgeApiKey: this.clearCurseforgeKey ? "" : (this.draftCurseforgeKey || apiKeyKeepSentinel),
                    modrinthApiKey: this.clearModrinthKey ? "" : (this.draftModrinthKey || apiKeyKeepSentinel),
                };
                this.view = await SaveApiKeys(req);
                this.draftCurseforgeKey = "";
                this.draftModrinthKey = "";
                this.clearCurseforgeKey = false;
                this.clearModrinthKey = false;
            } finally {
                this.isSavingKeys = false;
            }
        },
        async chooseCacheDir() {
            this.isChoosingCacheDir = true;
            try {
                const previous = this.view?.cacheDir || "";
                this.view = await ChooseCacheDir(currentLocale());
                return (this.view?.cacheDir || "") !== previous;
            } finally {
                this.isChoosingCacheDir = false;
            }
        },
        async resetCacheDir() {
            this.isSavingCacheDir = true;
            try {
                this.view = await SaveCacheDirPreference("");
                return this.view;
            } finally {
                this.isSavingCacheDir = false;
            }
        },
    },
});
