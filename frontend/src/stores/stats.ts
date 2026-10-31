import { create } from "zustand";
import { api } from "@/services/app";
import { appEventNames, onAppEvent } from "@/services/events";
import type { UsageStats } from "@/types";

const emptyStats: UsageStats = {
  downloadsCompleted: 0, modsEnabled: 0, modsDisabled: 0,
  modsDeleted: 0, favoritesAdded: 0, packwizExports: 0,
};

interface StatsState {
  stats: UsageStats;
  loading: boolean;
  error: string;
  updatedAt: Date | null;
  refresh: () => Promise<void>;
  start: () => Promise<void>;
  stop: () => void;
}

let unsubscribe: (() => void) | null = null;

export const useStatsStore = create<StatsState>((set, get) => ({
  stats: emptyStats, loading: false, error: "", updatedAt: null,

  async refresh() {
    set({ loading: true, error: "" });
    try {
      set({ stats: await api.GetUsageStats(), updatedAt: new Date() });
    } catch (error) {
      set({ error: error instanceof Error ? error.message : String(error) });
    } finally {
      set({ loading: false });
    }
  },

  async start() {
    if (!unsubscribe) unsubscribe = onAppEvent(appEventNames.UsageStatsUpdated, (stats) => set({ stats, updatedAt: new Date() }));
    await get().refresh();
  },

  stop() {
    unsubscribe?.();
    unsubscribe = null;
  },
}));
