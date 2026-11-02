import type { DragEvent, RefObject } from "react";
import {
  Copy,
  Eye,
  FolderSync,
  GripVertical,
  Link2,
  ListMusic,
  ListPlus,
  MoreVertical,
  Palette,
  Pencil,
  Pin,
  PinOff,
  Star,
  Trash2,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Progress } from "@/components/ui/progress";
import { cn } from "@/lib/utils";
import type { FavoriteList } from "@/types";
import type { CopyMode } from "@/components/favorites/shared";

export interface FavoriteListSidebarProps {
  lists: FavoriteList[];
  pinnedLists: FavoriteList[];
  ungroupedLists: FavoriteList[];
  selectedListId: string;
  loadingLists: boolean;
  canCreate: boolean;
  listRef: RefObject<HTMLDivElement>;
  draggedListId: string;
  dragOver: { id: string; position: "before" | "after" } | null;
  listIconUrl: (list: FavoriteList | null) => string;
  onCreate: () => void;
  onSelect: (id: string) => void;
  onOpenEdit: (list?: FavoriteList) => void;
  onPin: (list: FavoriteList) => void;
  onOpenMetadata: (list: FavoriteList) => void;
  onOpenCopy: (mode: CopyMode, list?: FavoriteList) => void;
  onOpenAsSearchResults: (list: FavoriteList) => void;
  onSetCrossList: (list: FavoriteList) => void;
  onDeleteTarget: (list: FavoriteList) => void;
  onDragStart: (event: DragEvent, list: FavoriteList) => void;
  onDragEnd: () => void;
  onDragOverRow: (event: DragEvent, listId: string) => void;
  onDropRow: (event: DragEvent, targetListId: string, sectionLists: FavoriteList[], pinned: boolean) => void;
  onSectionDrop: (event: DragEvent, pinned: boolean) => void;
}

export function FavoriteListSidebar({
  lists,
  pinnedLists,
  ungroupedLists,
  selectedListId,
  loadingLists,
  canCreate,
  listRef,
  draggedListId,
  dragOver,
  listIconUrl,
  onCreate,
  onSelect,
  onOpenEdit,
  onPin,
  onOpenMetadata,
  onOpenCopy,
  onOpenAsSearchResults,
  onSetCrossList,
  onDeleteTarget,
  onDragStart,
  onDragEnd,
  onDragOverRow,
  onDropRow,
  onSectionDrop,
}: FavoriteListSidebarProps) {
  const { t } = useTranslation();

  const renderListRow = (list: FavoriteList, sectionLists: FavoriteList[], pinned: boolean) => {
    const iconUrl = listIconUrl(list);
    const selected = list.id === selectedListId;
    return (
      <div
        key={list.id}
        data-list-id={list.id}
        data-flip-key={list.id}
        draggable
        onDragStart={(event) => onDragStart(event, list)}
        onDragEnd={onDragEnd}
        onDragOver={(event) => {
          event.stopPropagation();
          onDragOverRow(event, list.id);
        }}
        onDrop={(event) => {
          event.stopPropagation();
          onDropRow(event, list.id, sectionLists, pinned);
        }}
        className={cn(
          "group relative flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 text-sm transition-[background-color,color,opacity,transform] duration-md-fast",
          selected ? "bg-primary/10 text-foreground" : "text-muted-foreground hover:bg-accent/60 hover:text-foreground",
          draggedListId === list.id && "scale-[0.98] opacity-40",
        )}
        onClick={() => onSelect(list.id)}
      >
        {dragOver?.id === list.id ? (
          <span
            aria-hidden="true"
            data-drop-indicator={dragOver.position}
            className={cn(
              "md-drop-line pointer-events-none absolute inset-x-0",
              dragOver.position === "before" ? "-top-[5px]" : "-bottom-[5px]",
            )}
          />
        ) : null}
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
            <DropdownMenuItem onClick={() => onOpenEdit(list)}>
              <Pencil className="size-4" /> {t("favorites.actions.rename")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onPin(list)}>
              {list.pinned ? <PinOff className="size-4" /> : <Pin className="size-4" />}
              {t(list.pinned ? "favorites.actions.unpin" : "favorites.actions.pin")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onOpenMetadata(list)}>
              <Palette className="size-4" /> {t("favorites.actions.customize")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onOpenCopy("list", list)}>
              <Copy className="size-4" /> {t("favorites.actions.copyList")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onOpenAsSearchResults(list)}>
              <Eye className="size-4" /> {t("favorites.actions.asSearchResults")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onOpenCopy("reference", list)}>
              <Link2 className="size-4" /> {t("favorites.actions.referenceList")}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => onSetCrossList(list)}>
              <FolderSync className="size-4" /> {t("favorites.actions.copyAcrossVersions")}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem variant="destructive" onClick={() => onDeleteTarget(list)}>
              <Trash2 className="size-4" /> {t("favorites.actions.delete")}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    );
  };

  return (
    <aside className="flex min-h-0 flex-col border-b pb-3 lg:border-b-0 lg:border-r lg:pb-0 lg:pr-4">
      <div className="flex items-center justify-between gap-2">
        <h1 className="text-lg font-semibold">{t("favorites.title")}</h1>
        <div className="flex items-center gap-1">
          <Button type="button" size="icon-sm" variant="secondary" disabled={!canCreate} aria-label={t("favorites.dialog.createTitle")} onClick={onCreate}>
            <ListPlus className="size-4" />
          </Button>
        </div>
      </div>
      {loadingLists ? <Progress value={undefined} className="mt-2 h-0.5 animate-pulse" /> : null}
      <div className="mt-2 min-h-0 flex-1 overflow-y-auto pr-1">
        {lists.length === 0 ? (
          <div className="py-10 text-center text-sm text-muted-foreground">{t("favorites.empty.noLists")}</div>
        ) : (
          // Pinned and normal lists share one parent so React can move a row
          // between sections without remounting it, which lets FLIP animate
          // the pin/unpin flight.
          <div ref={listRef} className="relative grid gap-2">
            {pinnedLists.length ? (
              <>
                <div
                  className="flex items-center gap-1 px-2 py-1 text-[0.7rem] font-bold uppercase tracking-wide text-muted-foreground animate-in fade-in-0 slide-in-from-top-1 duration-md-fast"
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={(event) => onSectionDrop(event, true)}
                >
                  <Pin className="size-3" /> {t("favorites.sections.pinned")}
                </div>
                {pinnedLists.map((list) => renderListRow(list, pinnedLists, true))}
              </>
            ) : null}
            <div
              className="flex items-center gap-1 px-2 py-1 text-[0.7rem] font-bold uppercase tracking-wide text-muted-foreground"
              onDragOver={(event) => event.preventDefault()}
              onDrop={(event) => onSectionDrop(event, false)}
            >
              <Star className="size-3" /> {t("favorites.sections.lists")}
            </div>
            {ungroupedLists.map((list) => renderListRow(list, ungroupedLists, false))}
          </div>
        )}
      </div>
    </aside>
  );
}
