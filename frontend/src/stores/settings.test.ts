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

  it("treats a canceled cache-dir selection as no change", async () => {
    useSettingsStore.setState({ settings: { ...view, cacheDir: "/old" } });
    vi.mocked(api.ChooseCacheDir).mockResolvedValue({ settings: view, canceled: true });
    await expect(useSettingsStore.getState().chooseCacheDir()).resolves.toBe(false);
    const state = useSettingsStore.getState();
    expect(state.settings).toEqual(view);
    expect(state.error).toBe("");
  });

  it("surfaces a failed cache-dir switch", async () => {
    useSettingsStore.setState({ settings: { ...view, cacheDir: "/old" } });
    vi.mocked(api.ChooseCacheDir).mockResolvedValue({
      settings: { ...view, cacheDir: "/old" },
      canceled: false,
      error: "permission denied",
    });
    await expect(useSettingsStore.getState().chooseCacheDir()).resolves.toBe(false);
    expect(useSettingsStore.getState().error).toBe("permission denied");
  });

  it("rejects a failed cache-dir reset", async () => {
    vi.mocked(api.SaveCacheDirPreference).mockResolvedValue({ ...view, saveError: "disk full" });
    await expect(useSettingsStore.getState().resetCacheDir()).rejects.toThrow("disk full");
    expect(useSettingsStore.getState().error).toBe("disk full");
  });

  it("rejects a failed settings save and keeps the save error visible", async () => {
    vi.mocked(api.SaveMultiSelectSettings).mockResolvedValue({ ...view, saveError: "disk full" });
    useSettingsStore.setState({ settings: view, draftRequireCtrlForMultiSelect: true });
    await expect(useSettingsStore.getState().saveMultiSelectSettings()).rejects.toThrow("disk full");
    expect(useSettingsStore.getState().settings?.saveError).toBe("disk full");
  });

  it("rejects a failed theme save and reloads the persisted settings", async () => {
    vi.mocked(api.SaveTheme).mockRejectedValue(new Error("disk full"));
    vi.mocked(api.GetSettings).mockResolvedValue({ ...view, theme: "dark", saveError: "disk full" });
    useSettingsStore.setState({ settings: { ...view, theme: "dark" }, draftTheme: "light" });
    await expect(useSettingsStore.getState().saveTheme()).rejects.toThrow("disk full");
    const state = useSettingsStore.getState();
    expect(state.settings?.saveError).toBe("disk full");
    expect(state.draftTheme).toBe("dark");
  });

  it("rejects an autosave that returns SaveError", async () => {
    vi.mocked(api.SaveAnimationSettings).mockResolvedValue({ ...view, saveError: "disk full" });
    vi.mocked(api.GetSettings).mockResolvedValue({ ...view, saveError: "disk full" });
    useSettingsStore.setState({ settings: view, draftAnimationDurationMultiplier: 2 });
    await expect(useSettingsStore.getState().saveAnimationSettings()).rejects.toThrow("disk full");
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
