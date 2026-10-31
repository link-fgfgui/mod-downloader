import { create } from "zustand";

import { api } from "@/services/app";
import { applyLanguagePreference, currentLocale, languageSystem, normalizeLanguagePreference } from "@/i18n";
import {
  applyAnimationSettings,
  defaultAnimationDurationMultiplier,
  normalizeAnimationDurationMultiplier,
} from "@/lib/animations";
import { applyThemePreference, normalizeThemePreference } from "@/lib/theme";
import type { SettingsView } from "@/types";

export const apiKeyKeepSentinel = "<keep>";

export type AutoSaveKind =
  | "animations"
  | "simpleMode"
  | "cleanup"
  | "mcim"
  | "releaseType"
  | "network"
  | "keys";

export interface SettingsDrafts {
  draftTheme: string;
  draftLanguage: string;
  draftAnimationDurationMultiplier: number;
  draftSimpleMode: boolean;
  draftAutoScanUnusedDependencies: boolean;
  draftMCIMEnabled: boolean;
  draftEnableJijDetection: boolean;
  draftHardwareAcceleration: boolean;
  draftPreferredReleaseType: string;
  draftFileConcurrency: number;
  draftConcurrentDownloads: number;
  draftAdaptiveFileConcurrency: boolean;
  draftTargetDownloadRateMiB: number;
  draftVerifySHA1: boolean;
  draftRequestsPerSecond: number;
  draftCurseforgeKey: string;
  draftModrinthKey: string;
  clearCurseforgeKey: boolean;
  clearModrinthKey: boolean;
}

interface SettingsState extends SettingsDrafts {
  settings: SettingsView | null;
  loading: boolean;
  saving: boolean;
  autoSaveState: "idle" | "saving" | "saved" | "error";
  autoSaveError: string;
  error: string;
  isSavingTheme: boolean;
  isSavingLanguage: boolean;
  isSavingAnimations: boolean;
  isSavingSimpleMode: boolean;
  isSavingUnusedDependencyCleanup: boolean;
  isSavingMCIM: boolean;
  isSavingJijDetection: boolean;
  isSavingHardwareAcceleration: boolean;
  isSavingKeys: boolean;
  isSavingCacheDir: boolean;
  isChoosingCacheDir: boolean;
  isSavingNetwork: boolean;
  load: () => Promise<void>;
  patch: (patch: Partial<SettingsDrafts>) => void;
  scheduleAutoSave: (kind: AutoSaveKind, delay?: number) => void;
  saveTheme: () => Promise<string>;
  saveLanguage: () => Promise<string>;
  saveAnimationSettings: () => Promise<number>;
  saveSimpleModeSettings: () => Promise<SettingsView>;
  saveUnusedDependencyCleanupSettings: () => Promise<SettingsView>;
  saveMCIMSettings: () => Promise<SettingsView>;
  saveJijDetectionSettings: () => Promise<SettingsView>;
  saveHardwareAccelerationSettings: () => Promise<SettingsView>;
  savePreferredReleaseTypeSettings: () => Promise<SettingsView>;
  saveNetworkSettings: () => Promise<SettingsView>;
  saveApiKeys: () => Promise<SettingsView>;
  cancelApiKeyAutoSave: () => void;
  chooseCacheDir: () => Promise<boolean>;
  resetCacheDir: () => Promise<SettingsView>;
}

const autoSaveTimers = new Map<AutoSaveKind, number>();

