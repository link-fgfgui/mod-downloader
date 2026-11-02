import { useEffect, useMemo, useRef, useState } from "react";
import { Archive, FolderSync, Loader2, ListMusic, ListPlus, RefreshCw, Star } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { FavoriteListDialogs } from "@/components/favorites/FavoriteListDialogs";
import { FavoriteListSidebar } from "@/components/favorites/FavoriteListSidebar";
import { FavoriteModTable } from "@/components/favorites/FavoriteModTable";
import { FavoriteSelectionToolbar } from "@/components/favorites/FavoriteSelectionToolbar";
import { displayName, type CopyMode } from "@/components/favorites/shared";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { useDownloadSearchStore } from "@/stores/downloadSearch";
import {
  hasFavoriteScope,
  favoriteModKey,
  useFavoritesStore,
} from "@/stores/favorites";
import { useMinecraftStore } from "@/stores/minecraft";
import { useSettingsStore } from "@/stores/settings";
import { errorMessage } from "@/lib/format";
import { scaledDuration } from "@/lib/animations";
import { useFlipListAnimation } from "@/hooks/useFlipListAnimation";
import type { FavoriteList, FavoriteModEntry, ModProject } from "@/types";
import type {
  FavoriteCopyDialogState,
  FavoriteListEditState,
  FavoriteMetadataDraft,
} from "@/components/favorites/FavoriteListDialogs";

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

  const [listEdit, setListEdit] = useState<FavoriteListEditState>({ open: false, id: "", name: "" });
  const [deleteTarget, setDeleteTarget] = useState<FavoriteList | null>(null);
  const [metadata, setMetadata] = useState<FavoriteMetadataDraft>({ open: false, list: null, pinned: false, iconKind: "mdi", iconValue: "", iconPlatform: "modrinth" });
  const [copyDialog, setCopyDialog] = useState<FavoriteCopyDialogState>({ open: false, mode: "selected", sourceListId: "", targetListId: "", targetListIds: [], mods: [] });
  const [crossList, setCrossList] = useState<FavoriteList | null>(null);
  const [pinItem, setPinItem] = useState<FavoriteModEntry | null>(null);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(new Set());
  const [draggedListId, setDraggedListId] = useState("");
  const [dragOver, setDragOver] = useState<{ id: string; position: "before" | "after" } | null>(null);
  const [exitingKeys, setExitingKeys] = useState<Set<string>>(new Set());
  const [listRemoving, setListRemoving] = useState(false);
  const listRef = useRef<HTMLDivElement | null>(null);

  useFlipListAnimation(listRef);

  const wait = (ms: number) =>
    ms > 0 ? new Promise<void>((resolve) => window.setTimeout(resolve, ms)) : Promise.resolve();

  useEffect(() => {
    setDisplayScope(minecraftVersion, modLoader);
    void loadLists();
  }, [loadLists, minecraftVersion, modLoader, setDisplayScope]);

  useEffect(() => {
    setSelectedKeys(new Set());
  }, [selectedListId]);

  // Core owns scope filtering and section ordering.
  const overview = useFavoritesStore((state) => state.overview);
  const pinnedLists = overview.pinned ?? [];
  const ungroupedLists = overview.ungrouped ?? [];
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
    // Batch removal only fades the container; animating dozens of rows would
    // cost more than it communicates.
    setListRemoving(true);
    await wait(scaledDuration(0.15) * 1000);
    try {
      await removeMany(removable);
      toast.success(t("favorites.removed"));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    }
    setSelectedKeys(new Set());
    setListRemoving(false);
  };

  // Single-row removal plays the row exit first, then updates the store.
  const handleRemoveItem = async (item: FavoriteModEntry) => {
    const key = favoriteModKey(item);
    setExitingKeys((current) => new Set(current).add(key));
    await wait(scaledDuration(0.15) * 1000);
    try {
      const removed = await removeMod(item);
      if (removed) toast.success(t("favorites.removed"));
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    } finally {
      setExitingKeys((current) => {
        if (!current.has(key)) return current;
        const next = new Set(current);
        next.delete(key);
        return next;
      });
    }
  };

  const onDragStart = (event: React.DragEvent, list: FavoriteList) => {
    setDraggedListId(list.id);
    setDragOver(null);
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", list.id);
  };

  const onDragOver = (event: React.DragEvent, listId: string) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
    if (!draggedListId || draggedListId === listId) return;
    const rect = event.currentTarget.getBoundingClientRect();
    const position = event.clientY < rect.top + rect.height / 2 ? "before" : "after";
    setDragOver((current) =>
      current?.id === listId && current.position === position ? current : { id: listId, position },
    );
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
    const position = dragOver?.id === targetListId ? dragOver.position : "before";
    setDragOver(null);
    const sourceId = draggedListId;
    setDraggedListId("");
    if (!sourceId || sourceId === targetListId) return;
    const moved = await moveList(sourceId, pinned);
    if (!moved) return;
    const ids = sectionLists.map((list) => list.id).filter((id) => id !== sourceId);
    let targetIndex = ids.indexOf(targetListId);
    if (targetIndex < 0) return;
    if (position === "after") targetIndex += 1;
    ids.splice(targetIndex, 0, sourceId);
    await reorderLists(ids);
  };

  const onSectionDrop = async (event: React.DragEvent, pinned: boolean) => {
    event.preventDefault();
    const sourceId = draggedListId;
    setDraggedListId("");
    setDragOver(null);
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

  return (
    <div className="mx-auto grid h-full w-full max-w-6xl grid-cols-1 gap-5 overflow-hidden p-6 lg:grid-cols-[minmax(15rem,19rem)_minmax(0,1fr)]">
      <FavoriteListSidebar
        lists={lists}
        pinnedLists={pinnedLists}
        ungroupedLists={ungroupedLists}
        selectedListId={selectedListId}
        loadingLists={loadingLists}
        canCreate={canCreate}
        listRef={listRef}
        draggedListId={draggedListId}
        dragOver={dragOver}
        listIconUrl={listIconUrl}
        onCreate={() => openListEdit()}
        onSelect={(id) => void selectList(id)}
        onOpenEdit={openListEdit}
        onPin={(list) => void handlePinList(list)}
        onOpenMetadata={openMetadata}
        onOpenCopy={openCopy}
        onOpenAsSearchResults={(list) => void openListAsSearchResults(list)}
        onSetCrossList={setCrossList}
        onDeleteTarget={setDeleteTarget}
        onDragStart={onDragStart}
        onDragEnd={() => {
          setDraggedListId("");
          setDragOver(null);
        }}
        onDragOverRow={onDragOver}
        onDropRow={(event, targetListId, sectionLists, pinned) => void onDrop(event, targetListId, sectionLists, pinned)}
        onSectionDrop={(event, pinned) => void onSectionDrop(event, pinned)}
      />

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
          <FavoriteModTable
            items={items}
            selectedKeys={selectedKeys}
            exitingKeys={exitingKeys}
            listRemoving={listRemoving}
            selectedListId={selectedListId}
            selectedCount={selectedItems.length}
            onToggleItem={toggleItem}
            onPinItem={setPinItem}
            onRemoveItem={(item) => void handleRemoveItem(item)}
          />
        )}

        <FavoriteSelectionToolbar
          selectedCount={selectedItems.length}
          onCopySelected={() => openCopy("selected")}
          onRemoveSelected={() => void removeSelected()}
          onDeselectAll={() => setSelectedKeys(new Set())}
        />
      </section>

      <FavoriteListDialogs
        listEdit={listEdit}
        onListEditChange={setListEdit}
        onSaveListName={() => void saveListName()}
        metadata={metadata}
        onMetadataChange={setMetadata}
        onSaveMetadata={() => void saveMetadata()}
        onClearIcon={() => void handleClearIcon()}
        copyDialog={copyDialog}
        onCopyDialogChange={setCopyDialog}
        targetListOptions={targetListOptions}
        copyReady={copyReady}
        onApplyCopy={() => void applyCopy()}
        deleteTarget={deleteTarget}
        onDeleteTargetChange={setDeleteTarget}
        onConfirmDelete={() => void handleDelete()}
        crossList={crossList}
        onCrossListChange={setCrossList}
        pinItem={pinItem}
        onPinItemChange={setPinItem}
      />
    </div>
  );
}
