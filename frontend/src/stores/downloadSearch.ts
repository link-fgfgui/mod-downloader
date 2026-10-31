import { create, type StoreApi } from "zustand";
import { toast } from "sonner";

import i18n from "@/i18n";
import { api } from "@/services/app";
import { appEventNames, onAppEvent } from "@/services/events";
import { useMinecraftStore, selectHasSelectedInstance } from "@/stores/minecraft";
import type {
  BatchIncompatibleAnalysis,
  DependencyPreflightResult,
  DownloadQueueState,
  ModDownloadButtonState,
  ModProject,
  ModVersion,
} from "@/types";

export const searchPageSize = 10;

type DependencyPreflightEntry = NonNullable<DependencyPreflightResult["entries"]>[number];
type BatchIncompatibleConflict = NonNullable<BatchIncompatibleAnalysis["conflicts"]>[number];

export function modIDFromResult(result: ModProject): string {
  const id = result?.id || result?.projectId || "";
  return id.includes(":") ? id.split(":").slice(1).join(":") : id;
}

export function projectKey(result: ModProject): string {
  return result.id || `${result.platform}:${modIDFromResult(result)}`;
}

const downloadErrorKeys: Record<string, string> = {
  "invalid request": "invalidRequest",
  "no matching version": "noMatchingVersion",
  "missing download url": "missingDownloadUrl",
  "no selected version": "noSelectedVersion",
};

function disabledStates(results: ModProject[], status: string): ModDownloadButtonState[] {
  const icon = status === "noSelectedVersion" ? "noSelectedVersion" : "invalidRequest";
  return results.map((result) => ({
    key: modIDFromResult(result) || result.id || result.projectId || "",
    status,
    disabled: true,
    loading: false,
    icon,
    color: "",
  }));
}

function queueMembership(state: DownloadQueueState): string {
  return (state.items ?? []).map((item) => `${item.id}:${item.status}`).sort().join("|");
}

function target() {
  const minecraft = useMinecraftStore.getState();
  return {
    minecraftVersion: minecraft.minecraftVersion,
    modLoader: minecraft.modLoader,
    hasSelectedInstance: selectHasSelectedInstance(minecraft),
  };
}

export interface ConfirmDialogState {
  show: boolean;
  status: string;
  result: ModProject | null;
  key: string;
  conflictFileName: string;
}

export interface PreflightDialogState {
  show: boolean;
  result: ModProject | null;
  key: string;
  results: ModProject[];
  entries: DependencyPreflightEntry[];
  missing: number;
  conflicts: number;
}

export interface BatchConfirmDialogState {
  show: boolean;
  conflicts: BatchIncompatibleConflict[];
  pendingResults: ModProject[];
}

interface SearchState {
  query: string;
  lastSearchedQuery: string;
  results: ModProject[];
  downloadStates: ModDownloadButtonState[];
  searching: boolean;
  loadingMore: boolean;
  hasMore: boolean;
  error: string;
  activeRequestId: string;
  showVersionsOverlay: boolean;
  isLoadingVersions: boolean;
  selectedMod: ModProject | null;
  matchingVersions: ModVersion[];
  pinnedVersionId: string;
  pinningVersionId: string;
  downloadingKeys: Record<string, boolean>;
  confirmDialog: ConfirmDialogState;
  preflightDialog: PreflightDialogState;
  batchConfirmDialog: BatchConfirmDialogState;
  batchInstalling: boolean;
  autoSearchToken: number;
  requestAutoSearch: () => void;
  setQuery: (query: string) => void;
  runSearch: (append?: boolean) => Promise<void>;
  loadMoreSearchResults: () => Promise<void>;
  showResults: (results: ModProject[]) => Promise<void>;
  refreshDownloadStates: () => Promise<void>;
  installMod: (payload: {
    result?: ModProject;
    key?: string;
    status?: string;
    conflictFileName?: string;
    confirm?: boolean;
  }) => Promise<void>;
  confirmInstall: () => Promise<void>;
  closeConfirmDialog: () => void;
  batchInstall: (results: ModProject[]) => Promise<void>;
  confirmBatchInstall: () => Promise<void>;
  skipConflictedBatchInstall: () => Promise<void>;
  cancelBatchInstall: () => void;
  confirmPreflightInstall: () => Promise<void>;
  closePreflightDialog: () => void;
  openVersionsOverlay: (result: ModProject) => Promise<void>;
  closeVersionsOverlay: () => void;
  pinVersion: (version: ModVersion) => Promise<void>;
  batchUnpin: (results: ModProject[]) => Promise<void>;
  start: () => void;
  stop: () => void;
}

