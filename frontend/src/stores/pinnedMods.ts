import { defineStore } from "pinia";
import { ListPinnedMods, UnpinMod } from "../../wailsjs/go/main/App";
import type { database } from "../../wailsjs/go/models";

export const usePinnedModsStore = defineStore("pinnedMods", {
    state: () => ({
        pins: [] as database.PinnedMod[],
        isLoading: false,
        filterPlatform: "",
        filterMinecraftVersion: "",
        filterModLoader: "",
        pendingUnpinKeys: new Set<string>(),
    }),
    getters: {
        filteredPins(state): database.PinnedMod[] {
            const platform = state.filterPlatform.toLowerCase();
            const mc = state.filterMinecraftVersion.toLowerCase();
            const loader = state.filterModLoader.toLowerCase();
            return state.pins.filter((pin) => {
                if (platform && !pin.platform.toLowerCase().includes(platform)) {
                    return false;
                }
                if (mc && !pin.minecraftVersion.toLowerCase().includes(mc)) {
                    return false;
                }
                if (loader && !pin.modLoader.toLowerCase().includes(loader)) {
                    return false;
                }
                return true;
            });
        },
        pinKey(): (pin: database.PinnedMod) => string {
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
        async unpin(pin: database.PinnedMod) {
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
        async unpinAllFiltered() {
            const targets = [...this.filteredPins];
            for (const pin of targets) {
                await this.unpin(pin);
            }
        },
    },
});
