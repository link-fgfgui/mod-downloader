import { defineStore } from "pinia";

import { GetDownloadQueueState } from "../../wailsjs/go/main/App";
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
    actions: {
        async refresh() {
            this.queue = (await GetDownloadQueueState()) || emptyQueue();
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
