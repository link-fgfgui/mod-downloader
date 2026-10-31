import { create } from "zustand";
import { toast } from "sonner";
import i18n from "@/i18n";
import { api } from "@/services/app";
import { appEventNames, onAppEvent } from "@/services/events";
import type { DownloadQueueState } from "@/types";
import { appIsUnfocused, playDownloadCompletionSound } from "@/utils/completionSound";

const emptyQueue: DownloadQueueState = {
  active: false, pending: 0, running: 0, messageCount: 0,
  bytesComplete: 0, totalBytes: 0, bytesPerSecond: 0, items: [], optionalReminders: [],
};

interface QueueState {
  queue: DownloadQueueState;
  start: () => Promise<void>;
  stop: () => void;
  refresh: () => Promise<void>;
  cancel: (id: string) => Promise<void>;
  retry: (id: string) => Promise<void>;
  remove: (id: string) => Promise<void>;
  dismissOptional: (id: string) => Promise<void>;
  clearOptional: () => Promise<void>;
  installOptional: (id: string) => Promise<void>;
}

let subscriptions: Array<() => void> = [];

export const useDownloadQueueStore = create<QueueState>((set, get) => ({
  queue: emptyQueue,

  async start() {
    if (subscriptions.length) return;
    subscriptions = [
      onAppEvent(appEventNames.DownloadQueueUpdated, (queue) => set({ queue })),
      onAppEvent(appEventNames.DownloadFailed, ({ fileName, reason }) => {
        toast.error(i18n.t("download.errors.failedWithReason", { fileName, reason }));
      }),
      onAppEvent(appEventNames.DownloadCompleted, () => {
        if (appIsUnfocused()) void playDownloadCompletionSound();
      }),
    ];
    await get().refresh();
  },

  stop() {
    subscriptions.forEach((unsubscribe) => unsubscribe());
    subscriptions = [];
  },

  async refresh() {
    try {
      set({ queue: await api.GetDownloadQueueState() });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    }
  },

  async cancel(id) { await api.CancelDownload(id); await get().refresh(); },
  async retry(id) { await api.RetryDownload(id); await get().refresh(); },
  async remove(id) { await api.RemoveDownload(id); await get().refresh(); },
  async dismissOptional(id) { await api.DismissOptionalDependencyReminder(id); await get().refresh(); },
  async clearOptional() { await api.ClearOptionalDependencyReminders(); await get().refresh(); },
  async installOptional(id) {
    const results = await api.InstallOptionalDependencies(id);
    const failed = (results ?? []).filter((result) => result.skipped);
    if (failed.length) toast.error(failed.map((result) => result.reason).join("\n"));
    await get().refresh();
  },
}));
