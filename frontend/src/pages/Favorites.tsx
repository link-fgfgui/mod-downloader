import { useEffect, useMemo, useState } from "react";
import {
  Archive,
  Copy,
  Eye,
  FolderSync,
  GripVertical,
  Link2,
  Loader2,
  MoreVertical,
  Palette,
  Pencil,
  Pin,
  PinOff,
  ListMusic,
  ListPlus,
  RefreshCw,
  Star,
  Trash2,
  X,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { CrossVersionCopyDialog } from "@/components/CrossVersionCopyDialog";
import { AnimatedPresence } from "@/components/AnimatedPresence";
import { FavoritePinDialog } from "@/components/FavoritePinDialog";
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
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Switch } from "@/components/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useDownloadSearchStore } from "@/stores/downloadSearch";
import {
  hasFavoriteScope,
  listMatchesScope,
  listOrder,
  favoriteModKey,
  useFavoritesStore,
} from "@/stores/favorites";
import { useMinecraftStore } from "@/stores/minecraft";
import { useSettingsStore } from "@/stores/settings";
import { errorMessage } from "@/lib/format";
import type { FavoriteList, FavoriteModEntry, ModProject } from "@/types";
import { cn } from "@/lib/utils";

type CopyMode = "selected" | "list" | "reference";

const displayName = (item: FavoriteModEntry) => item.title || item.slug || item.modId;

const projectFromFavorite = (item: FavoriteModEntry): ModProject => ({
  id: `${item.platform}:${item.modId}`,
  platform: item.platform,
  projectId: item.modId,
  slug: item.slug || "",
  title: displayName(item),
  icon: "",
  iconUrl: item.iconUrl || "",
  description: item.description || "",
  downloads: 0,
  categories: item.categories || [],
  updatedAt: 0,
  cachedAt: 0,
});

