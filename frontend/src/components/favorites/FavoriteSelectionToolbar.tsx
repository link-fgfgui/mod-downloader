import { Copy, Trash2, X } from "lucide-react";
import { useTranslation } from "react-i18next";

import { AnimatedPresence } from "@/components/AnimatedPresence";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";

export interface FavoriteSelectionToolbarProps {
  selectedCount: number;
  onCopySelected: () => void;
  onRemoveSelected: () => void;
  onDeselectAll: () => void;
}

export function FavoriteSelectionToolbar({
  selectedCount,
  onCopySelected,
  onRemoveSelected,
  onDeselectAll,
}: FavoriteSelectionToolbarProps) {
  const { t } = useTranslation();

  return (
    <div className="pointer-events-none absolute inset-x-0 bottom-4 z-30 flex justify-center px-2">
      <AnimatedPresence
        open={selectedCount > 0}
        mode="float"
        preserveContent
        className="pointer-events-auto max-w-full"
      >
        <div className="md-selection-toolbar flex max-w-full flex-wrap items-center justify-center gap-1.5 rounded-2xl border bg-popover/95 p-2 shadow-2xl backdrop-blur-md">
          <span
            key={selectedCount}
            aria-label={t("manage.selectedCount", { n: selectedCount })}
            className="flex size-9 animate-in items-center justify-center rounded-full bg-sky-500/15 text-sm font-semibold text-sky-600 zoom-in-75 fade-in-0 duration-md-fast tabular-nums dark:text-sky-300"
          >
            {selectedCount}
          </span>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                size="icon"
                variant="ghost"
                className="rounded-full bg-muted text-muted-foreground hover:bg-muted/70 hover:text-foreground"
                aria-label={t("favorites.actions.copySelected")}
                onClick={onCopySelected}
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
                onClick={onRemoveSelected}
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
                onClick={onDeselectAll}
              >
                <X className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("download.selection.deselectAll")}</TooltipContent>
          </Tooltip>
        </div>
      </AnimatedPresence>
    </div>
  );
}
