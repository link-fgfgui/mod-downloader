import { defineStore } from "pinia";

import {
    ChooseMinecraftDir,
    GetMinecraftDir,
    GetMinecraftReleaseVersions,
    GetSelectedVersion,
    GetVersions,
    RefreshSelectedVersionMods,
    RefreshVersions,
    SelectVersion,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { structs } from "../../wailsjs/go/models";

const minecraftDirChangedEvent = "minecraft-dir-changed";
const selectedVersionChangedEvent = "selected-version-changed";

type VersionInfoSnapshot = Partial<structs.VersionInfo> & Record<string, any>;

const valueOf = (source: VersionInfoSnapshot | null, lowerKey: string, upperKey: string) =>
    source?.[lowerKey] || source?.[upperKey] || "";

const defaultModLoader = "Fabric";
const modLoaderOptions = ["Fabric", "Forge", "NeoForge"];

const matchingModLoader = (value: string) => {
    const normalized = value.trim().toLowerCase();
    return modLoaderOptions.find((item) => item.toLowerCase() === normalized) || value.trim();
};

export const useMinecraftStore = defineStore("minecraft", {
    state: () => ({
        selectedVersion: null as VersionInfoSnapshot | null,
        selectedMinecraftVersion: "",
        selectedModLoader: defaultModLoader,
        versions: [] as Array<string | VersionInfoSnapshot>,
        releaseVersions: [] as string[],
        modLoaderList: modLoaderOptions,
        minecraftDir: "",
        isRefreshing: false,
        isLoading: false,
        stopListeningMinecraftDirChanged: null as (() => void) | null,
        stopListeningSelectedVersionChanged: null as (() => void) | null,
    }),
    getters: {
        hasSelectedInstance: (state) => Boolean(valueOf(state.selectedVersion, "name", "Name") || valueOf(state.selectedVersion, "id", "ID")),
        selectedInstanceLabel: (state) => {
            const name = valueOf(state.selectedVersion, "name", "Name") || valueOf(state.selectedVersion, "id", "ID");
            const minecraftVersion = state.selectedMinecraftVersion;
            const modLoader = state.selectedModLoader;
            return name ? [name, minecraftVersion, modLoader].filter(Boolean).join(" / ") : "";
        },
        mods: (state) => state.selectedVersion?.mods || state.selectedVersion?.Mods || [],
    },
    actions: {
        async refreshMinecraftDir() {
            this.minecraftDir = await GetMinecraftDir();
        },
        async refreshVersions(force = false) {
            this.isRefreshing = true;
            try {
                const versions = (force ? await RefreshVersions() : await GetVersions()) || [];
                this.versions = versions;
                this.releaseVersions = await GetMinecraftReleaseVersions();
                this.applySelectedVersion(await GetSelectedVersion());
            } finally {
                this.isRefreshing = false;
            }
        },
        async refreshSelectedMods() {
            if (this.isRefreshing) return this.selectedVersion;
            this.isRefreshing = true;
            try {
                const version = await RefreshSelectedVersionMods();
                this.applySelectedVersion(version);
                return version;
            } finally {
                this.isRefreshing = false;
            }
        },
        async selectVersion(version: string) {
            this.isLoading = true;
            try {
                this.applySelectedVersion(await SelectVersion(version));
            } finally {
                this.isLoading = false;
            }
        },
        async chooseMinecraftDir() {
            const result = await ChooseMinecraftDir();
            if (result) {
                this.minecraftDir = result;
                this.isLoading = true;
                try {
                    await this.refreshVersions(true);
                } finally {
                    this.isLoading = false;
                }
            }
            return result;
        },
        applySelectedVersion(version: VersionInfoSnapshot | null) {
            this.selectedVersion = version;
            const minecraftVersion = valueOf(version, "minecraftVersion", "MinecraftVersion");
            if (minecraftVersion) {
                if (!this.releaseVersions.includes(minecraftVersion)) {
                    this.releaseVersions = [minecraftVersion, ...this.releaseVersions];
                }
                this.selectedMinecraftVersion = minecraftVersion;
            }

            const modLoader = valueOf(version, "modLoader", "ModLoader");
            if (modLoader) {
                this.selectedModLoader = matchingModLoader(modLoader);
            }
        },
        setSelectedMinecraftVersion(version: string) {
            this.selectedMinecraftVersion = version || "";
            this.selectedVersion = null;
        },
        setSelectedModLoader(modLoader: string) {
            this.selectedModLoader = matchingModLoader(modLoader || "");
            this.selectedVersion = null;
        },
        async start() {
            if (this.stopListeningMinecraftDirChanged || this.stopListeningSelectedVersionChanged) {
                return;
            }
            this.stopListeningMinecraftDirChanged = EventsOn(minecraftDirChangedEvent, async () => {
                await this.refreshMinecraftDir();
                await this.refreshVersions();
            });
            this.stopListeningSelectedVersionChanged = EventsOn(selectedVersionChangedEvent, this.applySelectedVersion);
            await this.refreshMinecraftDir();
            await this.refreshVersions();
            this.applySelectedVersion(await GetSelectedVersion());
        },
        stop() {
            this.stopListeningMinecraftDirChanged?.();
            this.stopListeningSelectedVersionChanged?.();
            this.stopListeningMinecraftDirChanged = null;
            this.stopListeningSelectedVersionChanged = null;
        },
    },
});
