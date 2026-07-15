import { defineStore } from "pinia";

import {
    CancelDownload,
    ClearOptionalDependencyReminders,
    DismissOptionalDependencyReminder,
    GetDownloadQueueState,
    InstallOptionalDependencies,
    RemoveCanceledDownload,
    RetryDownload,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { structs } from "../../wailsjs/go/models";
import {
    appIsUnfocused,
    playDownloadCompletionSound,
    prepareDownloadCompletionSound,
} from "../composables/useDownloadCompletionSound";

const downloadQueueUpdatedEvent = "download-queue-updated";
const downloadCompletedEvent = "download-completed";

type DownloadQueueSnapshot = Pick<
    structs.DownloadQueueState,
    "active" | "pending" | "running" | "messageCount" | "bytesComplete" | "totalBytes" | "bytesPerSecond" | "items" | "optionalReminders"
>;

const emptyQueue = (): DownloadQueueSnapshot => ({
    active: false,
    pending: 0,
    running: 0,
    messageCount: 0,
    bytesComplete: 0,
    totalBytes: 0,
    bytesPerSecond: 0,
});

export const useDownloadQueueStore = defineStore("downloadQueue", {
    state: () => ({
        queue: emptyQueue(),
        stopListening: null as (() => void) | null,
        stopListeningCompleted: null as (() => void) | null,
        stopPreparingSound: null as (() => void) | null,
        completedInActiveCycle: false,
    }),
    getters: {
        activeCount: (state) => (state.queue.pending || 0) + (state.queue.running || 0),
        messageCount: (state) => state.queue.messageCount || state.queue.optionalReminders?.length || 0,
        items: (state) => state.queue.items || [],
        optionalReminders: (state) => state.queue.optionalReminders || [],
        hasVisibleItems: (state) => Boolean(state.queue.items?.length || state.queue.optionalReminders?.length),
        hasAnyQueueSurface(): boolean {
            return Boolean(this.queue.active || this.messageCount);
        },
        reminderOnly(): boolean {
            return Boolean(!this.activeCount && this.messageCount);
        },
    },
    actions: {
        async refresh() {
            this.queue = (await GetDownloadQueueState()) || emptyQueue();
        },
        async cancel(id: string) {
            if (!id) {
                return false;
            }
            const canceled = await CancelDownload(id);
            if (canceled) {
                await this.refresh();
            }
            return canceled;
        },
        async retry(id: string) {
            if (!id) {
                return false;
            }
            const retried = await RetryDownload(id);
            if (retried) {
                await this.refresh();
            }
            return retried;
        },
        async removeCanceled(id: string) {
            if (!id) {
                return false;
            }
            const removed = await RemoveCanceledDownload(id);
            if (removed) {
                await this.refresh();
            }
            return removed;
        },
        async dismissOptionalReminder(id: string) {
            if (!id) {
                return false;
            }
            const dismissed = await DismissOptionalDependencyReminder(id);
            if (dismissed) {
                await this.refresh();
            }
            return dismissed;
        },
        async clearOptionalReminders() {
            const cleared = await ClearOptionalDependencyReminders();
            if (cleared) {
                await this.refresh();
            }
            return cleared;
        },
        async installOptionalDependencies(id: string) {
            if (!id) {
                return [];
            }
            const results = await InstallOptionalDependencies(id);
            await this.refresh();
            return results || [];
        },
        async start() {
            if (this.stopListening || this.stopListeningCompleted) {
                return;
            }
            await this.refresh();
            this.stopListening = EventsOn(downloadQueueUpdatedEvent, (state: DownloadQueueSnapshot) => {
                const previousActive = Boolean(this.queue.active);
                const nextQueue = state || emptyQueue();
                this.queue = nextQueue;
                if (previousActive && !nextQueue.active) {
                    const shouldPlay = this.completedInActiveCycle && appIsUnfocused();
                    this.completedInActiveCycle = false;
                    if (shouldPlay) void playDownloadCompletionSound();
                }
            });
            this.stopListeningCompleted = EventsOn(downloadCompletedEvent, () => {
                this.completedInActiveCycle = true;
            });
            const prepareSound = () => {
                void prepareDownloadCompletionSound();
                this.stopPreparingSound?.();
                this.stopPreparingSound = null;
            };
            document.addEventListener("pointerdown", prepareSound, { once: true });
            document.addEventListener("keydown", prepareSound, { once: true });
            this.stopPreparingSound = () => {
                document.removeEventListener("pointerdown", prepareSound);
                document.removeEventListener("keydown", prepareSound);
            };
        },
        stop() {
            this.stopListening?.();
            this.stopListeningCompleted?.();
            this.stopPreparingSound?.();
            this.stopListening = null;
            this.stopListeningCompleted = null;
            this.stopPreparingSound = null;
            this.completedInActiveCycle = false;
        },
    },
});
