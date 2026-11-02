import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/services/app", () => ({
  api: {
    GetLocalModView: vi.fn(),
    ScanUnusedDependencies: vi.fn(),
    OpenLocalModLocation: vi.fn(),
    GetMinecraftDir: vi.fn(),
    GetVersions: vi.fn(),
    GetMinecraftReleaseVersions: vi.fn(),
    GetSelectedVersion: vi.fn(),
    RefreshSelectedVersionMods: vi.fn(),
    GetSettings: vi.fn(),
  },
}));

import { api } from "@/services/app";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ManagePage } from "@/pages/Manage";
import { useMinecraftStore } from "@/stores/minecraft";
import { useSettingsStore } from "@/stores/settings";
import type { LocalModGroup, LocalModView, ModInfo, SettingsView } from "@/types";

function group(rowKey: string, name: string, incompatible = false): LocalModGroup {
  const primary = {
    id: rowKey,
    name,
    path: `/mods/${rowKey}.jar`,
    fileName: `${rowKey}.jar`,
    version: "1.0.0",
    enabled: true,
  } as unknown as ModInfo;
  return {
    primary,
    strong: [primary],
    displayName: name,
    searchText: name,
    incompatible,
    rowKey,
    canReplaceVersion: false,
  } as unknown as LocalModGroup;
}

const settingsView = {
  theme: "dark",
  language: "en",
  simpleMode: false,
  animationDurationMultiplier: 1,
  autoScanUnusedDependencies: false,
  mcimEnabled: false,
  enableJijDetection: true,
  hardwareAcceleration: true,
  requireCtrlForMultiSelect: false,
  preferredReleaseType: "latest",
  minecraftDir: "/tmp/.minecraft",
  cacheDir: "",
  cachePath: "/tmp/cache",
  hasCurseforgeKey: false,
  curseforgeKeyMask: "",
  hasModrinthKey: false,
  modrinthKeyMask: "",
  fileConcurrency: 4,
  concurrentDownloads: 1,
  adaptiveFileConcurrency: false,
  targetDownloadRateMiB: 1,
  verifySha1: false,
  requestsPerSecond: 0,
} satisfies SettingsView;

describe("ManagePage selection guard", () => {
  beforeEach(() => {
    vi.mocked(api.GetLocalModView).mockResolvedValue({
      groups: [group("alpha", "Alpha Mod"), group("beta", "Beta Mod"), group("gamma", "Gamma Mod", true)],
      graph: { roots: [], missing: [] },
      dependents: {},
    } as unknown as LocalModView);
    useMinecraftStore.setState({
      dir: "/tmp/.minecraft",
      minecraftVersion: "1.20.1",
      modLoader: "fabric",
      selected: {
        id: "inst-1",
        name: "Inst",
        minecraftVersion: "1.20.1",
        modLoader: "fabric",
        mods: [],
      } as never,
      start: vi.fn().mockResolvedValue(undefined),
      ensureSelectedModsLoaded: vi.fn().mockResolvedValue(undefined),
    });
    useSettingsStore.setState({
      settings: settingsView,
      load: vi.fn().mockResolvedValue(undefined),
      draftRequireCtrlForMultiSelect: false,
    });
  });

  it("drops hidden rows from the batch selection when a filter hides them", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <TooltipProvider>
          <ManagePage />
        </TooltipProvider>
      </MemoryRouter>,
    );

    const alpha = await screen.findByText("Alpha Mod");
    const beta = screen.getByText("Beta Mod");
    await user.click(alpha.closest(".md-row-interactive") as HTMLElement);
    await user.click(beta.closest(".md-row-interactive") as HTMLElement);
    expect(screen.getByRole("toolbar")).toHaveAttribute("aria-label", "2 selected");

    // Hide every enabled row through the status filter.
    await user.click(screen.getByRole("combobox", { name: "All" }));
    await user.click(await screen.findByRole("option", { name: "Disabled" }));
    await waitFor(() => expect(screen.queryByText("Alpha Mod")).not.toBeInTheDocument());

    // The selection must not survive the filter that hid the rows.
    await waitFor(() => expect(screen.queryByRole("toolbar")).not.toBeInTheDocument(), { timeout: 2000 });

    // Restoring the filter shows the rows again, still unselected.
    await user.click(screen.getByRole("combobox", { name: "All" }));
    await user.click(await screen.findByRole("option", { name: "All" }));
    const restored = await screen.findByText("Alpha Mod");
    expect(restored.closest(".md-row-interactive")).not.toHaveClass("border-primary/50");
  });
});
