import { defineStore } from "pinia";

import { CancelDownload, GetDownloadQueueState, RetryDownload } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { structs } from "../../wailsjs/go/models";

const downloadQueueUpdatedEvent = "download-queue-updated";

type DownloadQueueSnapshot = Pick<structs.DownloadQueueState, "active" | "pending" | "running" | "items">;

const emptyQueue = (): DownloadQueueSnapshot => ({
    active: false,
    pending: 0,
    running: 0,
});

export const useDownloadQueueStore = defineStore("downloadQueue", {
    state: () => ({
        queue: emptyQueue(),
        stopListening: null as (() => void) | null,
    }),
    getters: {
        activeCount: (state) => (state.queue.pending || 0) + (state.queue.running || 0),
        items: (state) => state.queue.items || [],
        hasVisibleItems: (state) => Boolean(state.queue.items?.length),
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
