import { useEffect, useMemo, useState } from "react";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

import { errorMessage } from "@/lib/format";
import { api } from "@/services/app";

import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  normalizeKeyPart,
  normalizeMinecraftVersion,
  useFavoritesStore,
  type FavoriteModDraft,
} from "@/stores/favorites";
import type { FavoriteList } from "@/types";

interface AddToFavoriteDialogProps {
  open: boolean;
  drafts: FavoriteModDraft[];
  onOpenChange: (open: boolean) => void;
  onAdded?: () => void;
}

export function AddToFavoriteDialog({ open, drafts, onOpenChange, onAdded }: AddToFavoriteDialogProps) {
  const { t } = useTranslation();
  const selectedListId = useFavoritesStore((state) => state.selectedListId);
  const createList = useFavoritesStore((state) => state.createList);
  const addDrafts = useFavoritesStore((state) => state.addDrafts);
  const [selectedListName, setSelectedListName] = useState("");
  const [saving, setSaving] = useState(false);
  const [scopeError, setScopeError] = useState(false);
  const [loadingScope, setLoadingScope] = useState(false);
  const [allLists, setAllLists] = useState<FavoriteList[]>([]);
  const [scopedLists, setScopedLists] = useState<FavoriteList[]>([]);

  const draftMinecraftVersion = normalizeMinecraftVersion(drafts[0]?.minecraftVersion);
  const draftModLoader = normalizeKeyPart(drafts[0]?.modLoader);
  const inScope = useMemo(() => scopedLists, [scopedLists]);
  const scopeLabel =
    draftMinecraftVersion && draftModLoader
      ? `${draftMinecraftVersion} / ${draftModLoader.toUpperCase()}`
      : "";
  const canCreate = drafts.every((draft) => Boolean(draft.minecraftVersion?.trim() && draft.modLoader?.trim()));

  // The dialog can target a different scope than the page, so ask core for the
  // draft scope's overview instead of filtering the page's lists locally.
  useEffect(() => {
    if (!open) return;
    setScopeError(false);
    let cancelled = false;
    setLoadingScope(true);
    void Promise.all([
      api.ListFavoriteLists(),
      api.ListFavoriteOverview(draftMinecraftVersion, draftModLoader),
    ])
      .then(([lists, overview]) => {
        if (cancelled) return;
        const scoped = [...(overview?.pinned ?? []), ...(overview?.ungrouped ?? [])];
        setAllLists(lists ?? []);
        setScopedLists(scoped);
        const current = useFavoritesStore.getState();
        setSelectedListName(
          scoped.find((list) => list.id === current.selectedListId)?.name || scoped[0]?.name || "",
        );
      })
      .catch((error) => {
        if (!cancelled) toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
      })
      .finally(() => {
        if (!cancelled) setLoadingScope(false);
      });
    return () => {
      cancelled = true;
    };
  }, [draftMinecraftVersion, draftModLoader, open, selectedListId, t]);

  const submit = async () => {
    const name = selectedListName.trim();
    if (!name || saving || drafts.length === 0 || !canCreate) return;
    setSaving(true);
    setScopeError(false);
    try {
      const existing = allLists.find(
        (list) => list.name.trim().toLocaleLowerCase() === name.toLocaleLowerCase(),
      );
      let listId = existing?.id ?? "";
      if (existing) {
        if (!scopedLists.some((list) => list.id === existing.id)) {
          setScopeError(true);
          return;
        }
      } else {
        const draft = drafts[0];
        listId = (await createList(name, draft?.minecraftVersion, draft?.modLoader))?.id ?? "";
      }
      if (!listId) {
        toast.error(t("favorites.operationFailed"));
        return;
      }
      await addDrafts(listId, drafts);
      onAdded?.();
      onOpenChange(false);
    } catch (error) {
      toast.error(t("favorites.operationFailed") + ": " + errorMessage(error));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(value) => !saving && onOpenChange(value)}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t("favorites.addDialog.title")}</DialogTitle>
          <DialogDescription>{t("favorites.addDialog.count", { n: drafts.length })}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-3">
          <div className="grid gap-1.5 text-sm">
            <span className="font-medium">{t("favorites.addDialog.list")}</span>
            <Combobox
              value={selectedListName}
              onValueChange={(value) => {
                setSelectedListName(value);
                setScopeError(false);
              }}
              options={inScope.map((list) => ({ value: list.name, label: list.name }))}
              placeholder={t("favorites.addDialog.newList")}
              disabled={saving || loadingScope}
              allowCustom
              autoFocus
              customLabel={(value) => t("common.useCustom", { value })}
              searchPlaceholder={t("common.search")}
              emptyText={t("common.noResults")}
              aria-label={t("favorites.addDialog.list")}
            />
          </div>
          {scopeLabel ? (
            <div className="text-xs text-muted-foreground">
              {t("favorites.addDialog.scopeLabel", { scope: scopeLabel })}
            </div>
          ) : null}
          {scopeError ? (
            <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-xs text-amber-700 dark:text-amber-300">
              {t("favorites.addDialog.scopeConflict")}
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="ghost" disabled={saving} onClick={() => onOpenChange(false)}>
            {t("favorites.actions.cancel")}
          </Button>
          <Button
            type="button"
            disabled={saving || loadingScope || !canCreate || !selectedListName.trim()}
            onClick={() => void submit()}
          >
            {saving ? <Loader2 className="size-4 animate-spin" /> : null}
            {t("favorites.actions.add")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
