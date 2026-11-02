import { Copy, ListPlus, ListTree, Loader2, RefreshCw, ToggleLeft, ToggleRight, Trash2, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { AnimatedPresence } from "@/components/AnimatedPresence";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import type { ManageSelectionAction } from "@/hooks/useLocalModSelection";
import type { LocalModGroup } from "@/types";

export interface ManageSelectionToolbarProps {
  selectedGroups: LocalModGroup[];
  canFavoriteGroup: (group: LocalModGroup) => boolean;
  onOpenFavorites: (groups: LocalModGroup[]) => void;
  actions: ManageSelectionAction[];
  batchBusy: boolean;
}

export function ManageSelectionToolbar({
  selectedGroups,
  canFavoriteGroup,
  onOpenFavorites,
  actions,
  batchBusy,
}: ManageSelectionToolbarProps) {
  const { t } = useTranslation();

  return (
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
            key={selectedGroups.length}
            aria-hidden="true"
            className="flex size-9 animate-in items-center justify-center rounded-full bg-sky-500/15 text-sm font-semibold text-sky-600 zoom-in-75 fade-in-0 duration-md-fast tabular-nums dark:text-sky-300"
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
                onClick={() => onOpenFavorites(selectedGroups)}
              >
                <ListPlus className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("favorites.addToFavorites")}</TooltipContent>
          </Tooltip>
          {actions.map((action) => {
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
                    disabled={batchBusy}
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
  );
}
