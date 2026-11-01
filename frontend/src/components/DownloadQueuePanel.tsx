import { useEffect, useRef, useState } from "react";
import {
  Bell,
  BellOff,
  Check,
  ChevronDown,
  CircleAlert,
  CircleX,
  Clock,
  Download,
  Loader2,
  RefreshCw,
  Trash2,
  X,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { AnimatedPresence } from "@/components/AnimatedPresence";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useSmoothedValue, useThrottledValue } from "@/hooks/useSmoothedValue";
import {
  animateQueueExit,
  animateQueuePanel,
  animateScaleExit,
  scaledDuration,
} from "@/lib/animations";
import { formatBytes, formatSpeed, progressPercent } from "@/lib/format";
import { useDownloadQueueStore } from "@/stores/downloadQueue";
import type { DownloadQueueItem, OptionalDependencyCandidate } from "@/types";
import { cn } from "@/lib/utils";

type QueueTab = "downloads" | "optional";

const RING_RADIUS = 21;
const RING_CIRCUMFERENCE = 2 * Math.PI * RING_RADIUS;

function statusIcon(status: string) {
  switch (status) {
    case "running":
      return <Download className="size-4" />;
    case "pending":
      return <Clock className="size-4" />;
    case "completed":
      return <Check className="size-4" />;
    case "failed":
      return <CircleAlert className="size-4" />;
    case "canceled":
      return <CircleX className="size-4" />;
    default:
      return <Download className="size-4" />;
  }
}

function optionalStatusIcon(status: string) {
  switch (status) {
    case "installed":
      return <Check className="size-4 text-emerald-500" />;
    case "update":
      return <RefreshCw className="size-4 text-sky-500" />;
    case "conflict":
    case "incompatible":
      return <CircleAlert className="size-4 text-destructive" />;
    default:
      return <Download className="size-4 text-muted-foreground" />;
  }
}

function isActiveItem(item: DownloadQueueItem) {
  return item.status === "pending" || item.status === "running";
}

function isRemovableItem(item: DownloadQueueItem) {
  return item.status === "pending" || item.status === "failed" || item.status === "canceled";
}

interface QueueRowProps {
  item: DownloadQueueItem;
  removing: boolean;
  completed: boolean;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
  onRemove: (id: string) => void;
}

/**
 * One download row. Progress is smoothed independently per row so a burst of
 * backend events does not make the bar and percentage jump, while the row
 * keeps its own collapse/exit transition when the user removes it.
 */
