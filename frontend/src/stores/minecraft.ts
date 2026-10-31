import { create } from "zustand";
import { toast } from "sonner";

import { api } from "@/services/app";
import { appEventNames, onAppEvent } from "@/services/events";
import i18n, { currentLocale } from "@/i18n";
import type { VersionInfo } from "@/types";

export const defaultModLoader = "Fabric";
export const modLoaderOptions = ["Fabric", "Forge", "NeoForge"];

type VersionInfoSnapshot = Partial<VersionInfo> & {
  ID?: string;
  Name?: string;
  MinecraftVersion?: string;
  ModLoader?: string;
};

type VersionField = "id" | "name" | "minecraftVersion" | "modLoader";

const UPPER_KEYS: Record<VersionField, string> = {
  id: "ID",
  name: "Name",
  minecraftVersion: "MinecraftVersion",
  modLoader: "ModLoader",
};

const valueOf = (source: VersionInfoSnapshot | null | undefined, lowerKey: VersionField) => {
  const upperKey = UPPER_KEYS[lowerKey];
  const value = source?.[lowerKey] ?? (upperKey ? source?.[upperKey as keyof VersionInfoSnapshot] : undefined);
  return typeof value === "string" ? value : "";
};

const selectedModsKey = (minecraftDir: string, version: VersionInfoSnapshot | null) => {
  const versionKey = valueOf(version, "id") || valueOf(version, "name");
  return minecraftDir && versionKey ? `${minecraftDir}\u0000${versionKey}` : "";
};

const matchingModLoader = (value: string) => {
  const normalized = value.trim().toLowerCase();
  return modLoaderOptions.find((item) => item.toLowerCase() === normalized) || value.trim();
};

interface MinecraftState {
  dir: string;
  versions: VersionInfo[];
  releases: string[];
  selected: VersionInfo | null;
  minecraftVersion: string;
  modLoader: string;
  loading: boolean;
  refreshing: boolean;
  error: string;
  start: () => Promise<void>;
  stop: () => void;
  refreshDir: () => Promise<void>;
  refreshVersions: (force?: boolean) => Promise<void>;
  refreshSelected: () => Promise<VersionInfo | null>;
  ensureSelectedModsLoaded: () => Promise<VersionInfo | null>;
  chooseDir: () => Promise<string>;
  selectVersion: (id: string) => Promise<void>;
  toggleConnector: () => Promise<void>;
  setMinecraftVersion: (version: string) => void;
  setModLoader: (loader: string) => void;
  applySelectedVersion: (version: VersionInfo | null | undefined) => void;
}

let subscriptions: Array<() => void> = [];
let starting: Promise<void> | null = null;
let initializedSelectedModsKey = "";

function applySelectedVersion(
  selected: VersionInfo | null | undefined,
  set: (patch: Partial<MinecraftState>) => void,
) {
  const version = (selected ?? null) as VersionInfo | null;
  const minecraftVersion = valueOf(version, "minecraftVersion");
  const modLoader = valueOf(version, "modLoader");
  set({
    selected: version,
    ...(minecraftVersion ? { minecraftVersion } : {}),
    ...(modLoader ? { modLoader: matchingModLoader(modLoader) } : {}),
  });
}

