import { beforeEach, describe, expect, it, vi } from "vitest";

import type { SettingsView } from "@/types";

vi.mock("@/services/app", () => ({
  api: {
    GetSettings: vi.fn(),
    SaveTheme: vi.fn(),
    SaveLanguage: vi.fn(),
    SaveAnimationSettings: vi.fn(),
    SaveSimpleModeSettings: vi.fn(),
    SaveUnusedDependencyCleanupSettings: vi.fn(),
    SaveMCIMSettings: vi.fn(),
    SaveJijDetectionSettings: vi.fn(),
    SaveHardwareAccelerationSettings: vi.fn(),
    SavePreferredReleaseTypeSettings: vi.fn(),
    SaveNetworkSettings: vi.fn(),
    SaveApiKeys: vi.fn(),
    ChooseCacheDir: vi.fn(),
    SaveCacheDirPreference: vi.fn(),
  },
}));

import { api } from "@/services/app";
import { apiKeyKeepSentinel, useSettingsStore } from "@/stores/settings";

const view: SettingsView = {
  theme: "light",
  language: "zh",
  simpleMode: false,
  animationDurationMultiplier: 1.5,
  autoScanUnusedDependencies: true,
  mcimEnabled: true,
  enableJijDetection: false,
  hardwareAcceleration: true,
  requireCtrlForMultiSelect: false,
  preferredReleaseType: "release",
  minecraftDir: "/tmp/.minecraft",
  cacheDir: "",
  cachePath: "/tmp/cache",
  hasCurseforgeKey: true,
  curseforgeKeyMask: "****abcd",
  hasModrinthKey: false,
  modrinthKeyMask: "",
  fileConcurrency: 8,
  concurrentDownloads: 2,
  adaptiveFileConcurrency: true,
  targetDownloadRateMiB: 2.5,
  verifySha1: true,
  requestsPerSecond: 6,
};

describe("settings store", () => {
  beforeEach(() => {
    useSettingsStore.setState({
      settings: null,
      loading: false,
      saving: false,
      autoSaveState: "idle",
      autoSaveError: "",
      error: "",
      draftTheme: "dark",
      draftLanguage: "system",
      draftCurseforgeKey: "",
      draftModrinthKey: "",
      clearCurseforgeKey: false,
      clearModrinthKey: false,
    });
  });

  it("loads settings into draft fields", async () => {
    vi.mocked(api.GetSettings).mockResolvedValue(view);
    await useSettingsStore.getState().load();
    const state = useSettingsStore.getState();
    expect(state.settings).toEqual(view);
    expect(state.draftTheme).toBe("light");
    expect(state.draftLanguage).toBe("zh");
    expect(state.draftAnimationDurationMultiplier).toBe(1.5);
    expect(state.draftFileConcurrency).toBe(8);
    expect(state.draftAutoScanUnusedDependencies).toBe(true);
    expect(state.draftEnableJijDetection).toBe(false);
  });

  it("persists network drafts in one request", async () => {
    vi.mocked(api.SaveNetworkSettings).mockResolvedValue(view);
    useSettingsStore.setState({
      settings: view,
      draftFileConcurrency: 12,
      draftConcurrentDownloads: 3,
      draftVerifySHA1: false,
    });
    await useSettingsStore.getState().saveNetworkSettings();
    expect(api.SaveNetworkSettings).toHaveBeenCalledWith({
      fileConcurrency: 12,
      concurrentDownloads: 3,
      adaptiveFileConcurrency: true,
      targetDownloadRateMiB: 2.5,
      verifySha1: false,
      requestsPerSecond: 6,
    });
  });

  it("keeps existing API keys with the sentinel", async () => {
    vi.mocked(api.SaveApiKeys).mockResolvedValue(view);
    useSettingsStore.setState({ settings: view, draftCurseforgeKey: "", draftModrinthKey: "new-key" });
    await useSettingsStore.getState().saveApiKeys();
    expect(api.SaveApiKeys).toHaveBeenCalledWith({
      curseforgeApiKey: apiKeyKeepSentinel,
      modrinthApiKey: "new-key",
    });
    expect(useSettingsStore.getState().draftModrinthKey).toBe("");
  });
});