export const useSettingsStore = create<SettingsState>((set, get) => ({
  settings: null,
  loading: false,
  saving: false,
  autoSaveState: "idle",
  autoSaveError: "",
  error: "",
  draftTheme: "dark",
  draftLanguage: languageSystem,
  draftAnimationDurationMultiplier: defaultAnimationDurationMultiplier,
  draftSimpleMode: false,
  draftAutoScanUnusedDependencies: false,
  draftMCIMEnabled: false,
  draftEnableJijDetection: true,
  draftHardwareAcceleration: true,
  draftPreferredReleaseType: "latest",
  draftFileConcurrency: 4,
  draftConcurrentDownloads: 1,
  draftAdaptiveFileConcurrency: false,
  draftTargetDownloadRateMiB: 1,
  draftVerifySHA1: false,
  draftRequestsPerSecond: 0,
  draftCurseforgeKey: "",
  draftModrinthKey: "",
  clearCurseforgeKey: false,
  clearModrinthKey: false,
  isSavingTheme: false,
  isSavingLanguage: false,
  isSavingAnimations: false,
  isSavingSimpleMode: false,
  isSavingUnusedDependencyCleanup: false,
  isSavingMCIM: false,
  isSavingJijDetection: false,
  isSavingHardwareAcceleration: false,
  isSavingKeys: false,
  isSavingCacheDir: false,
  isChoosingCacheDir: false,
  isSavingNetwork: false,

  async load() {
    set({ loading: true, error: "" });
    try {
      const view = await api.GetSettings();
      set({
        settings: view,
        draftTheme: normalizeThemePreference(view?.theme),
        draftLanguage: normalizeLanguagePreference(view?.language),
        draftAnimationDurationMultiplier: normalizeAnimationDurationMultiplier(
          view?.animationDurationMultiplier ?? defaultAnimationDurationMultiplier,
        ),
        draftSimpleMode: view?.simpleMode ?? false,
        draftAutoScanUnusedDependencies: view?.autoScanUnusedDependencies ?? false,
        draftMCIMEnabled: view?.mcimEnabled ?? false,
        draftEnableJijDetection: view?.enableJijDetection ?? true,
        draftHardwareAcceleration: view?.hardwareAcceleration ?? true,
        draftPreferredReleaseType: view?.preferredReleaseType || "latest",
        draftFileConcurrency: view?.fileConcurrency ?? 4,
        draftConcurrentDownloads: view?.concurrentDownloads ?? 1,
        draftAdaptiveFileConcurrency: view?.adaptiveFileConcurrency ?? false,
        draftTargetDownloadRateMiB: view?.targetDownloadRateMiB ?? 1,
        draftVerifySHA1: view?.verifySha1 ?? false,
        draftRequestsPerSecond: view?.requestsPerSecond ?? 0,
        draftCurseforgeKey: "",
        draftModrinthKey: "",
        clearCurseforgeKey: false,
        clearModrinthKey: false,
      });
      if (view) {
        applyThemePreference(view.theme);
        await applyLanguagePreference(view.language);
        applyAnimationSettings(view);
      }
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ loading: false });
    }
  },

  patch(patch) {
    set(patch);
  },

  scheduleAutoSave(kind, delay = 600) {
    const previous = autoSaveTimers.get(kind);
    if (previous) window.clearTimeout(previous);
    set({ autoSaveState: "saving", autoSaveError: "" });
    autoSaveTimers.set(
      kind,
      window.setTimeout(async () => {
        autoSaveTimers.delete(kind);
        try {
          switch (kind) {
            case "animations":
              await get().saveAnimationSettings();
              break;
            case "simpleMode":
              await get().saveSimpleModeSettings();
              break;
            case "cleanup":
              await get().saveUnusedDependencyCleanupSettings();
              break;
            case "mcim":
              await get().saveMCIMSettings();
              break;
            case "releaseType":
              await get().savePreferredReleaseTypeSettings();
              break;
            case "network":
              await get().saveNetworkSettings();
              break;
            case "keys":
              await get().saveApiKeys();
              break;
          }
          set({ autoSaveState: "saved" });
        } catch (error) {
          set({
            autoSaveState: "error",
            autoSaveError: error instanceof Error ? error.message : String(error),
          });
        }
      }, delay),
    );
  },

  async saveTheme() {
    const theme = normalizeThemePreference(get().draftTheme);
    set({ isSavingTheme: true, error: "" });
    try {
      const next = normalizeThemePreference(await api.SaveTheme(theme));
      set((state) => ({
        draftTheme: next,
        settings: state.settings ? { ...state.settings, theme: next } : state.settings,
      }));
      applyThemePreference(next);
      return next;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ isSavingTheme: false });
    }
  },

  async saveLanguage() {
    const language = normalizeLanguagePreference(get().draftLanguage);
    set({ isSavingLanguage: true, error: "" });
    try {
      const next = normalizeLanguagePreference(await api.SaveLanguage(language));
      set((state) => ({
        draftLanguage: next,
        settings: state.settings ? { ...state.settings, language: next } : state.settings,
      }));
      await applyLanguagePreference(next);
      return next;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ isSavingLanguage: false });
    }
  },

  async saveAnimationSettings() {
    set({ isSavingAnimations: true, error: "" });
    try {
      const multiplier = normalizeAnimationDurationMultiplier(get().draftAnimationDurationMultiplier);
      const view = await api.SaveAnimationSettings({ animationDurationMultiplier: multiplier });
      set({ settings: view, draftAnimationDurationMultiplier: view.animationDurationMultiplier });
      applyAnimationSettings(view);
      return view.animationDurationMultiplier;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      await get().load();
      throw error;
    } finally {
      set({ isSavingAnimations: false });
    }
  },

  async saveSimpleModeSettings() {
    set({ isSavingSimpleMode: true, error: "" });
    try {
      const view = await api.SaveSimpleModeSettings({ simpleMode: get().draftSimpleMode });
      set({ settings: view, draftSimpleMode: view.simpleMode });
      return view;
    } finally {
      set({ isSavingSimpleMode: false });
    }
  },

  async saveUnusedDependencyCleanupSettings() {
    set({ isSavingUnusedDependencyCleanup: true, error: "" });
    try {
      const view = await api.SaveUnusedDependencyCleanupSettings({
        autoScanUnusedDependencies: get().draftAutoScanUnusedDependencies,
      });
      set({ settings: view, draftAutoScanUnusedDependencies: view.autoScanUnusedDependencies });
      return view;
    } finally {
      set({ isSavingUnusedDependencyCleanup: false });
    }
  },

  async saveMCIMSettings() {
    set({ isSavingMCIM: true, error: "" });
    try {
      const view = await api.SaveMCIMSettings({ mcimEnabled: get().draftMCIMEnabled });
      set({ settings: view, draftMCIMEnabled: view.mcimEnabled });
      return view;
    } finally {
      set({ isSavingMCIM: false });
    }
  },

  async saveJijDetectionSettings() {
    set({ isSavingJijDetection: true, error: "" });
    try {
      const view = await api.SaveJijDetectionSettings({ enableJijDetection: get().draftEnableJijDetection });
      set({ settings: view, draftEnableJijDetection: view.enableJijDetection });
      return view;
    } finally {
      set({ isSavingJijDetection: false });
    }
  },

  async saveHardwareAccelerationSettings() {
    set({ isSavingHardwareAcceleration: true, error: "" });
    try {
      const view = await api.SaveHardwareAccelerationSettings({
        hardwareAcceleration: get().draftHardwareAcceleration,
      });
      set({ settings: view, draftHardwareAcceleration: view.hardwareAcceleration });
      return view;
    } finally {
      set({ isSavingHardwareAcceleration: false });
    }
  },

  async savePreferredReleaseTypeSettings() {
    set({ error: "" });
    const view = await api.SavePreferredReleaseTypeSettings({
      preferredReleaseType: get().draftPreferredReleaseType,
    });
    set({ settings: view, draftPreferredReleaseType: view.preferredReleaseType });
    return view;
  },

  async saveNetworkSettings() {
    set({ isSavingNetwork: true, error: "" });
    try {
      const view = await api.SaveNetworkSettings({
        fileConcurrency: get().draftFileConcurrency,
        concurrentDownloads: get().draftConcurrentDownloads,
        adaptiveFileConcurrency: get().draftAdaptiveFileConcurrency,
        targetDownloadRateMiB: get().draftTargetDownloadRateMiB,
        verifySha1: get().draftVerifySHA1,
        requestsPerSecond: get().draftRequestsPerSecond,
      });
      set({
        settings: view,
        draftFileConcurrency: view.fileConcurrency,
        draftConcurrentDownloads: view.concurrentDownloads,
        draftAdaptiveFileConcurrency: view.adaptiveFileConcurrency,
        draftTargetDownloadRateMiB: view.targetDownloadRateMiB,
        draftVerifySHA1: view.verifySha1,
        draftRequestsPerSecond: view.requestsPerSecond,
      });
      return view;
    } finally {
      set({ isSavingNetwork: false });
    }
  },

  cancelApiKeyAutoSave() {
    const pending = autoSaveTimers.get("keys");
    if (pending) {
      window.clearTimeout(pending);
      autoSaveTimers.delete("keys");
    }
  },

  async saveApiKeys() {
    // An explicit save supersedes any pending autosave; otherwise the timer
    // fires a second request with the already-cleared draft.
    get().cancelApiKeyAutoSave();
    const curseforgeDraft = get().draftCurseforgeKey;
    const modrinthDraft = get().draftModrinthKey;
    const clearCurseforge = get().clearCurseforgeKey;
    const clearModrinth = get().clearModrinthKey;
    set({ isSavingKeys: true, error: "" });
    try {
      const view = await api.SaveApiKeys({
        curseforgeApiKey: clearCurseforge ? "" : curseforgeDraft || apiKeyKeepSentinel,
        modrinthApiKey: clearModrinth ? "" : modrinthDraft || apiKeyKeepSentinel,
      });
      set((state) => ({
        settings: view,
        draftCurseforgeKey:
          state.draftCurseforgeKey === curseforgeDraft ? "" : state.draftCurseforgeKey,
        draftModrinthKey:
          state.draftModrinthKey === modrinthDraft ? "" : state.draftModrinthKey,
        clearCurseforgeKey:
          state.clearCurseforgeKey === clearCurseforge ? false : state.clearCurseforgeKey,
        clearModrinthKey:
          state.clearModrinthKey === clearModrinth ? false : state.clearModrinthKey,
      }));
      return view;
    } finally {
      set({ isSavingKeys: false });
    }
  },

  async chooseCacheDir() {
    set({ isChoosingCacheDir: true, error: "" });
    try {
      const previous = get().settings?.cacheDir || "";
      const view = await api.ChooseCacheDir(currentLocale());
      set({ settings: view });
      return (view?.cacheDir || "") !== previous;
    } finally {
      set({ isChoosingCacheDir: false });
    }
  },

  async resetCacheDir() {
    set({ isSavingCacheDir: true, error: "" });
    try {
      const view = await api.SaveCacheDirPreference("");
      set({ settings: view });
      return view;
    } finally {
      set({ isSavingCacheDir: false });
    }
  },
}));

export const selectHasCurseforgeKey = (state: SettingsState) => Boolean(state.settings?.hasCurseforgeKey);
export const selectHasModrinthKey = (state: SettingsState) => Boolean(state.settings?.hasModrinthKey);
