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
    SaveJIJDetectionSettings: vi.fn(),
    SaveHardwareAccelerationSettings: vi.fn(),
    SaveMultiSelectSettings: vi.fn(),
    SavePreferredReleaseTypeSettings: vi.fn(),
    SaveNetworkSettings: vi.fn(),
    SaveAPIKeys: vi.fn(),
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
      draftRequireCtrlForMultiSelect: false,
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
    vi.mocked(api.SaveAPIKeys).mockResolvedValue(view);
    useSettingsStore.setState({ settings: view, draftCurseforgeKey: "", draftModrinthKey: "new-key" });
    await useSettingsStore.getState().saveApiKeys();
    expect(api.SaveAPIKeys).toHaveBeenCalledWith({
      curseforgeApiKey: apiKeyKeepSentinel,
      modrinthApiKey: "new-key",
    });
    expect(useSettingsStore.getState().draftModrinthKey).toBe("");
  });

  it("persists the multi-select preference and mirrors the saved view", async () => {
    vi.mocked(api.SaveMultiSelectSettings).mockResolvedValue({ ...view, requireCtrlForMultiSelect: true });
    useSettingsStore.setState({ settings: view, draftRequireCtrlForMultiSelect: true });
    await useSettingsStore.getState().saveMultiSelectSettings();
    expect(api.SaveMultiSelectSettings).toHaveBeenCalledWith({ requireCtrlForMultiSelect: true });
    const state = useSettingsStore.getState();
    expect(state.draftRequireCtrlForMultiSelect).toBe(true);
    expect(state.settings?.requireCtrlForMultiSelect).toBe(true);
  });

  it("drops an in-flight save response after the draft is edited again", async () => {
    // The rerun-ui-notes.md autosave race: an older in-flight response must
    // not clobber a newer draft before the save it scheduled fires.
    let resolveFirst!: (value: SettingsView) => void;
    const first = new Promise<SettingsView>((resolve) => {
      resolveFirst = resolve;
    });
    vi.mocked(api.SaveSimpleModeSettings)
      .mockImplementationOnce(() => first as unknown as ReturnType<typeof api.SaveSimpleModeSettings>)
      .mockResolvedValueOnce({ ...view, simpleMode: false });

    useSettingsStore.setState({ settings: view, draftSimpleMode: true });
    const saving = useSettingsStore.getState().saveSimpleModeSettings();
    // The user flips the same switch again while the first save is in flight.
    useSettingsStore.getState().patch({ draftSimpleMode: false });
    resolveFirst({ ...view, simpleMode: true });
    await saving;
    // The stale response (simpleMode: true) must not overwrite the newer draft.
    expect(useSettingsStore.getState().draftSimpleMode).toBe(false);

    // The follow-up save persists the newer value and applies its response.
    await useSettingsStore.getState().saveSimpleModeSettings();
    expect(api.SaveSimpleModeSettings).toHaveBeenLastCalledWith({ simpleMode: false });
    expect(useSettingsStore.getState().draftSimpleMode).toBe(false);
    expect(useSettingsStore.getState().settings?.simpleMode).toBe(false);
  });
});
