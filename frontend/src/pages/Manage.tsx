import { forwardRef, useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  Copy,
  Download,
  Eye,
  ListTree,
  FolderOpen,
  Loader2,
  Package,
  ListPlus,
  RefreshCw,
  Search,
  Trash2,
  ToggleLeft,
  ToggleRight,
  X,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { AddToFavoriteDialog } from "@/components/AddToFavoriteDialog";
import { AnimatedPresence } from "@/components/AnimatedPresence";
import { ManageVersionDialog } from "@/components/ManageVersionDialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SearchInput } from "@/components/ui/search-input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { api } from "@/services/app";
import { errorMessage } from "@/lib/format";
import {
  collectMatchingTreeKeys,
  collectRelatedTreeKeys,
  flattenModDependencyTree,
  buildModDependencyGraph,
} from "@/lib/modDependencyGraph";
import {
  displayModName,
  groupMods,
  groupRowKey,
  groupTooltip,
  hasGroupedDetails,
  isGroupIncompatible,
  modCategories,
  searchableGroupText,
  strongModIds,
  type ModGroup,
} from "@/lib/modGroups";
import { useLocalModSelection } from "@/hooks/useLocalModSelection";
import { useUnusedDependencies } from "@/hooks/useUnusedDependencies";
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
import type { ModInfo } from "@/types";
import { cn } from "@/lib/utils";

type ViewMode = "list" | "tree";

const EMPTY_MODS: ModInfo[] = [];

const ViewModeButton = forwardRef<
  HTMLButtonElement,
  {
    active: boolean;
    overlay?: boolean;
    onClick?: () => void;
    children: ReactNode;
  }
>(function ViewModeButton({ active, overlay = false, onClick, children }, ref) {
  return (
    <button
      ref={ref}
      type="button"
      tabIndex={overlay ? -1 : undefined}
      aria-hidden={overlay || undefined}
      aria-pressed={overlay ? undefined : active}
      className={cn(
        "md-pressable relative z-10 inline-flex h-7 shrink-0 items-center gap-1.5 rounded-md px-3 text-sm font-medium whitespace-nowrap",
        overlay
          ? "text-primary-foreground"
          : active
            ? "text-foreground"
            : "text-muted-foreground hover:text-foreground",
      )}
      onClick={overlay ? undefined : onClick}
    >
      {children}
    </button>
  );
});

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

  const [searchInput, setSearchInput] = useState("");
  const [appliedSearch, setAppliedSearch] = useState("");
  const [enabledFilter, setEnabledFilter] = useState("all");
  const [sortMode, setSortMode] = useState("default");
  const [viewMode, setViewMode] = useState<ViewMode>("list");
  const viewModeRef = useRef<HTMLDivElement | null>(null);
  const listModeRef = useRef<HTMLButtonElement | null>(null);
  const treeModeRef = useRef<HTMLButtonElement | null>(null);
  const [viewIndicator, setViewIndicator] = useState({ left: 0, width: 0 });
  const [incompatibleExpanded, setIncompatibleExpanded] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [versionGroup, setVersionGroup] = useState<ModGroup | null>(null);
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

  const allGroups = useMemo(() => groupMods(mods), [mods]);
  const filteredGroups = useMemo(
    () =>
      allGroups.filter((group) => {
        const matches = matchesSearchExpression(searchableGroupText(group), appliedSearch);
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

  const compareNames = (a: ModGroup, b: ModGroup) =>
    displayModName(a).toLowerCase().localeCompare(displayModName(b).toLowerCase());

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

  const compatible = sortedGroups.filter((group) => !isGroupIncompatible(group, selectedLoader));
  const incompatible = sortedGroups.filter((group) => isGroupIncompatible(group, selectedLoader));
  const hasActiveFilter = appliedSearch !== "" || enabledFilter !== "all";

  const availableGroups = useMemo(() => {
    const rows: Array<ModGroup | "incompatible-toggle"> = [...compatible];
    if (incompatible.length > 0) rows.push("incompatible-toggle");
    if (incompatibleExpanded) rows.push(...incompatible);
    return rows;
  }, [compatible, incompatible, incompatibleExpanded]);

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

  const toggleSelection = (group: ModGroup, event: { ctrlKey: boolean; metaKey: boolean }) => {
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

  const canFavoriteGroup = (group: ModGroup) =>
    Boolean(
      minecraftVersion.trim() &&
        selectedLoader.trim() &&
        group.primary.onlinePlatform &&
        group.primary.onlineProjectId,
    );

  const favoriteDraftFromGroup = (group: ModGroup): FavoriteModDraft => ({
    platform: group.primary.onlinePlatform || "",
    modId: group.primary.onlineProjectId || "",
    minecraftVersion:
      minecraftVersion ||
      group.primary.onlineVersion ||
      selectedInstance?.minecraftVersion ||
      "",
    modLoader: selectedLoader || selectedInstance?.modLoader || "",
    title: displayModName(group),
    slug: group.primary.onlineSlug || "",
    iconUrl: group.primary.iconUrl || "",
    description: group.primary.description || "",
    categories: modCategories(group),
  });

  const openFavorites = (groups: ModGroup[]) => {
    if (!groups.length || !groups.every(canFavoriteGroup)) {
      toast.warning(t("favorites.invalidSelection"));
      return;
    }
    setFavoriteDrafts(groups.map(favoriteDraftFromGroup));
    setFavoriteOpen(true);
  };

  const openVersion = (group: ModGroup) => {
    if (!canReplaceVersion(group)) return;
    setVersionGroup(group);
    setVersionOpen(true);
  };

  const canReplaceVersion = (group: ModGroup) =>
    Boolean(group.primary.onlinePlatform && group.primary.onlineProjectId && hasSelectedInstance);

  const searchVersion = (group: ModGroup) => {
    const searchStore = useDownloadSearchStore.getState();
    searchStore.setQuery(displayModName(group));
    searchStore.requestAutoSearch();
    void navigate("/download");
  };

  const graph = useMemo(() => buildModDependencyGraph(mods), [mods]);
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

  const treeGroupFor = (path: string) =>
    allGroups.find((group) => group.strong.some((mod) => (mod.path || "") === path)) || null;

  const toggleTreeNode = (key: string) => {
    setSelectedTreeKey(key);
    setExpandedTreeKeys((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const treeStatus = (node: {
    isMissing: boolean;
    isJij: boolean;
    enabled: boolean;
    isOptional: boolean;
  }) => {
    if (node.isMissing)
      return { label: t("manage.tree.status.missing"), color: "text-destructive", icon: AlertTriangle };
    if (node.isJij)
      return { label: t("manage.tree.status.jij"), color: "text-violet-600 dark:text-violet-400", icon: Package };
    if (node.isOptional)
      return { label: t("manage.tree.status.optional"), color: "text-amber-600 dark:text-amber-400", icon: AlertTriangle };
    if (!node.enabled)
      return { label: t("manage.tree.status.disabled"), color: "text-muted-foreground", icon: ToggleLeft };
    return { label: t("manage.tree.status.enabled"), color: "text-emerald-600 dark:text-emerald-400", icon: ToggleRight };
  };

  const openLocation = async (group: ModGroup | null) => {
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
      <div className="flex shrink-0 flex-wrap items-center gap-3">
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-bold tracking-tight">{t("manage.title")}</h1>
          <div className="truncate text-sm text-muted-foreground">{selectedLabel}</div>
        </div>
        <Button type="button" size="sm" disabled={refreshing || isBatchBusy} onClick={() => void refreshSelectedMods()}>
          <RefreshCw className={refreshing ? "size-4 animate-spin" : "size-4"} />
          {t("manage.refresh")}
        </Button>
        <div ref={viewModeRef} className="relative flex rounded-lg border bg-muted/30 p-0.5">
          <span
            aria-hidden="true"
            className="pointer-events-none absolute top-0.5 bottom-0.5 left-0 rounded-md bg-primary transition-[transform,width] duration-md ease-out"
            style={{ transform: `translateX(${viewIndicator.left}px)`, width: viewIndicator.width }}
          />
          <ViewModeButton
            ref={listModeRef}
            active={viewMode === "list"}
            onClick={() => setViewMode("list")}
          >
            {t("manage.viewMode.list")}
          </ViewModeButton>
          <ViewModeButton
            ref={treeModeRef}
            active={viewMode === "tree"}
            onClick={() => setViewMode("tree")}
          >
            <ListTree className="size-4" />
            {t("manage.viewMode.tree")}
          </ViewModeButton>
          {/* Clipped duplicate keeps the label readable while the indicator
              slides between the two options. Without it the incoming label
              would switch to dark text before the light pill arrives. */}
          <span
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 z-20 flex overflow-hidden rounded-lg p-0.5 transition-[clip-path] duration-md ease-out"
            style={{
              clipPath: `inset(0 ${Math.max(
                0,
                (viewModeRef.current?.clientWidth ?? 0) - viewIndicator.left - viewIndicator.width,
              )}px 0 ${viewIndicator.left}px round 6px)`,
            }}
          >
            <ViewModeButton active overlay>
              {t("manage.viewMode.list")}
            </ViewModeButton>
            <ViewModeButton active overlay>
              <ListTree className="size-4" />
              {t("manage.viewMode.tree")}
            </ViewModeButton>
          </span>
        </div>
      </div>

      {hasSelectedInstance ? (
        <div className="mt-4 flex shrink-0 flex-wrap items-center gap-2">
          <div className="relative min-w-0 flex-1 basis-64">
            <SearchInput
              value={searchInput}
              onValueChange={setSearchInput}
              placeholder={t("manage.search.label")}
              aria-label={t("manage.search.label")}
              clearAriaLabel={t("common.clearSearch")}
            />
          </div>
          <HelpTooltip label={t("manage.search.syntaxTip")} />
          <Combobox
            value={enabledFilter}
            onValueChange={setEnabledFilter}
            options={[
              { value: "all", label: t("manage.filter.all") },
              { value: "enabled", label: t("manage.enabled") },
              { value: "disabled", label: t("manage.disabled") },
            ]}
            aria-label={t("manage.filter.all")}
            className="w-36"
          />
          {viewMode === "list" ? (
            <Combobox
              value={sortMode}
              onValueChange={setSortMode}
              options={[
                { value: "default", label: t("manage.sort.default") },
                { value: "name", label: t("manage.sort.name") },
                {
                  value: "unused",
                  label: t("manage.sort.unused"),
                  disabled: !jijDetectionEnabled,
                },
              ]}
              aria-label={t("manage.sort.default")}
              className="w-48"
            />
          ) : null}
          {isScanning ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
        </div>
      ) : null}

      {!hasSelectedInstance ? (
        <div className="mt-6 flex min-h-64 items-center justify-center rounded-xl border border-dashed text-sm text-muted-foreground">
          {t("manage.noInstance")}
        </div>
      ) : viewMode === "tree" ? (
        <div className="mt-4 min-h-0 flex-1 overflow-y-auto overflow-x-hidden rounded-xl border bg-card/40 animate-in fade-in-0 slide-in-from-right-2 duration-200">
          {graph.missing.length ? (
            <div className="border-b border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-300">
              {t("manage.tree.missingDep", { n: graph.missing.length })}
            </div>
          ) : null}
          {treeRows.length === 0 ? (
            <div className="flex min-h-64 flex-col items-center justify-center gap-2 text-muted-foreground">
              <ListTree className="size-10" />
              <span className="text-sm">{hasActiveFilter ? t("manage.search.empty") : t("manage.tree.empty")}</span>
            </div>
          ) : (
            treeRows.map((row) => {
              const node = row.node;
              const status = treeStatus(node);
              const StatusIcon = status.icon;
              const group = treeGroupFor(node.path);
              return (
                <div
                  key={row.key}
                  className={cn(
                    "md-row-interactive flex min-h-14 items-center gap-2 border-b px-3 py-2 last:border-b-0",
                    selectedTreeKey === node.key
                      ? "bg-primary/10 hover:bg-primary/15"
                      : treeRelations.dependencies.has(node.key)
                        ? "bg-sky-500/10 hover:bg-sky-500/15"
                        : treeRelations.dependents.has(node.key)
                          ? "bg-amber-500/10 hover:bg-amber-500/15"
                          : "hover:bg-accent/50",
                  )}
                  style={{ paddingLeft: 12 + row.depth * 20 }}
                >
                  {row.hasChildren ? (
                    <Button
                      type="button"
                      size="icon-xs"
                      variant="ghost"
                      aria-expanded={row.expanded}
                      onClick={() => toggleTreeNode(node.key)}
                    >
                      <ChevronRight className={cn("size-3.5 transition-transform duration-md", row.expanded && "rotate-90")} />
                    </Button>
                  ) : (
                    <span className="inline-block w-6" />
                  )}
                  <button
                    type="button"
                    className="flex min-w-0 flex-1 items-center gap-2 text-left transition-opacity duration-fast active:opacity-70"
                    onClick={() => {
                      setSelectedTreeKey(node.key);
                      if (row.hasChildren) toggleTreeNode(node.key);
                    }}
                  >
                    {node.iconUrl ? (
                      <img src={node.iconUrl} alt="" className="size-6 rounded object-cover" />
                    ) : (
                      <StatusIcon className={cn("size-5 shrink-0", status.color)} />
                    )}
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center gap-1">
                        <span className="truncate text-sm font-medium">{node.name}</span>
                        {node.version ? <span className="truncate text-xs text-muted-foreground">· {node.version}</span> : null}
                      </span>
                      <span className="block truncate text-xs text-muted-foreground">
                        {node.isMissing
                          ? t("manage.tree.missingNode")
                          : node.isJij
                            ? t("manage.tree.jijNode")
                            : [
                                node.children.length ? t("manage.tree.dependsOn", { n: node.children.length }) : "",
                                node.parents.length ? t("manage.tree.requiredBy", { n: node.parents.length }) : "",
                                node.isOptional ? t("manage.tree.optionalNode") : "",
                              ]
                                .filter(Boolean)
                                .join(" · ")}
                      </span>
                    </span>
                  </button>
                  {node.isRoot ? <Badge variant="outline">{t("manage.tree.root")}</Badge> : null}
                  <Badge variant="secondary" className={status.color}>{status.label}</Badge>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button type="button" size="icon-sm" variant="ghost" aria-label={t("manage.tree.actions.menu")}>
                        <ChevronDown className="size-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem disabled={!group} onClick={() => void openLocation(group)}>
                        <FolderOpen className="size-4" />
                        {t("manage.tree.actions.openLocation")}
                      </DropdownMenuItem>
                      <DropdownMenuItem disabled={!group || isBatchBusy} onClick={() => group && void toggleGroup(group)}>
                        {group?.primary?.enabled ? <ToggleLeft className="size-4" /> : <ToggleRight className="size-4" />}
                        {t(group?.primary?.enabled ? "manage.tree.actions.disable" : "manage.tree.actions.enable")}
                      </DropdownMenuItem>
                      <DropdownMenuItem disabled={!group || isBatchBusy} onClick={() => group && openDeleteDialog([group])}>
                        <Trash2 className="size-4" />
                        {t("manage.tree.actions.uninstall")}
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem disabled={!group || !canReplaceVersion(group)} onClick={() => group && searchVersion(group)}>
                        <Search className="size-4" />
                        {t("manage.tree.actions.searchNewVersion")}
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              );
            })
          )}
        </div>
      ) : availableGroups.length === 0 ? (
        <div className="mt-6 flex min-h-64 flex-col items-center justify-center gap-2 rounded-xl border border-dashed text-muted-foreground">
          <Package className="size-10" />
          <span className="text-sm">{hasActiveFilter ? t("manage.search.empty") : t("manage.noMods")}</span>
        </div>
      ) : (
        <div className="mt-4 min-h-0 flex-1 overflow-y-auto overflow-x-hidden animate-in fade-in-0 slide-in-from-left-2 duration-200">
          <div className={cn("grid gap-2", selectedGroups.length > 0 && "pb-24")}>
            {availableGroups.map((row) => {
              if (row === "incompatible-toggle") {
                return (
                  <button
                    key="incompatible-toggle"
                    type="button"
                    className="md-row-interactive flex w-full min-w-0 items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-left hover:bg-amber-500/20"
                    onClick={() => setIncompatibleExpanded((value) => !value)}
                  >
                    <AlertTriangle className="size-5 text-amber-600 dark:text-amber-400" />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-semibold">
                        {t("manage.incompatible.title", { n: incompatible.length })}
                      </span>
                      <span className="block text-xs text-muted-foreground">
                        {t(incompatibleExpanded ? "manage.incompatible.expanded" : "manage.incompatible.collapsed")}
                      </span>
                    </span>
                    <ChevronDown className={cn("size-4 transition-transform", incompatibleExpanded && "rotate-180")} />
                  </button>
                );
              }

              const group = row;
              const key = groupRowKey(group);
              const selected = selectedKeys.has(key);
              const unused = candidateFor(group);
              const categories = modCategories(group);
              const groupAction = group.primary.enabled ? "disable" : "enable";
              return (
                <div
                  key={key}
                  className={cn(
                    "md-row-interactive flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl border bg-card p-3 shadow-sm",
                    selected
                      ? "border-primary/50 bg-primary/10 shadow-sm hover:bg-primary/15"
                      : "hover:border-foreground/20 hover:bg-accent/60 hover:shadow-md",
                    isGroupIncompatible(group, selectedLoader) && "opacity-80",
                  )}
                  onClick={(event) => toggleSelection(group, event)}
                >
                  <div className={cn("flex size-5 shrink-0 items-center justify-center rounded border transition-colors duration-md", selected ? "border-primary bg-primary text-primary-foreground" : "border-input")}>
                    {selected ? <Eye className="size-3.5 animate-in fade-in-0 zoom-in-50 duration-150" /> : null}
                  </div>
                  <div className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted/40">
                    {group.primary.onlineMetadataLoading ? (
                      <Loader2 className="size-5 animate-spin text-muted-foreground" />
                    ) : group.primary.iconUrl ? (
                      <img src={group.primary.iconUrl} alt="" className="size-full object-cover" />
                    ) : (
                      <Package className="size-6 text-muted-foreground" />
                    )}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex min-w-0 items-center gap-2">
                      <span className="truncate text-sm font-semibold">{displayModName(group)}</span>
                      {hasGroupedDetails(group) ? (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Badge variant="outline" className="shrink-0">{group.strong.length + group.jij.length}</Badge>
                          </TooltipTrigger>
                          <TooltipContent className="whitespace-pre-line">{groupTooltip(group)}</TooltipContent>
                        </Tooltip>
                      ) : null}
                      {unused ? (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Badge variant="outline" className="shrink-0 border-amber-500/40 text-amber-700 dark:text-amber-300">
                              {t("manage.unused.chip")}
                            </Badge>
                          </TooltipTrigger>
                          <TooltipContent className="whitespace-pre-line">{tooltipFor(group)}</TooltipContent>
                        </Tooltip>
                      ) : null}
                    </div>
                    <div className="flex min-w-0 items-center gap-1 text-xs text-muted-foreground">
                      <span className="truncate">{strongModIds(group).join(", ")}</span>
                      {group.primary.version ? (
                        <>
                          <span className="shrink-0">·</span>
                          <button
                            type="button"
                            className="group/version shrink-0 transition-[color,opacity] duration-fast active:opacity-60 disabled:no-underline"
                            disabled={!canReplaceVersion(group)}
                            onClick={(event) => {
                              event.stopPropagation();
                              openVersion(group);
                            }}
                          >
                            <span className="underline-offset-2 group-hover/version:underline">
                              {group.primary.version}
                            </span>
                          </button>
                        </>
                      ) : null}
                      {group.primary.fileName || group.primary.path ? (
                        <span className="truncate">· {group.primary.fileName || group.primary.path}</span>
                      ) : null}
                    </div>
                    {categories.length ? (
                      <div className="mt-1 flex flex-wrap gap-1">
                        {categories.slice(0, 3).map((category) => (
                          <Badge key={category} variant="secondary" className="max-w-32 truncate text-[0.65rem]">
                            {category}
                          </Badge>
                        ))}
                      </div>
                    ) : null}
                  </div>
                  <div className="flex shrink-0 items-center gap-1.5">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          size="icon-sm"
                          variant="secondary"
                          disabled={!canFavoriteGroup(group)}
                          aria-label={t("favorites.addToFavorites")}
                          onClick={(event) => {
                            event.stopPropagation();
                            openFavorites([group]);
                          }}
                        >
                          <ListPlus className="size-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{t("favorites.addToFavorites")}</TooltipContent>
                    </Tooltip>
                    <Button
                      type="button"
                      size="sm"
                      variant="secondary"
                      className={group.primary.enabled ? "text-emerald-700 dark:text-emerald-300" : "text-amber-700 dark:text-amber-300"}
                      disabled={isBatchBusy}
                      onClick={(event) => {
                        event.stopPropagation();
                        void toggleGroup(group);
                      }}
                    >
                      {pendingTogglePath === group.primary.path ? (
                        <Loader2 className="size-4 animate-spin" />
                      ) : group.primary.enabled ? (
                        <ToggleRight className="size-4" />
                      ) : (
                        <ToggleLeft className="size-4" />
                      )}
                      <span className="hidden sm:inline">
                        {t(groupAction === "disable" ? "manage.disableSelected" : "manage.enableSelected")}
                      </span>
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      <div className="pointer-events-none absolute inset-x-6 bottom-5 z-30 flex justify-center">
        <AnimatedPresence
          open={selectedGroups.length > 0}
          mode="float"
          preserveContent
          className="pointer-events-auto max-w-full"
        >
          <div
            role="toolbar"
            aria-label={t("manage.selectedCount", { n: selectedGroups.length })}
            className="md-selection-toolbar flex max-w-full flex-wrap items-center justify-center gap-1.5 rounded-2xl border bg-popover/95 p-2 shadow-2xl backdrop-blur-md"
          >
            <span
              aria-hidden="true"
              className="flex size-9 items-center justify-center rounded-full bg-sky-500/15 text-sm font-semibold text-sky-600 dark:text-sky-300"
            >
              {selectedGroups.length}
            </span>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  size="icon"
                  variant="ghost"
                  className="rounded-full bg-teal-500/15 text-teal-700 hover:bg-teal-500/25 dark:text-teal-300"
                  disabled={!selectedGroups.every(canFavoriteGroup)}
                  aria-label={t("favorites.addToFavorites")}
                  onClick={() => openFavorites(selectedGroups)}
                >
                  <ListPlus className="size-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>{t("favorites.addToFavorites")}</TooltipContent>
            </Tooltip>
            {selectionActions.map((action) => {
              const Icon =
                action.icon === "copy"
                  ? Copy
                  : action.icon === "identifier"
                    ? ListTree
                    : action.icon === "delete"
                      ? Trash2
                      : action.icon === "deselect"
                        ? X
                        : action.icon === "invert"
                          ? RefreshCw
                          : action.icon === "enable"
                            ? ToggleRight
                            : ToggleLeft;
              const tone =
                action.key === "deselect"
                  ? "border border-destructive/40 bg-transparent text-destructive hover:bg-destructive/10"
                  : action.color === "warning"
                    ? "bg-amber-500/15 text-amber-700 hover:bg-amber-500/25 dark:text-amber-300"
                    : action.color === "destructive"
                      ? "bg-destructive/10 text-destructive hover:bg-destructive/20"
                      : action.color === "primary"
                        ? "bg-sky-500/15 text-sky-600 hover:bg-sky-500/25 dark:text-sky-300"
                        : "bg-muted/70 text-foreground/70 hover:bg-muted hover:text-foreground";
              return (
                <Tooltip key={action.key}>
                  <TooltipTrigger asChild>
                    <Button
                      type="button"
                      size="icon"
                      variant="ghost"
                      className={cn("rounded-full", tone)}
                      disabled={isBatchBusy}
                      aria-label={action.label}
                      onClick={action.run}
                    >
                      {action.loading ? <Loader2 className="size-4 animate-spin" /> : <Icon className="size-4" />}
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{action.label}</TooltipContent>
                </Tooltip>
              );
            })}
          </div>
        </AnimatedPresence>
      </div>

      <AlertDialog open={deleteDialog} onOpenChange={(value) => (value ? setDeleteDialog(true) : closeDeleteDialog())}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("manage.confirmDelete.title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("manage.confirmDelete.body", { n: pendingDeleteCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isBatchBusy} onClick={closeDeleteDialog}>
              {t("manage.confirmDelete.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={isBatchBusy}
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={(event) => {
                event.preventDefault();
                void confirmDelete();
              }}
            >
              {batchOperation === "delete" ? <Loader2 className="size-4 animate-spin" /> : null}
              {t("manage.confirmDelete.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={disableImpactDialog} onOpenChange={(value) => (value ? setDisableImpactDialog(true) : closeDisableImpactDialog())}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("manage.disableImpact.title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 text-sm">
            <p>{t("manage.disableImpact.body")}</p>
            <div className="max-h-72 overflow-y-auto rounded-md border">
              {disableImpacts.map((impact) => {
                const candidates = impact.candidates ?? [];
                const affected = (impact.affectedMods ?? [])
                  .map((mod) => mod.name || mod.fileName || mod.path)
                  .filter(Boolean)
                  .join(", ");
                return (
                  <div key={impact.modId} className="flex items-center gap-2 border-b px-2 py-2 last:border-b-0">
                    <AlertTriangle className="size-4 text-amber-500" />
                    <div className="min-w-0 flex-1">
                      <div className="truncate font-medium">
                        {t("manage.disableImpact.missing", { id: impact.modId })}
                      </div>
                      <div className="truncate text-xs text-muted-foreground">
                        {t("manage.disableImpact.affected", { mods: affected })}
                      </div>
                    </div>
                    {candidates.length === 0 ? (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="text-muted-foreground">
                            <Download className="size-4" />
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>{t("manage.disableImpact.cacheMiss")}</TooltipContent>
                      </Tooltip>
                    ) : candidates.length === 1 ? (
                      <Button
                        type="button"
                        size="icon-sm"
                        variant="secondary"
                        disabled={Boolean(restoringDependencyId) || restoredDependencies[impact.modId]}
                        onClick={() => void restoreDependency(impact, candidates[0])}
                      >
                        {restoringDependencyId === impact.modId ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                      </Button>
                    ) : (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button type="button" size="icon-sm" variant="secondary" disabled={Boolean(restoringDependencyId) || restoredDependencies[impact.modId]}>
                            {restoringDependencyId === impact.modId ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {candidates.map((candidate) => (
                            <DropdownMenuItem
                              key={`${candidate.platform}:${candidate.projectId}:${candidate.versionId}`}
                              onClick={() => void restoreDependency(impact, candidate)}
                            >
                              {candidate.title || candidate.projectId}
                            </DropdownMenuItem>
                          ))}
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" disabled={Boolean(restoringDependencyId)} onClick={closeDisableImpactDialog}>
              {t("manage.disableImpact.cancel")}
            </Button>
            <Button
              type="button"
              variant="secondary"
              className="text-amber-700 dark:text-amber-300"
              disabled={Boolean(restoringDependencyId) || isBatchBusy}
              onClick={() => void confirmDisableImpact()}
            >
              {isBatchBusy ? <Loader2 className="size-4 animate-spin" /> : null}
              {t("manage.disableImpact.continue")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ManageVersionDialog
        open={versionOpen}
        group={versionGroup}
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

/**
 * Search-syntax help. This is deliberately hover/focus only: a click does not
 * toggle or dismiss it, so there is just one interaction model.
 */
function HelpTooltip({ label }: { label: string }) {
  const [open, setOpen] = useState(false);
  const hovered = useRef(false);
  const focused = useRef(false);

  const show = () => setOpen(true);
  const sync = () => {
    if (!hovered.current && !focused.current) setOpen(false);
  };

  return (
    <Tooltip
      open={open}
      onOpenChange={(next) => {
        if (next) show();
        else if (!hovered.current && !focused.current) setOpen(false);
      }}
    >
      <TooltipTrigger asChild>
        <span
          tabIndex={0}
          role="img"
          aria-label={label}
          className="md-pressable inline-flex size-6 shrink-0 cursor-help items-center justify-center rounded-full text-sm font-bold text-muted-foreground outline-none hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"
          onPointerEnter={() => {
            hovered.current = true;
            show();
          }}
          onPointerLeave={() => {
            hovered.current = false;
            sync();
          }}
          onFocus={() => {
            focused.current = true;
            show();
          }}
          onBlur={() => {
            focused.current = false;
            sync();
          }}
          onPointerDown={(event) => {
            // Keep the click from moving focus/dismissing the Radix tooltip.
            event.preventDefault();
            event.stopPropagation();
          }}
          onClick={(event) => event.stopPropagation()}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.preventDefault();
              focused.current = false;
              setOpen(false);
            }
          }}
        >
          ?
        </span>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">{label}</TooltipContent>
    </Tooltip>
  );
}
