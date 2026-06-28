import { defineStore } from "pinia";

import {
    GetDownloadStates,
    GetPinnedModVersion,
    ListMatchingProjectVersions,
    PinModVersion,
    QueueModDownload,
    SearchMods,
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import type { structs, models } from "../../wailsjs/go/models";

const searchModsUpdatedEvent = "search-mods-updated";
const downloadQueueUpdatedEvent = "download-queue-updated";
const downloadFailedEvent = "download-failed";

const searchPageSize = 10;

type VersionInfoSnapshot = Partial<structs.VersionInfo> & Record<string, any>;
type SearchModSnapshot = models.ModProject;
type ProjectVersionSnapshot = models.ModVersion;
type DownloadStateSnapshot = structs.ModDownloadButtonState;

const downloadErrorKeys: Record<string, string> = {
    "invalid request": "invalidRequest",
    "no matching version": "noMatchingVersion",
    "missing download url": "missingDownloadUrl",
    "no selected version": "noSelectedVersion",
};

export const useDownloadSearchStore = defineStore("downloadSearch", {
    state: () => ({
        searchText: "",
        selectedVersion: "",
        selectedModLoader: "Fabric",
        showDirOverlay: false,
        isSearching: false,
        isLoadingMore: false,
        hasMoreResults: true,
        showVersionsOverlay: false,
        isLoadingVersions: false,
        versionList: [] as string[],
        modLoaderList: ["Fabric", "Forge", "NeoForge"],
        searchResults: [] as SearchModSnapshot[],
        activeSearchRequestID: "",
        nextSearchOffset: 0,
        activeSearchAppend: false,
        appendBaseResults: [] as SearchModSnapshot[],
        selectedMod: null as SearchModSnapshot | null,
        matchingVersions: [] as ProjectVersionSnapshot[],
        pinnedVersion: null as null | { versionId?: string },
        pinningVersionID: "",
        downloadStates: [] as DownloadStateSnapshot[],
        activeDownloadStateRequestID: "",
        downloadingKeys: {} as Record<string, boolean>,
        snackbar: { show: false, key: "", params: {} as Record<string, string>, color: "success" },
        confirmDialog: { show: false, status: "", result: null as SearchModSnapshot | null, key: "" },
        stopListeningSearchModsUpdated: null as (() => void) | null,
        stopListeningDownloadQueueUpdated: null as (() => void) | null,
        stopListeningDownloadFailed: null as (() => void) | null,
    }),
    getters: {
        confirmStatuses: () => new Set(["update", "conflict"]),
    },
    actions: {
        showSnackbar(key: string, color = "success", params: Record<string, string> = {}) {
            this.snackbar = { show: true, key, params, color };
        },
        setVersionList(versions: string[]) {
            this.versionList = versions;
            if (!this.selectedVersion && versions.length > 0) {
                this.selectedVersion = versions[0];
            }
        },
        applySelectedInstance(version: VersionInfoSnapshot | null) {
            const minecraftVersion = version?.minecraftVersion || version?.MinecraftVersion;
            if (minecraftVersion) {
                if (!this.versionList.includes(minecraftVersion)) {
                    this.versionList = [minecraftVersion, ...this.versionList];
                }
                this.selectedVersion = minecraftVersion;
            }

            const modLoader = version?.modLoader || version?.ModLoader;
            if (modLoader) {
                const normalizedModLoader = modLoader.toLowerCase();
                const matchingModLoader = this.modLoaderList.find((item) => item.toLowerCase() === normalizedModLoader);
                if (matchingModLoader) {
                    this.selectedModLoader = matchingModLoader;
                }
            }
        },
        clearDownloadStates() {
            this.downloadStates = [];
            void this.refreshDownloadStates();
        },
        async refreshDownloadStates() {
            if (this.isSearching || this.isLoadingMore) {
                return;
            }
            const requestID = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
            this.activeDownloadStateRequestID = requestID;
            const results = this.searchResults || [];
            const states = (await GetDownloadStates({
                results,
                minecraftVersion: this.selectedVersion,
                modLoader: this.selectedModLoader,
            } as structs.DownloadStatesRequest)) || [];
            if (this.activeDownloadStateRequestID !== requestID) {
                return;
            }
            this.downloadStates = states;
        },
        async runSearch(options: { append?: boolean } = {}) {
            const append = Boolean(options.append);
            if (append && (this.isSearching || this.isLoadingMore || !this.hasMoreResults)) {
                return;
            }

            const requestID = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
            this.activeSearchRequestID = requestID;
            this.activeSearchAppend = append;
            const offset = append ? this.nextSearchOffset : 0;
            if (append) {
                this.appendBaseResults = [...this.searchResults];
                this.isLoadingMore = true;
            } else {
                this.searchResults = [];
                this.downloadStates = [];
                this.appendBaseResults = [];
                this.nextSearchOffset = 0;
                this.hasMoreResults = true;
                this.isSearching = true;
            }

            try {
                await SearchMods({
                    requestId: requestID,
                    query: this.searchText,
                    version: this.selectedVersion,
                    modLoader: this.selectedModLoader,
                    offset,
                    limit: searchPageSize,
                } as structs.SearchModsRequest);
                if (this.activeSearchRequestID === requestID) {
                    this.nextSearchOffset = offset + searchPageSize;
                }
            } finally {
                if (this.activeSearchRequestID === requestID) {
                    if (append) {
                        this.isLoadingMore = false;
                    } else {
                        this.isSearching = false;
                    }
                }
            }
        },
        async loadMoreSearchResults() {
            await this.runSearch({ append: true });
        },
        async installMod(payload: { result?: SearchModSnapshot; key?: string; status?: string; confirm?: boolean }) {
            const { result, key, status, confirm } = payload || {};
            if (!key || this.downloadingKeys[key]) {
                return;
            }
            if (confirm && this.confirmStatuses.has(status || "")) {
                this.confirmDialog = { show: true, status: status || "", result: result || null, key };
                return;
            }
            await this.doInstall({ result, key });
        },
        async confirmInstall() {
            const { result, key } = this.confirmDialog;
            this.confirmDialog = { show: false, status: "", result: null, key: "" };
            await this.doInstall({ result: result || undefined, key });
        },
        async doInstall(payload: { result?: SearchModSnapshot; key?: string }) {
            const { result, key } = payload || {};
            if (!key || this.downloadingKeys[key]) {
                return;
            }

            this.downloadingKeys = { ...this.downloadingKeys, [key]: true };
            try {
                const res = await QueueModDownload({
                    projectId: key,
                    result: result as models.ModProject,
                    minecraftVersion: this.selectedVersion,
                    modLoader: this.selectedModLoader,
                } as structs.ModDownloadRequest);
                if (res?.skipped) {
                    const errorKey = downloadErrorKeys[res.reason];
                    this.showSnackbar(errorKey ? `download.errors.${errorKey}` : "download.errors.generic", "error");
                } else {
                    this.showSnackbar("download.queued", "success");
                    await this.refreshDownloadStates();
                }
            } finally {
                const next = { ...this.downloadingKeys };
                delete next[key];
                this.downloadingKeys = next;
            }
        },
        async batchUnpin(results: SearchModSnapshot[]) {
            for (const result of results) {
                const modId = this.modIDFromResult(result);
                const pin = await GetPinnedModVersion(result.platform, modId, this.selectedVersion, this.selectedModLoader);
                if (pin?.versionId) {
                    await PinModVersion({
                        platform: result.platform,
                        modId,
                        versionId: pin.versionId,
                        minecraftVersion: this.selectedVersion,
                        modLoader: this.selectedModLoader,
                    } as structs.ModVersionPinRequest);
                }
            }
            await this.refreshDownloadStates();
        },
        async openVersionsOverlay(result: SearchModSnapshot) {
            this.selectedMod = result;
            this.matchingVersions = [];
            this.pinnedVersion = null;
            this.pinningVersionID = "";
            this.showVersionsOverlay = true;
            this.isLoadingVersions = true;
            try {
                const [versions, pin] = await Promise.all([
                    ListMatchingProjectVersions(result, this.selectedVersion, this.selectedModLoader),
                    GetPinnedModVersion(result.platform, this.modIDFromResult(result), this.selectedVersion, this.selectedModLoader),
                ]);
                this.matchingVersions = versions || [];
                this.pinnedVersion = pin?.versionId ? pin : null;
            } finally {
                this.isLoadingVersions = false;
            }
        },
        async pinVersion(version: ProjectVersionSnapshot) {
            if (!this.selectedMod || this.pinningVersionID) {
                return;
            }

            this.pinningVersionID = version.id;
            try {
                const pin = await PinModVersion({
                    platform: this.selectedMod.platform,
                    modId: this.modIDFromResult(this.selectedMod),
                    versionId: version.id,
                    minecraftVersion: this.selectedVersion,
                    modLoader: this.selectedModLoader,
                } as structs.ModVersionPinRequest);
                this.pinnedVersion = pin?.versionId ? pin : null;
            } finally {
                this.pinningVersionID = "";
            }
        },
        modIDFromResult(result: SearchModSnapshot) {
            const id = result?.id || "";
            return id.includes(":") ? id.split(":").slice(1).join(":") : id;
        },
        isPinnedVersion(version: ProjectVersionSnapshot) {
            return this.pinnedVersion?.versionId === version.id;
        },
        isPinningAnotherVersion(version: ProjectVersionSnapshot) {
            return Boolean(this.pinningVersionID && this.pinningVersionID !== version.id);
        },
        versionFileName(version: ProjectVersionSnapshot) {
            return version.fileName || version.name || version.version || version.id;
        },
        async start() {
            if (this.stopListeningSearchModsUpdated || this.stopListeningDownloadQueueUpdated || this.stopListeningDownloadFailed) {
                return;
            }

            this.stopListeningDownloadQueueUpdated = EventsOn(downloadQueueUpdatedEvent, () => {
                void this.refreshDownloadStates();
            });
            this.stopListeningDownloadFailed = EventsOn(downloadFailedEvent, (event) => {
                const fileName = event?.fileName || event?.FileName || "File";
                const reason = event?.reason || event?.Reason || "Download failed";
                this.showSnackbar("download.errors.failedWithReason", "error", { fileName, reason });
                void this.refreshDownloadStates();
            });
            this.stopListeningSearchModsUpdated = EventsOn(searchModsUpdatedEvent, (update) => {
                const requestID = update?.requestId || update?.RequestID;
                if (requestID !== this.activeSearchRequestID) {
                    return;
                }

                const nextResults = update.results || update.Results || [];
                const append = Boolean(update.append ?? update.Append ?? this.activeSearchAppend);
                this.searchResults = append ? [...this.appendBaseResults, ...nextResults] : nextResults;
                this.hasMoreResults = nextResults.length >= searchPageSize;
                const loading = Boolean(update.loading ?? update.Loading);
                if (append) {
                    this.isLoadingMore = loading;
                } else {
                    this.isSearching = loading;
                }
                if (!loading) {
                    void this.refreshDownloadStates();
                }
            });
        },
        stop() {
            this.stopListeningSearchModsUpdated?.();
            this.stopListeningDownloadQueueUpdated?.();
            this.stopListeningDownloadFailed?.();
            this.stopListeningSearchModsUpdated = null;
            this.stopListeningDownloadQueueUpdated = null;
            this.stopListeningDownloadFailed = null;
        },
    },
});