export function FavoritesPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const lists = useFavoritesStore((state) => state.lists);
  const selectedListId = useFavoritesStore((state) => state.selectedListId);
  const displayMinecraftVersion = useFavoritesStore((state) => state.displayMinecraftVersion);
  const displayModLoader = useFavoritesStore((state) => state.displayModLoader);
  const contents = useFavoritesStore((state) => state.contents);
  const items = useFavoritesStore((state) => state.items);
  const loadingLists = useFavoritesStore((state) => state.loadingLists);
  const loadingItems = useFavoritesStore((state) => state.loadingItems);
  const exporting = useFavoritesStore((state) => state.exportingPackwiz);
  const error = useFavoritesStore((state) => state.error);
  const setDisplayScope = useFavoritesStore((state) => state.setDisplayScope);
  const loadLists = useFavoritesStore((state) => state.loadLists);
  const loadItems = useFavoritesStore((state) => state.loadItems);
  const selectList = useFavoritesStore((state) => state.selectList);
  const createList = useFavoritesStore((state) => state.createList);
  const renameList = useFavoritesStore((state) => state.renameList);
  const deleteList = useFavoritesStore((state) => state.deleteList);
  const updateListMetadata = useFavoritesStore((state) => state.updateListMetadata);
  const updateListIcon = useFavoritesStore((state) => state.updateListIcon);
  const clearListIcon = useFavoritesStore((state) => state.clearListIcon);
  const setListPinned = useFavoritesStore((state) => state.setListPinned);
  const reorderLists = useFavoritesStore((state) => state.reorderLists);
  const removeMod = useFavoritesStore((state) => state.remove);
  const removeMany = useFavoritesStore((state) => state.removeMany);
  const exportPackwiz = useFavoritesStore((state) => state.exportPackwiz);
  const copySelectedMods = useFavoritesStore((state) => state.copySelectedMods);
  const copyList = useFavoritesStore((state) => state.copyList);
  const addListReference = useFavoritesStore((state) => state.addListReference);
  const minecraftVersion = useMinecraftStore((state) => state.minecraftVersion);
  const modLoader = useMinecraftStore((state) => state.modLoader);
  const requireCtrlForMultiSelect = useSettingsStore((state) => state.settings?.requireCtrlForMultiSelect ?? false);
  const showResults = useDownloadSearchStore((state) => state.showResults);

  const [listEdit, setListEdit] = useState<{ open: boolean; id: string; name: string }>({ open: false, id: "", name: "" });
  const [deleteTarget, setDeleteTarget] = useState<FavoriteList | null>(null);
  const [metadata, setMetadata] = useState<{
    open: boolean;
    list: FavoriteList | null;
    pinned: boolean;
    iconKind: string;
    iconValue: string;
    iconPlatform: string;
  }>({ open: false, list: null, pinned: false, iconKind: "mdi", iconValue: "", iconPlatform: "modrinth" });
  const [copyDialog, setCopyDialog] = useState<{
    open: boolean;
    mode: CopyMode;
    sourceListId: string;
    targetListId: string;
    targetListIds: string[];
    mods: FavoriteModEntry[];
  }>({ open: false, mode: "selected", sourceListId: "", targetListId: "", targetListIds: [], mods: [] });
  const [crossList, setCrossList] = useState<FavoriteList | null>(null);
  const [pinItem, setPinItem] = useState<FavoriteModEntry | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [draggedListId, setDraggedListId] = useState("");
  const [dropTargetId, setDropTargetId] = useState("");

  useEffect(() => {
    setDisplayScope(minecraftVersion, modLoader);
    void loadLists();
  }, [loadLists, minecraftVersion, modLoader, setDisplayScope]);

  useEffect(() => {
    setSelectedKeys(new Set());
  }, [selectedListId]);

  const scopedLists = useMemo(
    () => lists.filter((list) => listMatchesScope(list, displayMinecraftVersion, displayModLoader)),
    [displayMinecraftVersion, displayModLoader, lists],
  );
  const pinnedLists = useMemo(() => scopedLists.filter((list) => list.pinned).sort(listOrder), [scopedLists]);
  const ungroupedLists = useMemo(() => scopedLists.filter((list) => !list.pinned).sort(listOrder), [scopedLists]);
  const selectedList = useMemo(
    () => lists.find((list) => list.id === selectedListId) ?? null,
    [lists, selectedListId],
  );
  const selectedItems = useMemo(
    () => items.filter((item) => selectedKeys.has(favoriteModKey(item))),
    [items, selectedKeys],
  );

  useEffect(() => {
    const valid = new Set(items.map(favoriteModKey));
    setSelectedKeys((current) => {
      const next = new Set([...current].filter((key) => valid.has(key)));
      return next.size === current.size ? current : next;
    });
  }, [items]);

  const canCreate = Boolean(minecraftVersion.trim() && modLoader.trim());
  const scopeReady = hasFavoriteScope(displayMinecraftVersion, displayModLoader);
  const targetListOptions = lists
    .filter((list) => copyDialog.mode === "selected" || list.id !== copyDialog.sourceListId)
    .map((list) => ({ value: list.id, label: list.name }));
  const copyReady =
    copyDialog.mode === "selected"
      ? copyDialog.targetListIds.length > 0 && copyDialog.mods.length > 0
      : Boolean(copyDialog.targetListId);

  const listIconUrl = (list: FavoriteList | null) =>
    list?.iconKind === "project" ? list.iconUrl || "" : "";

  const toggleItem = (item: FavoriteModEntry, event: { ctrlKey: boolean; metaKey: boolean }) => {
    const key = favoriteModKey(item);
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

  const openListEdit = (list?: FavoriteList) => {
    if (!list && !canCreate) return;
    setListEdit({ open: true, id: list?.id || "", name: list?.name || "" });
  };

  const saveListName = async () => {
    const name = listEdit.name.trim();
    if (!name) return;
    try {
      if (listEdit.id) await renameList(listEdit.id, name);
      else await createList(name);
      setListEdit({ open: false, id: "", name: "" });
    } catch (error) {
      toast.error(errorMessage(error));
    }
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;
    try {
      await deleteList(deleteTarget.id);
      setDeleteTarget(null);
      toast.success(t("favorites.listDeleted"));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    }
  };

  const openMetadata = (list: FavoriteList) => {
    setMetadata({
      open: true,
      list,
      pinned: Boolean(list.pinned),
      iconKind: list.iconKind || "mdi",
      iconValue: list.iconValue || "",
      iconPlatform: "modrinth",
    });
  };

  const saveMetadata = async () => {
    if (!metadata.list) return;
    try {
      // Re-read the list so fields changed while the dialog was open (for
      // example Clear icon, which persists immediately) are not overwritten.
      const fresh = useFavoritesStore.getState().lists.find((list) => list.id === metadata.list?.id) || metadata.list;
      const updated = await updateListMetadata(fresh, { pinned: metadata.pinned });
      if (updated && metadata.iconValue.trim()) {
        await updateListIcon(updated, metadata.iconKind === "project" ? "project" : "mdi", metadata.iconValue, metadata.iconPlatform);
      }
      setMetadata((current) => ({ ...current, open: false }));
      toast.success(t("favorites.listSaved"));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    }
  };

  const handleClearIcon = async () => {
    if (!metadata.list) return;
    try {
      const updated = await clearListIcon(metadata.list);
      setMetadata((current) => ({
        ...current,
        list: updated ?? current.list,
        iconKind: "",
        iconValue: "",
        iconPlatform: "modrinth",
      }));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    }
  };

  const handlePinList = async (list: FavoriteList) => {
    try {
      await setListPinned(list, !list.pinned);
      toast.success(t(list.pinned ? "favorites.listUnpinned" : "favorites.listPinned"));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    }
  };

  const openListAsSearchResults = async (list: FavoriteList) => {
    await selectList(list.id);
    const currentItems = useFavoritesStore.getState().items;
    await showResults(currentItems.map(projectFromFavorite));
    void navigate("/download");
  };

  const openCopy = (mode: CopyMode, list?: FavoriteList) => {
    setCopyDialog({
      open: true,
      mode,
      sourceListId: list?.id || "",
      targetListId: "",
      targetListIds: [],
      mods: mode === "selected" ? selectedItems.filter((item) => !item.referenced) : [],
    });
  };

  const applyCopy = async () => {
    try {
      if (copyDialog.mode === "selected") {
        const result = await copySelectedMods(copyDialog.targetListIds, copyDialog.mods);
        setSelectedKeys(new Set());
        if (result) {
          if (result.errors?.length) {
            toast.error(t("favorites.operationFailed") + ": " + result.errors.join("; "));
          } else {
            toast.success(
              t("favorites.resultSummary", {
                added: result.added,
                updated: result.updated,
                skipped: result.skipped,
              }),
            );
          }
        }
      } else if (copyDialog.mode === "list") {
        const result = await copyList(copyDialog.sourceListId, copyDialog.targetListId);
        if (result) {
          if (result.errors?.length) {
            toast.error(t("favorites.operationFailed") + ": " + result.errors.join("; "));
          } else {
            toast.success(
              t("favorites.resultSummary", {
                added: result.added,
                updated: result.updated,
                skipped: result.skipped,
              }),
            );
          }
        }
      } else {
        const ref = await addListReference(copyDialog.targetListId, copyDialog.sourceListId);
        if (!ref || !(ref as { id?: string }).id) {
          toast.error(t("favorites.references.failed"));
          return;
        }
        toast.success(t("favorites.references.added"));
      }
      setCopyDialog((current) => ({ ...current, open: false }));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    }
  };

  const removeSelected = async () => {
    const removable = selectedItems.filter((item) => !item.referenced);
    if (removable.length === 0) {
      setSelectedKeys(new Set());
      return;
    }
    try {
      await removeMany(removable);
      toast.success(t("favorites.removed"));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    }
    setSelectedKeys(new Set());
  };

  const onDragStart = (event: React.DragEvent, list: FavoriteList) => {
    setDraggedListId(list.id);
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", list.id);
  };

  const onDragOver = (event: React.DragEvent, listId: string) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    if (draggedListId && draggedListId !== listId) setDropTargetId(listId);
  };

  const moveList = async (sourceId: string, pinned: boolean) => {
    const list = lists.find((entry) => entry.id === sourceId);
    if (!list) return null;
    if (Boolean(list.pinned) === pinned) return list;
    return updateListMetadata(list, { pinned });
  };

  const onDrop = async (
    event: React.DragEvent,
    targetListId: string,
    sectionLists: FavoriteList[],
    pinned: boolean,
  ) => {
    event.preventDefault();
    setDropTargetId("");
    const sourceId = draggedListId;
    setDraggedListId("");
    if (!sourceId || sourceId === targetListId) return;
    const moved = await moveList(sourceId, pinned);
    if (!moved) return;
    const ids = sectionLists.map((list) => list.id).filter((id) => id !== sourceId);
    const targetIndex = ids.indexOf(targetListId);
    if (targetIndex < 0) return;
    ids.splice(targetIndex, 0, sourceId);
    await reorderLists(ids);
  };

  const onSectionDrop = async (event: React.DragEvent, pinned: boolean) => {
    event.preventDefault();
    const sourceId = draggedListId;
    setDraggedListId("");
    setDropTargetId("");
    if (!sourceId) return;
    await moveList(sourceId, pinned);
  };

  const handleExport = async () => {
    if (!selectedList || items.length === 0) return;
    try {
      const result = await exportPackwiz(selectedList.id);
      if (!result || result.canceled) return;
      toast.success(t("favorites.export.success"));
    } catch (error) {
      toast.error(t("favorites.export.failed", { reason: errorMessage(error) }));
    }
  };

  const renderListRow = (list: FavoriteList, sectionLists: FavoriteList[], pinned: boolean) => {
    const iconUrl = listIconUrl(list);
    const selected = list.id === selectedListId;
    return (
      <div
        key={list.id}
        draggable
        onDragStart={(event) => onDragStart(event, list)}
        onDragEnd={() => {
          setDraggedListId("");
          setDropTargetId("");
        }}
        onDragOver={(event) => {
          event.stopPropagation();
          onDragOver(event, list.id);
        }}
        onDrop={(event) => {
          event.stopPropagation();
          void onDrop(event, list.id, sectionLists, pinned);
        }}
        className={cn(
          "group relative flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-colors",
          selected ? "bg-primary/10 text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
          dropTargetId === list.id && "ring-2 ring-primary/50",
        )}
        onClick={() => void selectList(list.id)}
      >
        <GripVertical className="size-3.5 shrink-0 cursor-grab text-muted-foreground/60" />
        <span className="flex size-6 shrink-0 items-center justify-center overflow-hidden rounded-md border bg-muted/40">
          {iconUrl ? <img src={iconUrl} alt="" className="size-full object-cover" /> : <ListMusic className="size-3.5" />}
        </span>
        <span className="min-w-0 flex-1 truncate">{list.name}</span>
        {list.pinned ? <Pin className="size-3 text-primary" /> : null}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              size="icon-xs"
              variant="ghost"
              className="opacity-0 transition-opacity duration-fast group-hover:opacity-100 group-focus-within:opacity-100 focus-visible:opacity-100"
              aria-label={t("favorites.actions.customize")}
              onClick={(event) => event.stopPropagation()}
            >
              <MoreVertical className="size-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" onClick={(event) => event.stopPropagation()}>
            <DropdownMenuItem onClick={() => openListEdit(list)}>
              <Pencil className="size-4" /> {t("favorites.actions.rename")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => void handlePinList(list)}>
              {list.pinned ? <PinOff className="size-4" /> : <Pin className="size-4" />}
              {t(list.pinned ? "favorites.actions.unpin" : "favorites.actions.pin")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => openMetadata(list)}>
              <Palette className="size-4" /> {t("favorites.actions.customize")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => openCopy("list", list)}>
              <Copy className="size-4" /> {t("favorites.actions.copyList")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => void openListAsSearchResults(list)}>
              <Eye className="size-4" /> {t("favorites.actions.asSearchResults")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => openCopy("reference", list)}>
              <Link2 className="size-4" /> {t("favorites.actions.referenceList")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setCrossList(list)}>
              <FolderSync className="size-4" /> {t("favorites.actions.copyAcrossVersions")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => setDeleteTarget(list)}>
              <Trash2 className="size-4" /> {t("favorites.actions.delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    );
  };

  return (
    <div className="mx-auto grid h-full w-full max-w-6xl grid-cols-1 gap-5 overflow-hidden p-6 lg:grid-cols-[minmax(15rem,19rem)_minmax(0,1fr)]">
      <aside className="flex min-h-0 flex-col border-b pb-3 lg:border-b-0 lg:border-r lg:pb-0 lg:pr-4">
        <div className="flex items-center justify-between gap-2">
          <h1 className="text-lg font-semibold">{t("favorites.title")}</h1>
          <div className="flex items-center gap-1">
            <Button type="button" size="icon-sm" variant="secondary" disabled={!canCreate} aria-label={t("favorites.dialog.createTitle")} onClick={() => openListEdit()}>
              <ListPlus className="size-4" />
            </Button>
          </div>
        </div>
        {loadingLists ? <Progress value={undefined} className="mt-2 h-0.5 animate-pulse" /> : null}
        <div className="mt-2 min-h-0 flex-1 overflow-y-auto pr-1">
          {lists.length === 0 ? (
            <div className="py-10 text-center text-sm text-muted-foreground">{t("favorites.empty.noLists")}</div>
          ) : (
            <>
              {pinnedLists.length ? (
                <section
                  className="mb-2"
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={(event) => void onSectionDrop(event, true)}
                >
                  <div className="flex items-center gap-1 px-2 py-1 text-[0.7rem] font-bold uppercase tracking-wide text-muted-foreground">
                    <Pin className="size-3" /> {t("favorites.sections.pinned")}
                  </div>
                  {pinnedLists.map((list) => renderListRow(list, pinnedLists, true))}
                </section>
              ) : null}
              <section
                className="mb-2"
                onDragOver={(event) => event.preventDefault()}
                onDrop={(event) => void onSectionDrop(event, false)}
              >
                <div className="flex items-center gap-1 px-2 py-1 text-[0.7rem] font-bold uppercase tracking-wide text-muted-foreground">
                  <Star className="size-3" /> {t("favorites.sections.lists")}
                </div>
                {ungroupedLists.map((list) => renderListRow(list, ungroupedLists, false))}
              </section>
            </>
          )}
        </div>
      </aside>

      <section className="relative flex min-h-0 min-w-0 flex-col">
        {selectedList ? (
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <span className="flex size-8 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted/40">
                  {listIconUrl(selectedList) ? (
                    <img src={listIconUrl(selectedList)} alt="" className="size-full object-cover" />
                  ) : (
                    <ListMusic className="size-4" />
                  )}
                </span>
                <h2 className="truncate text-xl font-semibold">{selectedList.name}</h2>
              </div>
              <div className="mt-1 text-sm text-muted-foreground">
                {t("favorites.itemCount", { n: items.length })}
                {contents?.refs?.length ? ` · ${t("favorites.references.count", { n: contents.refs.length })}` : ""}
              </div>
            </div>
            <div className="flex flex-wrap gap-2">
              <Button type="button" size="sm" variant="secondary" disabled={items.length === 0 || exporting} onClick={() => void handleExport()}>
                {exporting ? <Loader2 className="size-4 animate-spin" /> : <Archive className="size-4" />}
                {t("favorites.actions.exportPackwiz")}
              </Button>
              <Button type="button" size="sm" variant="secondary" disabled={loadingItems} onClick={() => void loadItems(selectedList.id)}>
                <RefreshCw className={loadingItems ? "size-4 animate-spin" : "size-4"} />
                {t("favorites.actions.refresh")}
              </Button>
              <Button type="button" size="sm" variant="secondary" onClick={() => setCrossList(selectedList)}>
                <FolderSync className="size-4" />
                {t("favorites.actions.copyAcrossVersions")}
              </Button>
            </div>
          </div>
        ) : null}

        {error ? <div className="mt-2 rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{error}</div> : null}
        {loadingItems ? <Progress value={undefined} className="mt-3 h-0.5 animate-pulse" /> : null}

        {!scopeReady ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 text-muted-foreground">
            <ListMusic className="size-10" />
            <span className="text-sm">{t("favorites.empty.selectScope")}</span>
          </div>
        ) : !selectedList ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 text-muted-foreground">
            <ListPlus className="size-10" />
            <span className="text-sm">{t("favorites.empty.selectOrCreate")}</span>
          </div>
        ) : items.length === 0 && !loadingItems ? (
          <div className="flex flex-1 flex-col items-center justify-center gap-2 text-muted-foreground">
            <Star className="size-10" />
            <span className="text-sm">{t("favorites.empty.noItems")}</span>
          </div>
        ) : (
          <div className="mt-3 min-h-0 flex-1 overflow-y-auto overflow-x-hidden pr-1">
            <div className={cn("grid gap-2", selectedItems.length > 0 && "pb-24")}>
              {items.map((item) => {
                const key = favoriteModKey(item);
                const selected = selectedKeys.has(key);
                return (
                  <div
                    key={item.id || key}
                    className={cn(
                      "md-row-interactive flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl border bg-card p-3 shadow-sm",
                      selected
                        ? "border-primary/50 bg-primary/10 shadow-sm hover:bg-primary/15"
                        : "hover:border-foreground/20 hover:bg-accent/60 hover:shadow-md",
                      item.referenced && "border-l-4 border-l-cyan-500",
                    )}
                    onClick={(event) => {
                      if (!item.referenced) toggleItem(item, event);
                    }}
                  >
                    <div className={cn("flex size-5 shrink-0 items-center justify-center rounded border transition-colors duration-md", selected ? "border-primary bg-primary text-primary-foreground" : "border-input")}>
                      {item.referenced ? (
                        <Link2 className="size-3 text-cyan-600 dark:text-cyan-400" />
                      ) : selected ? (
                        <Eye className="size-3.5 animate-in fade-in-0 zoom-in-50 duration-150" />
                      ) : null}
                    </div>
                    <button
                      type="button"
                      className="size-12 shrink-0 overflow-hidden rounded-lg border bg-muted/40 transition-[scale] duration-fast active:scale-95"
                      onClick={(event) => {
                        event.stopPropagation();
                        setPinItem(item);
                      }}
                      aria-label={t("favorites.actions.pin")}
                    >
                      {item.iconUrl ? (
                        <img src={item.iconUrl} alt="" className="size-full object-cover" />
                      ) : (
                        <span className="flex size-full items-center justify-center text-lg font-bold text-muted-foreground">
                          {displayName(item).slice(0, 1).toUpperCase()}
                        </span>
                      )}
                    </button>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-sm font-semibold">{displayName(item)}</span>
                        {item.referenced ? <Badge variant="secondary">{item.sourceListName}</Badge> : null}
                      </div>
                      <div className="truncate text-xs text-muted-foreground">
                        {item.platform} / {item.modId}
                      </div>
                    </div>
                    {!item.referenced ? (
                      <Button
                        type="button"
                        size="icon-sm"
                        variant="secondary"
                        className="text-destructive"
                        aria-label={t("favorites.actions.removeSelected")}
                        onClick={(event) => {
                          event.stopPropagation();
                          void removeMod(item)
                            .then((removed) => {
                              if (removed) toast.success(t("favorites.removed"));
                            })
                            .catch((error: unknown) => {
                              toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
                            });
                        }}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    ) : null}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        <div className="pointer-events-none absolute inset-x-0 bottom-4 z-30 flex justify-center px-2">
          <AnimatedPresence
            open={selectedItems.length > 0}
            mode="float"
            preserveContent
            className="pointer-events-auto max-w-full"
          >
            <div className="md-selection-toolbar flex max-w-full flex-wrap items-center justify-center gap-1.5 rounded-2xl border bg-popover/95 p-2 shadow-2xl backdrop-blur-md">
              <span
                aria-label={t("manage.selectedCount", { n: selectedItems.length })}
                className="flex size-9 items-center justify-center rounded-full bg-sky-500/15 text-sm font-semibold text-sky-600 dark:text-sky-300"
              >
                {selectedItems.length}
              </span>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="rounded-full bg-muted text-muted-foreground hover:bg-muted/70 hover:text-foreground"
                    aria-label={t("favorites.actions.copySelected")}
                    onClick={() => openCopy("selected")}
                  >
                    <Copy className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("favorites.actions.copySelected")}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="rounded-full bg-destructive/15 text-destructive hover:bg-destructive/25"
                    aria-label={t("favorites.actions.removeSelected")}
                    onClick={() => void removeSelected()}
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("favorites.actions.removeSelected")}</TooltipContent>
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon"
                    variant="ghost"
                    className="rounded-full border border-input bg-transparent text-muted-foreground hover:bg-accent hover:text-foreground"
                    aria-label={t("download.selection.deselectAll")}
                    onClick={() => setSelectedKeys(new Set())}
                  >
                    <X className="size-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("download.selection.deselectAll")}</TooltipContent>
              </Tooltip>
            </div>
          </AnimatedPresence>
        </div>
      </section>

      <Dialog open={listEdit.open} onOpenChange={(open) => setListEdit((current) => ({ ...current, open }))}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t(listEdit.id ? "favorites.dialog.renameTitle" : "favorites.dialog.createTitle")}</DialogTitle>
          </DialogHeader>
          <Input
            value={listEdit.name}
            autoFocus
            placeholder={t("favorites.dialog.name")}
            onChange={(event) => setListEdit((current) => ({ ...current, name: event.target.value }))}
            onKeyDown={(event) => {
              if (event.key === "Enter") void saveListName();
            }}
          />
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setListEdit({ open: false, id: "", name: "" })}>
              {t("favorites.actions.cancel")}
            </Button>
            <Button type="button" disabled={!listEdit.name.trim()} onClick={() => void saveListName()}>
              {t("favorites.actions.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={metadata.open} onOpenChange={(open) => setMetadata((current) => ({ ...current, open }))}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("favorites.dialog.customizeTitle")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <label className="flex items-center justify-between gap-3 text-sm">
              <span>{t("favorites.actions.pin")}</span>
              <Switch checked={metadata.pinned} onCheckedChange={(value) => setMetadata((current) => ({ ...current, pinned: value }))} />
            </label>
            <label className="grid gap-1 text-sm">
              <span>{t("favorites.dialog.iconMode")}</span>
              <Combobox
                value={metadata.iconKind}
                onValueChange={(value) => setMetadata((current) => ({ ...current, iconKind: value }))}
                options={[
                  { value: "mdi", label: "MDI" },
                  { value: "project", label: t("favorites.dialog.projectIcon") },
                ]}
                aria-label={t("favorites.dialog.iconMode")}
                className="w-full"
              />
            </label>
            <Input
              value={metadata.iconValue}
              placeholder={metadata.iconKind === "mdi" ? t("favorites.dialog.mdiIcon") : t("favorites.dialog.projectSlug")}
              onChange={(event) => setMetadata((current) => ({ ...current, iconValue: event.target.value }))}
            />
            {metadata.iconKind === "project" ? (
              <Combobox
                value={metadata.iconPlatform}
                onValueChange={(value) => setMetadata((current) => ({ ...current, iconPlatform: value }))}
                options={[
                  { value: "modrinth", label: "Modrinth" },
                  { value: "curseforge", label: "CurseForge" },
                ]}
                aria-label={t("favorites.dialog.platform")}
                className="w-full"
              />
            ) : null}
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => void handleClearIcon()}>
              {t("favorites.actions.clearIcon")}
            </Button>
            <Button type="button" variant="ghost" onClick={() => setMetadata((current) => ({ ...current, open: false }))}>
              {t("favorites.actions.cancel")}
            </Button>
            <Button type="button" onClick={() => void saveMetadata()}>
              {t("favorites.actions.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={copyDialog.open} onOpenChange={(open) => setCopyDialog((current) => ({ ...current, open }))}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>
              {t(
                copyDialog.mode === "reference"
                  ? "favorites.dialog.referenceTitle"
                  : copyDialog.mode === "list"
                    ? "favorites.dialog.copyListTitle"
                    : "favorites.dialog.copySelectedTitle",
              )}
            </DialogTitle>
          </DialogHeader>
          <div className="grid gap-2">
            {copyDialog.mode === "selected" ? (
              <div className="max-h-60 space-y-1 overflow-y-auto rounded-md border p-2">
                {targetListOptions.map((option) => (
                  <label key={option.value} className="flex items-center gap-2 text-sm">
                    <input
                      type="checkbox"
                      className="size-4 accent-primary"
                      checked={copyDialog.targetListIds.includes(option.value)}
                      onChange={(event) =>
                        setCopyDialog((current) => ({
                          ...current,
                          targetListIds: event.target.checked
                            ? [...current.targetListIds, option.value]
                            : current.targetListIds.filter((id) => id !== option.value),
                        }))
                      }
                    />
                    {option.label}
                  </label>
                ))}
              </div>
            ) : (
              <Combobox
                value={copyDialog.targetListId}
                onValueChange={(value) => setCopyDialog((current) => ({ ...current, targetListId: value }))}
                options={targetListOptions}
                placeholder={t("favorites.dialog.targetList")}
                searchable
                searchPlaceholder={t("common.search")}
                emptyText={t("common.noResults")}
                aria-label={t("favorites.dialog.targetList")}
                className="w-full"
              />
            )}
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setCopyDialog((current) => ({ ...current, open: false }))}>
              {t("favorites.actions.cancel")}
            </Button>
            <Button type="button" disabled={!copyReady} onClick={() => void applyCopy()}>
              {t("favorites.actions.apply")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={Boolean(deleteTarget)} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("favorites.dialog.deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("favorites.dialog.deleteBody", { name: deleteTarget?.name || "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("favorites.actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive text-white hover:bg-destructive/90" onClick={() => void handleDelete()}>
              {t("favorites.actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <CrossVersionCopyDialog
        open={Boolean(crossList)}
        list={crossList}
        onOpenChange={(open) => !open && setCrossList(null)}
        onMessage={(message, color) => (color === "error" ? toast.error(message) : toast.success(message))}
      />

      <FavoritePinDialog open={Boolean(pinItem)} item={pinItem} onOpenChange={(open) => !open && setPinItem(null)} />
    </div>
  );
}
