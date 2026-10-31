import { create } from "zustand";
import { api } from "@/services/app";
import type { PinnedMod } from "@/types";

interface PinnedState {
  pins: PinnedMod[];
  loading: boolean;
  error: string;
  refresh: () => Promise<void>;
  unpin: (pin: PinnedMod) => Promise<boolean>;
  unpinAll: () => Promise<number>;
}

export const usePinnedModsStore = create<PinnedState>((set, get) => ({
  pins: [], loading: false, error: "",

  async refresh() {
    set({ loading: true, error: "" });
    try {
      set({ pins: (await api.ListPinnedMods()) ?? [] });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
    } finally {
      set({ loading: false });
    }
  },

  async unpin(pin) {
    const removed = await api.UnpinMod(pin.platform, pin.modId, pin.minecraftVersion, pin.modLoader);
    await get().refresh();
    return removed;
  },

  async unpinAll() {
    let failed = 0;
    for (const pin of get().pins) {
      if (!(await api.UnpinMod(pin.platform, pin.modId, pin.minecraftVersion, pin.modLoader))) failed++;
    }
    await get().refresh();
    return failed;
  },
}));
