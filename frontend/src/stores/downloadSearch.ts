import { defineStore } from "pinia";

import {
    AnalyzeBatchIncompatibleConflicts,
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
const extensionModsAcceptedEvent = "extension-mods-accepted";
const downloadStatesUpdatedEvent = "download-states-updated";

const searchPageSize = 10;

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
        hasSelectedInstance: false,
        showDirOverlay: false,
        isSearching: false,
        isLoadingMore: false,
        hasMoreResults: true,
        showVersionsOverlay: false,
        isLoadingVersions: false,
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
        confirmDialog: { show: false, status: "", result: null as SearchModSnapshot | null, key: "", conflictFileName: "" },
        batchConfirmDialog: {
            show: false,
            conflicts: [] as structs.BatchIncompatibleConflict[],
            pendingResults: [] as SearchModSnapshot[],
        },
        stopListeningSearchModsUpdated: null as (() => void) | null,
        stopListeningDownloadQueueUpdated: null as (() => void) | null,
        stopListeningDownloadFailed: null as (() => void) | null,
        stopListeningExtensionModsAccepted: null as (() => void) | null,
        stopListeningDownloadStatesUpdated: null as (() => void) | null,
    }),
    getters: {
        confirmStatuses: () => new Set(["update", "conflict", "incompatible"]),
    },
    actions: {
        showSnackbar(key: string, color = "success", params: Record<string, string> = {}) {
            this.snackbar = { show: true, key, params, color };
        },
        setTargetTuple(minecraftVersion: string, modLoader: string, hasSelectedInstance: boolean) {
            const nextVersion = minecraftVersion || "";
            const nextLoader = modLoader || "";
            const changed =
                this.selectedVersion !== nextVersion ||
                this.selectedModLoader !== nextLoader ||
                this.hasSelectedInstance !== hasSelectedInstance;
            this.selectedVersion = nextVersion;
            this.selectedModLoader = nextLoader;
            this.hasSelectedInstance = hasSelectedInstance;
            if (changed) {
                this.clearDownloadStates();
            }
        },
        clearDownloadStates() {
            this.downloadStates = [];
            void this.refreshDownloadStates();
        },
        disabledDownloadStates(reason: "noSelectedVersion" | "invalidRequest" = "noSelectedVersion") {
            const icon = reason === "noSelectedVersion" ? "mdi-cube-off-outline" : "mdi-alert-circle-outline";
            return (this.searchResults || []).map((result) => ({
                key: this.modIDFromResult(result) || result.id || result.slug || "",
                status: reason,
                disabled: true,
                loading: false,
                icon,
                color: "surface-variant",
            }) as DownloadStateSnapshot);
        },
        async refreshDownloadStates() {
            if (this.isSearching || this.isLoadingMore) {
                return;
            }
            if (!this.selectedVersion || !this.selectedModLoader) {
                this.downloadStates = this.disabledDownloadStates("invalidRequest");
                return;
            }
            if (!this.hasSelectedInstance) {
                this.downloadStates = this.disabledDownloadStates("noSelectedVersion");
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
            if (!this.selectedVersion || !this.selectedModLoader) {
                this.searchResults = [];
                this.downloadStates = [];
                this.hasMoreResults = false;
                return;
            }
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
        async installMod(payload: { result?: SearchModSnapshot; key?: string; status?: string; conflictFileName?: string; confirm?: boolean }) {
            const { result, key, status, conflictFileName, confirm } = payload || {};
            if (!key || this.downloadingKeys[key] || !this.hasSelectedInstance || !this.selectedVersion || !this.selectedModLoader) {
                return;
            }
            if (confirm && this.confirmStatuses.has(status || "")) {
                this.confirmDialog = { show: true, status: status || "", result: result || null, key, conflictFileName: conflictFileName || "" };
                return;
            }
            await this.doInstall({ result, key });
        },
        async confirmInstall() {
            const { result, key } = this.confirmDialog;
            this.closeConfirmDialog();
            await this.doInstall({ result: result || undefined, key });
        },
        async batchInstall(results: SearchModSnapshot[]) {
            const selected = (results || []).filter(Boolean);
            if (!selected.length || !this.hasSelectedInstance || !this.selectedVersion || !this.selectedModLoader) {
                return;
            }
            const analysis = await AnalyzeBatchIncompatibleConflicts({
                results: selected,
                minecraftVersion: this.selectedVersion,
                modLoader: this.selectedModLoader,
            } as structs.BatchDownloadRequest);
            const conflicts = analysis?.conflicts || [];
            if (conflicts.length > 0) {
                this.batchConfirmDialog = { show: true, conflicts, pendingResults: selected };
                return;
            }
            await this.installBatchResults(selected);
        },
        async confirmBatchInstall() {
            const selected = [...this.batchConfirmDialog.pendingResults];
            this.batchConfirmDialog = { show: false, conflicts: [], pendingResults: [] };
            await this.installBatchResults(selected);
        },
        async skipConflictedBatchInstall() {
            const conflicted = new Set((this.batchConfirmDialog.conflicts || []).map((conflict) => conflict.key));
            const selected = this.batchConfirmDialog.pendingResults.filter((result) => {
                const key = this.downloadKeyForResult(result);
                return key && !conflicted.has(key);
            });
            this.batchConfirmDialog = { show: false, conflicts: [], pendingResults: [] };
            await this.installBatchResults(selected);
        },
        cancelBatchInstall() {
            this.batchConfirmDialog = { show: false, conflicts: [], pendingResults: [] };
        },
        async installBatchResults(results: SearchModSnapshot[]) {
            for (const result of results) {
                const key = this.downloadKeyForResult(result);
                if (!key) continue;
                await this.doInstall({ result, key });
            }
        },
        closeConfirmDialog() {
            this.confirmDialog = { ...this.confirmDialog, show: false };
        },
        clearClosedConfirmDialog() {
            if (!this.confirmDialog.show) {
                this.confirmDialog = { show: false, status: "", result: null, key: "", conflictFileName: "" };
            }
        },
        async doInstall(payload: { result?: SearchModSnapshot; key?: string }) {
            const { result, key } = payload || {};
            if (!key || this.downloadingKeys[key] || !this.hasSelectedInstance || !this.selectedVersion || !this.selectedModLoader) {
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
        downloadKeyForResult(result: SearchModSnapshot) {
            const index = this.searchResults.indexOf(result);
            const stateKey = index >= 0 ? this.downloadStates[index]?.key : "";
            return stateKey || result?.id || "";
        },
        async start() {
            if (this.stopListeningSearchModsUpdated || this.stopListeningDownloadQueueUpdated || this.stopListeningDownloadFailed || this.stopListeningExtensionModsAccepted || this.stopListeningDownloadStatesUpdated) {
                return;
            }

            this.stopListeningDownloadQueueUpdated = EventsOn(downloadQueueUpdatedEvent, () => {
                void this.refreshDownloadStates();
            });
            this.stopListeningDownloadStatesUpdated = EventsOn(downloadStatesUpdatedEvent, () => {
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
            this.stopListeningExtensionModsAccepted = EventsOn(extensionModsAcceptedEvent, (update) => {
                const results = update?.results || update?.Results || [];
                if (results.length === 0) {
                    return;
                }
                this.searchResults = results;
                this.appendBaseResults = [];
                this.hasMoreResults = false;
                this.isSearching = false;
                this.isLoadingMore = false;
                this.activeSearchRequestID = "";
                void this.refreshDownloadStates();
            });
        },
        stop() {
            this.stopListeningSearchModsUpdated?.();
            this.stopListeningDownloadQueueUpdated?.();
            this.stopListeningDownloadFailed?.();
            this.stopListeningExtensionModsAccepted?.();
            this.stopListeningDownloadStatesUpdated?.();
            this.stopListeningSearchModsUpdated = null;
            this.stopListeningDownloadQueueUpdated = null;
            this.stopListeningDownloadFailed = null;
            this.stopListeningExtensionModsAccepted = null;
            this.stopListeningDownloadStatesUpdated = null;
        },
    },
});
