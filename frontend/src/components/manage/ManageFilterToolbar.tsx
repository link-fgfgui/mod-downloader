import { forwardRef, useRef, useState, type ReactNode, type RefObject } from "react";
import { ListTree, Loader2, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import { SearchInput } from "@/components/ui/search-input";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";

const ViewModeButton = forwardRef<
  HTMLButtonElement,
  {
    active: boolean;
    overlay?: boolean;
    onClick?: () => void;
    children: ReactNode;
  }
>(function ViewModeButton({ active, overlay = false, onClick, children }, ref) {
  return (
    <button
      ref={ref}
      type="button"
      tabIndex={overlay ? -1 : undefined}
      aria-hidden={overlay || undefined}
      aria-pressed={overlay ? undefined : active}
      className={cn(
        "md-pressable relative z-10 inline-flex h-7 shrink-0 items-center gap-1.5 rounded-md px-3 text-sm font-medium whitespace-nowrap",
        overlay
          ? "text-primary-foreground"
          : active
            ? "text-foreground"
            : "text-muted-foreground hover:text-foreground",
      )}
      onClick={overlay ? undefined : onClick}
    >
      {children}
    </button>
  );
});

export interface ManageFilterToolbarProps {
  selectedLabel: string;
  refreshing: boolean;
  batchBusy: boolean;
  onRefresh: () => void;
  viewMode: "list" | "tree";
  onViewModeChange: (mode: "list" | "tree") => void;
  viewModeRef: RefObject<HTMLDivElement>;
  listModeRef: RefObject<HTMLButtonElement>;
  treeModeRef: RefObject<HTMLButtonElement>;
  viewIndicator: { left: number; width: number };
  showSearch: boolean;
  searchInput: string;
  onSearchInputChange: (value: string) => void;
  enabledFilter: string;
  onEnabledFilterChange: (value: string) => void;
  sortMode: string;
  onSortModeChange: (value: string) => void;
  showSort: boolean;
  jijDetectionEnabled: boolean;
  scanning: boolean;
}

export function ManageFilterToolbar({
  selectedLabel,
  refreshing,
  batchBusy,
  onRefresh,
  viewMode,
  onViewModeChange,
  viewModeRef,
  listModeRef,
  treeModeRef,
  viewIndicator,
  showSearch,
  searchInput,
  onSearchInputChange,
  enabledFilter,
  onEnabledFilterChange,
  sortMode,
  onSortModeChange,
  showSort,
  jijDetectionEnabled,
  scanning,
}: ManageFilterToolbarProps) {
  const { t } = useTranslation();

  return (
    <>
      <div className="flex shrink-0 flex-wrap items-center gap-3">
        <div className="min-w-0 flex-1">
          <h1 className="text-2xl font-bold tracking-tight">{t("manage.title")}</h1>
          <div className="truncate text-sm text-muted-foreground">{selectedLabel}</div>
        </div>
        <Button type="button" size="sm" disabled={refreshing || batchBusy} onClick={() => void onRefresh()}>
          <RefreshCw className={refreshing ? "size-4 animate-spin" : "size-4"} />
          {t("manage.refresh")}
        </Button>
        <div ref={viewModeRef} className="relative flex rounded-lg border bg-muted/30 p-0.5">
          <span
            aria-hidden="true"
            className="pointer-events-none absolute top-0.5 bottom-0.5 left-0 rounded-md bg-primary transition-[transform,width] duration-md ease-out"
            style={{ transform: `translateX(${viewIndicator.left}px)`, width: viewIndicator.width }}
          />
          <ViewModeButton
            ref={listModeRef}
            active={viewMode === "list"}
            onClick={() => onViewModeChange("list")}
          >
            {t("manage.viewMode.list")}
          </ViewModeButton>
          <ViewModeButton
            ref={treeModeRef}
            active={viewMode === "tree"}
            onClick={() => onViewModeChange("tree")}
          >
            <ListTree className="size-4" />
            {t("manage.viewMode.tree")}
          </ViewModeButton>
          {/* Clipped duplicate keeps the label readable while the indicator
              slides between the two options. Without it the incoming label
              would switch to dark text before the light pill arrives. */}
          <span
            aria-hidden="true"
            className="pointer-events-none absolute inset-0 z-20 flex overflow-hidden rounded-lg p-0.5 transition-[clip-path] duration-md ease-out"
            style={{
              clipPath: `inset(0 ${Math.max(
                0,
                (viewModeRef.current?.clientWidth ?? 0) - viewIndicator.left - viewIndicator.width,
              )}px 0 ${viewIndicator.left}px round 6px)`,
            }}
          >
            <ViewModeButton active overlay>
              {t("manage.viewMode.list")}
            </ViewModeButton>
            <ViewModeButton active overlay>
              <ListTree className="size-4" />
              {t("manage.viewMode.tree")}
            </ViewModeButton>
          </span>
        </div>
      </div>

      {showSearch ? (
        <div className="mt-4 flex shrink-0 flex-wrap items-center gap-2">
          <div className="relative min-w-0 flex-1 basis-64">
            <SearchInput
              value={searchInput}
              onValueChange={onSearchInputChange}
              placeholder={t("manage.search.label")}
              aria-label={t("manage.search.label")}
              clearAriaLabel={t("common.clearSearch")}
            />
          </div>
          <HelpTooltip label={t("manage.search.syntaxTip")} />
          <Combobox
            value={enabledFilter}
            onValueChange={onEnabledFilterChange}
            options={[
              { value: "all", label: t("manage.filter.all") },
              { value: "enabled", label: t("manage.enabled") },
              { value: "disabled", label: t("manage.disabled") },
            ]}
            aria-label={t("manage.filter.all")}
            className="w-36"
          />
          {showSort ? (
            <Combobox
              value={sortMode}
              onValueChange={onSortModeChange}
              options={[
                { value: "default", label: t("manage.sort.default") },
                { value: "name", label: t("manage.sort.name") },
                {
                  value: "unused",
                  label: t("manage.sort.unused"),
                  disabled: !jijDetectionEnabled,
                },
              ]}
              aria-label={t("manage.sort.default")}
              className="w-48"
            />
          ) : null}
          {scanning ? <Loader2 className="size-4 animate-spin text-muted-foreground" /> : null}
        </div>
      ) : null}
    </>
  );
}

/**
 * Search-syntax help. This is deliberately hover/focus only: a click does not
 * toggle or dismiss it, so there is just one interaction model.
 */
function HelpTooltip({ label }: { label: string }) {
  const [open, setOpen] = useState(false);
  const hovered = useRef(false);
  const focused = useRef(false);

  const show = () => setOpen(true);
  const sync = () => {
    if (!hovered.current && !focused.current) setOpen(false);
  };

  return (
    <Tooltip
      open={open}
      onOpenChange={(next) => {
        if (next) show();
        else if (!hovered.current && !focused.current) setOpen(false);
      }}
    >
      <TooltipTrigger asChild>
        <span
          tabIndex={0}
          role="img"
          aria-label={label}
          className="md-pressable inline-flex size-6 shrink-0 cursor-help items-center justify-center rounded-full text-sm font-bold text-muted-foreground outline-none hover:text-foreground focus-visible:ring-[3px] focus-visible:ring-ring/50"
          onPointerEnter={() => {
            hovered.current = true;
            show();
          }}
          onPointerLeave={() => {
            hovered.current = false;
            sync();
          }}
          onFocus={() => {
            focused.current = true;
            show();
          }}
          onBlur={() => {
            focused.current = false;
            sync();
          }}
          onPointerDown={(event) => {
            // Keep the click from moving focus/dismissing the Radix tooltip.
            event.preventDefault();
            event.stopPropagation();
          }}
          onClick={(event) => event.stopPropagation()}
          onKeyDown={(event) => {
            if (event.key === "Escape") {
              event.preventDefault();
              focused.current = false;
              setOpen(false);
            }
          }}
        >
          ?
        </span>
      </TooltipTrigger>
      <TooltipContent className="max-w-xs">{label}</TooltipContent>
    </Tooltip>
  );
}