const emptyConfirmDialog: ConfirmDialogState = {
  show: false,
  status: "",
  result: null,
  key: "",
  conflictFileName: "",
};

const emptyPreflightDialog: PreflightDialogState = {
  show: false,
  result: null,
  key: "",
  results: [],
  entries: [],
  missing: 0,
  conflicts: 0,
};

const emptyBatchConfirmDialog: BatchConfirmDialogState = {
  show: false,
  conflicts: [],
  pendingResults: [],
};

let subscriptions: Array<() => void> = [];
let stateRequestId = "";
let queueSignature = "";
let nextOffset = 0;
let appendBase: ModProject[] = [];

function beginBatchInstall(
  set: StoreApi<SearchState>["setState"],
  get: () => SearchState,
): boolean {
  if (get().batchInstalling) return false;
  set({ batchInstalling: true });
  return true;
}

export const useDownloadSearchStore = create<SearchState>((set, get) => ({
  query: "",
  lastSearchedQuery: "",
  results: [],
  downloadStates: [],
  searching: false,
  loadingMore: false,
  hasMore: true,
  error: "",
  activeRequestId: "",
  showVersionsOverlay: false,
  isLoadingVersions: false,
  selectedMod: null,
  matchingVersions: [],
  pinnedVersionId: "",
  pinningVersionId: "",
  downloadingKeys: {},
  confirmDialog: emptyConfirmDialog,
  preflightDialog: emptyPreflightDialog,
  batchConfirmDialog: emptyBatchConfirmDialog,
  batchInstalling: false,
  autoSearchToken: 0,

  setQuery(query) {
    set({ query });
  },

  requestAutoSearch() {
    set({ autoSearchToken: get().autoSearchToken + 1 });
  },

  async runSearch(append = false) {
    const currentTarget = target();
    if (!currentTarget.minecraftVersion || !currentTarget.modLoader) {
      set({ results: [], downloadStates: [], hasMore: false });
      return;
    }
    if (append && (get().searching || get().loadingMore || !get().hasMore)) return;
    const requestId = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
    const offset = append ? nextOffset : 0;
    appendBase = append ? [...get().results] : [];
    set({
      activeRequestId: requestId,
      lastSearchedQuery: get().query,
      error: "",
      searching: !append,
      loadingMore: append,
      ...(append ? {} : { results: [], downloadStates: [], hasMore: true }),
    });
    nextOffset = append ? nextOffset : 0;
    try {
      await api.SearchMods({
        requestId,
        query: get().query,
        version: currentTarget.minecraftVersion,
        modLoader: currentTarget.modLoader,
        offset,
        limit: searchPageSize,
      });
      if (get().activeRequestId === requestId) nextOffset = offset + searchPageSize;
    } catch (error) {
      if (get().activeRequestId === requestId) {
        const message = error instanceof Error ? error.message : String(error);
        set({ error: message, searching: false, loadingMore: false });
        toast.error(i18n.t("download.searchFailed", { reason: message }));
      }
    } finally {
      if (get().activeRequestId === requestId) {
        set({ searching: false, loadingMore: false });
      }
    }
  },

  async loadMoreSearchResults() {
    await get().runSearch(true);
  },

  async showResults(results) {
    nextOffset = 0;
    appendBase = [];
    set({
      query: "",
      lastSearchedQuery: "",
      results,
      downloadStates: [],
      hasMore: false,
      searching: false,
      loadingMore: false,
      error: "",
      activeRequestId: "",
      selectedMod: null,
      matchingVersions: [],
      showVersionsOverlay: false,
    });
    await get().refreshDownloadStates();
  },

  async refreshDownloadStates() {
    if (get().searching || get().loadingMore) return;
    const results = get().results;
    const currentTarget = target();
    if (!currentTarget.minecraftVersion || !currentTarget.modLoader) {
      set({ downloadStates: disabledStates(results, "invalidRequest") });
      return;
    }
    if (!currentTarget.hasSelectedInstance) {
      set({ downloadStates: disabledStates(results, "noSelectedVersion") });
      return;
    }
    const requestId = `${Date.now()}-${Math.random().toString(36).slice(2)}`;
    stateRequestId = requestId;
    try {
      const states = await api.GetDownloadStates({
        results,
        minecraftVersion: currentTarget.minecraftVersion,
        modLoader: currentTarget.modLoader,
      });
      if (stateRequestId === requestId) set({ downloadStates: states ?? [] });
    } catch {
      // Keep the last usable states during a transient refresh failure.
    }
  },

  async installMod(payload) {
    const { result, key, status, conflictFileName, confirm } = payload;
    const currentTarget = target();
    if (!currentTarget.hasSelectedInstance || !currentTarget.minecraftVersion || !currentTarget.modLoader) {
      toast.warning(i18n.t("download.errors.noSelectedVersion"));
      return;
    }
    if (!key) {
      await get().refreshDownloadStates();
      toast.warning(i18n.t("download.errors.preparing"));
      return;
    }
    if (get().downloadingKeys[key]) return;
    if (status === "metadataLoading" || status === "metadataFailed") {
      await get().refreshDownloadStates();
      return;
    }
    if (confirm && ["update", "conflict", "incompatible"].includes(status || "")) {
      set({ confirmDialog: { show: true, status: status || "", result: result ?? null, key, conflictFileName: conflictFileName || "" } });
      return;
    }
    await doInstall(set, get, { result, key, preflight: true });
  },

  async confirmInstall() {
    const { result, key } = get().confirmDialog;
    set({ confirmDialog: { ...emptyConfirmDialog } });
    await doInstall(set, get, { result: result ?? undefined, key, preflight: true });
  },

  closeConfirmDialog() {
    set({ confirmDialog: { ...get().confirmDialog, show: false } });
  },

  async batchInstall(results) {
    const selected = (results || []).filter(Boolean);
    const currentTarget = target();
    if (
      !selected.length ||
      !currentTarget.hasSelectedInstance ||
      !currentTarget.minecraftVersion ||
      !currentTarget.modLoader
    ) {
      return;
    }
    if (!beginBatchInstall(set, get)) return;
    try {
      try {
        const analysis = await api.AnalyzeBatchIncompatibleConflicts({
          results: selected,
          minecraftVersion: currentTarget.minecraftVersion,
          modLoader: currentTarget.modLoader,
        });
        const conflicts = analysis?.conflicts ?? [];
        if (conflicts.length > 0) {
          set({ batchConfirmDialog: { show: true, conflicts, pendingResults: selected } });
          return;
        }
      } catch (error) {
        toast.error(error instanceof Error ? error.message : String(error));
        return;
      }
      const preflight = await checkBatchDependencyPreflight(currentTarget, selected);
      if (preflight.missing > 0 || preflight.conflicts > 0) {
        set({
          preflightDialog: {
            show: true,
            result: null,
            key: "",
            results: selected,
            entries: preflight.entries,
            missing: preflight.missing,
            conflicts: preflight.conflicts,
          },
        });
        return;
      }
      await installBatchResults(set, get, selected);
    } finally {
      set({ batchInstalling: false });
    }
  },

  async confirmBatchInstall() {
    const selected = [...get().batchConfirmDialog.pendingResults];
    set({ batchConfirmDialog: emptyBatchConfirmDialog });
    if (!beginBatchInstall(set, get)) return;
    try {
      await installBatchResults(set, get, selected);
    } finally {
      set({ batchInstalling: false });
    }
  },

  async skipConflictedBatchInstall() {
    const conflicted = new Set((get().batchConfirmDialog.conflicts || []).map((conflict) => conflict.key));
    const selected = get().batchConfirmDialog.pendingResults.filter((result) => {
      const key = downloadKeyForResult(get(), result);
      return key && !conflicted.has(key);
    });
    set({ batchConfirmDialog: emptyBatchConfirmDialog });
    if (!beginBatchInstall(set, get)) return;
    try {
      await installBatchResults(set, get, selected);
    } finally {
      set({ batchInstalling: false });
    }
  },

  cancelBatchInstall() {
    set({ batchConfirmDialog: emptyBatchConfirmDialog });
  },

  async confirmPreflightInstall() {
    const { result, key, results } = get().preflightDialog;
    set({ preflightDialog: emptyPreflightDialog });
    if (results.length > 0) {
      if (!beginBatchInstall(set, get)) return;
      try {
        await installBatchResults(set, get, results);
      } finally {
        set({ batchInstalling: false });
      }
      return;
    }
    await doInstall(set, get, { result: result ?? undefined, key, preflight: false });
  },

  closePreflightDialog() {
    set({ preflightDialog: emptyPreflightDialog });
  },

  async openVersionsOverlay(result) {
    const currentTarget = target();
    set({
      selectedMod: result,
      matchingVersions: [],
      pinnedVersionId: "",
      pinningVersionId: "",
      showVersionsOverlay: true,
      isLoadingVersions: true,
    });
    try {
      const [versions, pin] = await Promise.all([
        api.ListMatchingProjectVersions(result, currentTarget.minecraftVersion, currentTarget.modLoader),
        api.GetPinnedModVersion(
          result.platform,
          modIDFromResult(result),
          currentTarget.minecraftVersion,
          currentTarget.modLoader,
        ),
      ]);
      set({ matchingVersions: versions ?? [], pinnedVersionId: pin?.versionId || "" });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    } finally {
      set({ isLoadingVersions: false });
    }
  },

  closeVersionsOverlay() {
    set({ showVersionsOverlay: false });
  },

  async pinVersion(version) {
    const selected = get().selectedMod;
    const currentTarget = target();
    if (!selected || get().pinningVersionId) return;
    set({ pinningVersionId: version.id });
    try {
      const pin = await api.PinModVersion({
        platform: selected.platform,
        modId: modIDFromResult(selected),
        versionId: version.id,
        minecraftVersion: currentTarget.minecraftVersion,
        modLoader: currentTarget.modLoader,
      });
      set({ pinnedVersionId: pin?.versionId || "" });
      await get().refreshDownloadStates();
    } finally {
      set({ pinningVersionId: "" });
    }
  },

  async batchUnpin(results) {
    const currentTarget = target();
    try {
      for (const result of results) {
        const modId = modIDFromResult(result);
        const pin = await api.GetPinnedModVersion(
          result.platform,
          modId,
          currentTarget.minecraftVersion,
          currentTarget.modLoader,
        );
        if (pin?.versionId) {
          await api.PinModVersion({
            platform: result.platform,
            modId,
            versionId: pin.versionId,
            minecraftVersion: currentTarget.minecraftVersion,
            modLoader: currentTarget.modLoader,
          });
        }
      }
      await get().refreshDownloadStates();
      toast.success(i18n.t("download.selection.unpinned"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    }
  },

  start() {
    if (subscriptions.length) return;
    subscriptions = [
      onAppEvent(appEventNames.SearchModsUpdated, (update) => {
        if (update.requestId !== get().activeRequestId) return;
        const incoming = update.results ?? [];
        const results = update.append ? [...appendBase, ...incoming] : incoming;
        set({
          results,
          hasMore: incoming.length >= searchPageSize,
          searching: update.append ? get().searching : update.loading,
          loadingMore: update.append ? update.loading : get().loadingMore,
        });
        if (!update.loading) void get().refreshDownloadStates();
      }),
      onAppEvent(appEventNames.ExtensionModsAccepted, (update) => {
        if (!update.results?.length) return;
        nextOffset = 0;
        appendBase = [];
        set({
          results: update.results,
          hasMore: false,
          searching: false,
          loadingMore: false,
          activeRequestId: "",
        });
        void get().refreshDownloadStates();
      }),
      onAppEvent(appEventNames.DownloadStatesUpdated, () => void get().refreshDownloadStates()),
      onAppEvent(appEventNames.DownloadQueueUpdated, (queue) => {
        const signature = queueMembership(queue);
        if (signature === queueSignature) return;
        queueSignature = signature;
        void get().refreshDownloadStates();
      }),
    ];
  },

  stop() {
    subscriptions.forEach((unsubscribe) => unsubscribe());
    subscriptions = [];
    queueSignature = "";
    stateRequestId = "";
    nextOffset = 0;
    appendBase = [];
  },
}));

function downloadKeyForResult(state: SearchState, result: ModProject): string {
  const index = state.results.indexOf(result);
  const stateKey = index >= 0 ? state.downloadStates[index]?.key : "";
  return stateKey || result?.id || "";
}

async function checkDependencyPreflight(
  currentTarget: ReturnType<typeof target>,
  result: ModProject,
): Promise<DependencyPreflightResult | null> {
  try {
    return await api.CheckDependencyFulfillment({
      projectId: modIDFromResult(result) || result.id || "",
      result,
      minecraftVersion: currentTarget.minecraftVersion,
      modLoader: currentTarget.modLoader,
    });
  } catch {
    return null;
  }
}

async function checkBatchDependencyPreflight(
  currentTarget: ReturnType<typeof target>,
  results: ModProject[],
): Promise<{ entries: DependencyPreflightEntry[]; missing: number; conflicts: number }> {
  const analyses = await Promise.all(results.map((result) => checkDependencyPreflight(currentTarget, result)));
  const entries: DependencyPreflightEntry[] = [];
  let missing = 0;
  let conflicts = 0;
  for (const analysis of analyses) {
    if (!analysis) continue;
    entries.push(...(analysis.entries ?? []));
    missing += analysis.missing || 0;
    conflicts += analysis.conflicts || 0;
  }
  return { entries, missing, conflicts };
}

async function doInstall(
  set: StoreApi<SearchState>["setState"],
  get: () => SearchState,
  payload: { result?: ModProject; key?: string; preflight?: boolean },
) {
  const { result, key, preflight } = payload;
  const currentTarget = target();
  if (
    !key ||
    get().downloadingKeys[key] ||
    !currentTarget.hasSelectedInstance ||
    !currentTarget.minecraftVersion ||
    !currentTarget.modLoader
  ) {
    return;
  }

  set((state) => ({ downloadingKeys: { ...state.downloadingKeys, [key]: true } }));
  try {
    if (preflight && result) {
      const analysis = await checkDependencyPreflight(currentTarget, result);
      if (analysis && (analysis.missing > 0 || analysis.conflicts > 0)) {
        set({
          preflightDialog: {
            show: true,
            result,
            key,
            results: [],
            entries: analysis.entries ?? [],
            missing: analysis.missing,
            conflicts: analysis.conflicts,
          },
        });
        return;
      }
    }

    const res = await api.QueueModDownload({
      projectId: key,
      result: result as ModProject,
      minecraftVersion: currentTarget.minecraftVersion,
      modLoader: currentTarget.modLoader,
    });
    if (res?.skipped) {
      const errorKey = downloadErrorKeys[res.reason];
      toast.error(i18n.t(errorKey ? `download.errors.${errorKey}` : "download.errors.generic"));
    } else {
      toast.success(i18n.t("download.queued"));
      await get().refreshDownloadStates();
    }
  } catch (error) {
    toast.error(error instanceof Error ? error.message : String(error));
  } finally {
    set((state) => {
      const next = { ...state.downloadingKeys };
      delete next[key];
      return { downloadingKeys: next };
    });
  }
}

async function installBatchResults(
  set: StoreApi<SearchState>["setState"],
  get: () => SearchState,
  results: ModProject[],
) {
  let skipped = 0;
  for (const result of results) {
    const key = downloadKeyForResult(get(), result);
    if (!key) {
      skipped += 1;
      continue;
    }
    await doInstall(set, get, { result, key });
  }
  if (skipped > 0) {
    await get().refreshDownloadStates();
    toast.warning(i18n.t("download.errors.preparing"));
  }
}