function QueueRow({ item, removing, completed, onCancel, onRetry, onRemove }: QueueRowProps) {
  const { t } = useTranslation();
  const percent = progressPercent(item.bytesComplete, item.totalBytes);
  const smoothedPercent = useSmoothedValue(percent);
  const displayPercent = Math.round(smoothedPercent);
  const active = isActiveItem(item);

  return (
    <div
      data-queue-item={item.id}
      className={cn(
        "grid overflow-hidden transition-[grid-template-rows,opacity,transform] duration-md-fast",
        removing
          ? "pointer-events-none translate-x-2 grid-rows-[0fr] opacity-0"
          : "translate-x-0 grid-rows-[1fr] opacity-100",
      )}
    >
      <div className="min-h-0">
        <div
          className={cn(
            "grid grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-2 rounded-lg p-2 transition-colors duration-md-fast hover:bg-muted/50 animate-in fade-in-0 slide-in-from-bottom-1 duration-md",
            completed && "bg-emerald-500/10 ring-1 ring-emerald-500/30",
          )}
        >
          <div
            className={cn(
              "flex size-8 items-center justify-center rounded-full transition-colors duration-md-fast",
              item.status === "running" && "bg-primary/10 text-primary",
              item.status === "pending" && "bg-sky-500/10 text-sky-600 dark:text-sky-400",
              item.status === "completed" && "bg-emerald-500/10 text-emerald-600 dark:text-emerald-400",
              item.status === "failed" && "bg-destructive/10 text-destructive",
              item.status === "canceled" && "bg-muted text-muted-foreground",
            )}
          >
            {item.status === "running" ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <span key={item.status} className="inline-flex animate-in fade-in-0 zoom-in-75 duration-md-fast">
                {statusIcon(item.status)}
              </span>
            )}
          </div>
          <div className="min-w-0">
            <div className="truncate text-sm font-medium">{item.title || item.fileName || item.versionId}</div>
            <div className="truncate text-xs text-muted-foreground">
              {t(`download.queue.status.${item.status}`)}
              {[item.platform, item.fileName].filter(Boolean).length
                ? ` · ${[item.platform, item.fileName].filter(Boolean).join(" · ")}`
                : ""}
            </div>
            {item.reason ? <div className="truncate text-xs text-destructive">{item.reason}</div> : null}
            {active ? (
              <div className="mt-1 grid gap-1">
                <Progress value={item.totalBytes > 0 ? smoothedPercent : undefined} className="h-1" />
                <div className="flex justify-between text-[0.68rem] text-muted-foreground tabular-nums">
                  <span>
                    {item.totalBytes > 0
                      ? `${formatBytes(item.bytesComplete)} / ${formatBytes(item.totalBytes)}`
                      : t("download.queue.unknownSize")}
                  </span>
                  {item.totalBytes > 0 ? <span>{displayPercent}%</span> : null}
                </div>
              </div>
            ) : null}
          </div>
          <div className="flex items-center gap-1">
            {item.cancelable ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon-xs"
                    variant="ghost"
                    className="text-destructive"
                    aria-label={t("download.queue.cancel")}
                    onClick={() => onCancel(item.id)}
                    onContextMenu={(event) => {
                      if (!isRemovableItem(item)) return;
                      event.preventDefault();
                      onRemove(item.id);
                    }}
                  >
                    <CircleX className="size-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("download.queue.cancel")}</TooltipContent>
              </Tooltip>
            ) : null}
            {item.retryable ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon-xs"
                    variant="ghost"
                    aria-label={t("download.queue.retry")}
                    onClick={() => onRetry(item.id)}
                    onContextMenu={(event) => {
                      if (!isRemovableItem(item)) return;
                      event.preventDefault();
                      onRemove(item.id);
                    }}
                  >
                    <RefreshCw className="size-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("download.queue.retry")}</TooltipContent>
              </Tooltip>
            ) : null}
            {isRemovableItem(item) ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon-xs"
                    variant="ghost"
                    aria-label={t("download.queue.removeItem")}
                    onClick={() => onRemove(item.id)}
                  >
                    <Trash2 className="size-3.5" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{t("download.queue.removeItem")}</TooltipContent>
              </Tooltip>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}

