import type { RefObject } from "react";
import {
  AlertTriangle,
  ChevronDown,
  Eye,
  ListPlus,
  Loader2,
  Package,
  ToggleLeft,
  ToggleRight,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { AnimatedPresence } from "@/components/AnimatedPresence";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import {
  groupRowKey,
  groupTooltip,
  hasGroupedDetails,
  strongModIds,
} from "@/lib/modGroupDisplay";
import { cn } from "@/lib/utils";
import type { LocalModGroup, UnusedDependencyCandidate } from "@/types";

export interface ManageModListViewProps {
  scrollRef: RefObject<HTMLDivElement>;
  listExiting: boolean;
  compatible: LocalModGroup[];
  incompatible: LocalModGroup[];
  incompatibleExpanded: boolean;
  onToggleIncompatible: () => void;
  selectedKeys: Set<string>;
  exitingGroupKeys: Set<string>;
  candidateFor: (group: LocalModGroup) => UnusedDependencyCandidate | null;
  tooltipFor: (group: LocalModGroup) => string;
  canFavoriteGroup: (group: LocalModGroup) => boolean;
  canReplaceVersion: (group: LocalModGroup) => boolean;
  onToggleSelection: (group: LocalModGroup, event: { ctrlKey: boolean; metaKey: boolean }) => void;
  onOpenFavorites: (groups: LocalModGroup[]) => void;
  onOpenVersion: (group: LocalModGroup) => void;
  onToggleGroup: (group: LocalModGroup) => void;
  batchBusy: boolean;
  pendingTogglePath: string;
  hasSelection: boolean;
}

export function ManageModListView({
  scrollRef,
  listExiting,
  compatible,
  incompatible,
  incompatibleExpanded,
  onToggleIncompatible,
  selectedKeys,
  exitingGroupKeys,
  candidateFor,
  tooltipFor,
  canFavoriteGroup,
  canReplaceVersion,
  onToggleSelection,
  onOpenFavorites,
  onOpenVersion,
  onToggleGroup,
  batchBusy,
  pendingTogglePath,
  hasSelection,
}: ManageModListViewProps) {
  const { t } = useTranslation();

  const renderGroupCard = (group: LocalModGroup) => {
    const key = groupRowKey(group);
    const selected = selectedKeys.has(key);
    const unused = candidateFor(group);
    const categories = group.categories ?? [];
    const groupAction = group.primary.enabled ? "disable" : "enable";
    return (
      <div
        key={key}
        data-flip-key={key}
        className={cn(
          "md-row-interactive flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl border bg-card p-3 shadow-sm",
          selected
            ? "border-primary/50 bg-primary/10 shadow-sm hover:bg-primary/15"
            : "hover:border-foreground/20 hover:bg-accent/60 hover:shadow-md",
          group.incompatible && "opacity-80",
          exitingGroupKeys.has(key) && "md-row-exiting",
        )}
        onClick={(event) => onToggleSelection(group, event)}
      >
        <div className={cn("flex size-5 shrink-0 items-center justify-center rounded border transition-colors duration-md", selected ? "border-primary bg-primary text-primary-foreground" : "border-input")}>
          {selected ? <Eye className="size-3.5 animate-in fade-in-0 zoom-in-50 duration-md-fast" /> : null}
        </div>
        <div className="flex size-12 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-muted/40">
          {group.primary.onlineMetadataLoading ? (
            <Loader2 className="size-5 animate-spin text-muted-foreground" />
          ) : group.primary.iconUrl ? (
            <img src={group.primary.iconUrl} alt="" className="size-full object-cover" />
          ) : (
            <Package className="size-6 text-muted-foreground" />
          )}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex min-w-0 items-center gap-2">
            <span className="truncate text-sm font-semibold">{group.displayName}</span>
            {hasGroupedDetails(group) ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge variant="outline" className="shrink-0">{(group.strong?.length || 0) + (group.jij?.length || 0)}</Badge>
                </TooltipTrigger>
                <TooltipContent className="whitespace-pre-line">{groupTooltip(group)}</TooltipContent>
              </Tooltip>
            ) : null}
            {unused ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge variant="outline" className="shrink-0 border-amber-500/40 text-amber-700 dark:text-amber-300">
                    {t("manage.unused.chip")}
                  </Badge>
                </TooltipTrigger>
                <TooltipContent className="whitespace-pre-line">{tooltipFor(group)}</TooltipContent>
              </Tooltip>
            ) : null}
          </div>
          <div className="flex min-w-0 items-center gap-1 text-xs text-muted-foreground">
            <span className="truncate">{strongModIds(group).join(", ")}</span>
            {group.primary.version ? (
              <>
                <span className="shrink-0">·</span>
                <button
                  type="button"
                  className="group/version shrink-0 transition-[color,opacity] duration-fast active:opacity-60 disabled:no-underline"
                  disabled={!canReplaceVersion(group)}
                  onClick={(event) => {
                    event.stopPropagation();
                    onOpenVersion(group);
                  }}
                >
                  <span className="underline-offset-2 group-hover/version:underline">
                    {group.primary.version}
                  </span>
                </button>
              </>
            ) : null}
            {group.primary.fileName || group.primary.path ? (
              <span className="truncate">· {group.primary.fileName || group.primary.path}</span>
            ) : null}
          </div>
          {categories.length ? (
            <div className="mt-1 flex flex-wrap gap-1">
              {categories.slice(0, 3).map((category) => (
                <Badge key={category} variant="secondary" className="max-w-32 truncate text-[0.65rem]">
                  {category}
                </Badge>
              ))}
            </div>
          ) : null}
        </div>
        <div className="flex shrink-0 items-center gap-1.5">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                size="icon-sm"
                variant="secondary"
                disabled={!canFavoriteGroup(group)}
                aria-label={t("favorites.addToFavorites")}
                onClick={(event) => {
                  event.stopPropagation();
                  onOpenFavorites([group]);
                }}
              >
                <ListPlus className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("favorites.addToFavorites")}</TooltipContent>
          </Tooltip>
          <Button
            type="button"
            size="sm"
            variant="secondary"
            className={group.primary.enabled ? "text-emerald-700 dark:text-emerald-300" : "text-amber-700 dark:text-amber-300"}
            disabled={batchBusy}
            onClick={(event) => {
              event.stopPropagation();
              void onToggleGroup(group);
            }}
          >
            {pendingTogglePath === group.primary.path ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <span
                key={String(group.primary.enabled)}
                className="inline-flex animate-in fade-in-0 zoom-in-75 duration-md-fast"
              >
                {group.primary.enabled ? (
                  <ToggleRight className="size-4" />
                ) : (
                  <ToggleLeft className="size-4" />
                )}
              </span>
            )}
            <span className="hidden sm:inline">
              {t(groupAction === "disable" ? "manage.disableSelected" : "manage.enableSelected")}
            </span>
          </Button>
        </div>
      </div>
    );
  };

  return (
    <div
      ref={scrollRef}
      className={cn(
        "relative mt-4 min-h-0 flex-1 overflow-y-auto overflow-x-hidden transition-[opacity,transform] duration-md-fast animate-in fade-in-0 slide-in-from-left-2",
        listExiting && "pointer-events-none translate-y-1 opacity-0",
      )}
    >
      <div className={cn("grid gap-2", hasSelection && "pb-24")}>
        {compatible.map(renderGroupCard)}
        {incompatible.length > 0 ? (
          <>
            <button
              key="incompatible-toggle"
              type="button"
              className="md-row-interactive flex w-full min-w-0 items-center gap-3 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-left hover:bg-amber-500/20"
              onClick={onToggleIncompatible}
            >
              <AlertTriangle className="size-5 text-amber-600 dark:text-amber-400" />
              <span className="min-w-0 flex-1">
                <span className="block text-sm font-semibold">
                  {t("manage.incompatible.title", { n: incompatible.length })}
                </span>
                <span className="block text-xs text-muted-foreground">
                  {t(incompatibleExpanded ? "manage.incompatible.expanded" : "manage.incompatible.collapsed")}
                </span>
              </span>
              <ChevronDown className={cn("size-4 transition-transform", incompatibleExpanded && "rotate-180")} />
            </button>
            <AnimatedPresence open={incompatibleExpanded} mode="collapse" preserveContent>
              {incompatibleExpanded ? (
                <div className="grid gap-2 pt-2">{incompatible.map(renderGroupCard)}</div>
              ) : null}
            </AnimatedPresence>
          </>
        ) : null}
      </div>
    </div>
  );
}
