import { useEffect, useMemo, useRef, useState } from "react";
import {
  AlertCircle,
  AlertTriangle,
  Ban,
  Check,
  CheckCircle2,
  Copy,
  Download,
  FileWarning,
  Loader2,
  Lock,
  PinOff,
  ListPlus,
  RefreshCw,
  Search,
  X,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { AddToFavoriteDialog } from "@/components/AddToFavoriteDialog";
import { AnimatedPresence } from "@/components/AnimatedPresence";
import { ModVersionList } from "@/components/ModVersionList";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { SearchInput } from "@/components/ui/search-input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { api } from "@/services/app";
import type { FavoriteModDraft } from "@/stores/favorites";
import {
  modIDFromResult,
  projectKey,
  useDownloadSearchStore,
} from "@/stores/downloadSearch";
import { useMinecraftStore } from "@/stores/minecraft";
import { useSettingsStore } from "@/stores/settings";
import type {
  ModDownloadButtonState,
  ModProject,
  ModVersion,
} from "@/types";
import { cn } from "@/lib/utils";

type DependencyPreflightEntry = NonNullable<import("@/types").DependencyPreflightResult["entries"]>[number];

function resultStateIcon(state: ModDownloadButtonState | undefined, busy: boolean) {
  if (busy) return <Loader2 className="size-4 animate-spin" />;
  switch (state?.status) {
    case "installed":
      return <CheckCircle2 className="size-4" />;
    case "update":
      return <RefreshCw className="size-4" />;
    case "conflict":
    case "incompatible":
      return <AlertTriangle className="size-4" />;
    case "metadataLoading":
      return <Loader2 className="size-4 animate-spin" />;
    case "metadataFailed":
      return <AlertCircle className="size-4" />;
    case "noSelectedVersion":
    case "invalidRequest":
      return <Ban className="size-4" />;
    default:
      return <Download className="size-4" />;
  }
}

const statusLabelKey = (status: string | undefined) => {
  switch (status) {
    case "installed":
      return "versions.installed";
    case "update":
      return "download.confirmReplace.updateTitle";
    case "conflict":
      return "download.confirmReplace.conflictTitle";
    case "incompatible":
      return "download.confirmReplace.incompatibleTitle";
    case "noSelectedVersion":
      return "download.errors.noSelectedVersion";
    case "metadataLoading":
      return "download.status.metadataLoading";
    case "metadataFailed":
      return "download.status.metadataFailed";
    default:
      return "download.status.download";
  }
};

// Use each provider's brand color so search results are recognizable at a glance.
const platformBadgeClass = (platform: string) => {
  switch (platform.toLowerCase()) {
    case "modrinth":
      return "bg-[#1bd96a]/15 text-[#087a3c] dark:bg-[#1bd96a]/15 dark:text-[#7bf0ad]";
    case "curseforge":
      return "bg-[#f16436]/15 text-[#a83a17] dark:bg-[#f16436]/15 dark:text-[#ff9a7a]";
    default:
      return "";
  }
};

const platformBorderColor = (platform: string) => {
  switch (platform.toLowerCase()) {
    case "modrinth":
      return "rgb(27 217 106 / 0.45)";
    case "curseforge":
      return "rgb(241 100 54 / 0.45)";
    default:
      return undefined;
  }
};

export function DownloadPage() {
  const { t } = useTranslation();
  const query = useDownloadSearchStore((state) => state.query);
  const lastSearchedQuery = useDownloadSearchStore((state) => state.lastSearchedQuery);
  const setQuery = useDownloadSearchStore((state) => state.setQuery);
  const results = useDownloadSearchStore((state) => state.results);
  const downloadStates = useDownloadSearchStore((state) => state.downloadStates);
  const searching = useDownloadSearchStore((state) => state.searching);
  const loadingMore = useDownloadSearchStore((state) => state.loadingMore);
  const hasMore = useDownloadSearchStore((state) => state.hasMore);
  const error = useDownloadSearchStore((state) => state.error);
  const downloadingKeys = useDownloadSearchStore((state) => state.downloadingKeys);
  const runSearch = useDownloadSearchStore((state) => state.runSearch);
  const loadMore = useDownloadSearchStore((state) => state.loadMoreSearchResults);
  const refreshDownloadStates = useDownloadSearchStore((state) => state.refreshDownloadStates);
  const installMod = useDownloadSearchStore((state) => state.installMod);
  const confirmInstall = useDownloadSearchStore((state) => state.confirmInstall);
  const closeConfirm = useDownloadSearchStore((state) => state.closeConfirmDialog);
  const batchInstall = useDownloadSearchStore((state) => state.batchInstall);
  const confirmBatch = useDownloadSearchStore((state) => state.confirmBatchInstall);
  const skipBatch = useDownloadSearchStore((state) => state.skipConflictedBatchInstall);
  const cancelBatch = useDownloadSearchStore((state) => state.cancelBatchInstall);
  const confirmPreflight = useDownloadSearchStore((state) => state.confirmPreflightInstall);
  const closePreflight = useDownloadSearchStore((state) => state.closePreflightDialog);
  const batchUnpin = useDownloadSearchStore((state) => state.batchUnpin);
  const showVersionsOverlay = useDownloadSearchStore((state) => state.showVersionsOverlay);
  const closeVersionsOverlay = useDownloadSearchStore((state) => state.closeVersionsOverlay);
  const selectedMod = useDownloadSearchStore((state) => state.selectedMod);
  const matchingVersions = useDownloadSearchStore((state) => state.matchingVersions);
  const isLoadingVersions = useDownloadSearchStore((state) => state.isLoadingVersions);
  const pinnedVersionId = useDownloadSearchStore((state) => state.pinnedVersionId);
  const pinningVersionId = useDownloadSearchStore((state) => state.pinningVersionId);
  const openVersionsOverlay = useDownloadSearchStore((state) => state.openVersionsOverlay);
  const pinVersion = useDownloadSearchStore((state) => state.pinVersion);
  const confirmDialog = useDownloadSearchStore((state) => state.confirmDialog);
  const preflightDialog = useDownloadSearchStore((state) => state.preflightDialog);
  const batchConfirmDialog = useDownloadSearchStore((state) => state.batchConfirmDialog);
  const batchInstalling = useDownloadSearchStore((state) => state.batchInstalling);
  const autoSearchToken = useDownloadSearchStore((state) => state.autoSearchToken);

  const minecraftVersion = useMinecraftStore((state) => state.minecraftVersion);
  const modLoader = useMinecraftStore((state) => state.modLoader);
  const chooseDir = useMinecraftStore((state) => state.chooseDir);
  const hasSelectedInstance = useMinecraftStore((state) => Boolean(state.selected?.id || state.selected?.name));
  const requireCtrlForMultiSelect = useSettingsStore((state) => state.settings?.requireCtrlForMultiSelect ?? false);

  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [favoriteDrafts, setFavoriteDrafts] = useState<FavoriteModDraft[]>([]);
  const [favoriteOpen, setFavoriteOpen] = useState(false);
  const [dirInvalid, setDirInvalid] = useState(false);
  const resultsScrollRef = useRef<HTMLDivElement | null>(null);
  const loadMoreSentinelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    useDownloadSearchStore.getState().start();
    return () => useDownloadSearchStore.getState().stop();
  }, []);

  useEffect(() => {
    if (autoSearchToken <= 0) return;
    useDownloadSearchStore.setState({ autoSearchToken: 0 });
    void runSearch();
  }, [autoSearchToken, runSearch]);

  useEffect(() => {
    void api.ValidateMinecraftDir().then((valid) => setDirInvalid(!valid));
    void refreshDownloadStates();
  }, [minecraftVersion, modLoader, hasSelectedInstance, refreshDownloadStates]);

  useEffect(() => {
    setSelectedKeys((current) => {
      const valid = new Set(results.map((result) => projectKey(result)));
      const next = new Set([...current].filter((key) => valid.has(key)));
      return next.size === current.size ? current : next;
    });
  }, [results]);

  useEffect(() => {
    const root = resultsScrollRef.current;
    const sentinel = loadMoreSentinelRef.current;
    if (!root || !sentinel || !hasMore || error) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) void loadMore();
      },
      { root, rootMargin: "240px 0px", threshold: 0 },
    );
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [error, hasMore, loadMore, loadingMore, searching]);

  const selectedResults = useMemo(
    () => results.filter((result) => selectedKeys.has(projectKey(result))),
    [results, selectedKeys],
  );

  const hasQuery = query.trim().length > 0;
  // Typing alone should not claim there are no matches: only switch to the
  // empty-results message after the current query has actually been searched.
  const showNoResults = hasQuery && lastSearchedQuery.trim() === query.trim();

  const toggleSelection = (result: ModProject, event: { ctrlKey: boolean; metaKey: boolean }) => {
    const key = projectKey(result);
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

  const copyNames = async () => {
    try {
      await navigator.clipboard.writeText(selectedResults.map((result) => result.title).join("\n"));
      toast.success(t("download.selection.copiedNames"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    }
  };

  const favoriteDraftFromResult = (result: ModProject): FavoriteModDraft => ({
    platform: result.platform,
    modId: modIDFromResult(result),
    minecraftVersion,
    modLoader,
    title: result.title,
    slug: result.slug,
    iconUrl: result.iconUrl,
    description: result.description,
    categories: result.categories || [],
  });

  const openFavorites = (items: ModProject[]) => {
    if (!minecraftVersion.trim() || !modLoader.trim()) {
      toast.warning(t("download.errors.noSelectedVersion"));
      return;
    }
    const drafts = items
      .map(favoriteDraftFromResult)
      .filter((draft) => draft.platform && draft.modId);
    if (!drafts.length) {
      toast.warning(t("favorites.invalidSelection"));
      return;
    }
    setFavoriteDrafts(drafts);
    setFavoriteOpen(true);
  };

  const stateFor = (index: number) => downloadStates[index];
  const busyFor = (state: ModDownloadButtonState | undefined) =>
    Boolean(state?.loading || (state?.key && downloadingKeys[state.key]));

  const confirmTitleKey = () => {
    switch (confirmDialog.status) {
      case "update":
        return "download.confirmReplace.updateTitle";
      case "incompatible":
        return "download.confirmReplace.incompatibleTitle";
      default:
        return "download.confirmReplace.conflictTitle";
    }
  };

  const confirmBodyKey = () => {
    switch (confirmDialog.status) {
      case "update":
        return "download.confirmReplace.updateBody";
      case "incompatible":
        return "download.confirmReplace.incompatibleBody";
      default:
        return "download.confirmReplace.conflictBody";
    }
  };

  const preflightMeta = (entry: DependencyPreflightEntry) => {
    switch (entry.status) {
      case "missing":
        return { icon: <Download className="size-4 text-sky-500" />, label: t("download.preflight.status.missing") };
      case "conflict":
        return { icon: <AlertTriangle className="size-4 text-destructive" />, label: t("download.preflight.status.conflict") };
      default:
        return { icon: <CheckCircle2 className="size-4 text-emerald-500" />, label: t("download.preflight.status.fulfilled") };
    }
  };

  const showEmpty = !searching && results.length === 0;

  return (
    <div className="relative mx-auto flex h-full w-full max-w-5xl flex-col overflow-hidden p-6">
      <form
        className="flex flex-col gap-2 pb-4 sm:flex-row"
        onSubmit={(event) => {
          event.preventDefault();
          void runSearch();
        }}
      >
        <div className="min-w-0 flex-1">
          <SearchInput
            value={query}
            onValueChange={setQuery}
            placeholder="emi"
            aria-label={t("search.label")}
            clearAriaLabel={t("common.clearSearch")}
            className="h-10"
            autoComplete="off"
          />
        </div>
        <Button type="submit" className="h-10 shrink-0" disabled={searching}>
          {searching ? <Loader2 className="size-4 animate-spin" /> : <Search className="size-4" />}
          {t("search.action")}
        </Button>
      </form>

      {error ? (
        <div className="mb-3 rounded-md border border-destructive/30 bg-destructive/5 p-2 text-sm text-destructive animate-in fade-in-0 slide-in-from-top-1 duration-200">
          {t("download.searchFailed", { reason: error })}
        </div>
      ) : null}

      {searching ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 text-muted-foreground">
          <Loader2 className="size-8 animate-spin" />
          <span className="text-sm">{t("search.action")}…</span>
        </div>
      ) : showEmpty ? (
        <div className="flex flex-1 flex-col items-center justify-center gap-3 pb-24 text-center animate-in fade-in-0 duration-200">
          <div className="flex size-16 items-center justify-center rounded-full bg-primary/10 text-primary">
            {showNoResults ? <AlertCircle className="size-8" /> : <Search className="size-8" />}
          </div>
          <div className="text-base font-semibold">
            {showNoResults ? t("download.searchEmpty.title") : t("download.searchIdle.title")}
          </div>
          <p className="max-w-md text-sm text-muted-foreground">
            {showNoResults ? t("download.searchEmpty.hint") : t("download.searchIdle.hint")}
          </p>
        </div>
      ) : (
        <div ref={resultsScrollRef} className="min-h-0 flex-1 overflow-y-auto overflow-x-hidden pr-1 animate-in fade-in-0 duration-200">
          <div className={cn("grid gap-2", selectedResults.length > 0 && "pb-24")}>
            {results.map((result, index) => {
              const state = stateFor(index);
              const key = projectKey(result);
              const selected = selectedKeys.has(key);
              const busy = busyFor(state);
              return (
                <div
                  key={result.id || `${result.platform}-${result.slug}-${index}`}
                  className={cn(
                    "flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl border bg-card p-3 shadow-sm transition-[background-color,border-color,box-shadow,color,opacity] duration-md ease-[var(--md-ease-out)] animate-in fade-in-0 slide-in-from-bottom-1 duration-200 active:opacity-80 active:duration-75",
                    selected
                      ? "border-primary/50 bg-primary/10 shadow-sm hover:bg-primary/15"
                      : "hover:border-foreground/20 hover:bg-accent/60 hover:shadow-md",
                  )}
                  onClick={(event) => toggleSelection(result, event)}
                >
                  <div className={cn("flex size-5 shrink-0 items-center justify-center rounded border transition-colors duration-md", selected ? "border-primary bg-primary text-primary-foreground" : "border-input")}>
                    {selected ? <Check className="size-3.5 animate-in fade-in-0 zoom-in-50 duration-150" /> : null}
                  </div>
                  <button
                    type="button"
                    className="size-12 shrink-0 overflow-hidden rounded-lg border bg-muted/40 transition-[scale] duration-fast active:scale-95"
                    onClick={(event) => {
                      event.stopPropagation();
                      void openVersionsOverlay(result);
                    }}
                    aria-label={result.title}
                  >
                    {result.iconUrl ? (
                      <img src={result.iconUrl} alt="" className="size-full object-cover" />
                    ) : (
                      <span className="flex size-full items-center justify-center text-lg font-bold text-muted-foreground">
                        {result.title?.slice(0, 1).toUpperCase()}
                      </span>
                    )}
                  </button>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-semibold">{result.title}</div>
                    <div className="line-clamp-2 text-xs text-muted-foreground">{result.description}</div>
                  </div>
                  <Badge
                    variant="outline"
                    className={cn("hidden shrink-0 sm:inline-flex", platformBadgeClass(result.platform))}
                    style={{ borderColor: platformBorderColor(result.platform) }}
                  >
                    {result.platform}
                  </Badge>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        type="button"
                        size="icon-sm"
                        variant="secondary"
                        className="hidden shrink-0 sm:inline-flex"
                        aria-label={t("favorites.addToFavorites")}
                        onClick={(event) => {
                          event.stopPropagation();
                          openFavorites([result]);
                        }}
                      >
                        <ListPlus className="size-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t("favorites.addToFavorites")}</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        type="button"
                        size="icon-sm"
                        variant={state?.status === "conflict" || state?.status === "incompatible" ? "destructive" : "outline"}
                        className="shrink-0"
                        disabled={!state || Boolean(state.disabled) || busy}
                        aria-label={t(statusLabelKey(state?.status))}
                        onClick={(event) => {
                          event.stopPropagation();
                          void installMod({
                            result,
                            key: state?.key,
                            status: state?.status,
                            conflictFileName: state?.conflictFileName,
                            confirm: true,
                          });
                        }}
                        onContextMenu={(event) => {
                          event.preventDefault();
                          event.stopPropagation();
                          void installMod({
                            result,
                            key: state?.key,
                            status: state?.status,
                            conflictFileName: state?.conflictFileName,
                            confirm: false,
                          });
                        }}
                      >
                        {resultStateIcon(state, busy)}
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t(statusLabelKey(state?.status))}</TooltipContent>
                  </Tooltip>
                </div>
              );
            })}
          </div>

          <div ref={loadMoreSentinelRef} className="flex min-h-12 items-center justify-center py-4 text-xs text-muted-foreground">
            {loadingMore ? (
              <span className="flex items-center gap-2">
                <Loader2 className="size-4 animate-spin" />
                {t("search.loadingMore")}
              </span>
            ) : error && hasMore ? (
              <Button type="button" variant="outline" size="sm" onClick={() => void loadMore()}>
                <RefreshCw className="size-4" />
                {t("search.retryLoadMore")}
              </Button>
            ) : hasMore ? (
              <span>{t("search.scrollForMore")}</span>
            ) : (
              <span>{t("search.noMoreResults")}</span>
            )}
          </div>
        </div>
      )}

      <AnimatedPresence open={selectedResults.length > 0} mode="float" preserveContent className="absolute inset-x-6 bottom-4 z-10">
        <div className="md-selection-toolbar flex flex-wrap items-center gap-2 rounded-2xl border bg-popover/95 p-2 shadow-2xl backdrop-blur">
          <span className="px-2 text-xs text-muted-foreground">
            {t("download.selection.count", { n: selectedResults.length })}
          </span>
          <Button type="button" size="sm" disabled={batchInstalling} onClick={() => void batchInstall(selectedResults)}>
            {batchInstalling ? <Loader2 className="size-4 animate-spin" /> : <Download className="size-4" />}
            {t("download.selection.downloadAll")}
          </Button>
          <Button type="button" size="sm" variant="secondary" onClick={() => openFavorites(selectedResults)}>
            <ListPlus className="size-4" />
            {t("favorites.addToFavorites")}
          </Button>
          <Button type="button" size="sm" variant="outline" onClick={() => void batchUnpin(selectedResults)}>
            <PinOff className="size-4" />
            {t("download.selection.unpin")}
          </Button>
          <Button type="button" size="sm" variant="ghost" onClick={() => void copyNames()}>
            <Copy className="size-4" />
            {t("download.selection.copyNames")}
          </Button>
          <Button type="button" size="sm" variant="ghost" className="ml-auto text-destructive" onClick={() => setSelectedKeys(new Set())}>
            <X className="size-4" />
            {t("download.selection.deselectAll")}
          </Button>
        </div>
      </AnimatedPresence>

      {dirInvalid ? (
        <div className="absolute inset-0 z-20 flex items-center justify-center bg-background/80 backdrop-blur-sm">
          <div className="mx-4 max-w-md rounded-xl border bg-card p-6 text-center shadow-xl">
            <FileWarning className="mx-auto size-12 text-amber-500" />
            <h2 className="mt-3 text-lg font-semibold">{t("download.selectDirTitle")}</h2>
            <p className="mt-2 text-sm text-muted-foreground">{t("download.selectDirDesc")}</p>
            <Button
              type="button"
              className="mt-4"
              onClick={() => {
                void chooseDir()
                  .then((dir) => {
                    if (dir) setDirInvalid(false);
                  })
                  .catch((error) => toast.error(error instanceof Error ? error.message : String(error)));
              }}
            >
              {t("download.chooseDir")}
            </Button>
          </div>
        </div>
      ) : null}

      <Dialog open={showVersionsOverlay} onOpenChange={(value) => (value ? undefined : closeVersionsOverlay())}>
        <DialogContent className="max-w-3xl gap-0 p-0">
          <DialogHeader className="border-b px-4 py-3 pr-12">
            <DialogTitle>{selectedMod?.title || t("download.versionsTitle")}</DialogTitle>
          </DialogHeader>
          <ModVersionList
            versions={matchingVersions}
            loading={isLoadingVersions}
            action="pin"
            pinnedVersionId={pinnedVersionId}
            busyVersionId={pinningVersionId}
            onSelect={(version: ModVersion) => void pinVersion(version)}
          />
        </DialogContent>
      </Dialog>

      <Dialog open={confirmDialog.show} onOpenChange={(value) => (value ? undefined : closeConfirm())}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t(confirmTitleKey())}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 text-sm">
            {confirmDialog.status === "conflict" && confirmDialog.conflictFileName ? (
              <div className="font-medium">
                {t("download.confirmReplace.conflictFile", { file: confirmDialog.conflictFileName })}
              </div>
            ) : null}
            <p>{t(confirmBodyKey())}</p>
            {confirmDialog.status === "conflict" ? (
              <p className="text-xs text-muted-foreground">{t("download.confirmReplace.conflictHint")}</p>
            ) : null}
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={closeConfirm}>
              {t("download.confirmReplace.cancel")}
            </Button>
            <Button type="button" className="bg-amber-500 text-white hover:bg-amber-600" onClick={() => void confirmInstall()}>
              {t("download.confirmReplace.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={preflightDialog.show} onOpenChange={(value) => (value ? undefined : closePreflight())}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("download.preflight.title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 text-sm">
            <p>
              {t("download.preflight.summary", {
                missing: preflightDialog.missing,
                conflicts: preflightDialog.conflicts,
              })}
            </p>
            <div className="max-h-72 overflow-y-auto rounded-md border">
              {preflightDialog.entries.map((entry, index) => (
                <div key={`${entry.projectId}-${index}`} className="flex items-center gap-2 border-b px-2 py-2 last:border-b-0">
                  {preflightMeta(entry).icon}
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">{entry.title || entry.projectId}</div>
                    <div className="truncate text-xs text-muted-foreground">
                      {entry.localName
                        ? t("download.preflight.localVersion", {
                            name: entry.localName,
                            version: entry.localVersion || t("download.preflight.unknownVersion"),
                          })
                        : entry.projectId}
                    </div>
                  </div>
                  <Badge variant="outline">{preflightMeta(entry).label}</Badge>
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={closePreflight}>
              {t("download.preflight.cancel")}
            </Button>
            <Button type="button" onClick={() => void confirmPreflight()}>
              {t("download.preflight.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={batchConfirmDialog.show} onOpenChange={(value) => (value ? undefined : cancelBatch())}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("download.confirmReplace.batchIncompatibleTitle")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 text-sm">
            <p>{t("download.confirmReplace.batchIncompatibleBody")}</p>
            <div className="max-h-72 overflow-y-auto">
              {batchConfirmDialog.conflicts.map((conflict) => (
                <div key={conflict.key} className="mb-2 rounded-md border p-2">
                  <div className="font-semibold">{conflict.title || conflict.key}</div>
                  {conflict.conflicts?.map((entry) => (
                    <div key={`${conflict.key}-${entry.incompatibleProjectKey}`} className="mt-1 text-xs">
                      <div>{entry.incompatibleTitle || entry.incompatibleProjectKey}</div>
                      {entry.paths?.map((path) => (
                        <div key={path.path} className="truncate text-muted-foreground">
                          {path.fileName || path.path}
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={cancelBatch}>
              {t("download.confirmReplace.cancel")}
            </Button>
            <Button type="button" variant="secondary" className="text-amber-700 dark:text-amber-300" onClick={() => void skipBatch()}>
              {t("download.confirmReplace.skipConflicted")}
            </Button>
            <Button type="button" className="bg-amber-500 text-white hover:bg-amber-600" onClick={() => void confirmBatch()}>
              {t("download.confirmReplace.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AddToFavoriteDialog
        open={favoriteOpen}
        drafts={favoriteDrafts}
        onOpenChange={setFavoriteOpen}
        onAdded={() => toast.success(t("favorites.added"))}
      />

      {hasSelectedInstance ? null : (
        <div className="pointer-events-none absolute bottom-20 left-1/2 flex -translate-x-1/2 items-center gap-1 rounded-full border bg-popover px-3 py-1 text-xs text-muted-foreground shadow">
          <Lock className="size-3" />
          {t("download.errors.noSelectedVersion")}
        </div>
      )}
    </div>
  );
}
