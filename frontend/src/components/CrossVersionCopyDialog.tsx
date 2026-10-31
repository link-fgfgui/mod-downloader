import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, Check, Eye, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { MinecraftTargetFields } from "@/components/MinecraftTargetFields";
import { modLoaderOptions, useMinecraftStore } from "@/stores/minecraft";
import { useFavoritesStore } from "@/stores/favorites";
import { errorMessage } from "@/lib/format";
import type { FavoriteCrossVersionCopyPreview, FavoriteList } from "@/types";

interface CrossVersionCopyDialogProps {
  open: boolean;
  list: FavoriteList | null;
  onOpenChange: (open: boolean) => void;
  onMessage: (message: string, color?: "success" | "error") => void;
}

const displayName = (item: { title?: string; slug?: string; modId: string }) =>
  item.title || item.slug || item.modId;

export function CrossVersionCopyDialog({ open, list, onOpenChange, onMessage }: CrossVersionCopyDialogProps) {
  const { t } = useTranslation();
  const releaseVersions = useMinecraftStore((state) => state.releases);
  const selectedVersion = useMinecraftStore((state) => state.minecraftVersion);
  const selectedLoader = useMinecraftStore((state) => state.modLoader);
  const previewCopy = useFavoritesStore((state) => state.previewCrossVersionCopy);
  const applyCopy = useFavoritesStore((state) => state.applyCrossVersionCopy);
  const [minecraftVersion, setMinecraftVersion] = useState("");
  const [modLoader, setModLoader] = useState("");
  const [ignoreConflicts, setIgnoreConflicts] = useState(false);
  const [preview, setPreview] = useState<FavoriteCrossVersionCopyPreview | null>(null);
  const [previewing, setPreviewing] = useState(false);
  const [applying, setApplying] = useState(false);

  useEffect(() => {
    if (!open || !list) return;
    setMinecraftVersion(selectedVersion);
    setModLoader(selectedLoader || "Fabric");
    setIgnoreConflicts(false);
    setPreview(null);
    setPreviewing(false);
    setApplying(false);
  }, [list, open, selectedLoader, selectedVersion]);

  useEffect(() => {
    setPreview(null);
  }, [minecraftVersion, modLoader, ignoreConflicts]);

  const copyReady = useMemo(() => {
    if (!list) return false;
    const targetVersion = minecraftVersion.trim();
    const targetLoader = modLoader.trim().toLowerCase();
    const sourceVersion = (list.minecraftVersion || "").trim();
    const sourceLoader = (list.modLoader || "").trim().toLowerCase();
    return Boolean(
      list.id &&
        targetVersion &&
        targetLoader &&
        (targetVersion !== sourceVersion || targetLoader !== sourceLoader),
    );
  }, [list, minecraftVersion, modLoader]);

  const previewIsCurrent = useMemo(() => {
    if (!preview || !list) return false;
    return (
      preview.sourceListId === list.id &&
      preview.minecraftVersion === minecraftVersion.trim() &&
      preview.modLoader === modLoader.trim().toLowerCase()
    );
  }, [list, minecraftVersion, modLoader, preview]);

  const applyReady =
    Boolean(list) &&
    copyReady &&
    previewIsCurrent &&
    !previewing &&
    !applying &&
    Boolean(preview) &&
    !preview?.nameConflict &&
    (preview?.errors || []).length === 0 &&
    (preview?.matched || []).length > 0 &&
    (ignoreConflicts || (preview?.conflicts || []).length === 0);

  const request = () => ({
    sourceListId: list?.id || "",
    minecraftVersion: minecraftVersion.trim(),
    modLoader: modLoader.trim().toLowerCase(),
    ignoreConflicts,
  });

  const handlePreview = async () => {
    if (!list || !copyReady || previewing) return;
    setPreviewing(true);
    try {
      setPreview(await previewCopy(request()));
    } catch (error) {
      onMessage(
        t("favorites.crossVersionCopy.notApplied", { reason: errorMessage(error) }),
        "error",
      );
    } finally {
      setPreviewing(false);
    }
  };

  const handleApply = async () => {
    if (!list || !applyReady) return;
    setApplying(true);
    try {
      const result = await applyCopy(request());
      if (result?.applied && result.targetList?.id) {
        onOpenChange(false);
        onMessage(
          t("favorites.crossVersionCopy.success", {
            name: result.targetList.name,
            added: result.result?.added || 0,
            skipped: result.result?.skipped || 0,
          }),
          "success",
        );
        return;
      }
      const reason =
        result?.result?.errors?.[0] ||
        result?.preview?.errors?.[0] ||
        t("favorites.crossVersionCopy.unknownError");
      onMessage(t("favorites.crossVersionCopy.notApplied", { reason }), "error");
    } catch (error) {
      onMessage(
        t("favorites.crossVersionCopy.notApplied", { reason: errorMessage(error) }),
        "error",
      );
    } finally {
      setApplying(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(value) => !applying && onOpenChange(value)}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("favorites.dialog.crossVersionCopyTitle")}</DialogTitle>
        </DialogHeader>
        <div className="grid gap-3">
          <MinecraftTargetFields
            versions={releaseVersions}
            modLoaders={modLoaderOptions}
            minecraftVersion={minecraftVersion}
            modLoader={modLoader}
            onMinecraftVersionChange={setMinecraftVersion}
            onModLoaderChange={setModLoader}
            minecraftVersionLabel={t("favorites.dialog.minecraftVersion")}
            modLoaderLabel={t("favorites.dialog.modLoader")}
            disabled={applying}
          />
          <div className="rounded-md border bg-muted/30 p-2 text-xs text-muted-foreground">
            {t("favorites.crossVersionCopy.target", {
              name: list?.name || "",
              minecraftVersion,
              modLoader,
            })}
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={ignoreConflicts}
              disabled={applying}
              onChange={(event) => setIgnoreConflicts(event.target.checked)}
              className="size-4 accent-primary"
            />
            {t("favorites.dialog.ignoreConflicts")}
          </label>
          <div>
            <Button type="button" variant="secondary" size="sm" disabled={!copyReady || applying || previewing} onClick={() => void handlePreview()}>
              {previewing ? <Loader2 className="size-4 animate-spin" /> : <Eye className="size-4" />}
              {t("favorites.actions.preview")}
            </Button>
          </div>

          {preview ? (
            <div className="grid gap-2">
              {preview.nameConflict ? (
                <div className="rounded-md border border-destructive/40 bg-destructive/10 p-2 text-xs text-destructive">
                  {t("favorites.crossVersionCopy.nameConflict")}
                </div>
              ) : (preview.errors || []).length ? (
                <div className="rounded-md border border-destructive/40 bg-destructive/10 p-2 text-xs text-destructive">
                  {preview.errors?.map((message) => <div key={message}>{message}</div>)}
                </div>
              ) : null}
              <div className="flex gap-2">
                <Badge variant="secondary" className="text-emerald-700 dark:text-emerald-300">
                  {t("favorites.crossVersionCopy.matched", { n: preview.matched?.length || 0 })}
                </Badge>
                <Badge variant="outline" className="text-amber-700 dark:text-amber-300">
                  {t("favorites.crossVersionCopy.conflicts", { n: preview.conflicts?.length || 0 })}
                </Badge>
              </div>
              <div className="max-h-64 overflow-y-auto rounded-md border">
                {(preview.matched || []).map((match) => (
                  <div key={`m-${match.source.id}`} className="flex items-center gap-2 border-b px-2 py-1.5 text-sm last:border-b-0">
                    <Check className="size-4 text-emerald-500" />
                    <span className="min-w-0 flex-1 truncate">{displayName(match.source)}</span>
                    <span className="font-mono text-xs text-muted-foreground">{match.version.versionId || match.version.id}</span>
                  </div>
                ))}
                {(preview.conflicts || []).map((conflict) => (
                  <div key={`c-${conflict.source.id}`} className="flex items-center gap-2 border-b px-2 py-1.5 text-sm last:border-b-0">
                    <AlertTriangle className="size-4 text-amber-500" />
                    <span className="min-w-0 flex-1 truncate">{displayName(conflict.source)}</span>
                    <span className="max-w-60 truncate text-xs text-muted-foreground">{conflict.reason}</span>
                  </div>
                ))}
              </div>
            </div>
          ) : null}
        </div>
        <DialogFooter>
          <Button type="button" variant="ghost" disabled={applying} onClick={() => onOpenChange(false)}>
            {t("favorites.actions.cancel")}
          </Button>
          <Button type="button" disabled={!applyReady} onClick={() => void handleApply()}>
            {applying ? <Loader2 className="size-4 animate-spin" /> : null}
            {t("favorites.actions.apply")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
