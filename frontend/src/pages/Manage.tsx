import { useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
import { Package } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { AddToFavoriteDialog } from "@/components/AddToFavoriteDialog";
import { ManageVersionDialog } from "@/components/ManageVersionDialog";
import { ManageFilterToolbar } from "@/components/manage/ManageFilterToolbar";
import { ManageImpactDialogs } from "@/components/manage/ManageImpactDialogs";
import { ManageModListView } from "@/components/manage/ManageModListView";
import { ManageModTreeView } from "@/components/manage/ManageModTreeView";
import { ManageSelectionToolbar } from "@/components/manage/ManageSelectionToolbar";
import { api } from "@/services/app";
import { errorMessage } from "@/lib/format";
import {
  collectMatchingTreeKeys,
  collectRelatedTreeKeys,
  flattenModDependencyTree,
  hydrateModDependencyGraph,
} from "@/lib/modDependencyGraph";
import { groupRowKey, groupSearchText } from "@/lib/modGroupDisplay";
import { useFlipListAnimation } from "@/hooks/useFlipListAnimation";
import { useFlipTreeAnimation } from "@/hooks/useFlipTreeAnimation";
import { useLocalModSelection } from "@/hooks/useLocalModSelection";
import { useManageViewMode } from "@/hooks/useManageViewMode";
import { useUnusedDependencies } from "@/hooks/useUnusedDependencies";
import { scaledDuration } from "@/lib/animations";
import { useDownloadQueueStore } from "@/stores/downloadQueue";
import {
  selectHasSelectedInstance,
  selectSelectedInstanceLabel,
  useMinecraftStore,
} from "@/stores/minecraft";
import { useDownloadSearchStore } from "@/stores/downloadSearch";
import { useSettingsStore } from "@/stores/settings";
import { matchesSearchExpression } from "@/utils/searchExpression";
import type { FavoriteModDraft } from "@/stores/favorites";
import type { LocalModGroup, LocalModView, ModInfo } from "@/types";

const EMPTY_MODS: ModInfo[] = [];

export function ManagePage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const mods = useMinecraftStore((state) => state.selected?.mods ?? EMPTY_MODS);
  const selectedInstance = useMinecraftStore((state) => state.selected);
  const selectedLoader = useMinecraftStore((state) => state.modLoader);
  const minecraftVersion = useMinecraftStore((state) => state.minecraftVersion);
  const hasSelectedInstance = useMinecraftStore(selectHasSelectedInstance);
  const selectedLabel = useMinecraftStore(selectSelectedInstanceLabel);
  const refreshing = useMinecraftStore((state) => state.refreshing);
  const refreshSelectedMods = useMinecraftStore((state) => state.refreshSelected);
  const applySelectedVersion = useMinecraftStore((state) => state.applySelectedVersion);
  const queueActive = useDownloadQueueStore((state) => state.queue.active);
  const requireCtrlForMultiSelect = useSettingsStore((state) => state.settings?.requireCtrlForMultiSelect ?? false);

  const selectedVersionKey = selectedInstance?.id || selectedInstance?.name || "";
  const groupsKey = `${selectedVersionKey}\u0000${selectedLoader}`;
  const [modViewState, setModViewState] = useState<{ key: string; view: LocalModView | null }>({
    key: "",
    view: null,
  });

  // Groups, dependency forest and reverse dependents are built by core in one
  // scan; refetch whenever the scanned mods or the active loader change so the
  // incompatible partition always reflects core's verdict. The key guard hides
  // the previous instance's rows while the new set loads.
  useEffect(() => {
    if (!selectedVersionKey) {
      setModViewState({ key: groupsKey, view: null });
      return;
    }
    let cancelled = false;
    void api
      .GetLocalModView(selectedVersionKey, selectedLoader)
      .then((result) => {
        if (!cancelled) setModViewState({ key: groupsKey, view: result ?? null });
      })
      .catch((error) => {
        if (cancelled) return;
        setModViewState({ key: groupsKey, view: null });
        toast.error(`${t("manage.operationFailed")}: ${errorMessage(error)}`);
      });
    return () => {
      cancelled = true;
    };
  }, [groupsKey, mods, selectedLoader, selectedVersionKey, t]);

  const [searchInput, setSearchInput] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [enabledFilter, setEnabledFilter] = useState("all");
  const [sortMode, setSortMode] = useState("default");
  const [viewMode, setViewMode] = useManageViewMode();
  const viewModeRef = useRef<HTMLDivElement | null>(null);
  const listModeRef = useRef<HTMLButtonElement | null>(null);
  const treeModeRef = useRef<HTMLButtonElement | null>(null);
  const treeScrollRef = useRef<HTMLDivElement | null>(null);
  const listScrollRef = useRef<HTMLDivElement | null>(null);

  const [exitingGroupKeys, setExitingGroupKeys] = useState<Set<string>>(new Set());
  const [listExiting, setListExiting] = useState(false);
  const [viewIndicator, setViewIndicator] = useState({ left: 0, width: 0 });
  const [incompatibleExpanded, setIncompatibleExpanded] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [versionGroup, setVersionGroup] = useState<LocalModGroup | null>(null);
  const [versionOpen, setVersionOpen] = useState(false);
  const [favoriteDrafts, setFavoriteDrafts] = useState<FavoriteModDraft[]>([]);
  const [favoriteOpen, setFavoriteOpen] = useState(false);
  const [replacementQueued, setReplacementQueued] = useState(false);
  const [expandedTreeKeys, setExpandedTreeKeys] = useState<Set<string>>(new Set());
  const [selectedTreeKey, setSelectedTreeKey] = useState("");

  useLayoutEffect(() => {
    const update = () => {
      const active = viewMode === "list" ? listModeRef.current : treeModeRef.current;
      if (!active) return;
      setViewIndicator({ left: active.offsetLeft, width: active.offsetWidth });
    };
    update();
    if (typeof ResizeObserver === "undefined") return;
    const observer = new ResizeObserver(update);
    for (const node of [viewModeRef.current, listModeRef.current, treeModeRef.current]) {
      if (node) observer.observe(node);
    }
    return () => observer.disconnect();
  }, [viewMode, i18n.language]);

  const notify = (key: string, color: "success" | "warning" | "error" = "success", params: Record<string, unknown> = {}) => {
    const message = t(key, params);
    if (color === "error") toast.error(message);
    else if (color === "warning") toast.warning(message);
    else toast.success(message);
  };

  const handleError = (error: unknown) => {
    toast.error(`${t("manage.operationFailed")}: ${errorMessage(error)}`);
  };

  const {
    batchOperation,
    isBatchBusy,
    pendingTogglePath,
    getPendingDeleteGroups,
    deleteDialog,
    setDeleteDialog,
    pendingDeleteCount,
    disableImpactDialog,
    setDisableImpactDialog,
    disableImpacts,
    restoringDependencyId,
    restoredDependencies,
    manageSelectionActions,
    restoreDependency,
    confirmDisableImpact,
    closeDisableImpactDialog,
    toggleGroup,
    openDeleteDialog,
    confirmDelete,
    closeDeleteDialog,
  } = useLocalModSelection({
    minecraftMods: mods,
    t,
    applySelectedVersion,
    refreshSelectedMods,
    onError: handleError,
    onClearError: () => undefined,
    onNotify: notify,
    onReplacementQueued: () => setReplacementQueued(true),
  });

  const usedGroupSignature = `${selectedInstance?.id || selectedInstance?.name || ""}\u0000${selectedLoader}`;
  useEffect(() => {
    setIncompatibleExpanded(false);
    setSelectedKeys(new Set());
    setExpandedTreeKeys(new Set());
  }, [usedGroupSignature]);

  useEffect(() => {
    if (!hasSelectedInstance) return;
    if (replacementQueued && !queueActive) {
      setReplacementQueued(false);
      void refreshSelectedMods();
    }
  }, [hasSelectedInstance, queueActive, refreshSelectedMods, replacementQueued]);

  useEffect(() => {
    void useMinecraftStore.getState().start();
    void useSettingsStore.getState().load().catch(() => undefined);
    void useMinecraftStore.getState().ensureSelectedModsLoaded().catch(() => undefined);
  }, []);

  useEffect(() => {
    const normalized = searchInput.trim().toLocaleLowerCase();
    if (!normalized) {
      setAppliedSearch("");
      return;
    }
    const timer = window.setTimeout(() => setAppliedSearch(normalized), 1000);
    return () => window.clearTimeout(timer);
  }, [searchInput]);

  const allGroups = modViewState.key === groupsKey ? (modViewState.view?.groups ?? []) : [];
  const filteredGroups = useMemo(
    () =>
      allGroups.filter((group) => {
        const matches = matchesSearchExpression(groupSearchText(group), appliedSearch);
        const enabled =
          enabledFilter === "all" || group.primary.enabled === (enabledFilter === "enabled");
        return matches && enabled;
      }),
    [allGroups, appliedSearch, enabledFilter],
  );

  const {
    isScanning,
    candidateFor,
    tooltipFor,
    jijDetectionEnabled,
  } = useUnusedDependencies({
    mods,
    hasSelectedInstance,
    sortMode,
    setSortMode,
    onError: handleError,
    t,
  });

  const compareNames = (a: LocalModGroup, b: LocalModGroup) =>
    a.displayName.toLowerCase().localeCompare(b.displayName.toLowerCase());

  const sortedGroups = useMemo(() => {
    if (sortMode === "default") return filteredGroups;
    const sorted = [...filteredGroups];
    if (sortMode === "name") sorted.sort(compareNames);
    else
      sorted.sort(
        (a, b) =>
          Number(Boolean(candidateFor(b))) - Number(Boolean(candidateFor(a))) ||
          compareNames(a, b),
      );
    return sorted;
    // candidateFor changes when a new scan completes.
  }, [candidateFor, filteredGroups, sortMode]);

  const compatible = sortedGroups.filter((group) => !group.incompatible);
  const incompatible = sortedGroups.filter((group) => group.incompatible);
  const hasActiveFilter = appliedSearch !== "" || enabledFilter !== "all";

  // Only groups the user can currently see participate in batch actions, so a
  // search/status filter never silently operates on hidden rows. The tree view
  // has its own single-node selection model, so the list batch selection is
  // empty there.
  const selectableGroups = useMemo(() => {
    if (viewMode === "tree") return [];
    return incompatibleExpanded ? filteredGroups : compatible;
  }, [compatible, filteredGroups, incompatibleExpanded, viewMode]);

  const selectedGroups = useMemo(
    () => selectableGroups.filter((group) => selectedKeys.has(groupRowKey(group))),
    [selectableGroups, selectedKeys],
  );

  // Drop stale keys when rows disappear because of a filter, refresh or a
  // view switch. Otherwise the bar can stay mounted with a zero count.
  useEffect(() => {
    const valid = new Set(selectableGroups.map(groupRowKey));
    setSelectedKeys((current) => {
      const next = new Set([...current].filter((key) => valid.has(key)));
      return next.size === current.size ? current : next;
    });
  }, [selectableGroups]);

  useEffect(() => {
    if (viewMode === "tree") setSelectedTreeKey("");
  }, [viewMode]);

  const toggleSelection = (group: LocalModGroup, event: { ctrlKey: boolean; metaKey: boolean }) => {
    const key = groupRowKey(group);
    const additive = !requireCtrlForMultiSelect || event.ctrlKey || event.metaKey;
    setSelectedKeys((current) => {
      if (!additive) {
        if (current.size === 1 && current.has(key)) return current;
        return new Set([key]);
      }
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const clearSelection = () => setSelectedKeys(new Set());

  const canFavoriteGroup = (group: LocalModGroup) =>
    Boolean(
      minecraftVersion.trim() &&
        selectedLoader.trim() &&
        group.primary.onlinePlatform &&
        group.primary.onlineProjectId,
    );

  const favoriteDraftFromGroup = (group: LocalModGroup): FavoriteModDraft => ({
    platform: group.primary.onlinePlatform || "",
    modId: group.primary.onlineProjectId || "",
    minecraftVersion:
      minecraftVersion ||
      group.primary.onlineVersion ||
      selectedInstance?.minecraftVersion ||
      "",
    modLoader: selectedLoader || selectedInstance?.modLoader || "",
    title: group.displayName,
    slug: group.primary.onlineSlug || "",
    iconUrl: group.primary.iconUrl || "",
    description: group.primary.description || "",
    categories: group.categories ?? [],
  });

  const openFavorites = (groups: LocalModGroup[]) => {
    if (!groups.length || !groups.every(canFavoriteGroup)) {
      toast.warning(t("favorites.invalidSelection"));
      return;
    }
    setFavoriteDrafts(groups.map(favoriteDraftFromGroup));
    setFavoriteOpen(true);
  };

  const openVersion = (group: LocalModGroup) => {
    if (!canReplaceVersion(group)) return;
    setVersionGroup(group);
    setVersionOpen(true);
  };

  const canReplaceVersion = (group: LocalModGroup) =>
    Boolean(group.canReplaceVersion && hasSelectedInstance);

  const searchVersion = (group: LocalModGroup) => {
    const searchStore = useDownloadSearchStore.getState();
    searchStore.setQuery(group.displayName);
    searchStore.requestAutoSearch();
    void navigate("/download");
  };

  const graph = useMemo(
    () => hydrateModDependencyGraph(modViewState.key === groupsKey ? modViewState.view?.graph : null),
    [groupsKey, modViewState],
  );
  const treeVisibleKeys = useMemo(() => {
    if (!hasActiveFilter) return null;
    return collectMatchingTreeKeys(graph.roots, (node) => {
      const enabled =
        enabledFilter === "all" || node.enabled === (enabledFilter === "enabled");
      return enabled && matchesSearchExpression(node.searchText, appliedSearch);
    });
  }, [appliedSearch, enabledFilter, graph.roots, hasActiveFilter]);
  const treeRows = useMemo(
    () =>
      flattenModDependencyTree(graph.roots, expandedTreeKeys, {
        visibleKeys: treeVisibleKeys,
        forceExpand: appliedSearch !== "",
      }),
    [appliedSearch, expandedTreeKeys, graph.roots, treeVisibleKeys],
  );
  const treeRelations = useMemo(
    () => collectRelatedTreeKeys(graph.roots, selectedTreeKey),
    [graph.roots, selectedTreeKey],
  );

  useFlipTreeAnimation(treeScrollRef);
  useFlipListAnimation(listScrollRef);

  const wait = (ms: number) =>
    ms > 0 ? new Promise<void>((resolve) => window.setTimeout(resolve, ms)) : Promise.resolve();

  // Play the row/container exit before the store refreshes. The delete dialog
  // itself closes only after the operation, so the exit never fights the
  // dialog's own close animation.
  const handleConfirmDelete = async () => {
    const groups = getPendingDeleteGroups();
    const duration = scaledDuration(0.14) * 1000;
    if (groups.length > 1) {
      setListExiting(true);
      await wait(duration);
      await confirmDelete();
      setListExiting(false);
      return;
    }
    if (groups.length === 1) {
      const key = groupRowKey(groups[0]);
      setExitingGroupKeys((current) => new Set(current).add(key));
      await wait(duration);
      await confirmDelete();
      setExitingGroupKeys((current) => {
        if (!current.has(key)) return current;
        const next = new Set(current);
        next.delete(key);
        return next;
      });
      return;
    }
    await confirmDelete();
  };

  const treeGroupFor = (path: string) =>
    allGroups.find((group) => (group.strong || []).some((mod) => (mod.path || "") === path)) || null;

  const toggleTreeNode = (key: string) => {
    setSelectedTreeKey(key);
    setExpandedTreeKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const openLocation = async (group: LocalModGroup | null) => {
    if (!group?.primary?.path) return;
    try {
      await api.OpenLocalModLocation(group.primary.path);
    } catch (error) {
      handleError(error);
    }
  };

  const selectionActions = manageSelectionActions(selectedGroups, clearSelection);

  return (
    <div className="relative mx-auto flex h-full w-full max-w-6xl flex-col overflow-hidden p-6">
      <ManageFilterToolbar
        selectedLabel={selectedLabel}
        refreshing={refreshing}
        batchBusy={isBatchBusy}
        onRefresh={refreshSelectedMods}
        viewMode={viewMode}
        onViewModeChange={setViewMode}
        viewModeRef={viewModeRef}
        listModeRef={listModeRef}
        treeModeRef={treeModeRef}
        viewIndicator={viewIndicator}
        showSearch={hasSelectedInstance}
        searchInput={searchInput}
        onSearchInputChange={setSearchInput}
        enabledFilter={enabledFilter}
        onEnabledFilterChange={setEnabledFilter}
        sortMode={sortMode}
        onSortModeChange={setSortMode}
        showSort={viewMode === "list"}
        jijDetectionEnabled={jijDetectionEnabled}
        scanning={isScanning}
      />

      {!hasSelectedInstance ? (
        <div className="mt-6 flex min-h-64 items-center justify-center rounded-xl border border-dashed text-sm text-muted-foreground">
          {t("manage.noInstance")}
        </div>
      ) : viewMode === "tree" ? (
        <ManageModTreeView
          scrollRef={treeScrollRef}
          missingCount={graph.missing.length}
          rows={treeRows}
          relations={treeRelations}
          selectedKey={selectedTreeKey}
          hasActiveFilter={hasActiveFilter}
          groupForPath={treeGroupFor}
          batchBusy={isBatchBusy}
          canReplaceVersion={canReplaceVersion}
          onSelectNode={setSelectedTreeKey}
          onToggleNode={toggleTreeNode}
          onOpenLocation={openLocation}
          onToggleGroup={toggleGroup}
          onOpenDeleteDialog={openDeleteDialog}
          onSearchVersion={searchVersion}
        />
      ) : sortedGroups.length === 0 ? (
        <div className="mt-6 flex min-h-64 flex-col items-center justify-center gap-2 rounded-xl border border-dashed text-muted-foreground">
          <Package className="size-10" />
          <span className="text-sm">{hasActiveFilter ? t("manage.search.empty") : t("manage.noMods")}</span>
        </div>
      ) : (
        <ManageModListView
          scrollRef={listScrollRef}
          listExiting={listExiting}
          compatible={compatible}
          incompatible={incompatible}
          incompatibleExpanded={incompatibleExpanded}
          onToggleIncompatible={() => setIncompatibleExpanded((value) => !value)}
          selectedKeys={selectedKeys}
          exitingGroupKeys={exitingGroupKeys}
          candidateFor={candidateFor}
          tooltipFor={tooltipFor}
          canFavoriteGroup={canFavoriteGroup}
          canReplaceVersion={canReplaceVersion}
          onToggleSelection={toggleSelection}
          onOpenFavorites={openFavorites}
          onOpenVersion={openVersion}
          onToggleGroup={toggleGroup}
          batchBusy={isBatchBusy}
          pendingTogglePath={pendingTogglePath}
          hasSelection={selectedGroups.length > 0}
        />
      )}

      <ManageSelectionToolbar
        selectedGroups={selectedGroups}
        canFavoriteGroup={canFavoriteGroup}
        onOpenFavorites={openFavorites}
        actions={selectionActions}
        batchBusy={isBatchBusy}
      />

      <ManageImpactDialogs
        deleteOpen={deleteDialog}
        onDeleteOpenChange={setDeleteDialog}
        onCloseDelete={closeDeleteDialog}
        pendingDeleteCount={pendingDeleteCount}
        batchOperation={batchOperation}
        batchBusy={isBatchBusy}
        onConfirmDelete={handleConfirmDelete}
        impactOpen={disableImpactDialog}
        onImpactOpenChange={setDisableImpactDialog}
        onCloseImpact={closeDisableImpactDialog}
        impacts={disableImpacts}
        restoringDependencyId={restoringDependencyId}
        restoredDependencies={restoredDependencies}
        onRestoreDependency={restoreDependency}
        onConfirmImpact={confirmDisableImpact}
      />

      <ManageVersionDialog
        open={versionOpen}
        group={versionGroup}
        dependents={
          versionGroup && modViewState.key === groupsKey
            ? (modViewState.view?.dependents?.[versionGroup.rowKey] ?? [])
            : []
        }
        onOpenChange={setVersionOpen}
        onQueued={() => setReplacementQueued(true)}
        onError={handleError}
        onNotify={notify}
      />

      <AddToFavoriteDialog
        open={favoriteOpen}
        drafts={favoriteDrafts}
        onOpenChange={setFavoriteOpen}
        onAdded={() => toast.success(t("favorites.added"))}
      />
    </div>
  );
}
