import { defineStore } from "pinia";
import { ListPinnedMods, UnpinMod } from "../../wailsjs/go/main/App";
import type { storage } from "../../wailsjs/go/models";

export const usePinnedModsStore = defineStore("pinnedMods", {
    state: () => ({
        pins: [] as storage.PinnedMod[],
        isLoading: false,
        pendingUnpinKeys: new Set<string>(),
    }),
    getters: {
        pinKey(): (pin: storage.PinnedMod) => string {
            return (pin) => [pin.platform, pin.modId, pin.minecraftVersion, pin.modLoader].join("|");
        },
        hasPins(state): boolean {
            return state.pins.length > 0;
        },
    },
    actions: {
        async load() {
            this.isLoading = true;
            try {
                this.pins = (await ListPinnedMods()) || [];
            } finally {
                this.isLoading = false;
            }
        },
        async unpin(pin: storage.PinnedMod) {
            const key = this.pinKey(pin);
            if (this.pendingUnpinKeys.has(key)) return;
            this.pendingUnpinKeys.add(key);
            try {
                const ok = await UnpinMod(pin.platform, pin.modId, pin.minecraftVersion, pin.modLoader);
                if (ok) {
                    this.pins = this.pins.filter((p) => this.pinKey(p) !== key);
                } else {
                    await this.load();
                }
                return ok;
            } finally {
                this.pendingUnpinKeys.delete(key);
            }
        },
        async unpinAll() {
            const targets = [...this.pins];
            for (const pin of targets) {
                await this.unpin(pin);
            }
        },
    },
});
