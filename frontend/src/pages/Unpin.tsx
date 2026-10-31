import { useEffect, useState } from "react";
import { Loader2, PinOff, PinOffIcon, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";

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
import { Progress } from "@/components/ui/progress";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { usePinnedModsStore } from "@/stores/pinnedMods";
import type { PinnedMod } from "@/types";

const pinKey = (pin: PinnedMod) =>
  [pin.platform, pin.modId, pin.minecraftVersion, pin.modLoader].join("|");

export function UnpinPage() {
  const { t } = useTranslation();
  const pins = usePinnedModsStore((state) => state.pins);
  const loading = usePinnedModsStore((state) => state.loading);
  const refresh = usePinnedModsStore((state) => state.refresh);
  const unpin = usePinnedModsStore((state) => state.unpin);
  const unpinAll = usePinnedModsStore((state) => state.unpinAll);
  const error = usePinnedModsStore((state) => state.error);
  const [pending, setPending] = useState<Set<string>>(new Set());
  const [confirmAll, setConfirmAll] = useState(false);
  const [bulkBusy, setBulkBusy] = useState(false);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const handleUnpin = async (pin: PinnedMod) => {
    const key = pinKey(pin);
    if (pending.has(key)) return;
    setPending((current) => new Set(current).add(key));
    try {
      const removed = await unpin(pin);
      if (removed) toast.success(t("unpin.removed"));
      else toast.warning(t("unpin.removeFailed"));
    } catch {
      await refresh();
      toast.error(t("unpin.removeFailed"));
    } finally {
      setPending((current) => {
        const next = new Set(current);
        next.delete(key);
        return next;
      });
    }
  };

  const handleUnpinAll = async () => {
    if (bulkBusy) return;
    setConfirmAll(false);
    setBulkBusy(true);
    try {
      const failed = await unpinAll();
      if (failed > 0) toast.warning(t("unpin.unpinAllPartial", { failed }));
      else toast.success(t("unpin.removedAll"));
    } catch (error) {
      toast.error(error instanceof Error ? error.message : String(error));
    } finally {
      setBulkBusy(false);
    }
  };

  return (
    <div className="mx-auto flex h-full w-full max-w-6xl flex-col gap-4 overflow-y-auto p-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h1 className="text-2xl font-bold tracking-tight">{t("unpin.title")}</h1>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="outline" disabled={loading || bulkBusy} onClick={() => void refresh()}>
            <RefreshCw className={loading ? "size-4 animate-spin" : "size-4"} />
            {t("unpin.refresh")}
          </Button>
          <Button type="button" variant="destructive" disabled={!pins.length || loading || bulkBusy} onClick={() => setConfirmAll(true)}>
            {bulkBusy ? <Loader2 className="size-4 animate-spin" /> : <PinOff className="size-4" />}
            {t("unpin.unpinAll")}
          </Button>
        </div>
      </div>

      {loading ? <Progress value={undefined} className="h-0.5 animate-pulse" /> : null}
      {error ? <div className="rounded-md border border-destructive/30 bg-destructive/5 p-2 text-xs text-destructive">{error}</div> : null}

      {!loading && pins.length === 0 ? (
        <div className="flex min-h-64 flex-col items-center justify-center gap-3 rounded-xl border border-dashed text-muted-foreground">
          <PinOffIcon className="size-10" />
          <span className="text-sm">{t("unpin.empty")}</span>
        </div>
      ) : pins.length === 0 ? null : (
        <div className="overflow-x-auto rounded-xl border">
          <Table className="min-w-[42rem]">
            <TableHeader>
              <TableRow>
                <TableHead>{t("unpin.columns.platform")}</TableHead>
                <TableHead>{t("unpin.columns.modId")}</TableHead>
                <TableHead>{t("unpin.columns.versionId")}</TableHead>
                <TableHead>{t("unpin.columns.minecraftVersion")}</TableHead>
                <TableHead>{t("unpin.columns.modLoader")}</TableHead>
                <TableHead className="text-right">{t("unpin.unpin")}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {pins.map((pin) => {
                const busy = pending.has(pinKey(pin));
                return (
                  <TableRow key={pin.id || pinKey(pin)} className="animate-in fade-in-0 duration-150">
                    <TableCell>{pin.platform}</TableCell>
                    <TableCell className="font-medium">{pin.modId}</TableCell>
                    <TableCell className="max-w-52 truncate font-mono text-xs">{pin.versionId}</TableCell>
                    <TableCell>{pin.minecraftVersion}</TableCell>
                    <TableCell>{pin.modLoader}</TableCell>
                    <TableCell className="text-right">
                      <Button
                        type="button"
                        size="sm"
                        variant="outline"
                        disabled={busy}
                        onClick={() => void handleUnpin(pin)}
                      >
                        {busy ? <Loader2 className="size-4 animate-spin" /> : <PinOff className="size-4" />}
                        {t("unpin.unpin")}
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </div>
      )}

      <AlertDialog open={confirmAll} onOpenChange={setConfirmAll}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("unpin.confirmAll.title")}</AlertDialogTitle>
            <AlertDialogDescription>{t("unpin.confirmAll.body")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("unpin.confirmAll.cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={() => void handleUnpinAll()}>
              {t("unpin.confirmAll.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