export const useMinecraftStore = create<MinecraftState>((set, get) => ({
  dir: "",
  versions: [],
  releases: [],
  selected: null,
  minecraftVersion: "",
  modLoader: defaultModLoader,
  loading: false,
  refreshing: false,
  error: "",

  async start() {
    if (subscriptions.length) return;
    if (starting) return starting;
    starting = (async () => {
      set({ loading: true, error: "" });
      subscriptions = [
        onAppEvent(appEventNames.MinecraftDirChanged, () => {
          void (async () => {
            await get().refreshDir();
            await get().refreshVersions();
            await get().ensureSelectedModsLoaded();
          })();
        }),
        onAppEvent(appEventNames.SelectedVersionChanged, (selected) => {
          applySelectedVersion(selected, set);
        }),
      ];
      try {
        await get().refreshDir();
        await get().refreshVersions();
        await get().ensureSelectedModsLoaded();
      } catch (error) {
        set({ error: error instanceof Error ? error.message : String(error) });
      } finally {
        set({ loading: false });
        starting = null;
      }
    })();
    return starting;
  },

  stop() {
    subscriptions.forEach((unsubscribe) => unsubscribe());
    subscriptions = [];
    starting = null;
    initializedSelectedModsKey = "";
  },

  async refreshDir() {
    const dir = await api.GetMinecraftDir();
    set({ dir });
  },

  async refreshVersions(force = false) {
    set({ refreshing: true, error: "" });
    try {
      if (force) initializedSelectedModsKey = "";
      const [versions, releases, selected] = await Promise.all([
        force ? api.RefreshVersions() : api.GetVersions(),
        api.GetMinecraftReleaseVersions(),
        api.GetSelectedVersion(),
      ]);
      set({ versions: versions ?? [], releases: releases ?? [] });
      applySelectedVersion(selected, set);
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ refreshing: false });
    }
  },

  async refreshSelected() {
    if (get().refreshing) return get().selected;
    set({ refreshing: true, error: "" });
    try {
      const selected = await api.RefreshSelectedVersionMods();
      applySelectedVersion(selected, set);
      initializedSelectedModsKey = selectedModsKey(get().dir, selected);
      return selected ?? null;
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ refreshing: false });
    }
  },

  async ensureSelectedModsLoaded() {
    const key = selectedModsKey(get().dir, get().selected);
    if (!key || key === initializedSelectedModsKey) return get().selected;
    return get().refreshSelected();
  },

  async chooseDir() {
    set({ loading: true, error: "" });
    try {
      const result = await api.ChooseMinecraftDir(currentLocale());
      if (result.canceled) return "";
      if (result.invalid) {
        toast.error(i18n.t("sidebar.invalidDir"));
        return "";
      }
      if (!result.path) return "";
      set({ dir: result.path });
      await get().refreshVersions(true);
      await get().ensureSelectedModsLoaded();
      return result.path;
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      set({ error: message });
      toast.error(i18n.t("sidebar.dirPickerFailed", { reason: message }));
      return "";
    } finally {
      set({ loading: false });
    }
  },

  async selectVersion(id) {
    if (!id) return;
    set({ loading: true, error: "" });
    try {
      const selected = await api.SelectVersion(id);
      applySelectedVersion(selected, set);
      initializedSelectedModsKey = selectedModsKey(get().dir, selected);
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ loading: false });
    }
  },

  async toggleConnector() {
    if (!get().selected?.connectorAvailable || get().loading) return;
    set({ loading: true, error: "" });
    try {
      const selected = await api.ToggleConnectorLoader();
      applySelectedVersion(selected, set);
      initializedSelectedModsKey = selectedModsKey(get().dir, selected);
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
      throw error;
    } finally {
      set({ loading: false });
    }
  },

  setMinecraftVersion(version) {
    set({ minecraftVersion: version || "", selected: null });
  },

  setModLoader(loader) {
    set({ modLoader: matchingModLoader(loader || ""), selected: null });
  },

  applySelectedVersion(selected) {
    applySelectedVersion(selected, set);
  },
}));

export const selectHasSelectedInstance = (state: MinecraftState) =>
  Boolean(valueOf(state.selected, "name") || valueOf(state.selected, "id"));

export const selectSelectedInstanceLabel = (state: MinecraftState) => {
  const name = valueOf(state.selected, "name") || valueOf(state.selected, "id");
  return name ? [name, state.minecraftVersion, state.modLoader].filter(Boolean).join(" / ") : "";
};

export const selectConnectorDestination = (state: MinecraftState) => {
  const selected = state.selected;
  const loader = selected?.connectorVirtual ? selected.actualModLoader : "fabric";
  return (
    modLoaderOptions.find((item) => item.toLowerCase() === (loader || "").toLowerCase()) || loader || ""
  );
};
