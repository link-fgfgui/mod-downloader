import { defineStore } from "pinia";

import {
    CancelDownload,
    ClearOptionalDependencyReminders,
    DismissOptionalDependencyReminder,
    GetDownloadQueueState,
    InstallOptionalDependencies,
    RetryDownload,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { structs } from "../../wailsjs/go/models";

const downloadQueueUpdatedEvent = "download-queue-updated";

type DownloadQueueSnapshot = Pick<structs.DownloadQueueState, "active" | "pending" | "running" | "messageCount" | "items" | "optionalReminders">;

const emptyQueue = (): DownloadQueueSnapshot => ({
    active: false,
    pending: 0,
    running: 0,
    messageCount: 0,
});

export const useDownloadQueueStore = defineStore("downloadQueue", {
    state: () => ({
        queue: emptyQueue(),
        stopListening: null as (() => void) | null,
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
            if (this.stopListening) {
                return;
            }
            await this.refresh();
            this.stopListening = EventsOn(downloadQueueUpdatedEvent, (state: DownloadQueueSnapshot) => {
                this.queue = state || emptyQueue();
            });
        },
        stop() {
            this.stopListening?.();
            this.stopListening = null;
        },
    },
});