export function DownloadQueuePanel() {
  const { t } = useTranslation();
  const queue = useDownloadQueueStore((state) => state.queue);
  const cancel = useDownloadQueueStore((state) => state.cancel);
  const retry = useDownloadQueueStore((state) => state.retry);
  const remove = useDownloadQueueStore((state) => state.remove);
  const dismissOptional = useDownloadQueueStore((state) => state.dismissOptional);
  const clearOptional = useDownloadQueueStore((state) => state.clearOptional);
  const installOptional = useDownloadQueueStore((state) => state.installOptional);

  const items = queue.items ?? [];
  const reminders = queue.optionalReminders ?? [];
  const activeCount = (queue.pending || 0) + (queue.running || 0);
  const messageCount = queue.messageCount || reminders.length;
  const active = Boolean(queue.active || messageCount || items.length);
  const reminderOnly = !activeCount && messageCount > 0;
  const badgeCount = activeCount + messageCount;

  const [rendered, setRendered] = useState(active);
  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<QueueTab>("downloads");
  const [removingIds, setRemovingIds] = useState<Set<string>>(new Set());
  const [completedIds, setCompletedIds] = useState<Set<string>>(new Set());
  const panelRef = useRef<HTMLDivElement | null>(null);
  const fabRef = useRef<HTMLButtonElement | null>(null);
  const openRef = useRef(open);
  const completedTimers = useRef(new Map<string, number>());
  const removeTimers = useRef(new Map<string, number>());
  const previousStatuses = useRef(new Map<string, string>());

  useEffect(() => {
    openRef.current = open;
  }, [open]);

  useEffect(() => {
    if (active) setRendered(true);
  }, [active]);

  // Exit the whole floating layer before unmounting: the open panel retracts
  // and the FAB scales/fades together, so clearing the queue never pops.
  useEffect(() => {
    if (active) return;
    let cancelled = false;
    void (async () => {
      await Promise.all([
        openRef.current ? animateQueueExit(panelRef.current) : Promise.resolve(),
        animateScaleExit(fabRef.current, { scale: 0.75, baseDuration: 0.16 }),
      ]);
      if (cancelled) return;
      setOpen(false);
      setRendered(false);
    })();
    return () => {
      cancelled = true;
    };
  }, [active]);

  useEffect(() => {
    if (reminders.length > 0 && items.length === 0) setTab("optional");
    else if (reminders.length === 0 && tab === "optional") setTab("downloads");
  }, [items.length, reminders.length, tab]);

  useEffect(() => {
    if (!open) return;
    return animateQueuePanel(panelRef.current);
  }, [open]);

  // Track status transitions so a row that just completed can flash a success
  // highlight. Backend-driven removal is intentionally not animated.
  useEffect(() => {
    const previous = previousStatuses.current;
    const next = new Map<string, string>();
    const newlyCompleted: string[] = [];
    for (const item of items) {
      next.set(item.id, item.status);
      const before = previous.get(item.id);
      if (before && before !== "completed" && item.status === "completed") {
        newlyCompleted.push(item.id);
      }
    }
    previousStatuses.current = next;
    if (newlyCompleted.length === 0) return;
    setCompletedIds((current) => {
      const merged = new Set(current);
      for (const id of newlyCompleted) merged.add(id);
      return merged;
    });
    const highlight = Math.max(320, scaledDuration(0.32) * 1000) + 60;
    for (const id of newlyCompleted) {
      const existing = completedTimers.current.get(id);
      if (existing !== undefined) window.clearTimeout(existing);
      const timer = window.setTimeout(() => {
        completedTimers.current.delete(id);
        setCompletedIds((current) => {
          if (!current.has(id)) return current;
          const nextIds = new Set(current);
          nextIds.delete(id);
          return nextIds;
        });
      }, highlight);
      completedTimers.current.set(id, timer);
    }
  }, [items]);

  useEffect(
    () => () => {
      for (const timer of completedTimers.current.values()) window.clearTimeout(timer);
      for (const timer of removeTimers.current.values()) window.clearTimeout(timer);
      completedTimers.current.clear();
      removeTimers.current.clear();
    },
    [],
  );

  const totalPercent = progressPercent(queue.bytesComplete, queue.totalBytes);
  const smoothedTotalPercent = useSmoothedValue(totalPercent);
  const displayTotalPercent = Math.round(smoothedTotalPercent);
  const smoothedSpeed = useSmoothedValue(queue.bytesPerSecond);
  const displaySpeed = useThrottledValue(formatSpeed(smoothedSpeed), 350);
  const ringPercent = Math.max(0, Math.min(100, smoothedTotalPercent));
  const showRing = activeCount > 0;

  if (!rendered) return null;

  const close = () => {
    void animateQueueExit(panelRef.current).then(() => setOpen(false));
  };

  // User-initiated removal plays the row exit first, then lets the store
  // update. Backend events (cancel, completion) never wait for animation.
  const handleRemove = (id: string) => {
    if (removingIds.has(id)) return;
    setRemovingIds((current) => new Set(current).add(id));
    const delay = scaledDuration(0.15) * 1000;
    const finish = () => {
      removeTimers.current.delete(id);
      void remove(id).finally(() => {
        setRemovingIds((current) => {
          if (!current.has(id)) return current;
          const next = new Set(current);
          next.delete(id);
          return next;
        });
      });
    };
    if (delay <= 0) {
      finish();
      return;
    }
    removeTimers.current.set(id, window.setTimeout(finish, delay));
  };

  return (
    <div className="fixed bottom-5 right-5 z-40 flex flex-col items-end gap-3">
      {open ? (
        <div
          ref={panelRef}
          className="w-[min(26rem,calc(100vw-2rem))] overflow-hidden rounded-xl border bg-popover text-popover-foreground shadow-2xl"
        >
          <div className="grid gap-3 border-b p-3">
            <div className="flex items-start justify-between gap-3">
              <div>
                <div className="text-sm font-semibold">{t("download.queue.title")}</div>
                <div className="text-xs text-muted-foreground">
                  {t("download.queue.summary", { running: queue.running, pending: queue.pending })}
                </div>
              </div>
              <Button type="button" size="icon-xs" variant="ghost" aria-label={t("download.queue.close")} onClick={close}>
                <X className="size-3.5" />
              </Button>
            </div>

            <AnimatedPresence open={activeCount > 0} mode="collapse" preserveContent>
              <div className="grid gap-1.5">
                <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground tabular-nums">
                  <span>{t("download.queue.totalProgress")}</span>
                  <span>
                    {queue.totalBytes > 0
                      ? `${formatBytes(queue.bytesComplete)} / ${formatBytes(queue.totalBytes)} · ${displayTotalPercent}%`
                      : t("download.queue.unknownSize")}
                  </span>
                </div>
                <Progress
                  value={queue.totalBytes > 0 ? smoothedTotalPercent : undefined}
                  className={queue.totalBytes > 0 ? "h-1.5" : "h-1.5 animate-pulse"}
                />
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{t("download.queue.totalSpeed")}</span>
                  <strong className="text-foreground tabular-nums">{displaySpeed}</strong>
                </div>
              </div>
            </AnimatedPresence>
          </div>

          <div className="relative flex border-b">
            <span
              aria-hidden="true"
              className="pointer-events-none absolute bottom-0 left-0 h-0.5 w-1/2 rounded-full bg-primary transition-transform duration-md ease-[var(--md-ease-out)]"
              style={{ transform: tab === "optional" ? "translateX(100%)" : "translateX(0)" }}
            />
            {(["downloads", "optional"] as QueueTab[]).map((value) => (
              <button
                key={value}
                type="button"
                role="tab"
                aria-selected={tab === value}
                className={cn(
                  "relative z-10 flex-1 px-3 py-2 text-xs font-medium transition-[color,opacity] duration-md-fast active:opacity-60",
                  tab === value ? "text-foreground" : "text-muted-foreground hover:text-foreground",
                )}
                onClick={() => setTab(value)}
              >
                {t(`download.queue.tabs.${value}`)}
                {value === "optional" && reminders.length ? ` (${reminders.length})` : ""}
              </button>
            ))}
          </div>

          <div key={tab} className="max-h-[22rem] overflow-y-auto p-1.5 animate-in fade-in-0 slide-in-from-bottom-1 duration-md-fast">
            {tab === "downloads" ? (
              items.length === 0 ? (
                <div className="p-6 text-center text-xs text-muted-foreground">{t("download.queue.emptyDownloads")}</div>
              ) : (
                items.map((item) => (
                  <QueueRow
                    key={item.id}
                    item={item}
                    removing={removingIds.has(item.id)}
                    completed={completedIds.has(item.id)}
                    onCancel={(id) => void cancel(id)}
                    onRetry={(id) => void retry(id)}
                    onRemove={handleRemove}
                  />
                ))
              )
            ) : reminders.length === 0 ? (
              <div className="p-6 text-center text-xs text-muted-foreground">{t("download.queue.emptyOptional")}</div>
            ) : (
              reminders.map((reminder) => (
                <div key={reminder.id} className="rounded-lg p-2 transition-colors duration-md-fast hover:bg-muted/50">
                  <div className="flex items-center justify-between gap-2">
                    <div className="min-w-0 truncate text-sm font-medium">{reminder.mainTitle || reminder.mainProjectKey}</div>
                    <div className="flex shrink-0 items-center gap-1">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button type="button" size="icon-xs" variant="ghost" aria-label={t("download.queue.installOptional")} onClick={() => void installOptional(reminder.id)}>
                            <Download className="size-3.5" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t("download.queue.installOptional")}</TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button type="button" size="icon-xs" variant="ghost" aria-label={t("download.queue.dismissOptional")} onClick={() => void dismissOptional(reminder.id)}>
                            <BellOff className="size-3.5" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t("download.queue.dismissOptional")}</TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                  <div className="mt-1 grid gap-1">
                    {(reminder.dependencies ?? []).map((candidate: OptionalDependencyCandidate) => (
                      <div
                        key={candidate.projectKey}
                        className={cn(
                          "flex items-center gap-2 rounded-md px-1 py-1 text-xs",
                          candidate.disabled && "opacity-55",
                        )}
                      >
                        {optionalStatusIcon(candidate.status)}
                        <div className="min-w-0">
                          <div className="truncate font-medium">{candidate.title || candidate.projectKey}</div>
                          <div className="truncate text-muted-foreground">
                            {[candidate.platform, candidate.versionId, candidate.status, candidate.reason].filter(Boolean).join(" · ")}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>
      ) : null}

      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            ref={fabRef}
            type="button"
            size="icon-lg"
            className="relative size-12 rounded-full shadow-lg"
            aria-label={t("download.queue.title")}
            onClick={() => (open ? close() : setOpen(true))}
            onContextMenu={(event) => {
              if (!reminderOnly) return;
              event.preventDefault();
              void clearOptional();
            }}
          >
            {showRing ? (
              <svg
                aria-hidden="true"
                data-queue-ring
                viewBox="0 0 48 48"
                className="pointer-events-none absolute inset-0 size-12 -rotate-90"
              >
                <circle cx="24" cy="24" r={RING_RADIUS} fill="none" stroke="currentColor" strokeWidth="2" className="opacity-20" />
                <circle
                  cx="24"
                  cy="24"
                  r={RING_RADIUS}
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeDasharray={RING_CIRCUMFERENCE}
                  strokeDashoffset={RING_CIRCUMFERENCE * (1 - ringPercent / 100)}
                  className="transition-[stroke-dashoffset] duration-md ease-[var(--md-ease-out)]"
                />
              </svg>
            ) : null}
            {open ? (
              <ChevronDown key="open" className="size-5 animate-in fade-in-0 zoom-in-75 duration-md-fast" />
            ) : reminderOnly ? (
              <Bell key="bell" className="size-5 animate-in fade-in-0 zoom-in-75 duration-md-fast" />
            ) : (
              <Download key="download" className="size-5 animate-in fade-in-0 zoom-in-75 duration-md-fast" />
            )}
            {badgeCount > 0 ? (
              <Badge
                key={badgeCount}
                className="absolute -right-1 -top-1 min-w-5 justify-center rounded-full px-1 text-[0.65rem] animate-in fade-in-0 zoom-in-75 duration-md-fast tabular-nums"
              >
                {badgeCount}
              </Badge>
            ) : null}
          </Button>
        </TooltipTrigger>
        <TooltipContent>{t("download.queue.title")}</TooltipContent>
      </Tooltip>
    </div>
  );
}
