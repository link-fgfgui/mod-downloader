import { AlertTriangle, Download, Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";

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
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type { LocalModDependencyImpact } from "@/types";

type RestoreCandidate = NonNullable<LocalModDependencyImpact["candidates"]>[number];

export interface ManageImpactDialogsProps {
  deleteOpen: boolean;
  onDeleteOpenChange: (open: boolean) => void;
  onCloseDelete: () => void;
  pendingDeleteCount: number;
  batchOperation: string;
  batchBusy: boolean;
  onConfirmDelete: () => void;
  impactOpen: boolean;
  onImpactOpenChange: (open: boolean) => void;
  onCloseImpact: () => void;
  impacts: LocalModDependencyImpact[];
  restoringDependencyId: string;
  restoredDependencies: Record<string, boolean>;
  onRestoreDependency: (impact: LocalModDependencyImpact, candidate: RestoreCandidate) => void;
  onConfirmImpact: () => void;
}

export function ManageImpactDialogs({
  deleteOpen,
  onDeleteOpenChange,
  onCloseDelete,
  pendingDeleteCount,
  batchOperation,
  batchBusy,
  onConfirmDelete,
  impactOpen,
  onImpactOpenChange,
  onCloseImpact,
  impacts,
  restoringDependencyId,
  restoredDependencies,
  onRestoreDependency,
  onConfirmImpact,
}: ManageImpactDialogsProps) {
  const { t } = useTranslation();

  return (
    <>
      <AlertDialog open={deleteOpen} onOpenChange={(value) => (value ? onDeleteOpenChange(true) : onCloseDelete())}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("manage.confirmDelete.title")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("manage.confirmDelete.body", { n: pendingDeleteCount })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={batchBusy} onClick={onCloseDelete}>
              {t("manage.confirmDelete.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={batchBusy}
              className="bg-destructive text-white hover:bg-destructive/90"
              onClick={(event) => {
                event.preventDefault();
                void onConfirmDelete();
              }}
            >
              {batchOperation === "delete" ? <Loader2 className="size-4 animate-spin" /> : null}
              {t("manage.confirmDelete.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={impactOpen} onOpenChange={(value) => (value ? onImpactOpenChange(true) : onCloseImpact())}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t("manage.disableImpact.title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 text-sm">
            <p>{t("manage.disableImpact.body")}</p>
            <div className="max-h-72 overflow-y-auto rounded-md border">
              {impacts.map((impact) => {
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
                        onClick={() => void onRestoreDependency(impact, candidates[0])}
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
                              onClick={() => void onRestoreDependency(impact, candidate)}
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
            <Button type="button" variant="ghost" disabled={Boolean(restoringDependencyId)} onClick={onCloseImpact}>
              {t("manage.disableImpact.cancel")}
            </Button>
            <Button
              type="button"
              variant="secondary"
              className="text-amber-700 dark:text-amber-300"
              disabled={Boolean(restoringDependencyId) || batchBusy}
              onClick={() => void onConfirmImpact()}
            >
              {batchBusy ? <Loader2 className="size-4 animate-spin" /> : null}
              {t("manage.disableImpact.continue")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
