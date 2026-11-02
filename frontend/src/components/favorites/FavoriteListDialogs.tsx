import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { CrossVersionCopyDialog } from "@/components/CrossVersionCopyDialog";
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
import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Switch } from "@/components/ui/switch";
import type { CopyMode } from "@/components/favorites/shared";
import type { FavoriteList, FavoriteModEntry } from "@/types";

export interface FavoriteListEditState {
  open: boolean;
  id: string;
  name: string;
}

export interface FavoriteMetadataDraft {
  open: boolean;
  list: FavoriteList | null;
  pinned: boolean;
  iconKind: string;
  iconValue: string;
  iconPlatform: string;
}

export interface FavoriteCopyDialogState {
  open: boolean;
  mode: CopyMode;
  sourceListId: string;
  targetListId: string;
  targetListIds: string[];
  mods: FavoriteModEntry[];
}

export interface FavoriteListDialogsProps {
  listEdit: FavoriteListEditState;
  onListEditChange: (next: FavoriteListEditState) => void;
  onSaveListName: () => void;
  metadata: FavoriteMetadataDraft;
  onMetadataChange: (updater: (current: FavoriteMetadataDraft) => FavoriteMetadataDraft) => void;
  onSaveMetadata: () => void;
  onClearIcon: () => void;
  copyDialog: FavoriteCopyDialogState;
  onCopyDialogChange: (updater: (current: FavoriteCopyDialogState) => FavoriteCopyDialogState) => void;
  targetListOptions: { value: string; label: string }[];
  copyReady: boolean;
  onApplyCopy: () => void;
  deleteTarget: FavoriteList | null;
  onDeleteTargetChange: (list: FavoriteList | null) => void;
  onConfirmDelete: () => void;
  crossList: FavoriteList | null;
  onCrossListChange: (list: FavoriteList | null) => void;
  pinItem: FavoriteModEntry | null;
  onPinItemChange: (item: FavoriteModEntry | null) => void;
}

export function FavoriteListDialogs({
  listEdit,
  onListEditChange,
  onSaveListName,
  metadata,
  onMetadataChange,
  onSaveMetadata,
  onClearIcon,
  copyDialog,
  onCopyDialogChange,
  targetListOptions,
  copyReady,
  onApplyCopy,
  deleteTarget,
  onDeleteTargetChange,
  onConfirmDelete,
  crossList,
  onCrossListChange,
  pinItem,
  onPinItemChange,
}: FavoriteListDialogsProps) {
  const { t } = useTranslation();

  return (
    <>
      <Dialog open={listEdit.open} onOpenChange={(open) => onListEditChange({ ...listEdit, open })}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>{t(listEdit.id ? "favorites.dialog.renameTitle" : "favorites.dialog.createTitle")}</DialogTitle>
          </DialogHeader>
          <Input
            value={listEdit.name}
            autoFocus
            placeholder={t("favorites.dialog.name")}
            onChange={(event) => onListEditChange({ ...listEdit, name: event.target.value })}
            onKeyDown={(event) => {
              if (event.key === "Enter") onSaveListName();
            }}
          />
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => onListEditChange({ open: false, id: "", name: "" })}>
              {t("favorites.actions.cancel")}
            </Button>
            <Button type="button" disabled={!listEdit.name.trim()} onClick={onSaveListName}>
              {t("favorites.actions.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={metadata.open} onOpenChange={(open) => onMetadataChange((current) => ({ ...current, open }))}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>{t("favorites.dialog.customizeTitle")}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-3">
            <label className="flex items-center justify-between gap-3 text-sm">
              <span>{t("favorites.actions.pin")}</span>
              <Switch checked={metadata.pinned} onCheckedChange={(value) => onMetadataChange((current) => ({ ...current, pinned: value }))} />
            </label>
            <label className="grid gap-1 text-sm">
              <span>{t("favorites.dialog.iconMode")}</span>
              <Combobox
                value={metadata.iconKind}
                onValueChange={(value) => onMetadataChange((current) => ({ ...current, iconKind: value }))}
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
              onChange={(event) => onMetadataChange((current) => ({ ...current, iconValue: event.target.value }))}
            />
            {metadata.iconKind === "project" ? (
              <Combobox
                value={metadata.iconPlatform}
                onValueChange={(value) => onMetadataChange((current) => ({ ...current, iconPlatform: value }))}
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
            <Button type="button" variant="ghost" onClick={onClearIcon}>
              {t("favorites.actions.clearIcon")}
            </Button>
            <Button type="button" variant="ghost" onClick={() => onMetadataChange((current) => ({ ...current, open: false }))}>
              {t("favorites.actions.cancel")}
            </Button>
            <Button type="button" onClick={onSaveMetadata}>
              {t("favorites.actions.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={copyDialog.open} onOpenChange={(open) => onCopyDialogChange((current) => ({ ...current, open }))}>
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
                        onCopyDialogChange((current) => ({
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
                onValueChange={(value) => onCopyDialogChange((current) => ({ ...current, targetListId: value }))}
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
            <Button type="button" variant="ghost" onClick={() => onCopyDialogChange((current) => ({ ...current, open: false }))}>
              {t("favorites.actions.cancel")}
            </Button>
            <Button type="button" disabled={!copyReady} onClick={onApplyCopy}>
              {t("favorites.actions.apply")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={Boolean(deleteTarget)} onOpenChange={(open) => !open && onDeleteTargetChange(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("favorites.dialog.deleteTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("favorites.dialog.deleteBody", { name: deleteTarget?.name || "" })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("favorites.actions.cancel")}</AlertDialogCancel>
            <AlertDialogAction className="bg-destructive text-white hover:bg-destructive/90" onClick={onConfirmDelete}>
              {t("favorites.actions.delete")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <CrossVersionCopyDialog
        open={Boolean(crossList)}
        list={crossList}
        onOpenChange={(open) => !open && onCrossListChange(null)}
        onMessage={(message, color) => (color === "error" ? toast.error(message) : toast.success(message))}
      />

      <FavoritePinDialog open={Boolean(pinItem)} item={pinItem} onOpenChange={(open) => !open && onPinItemChange(null)} />
    </>
  );
}
