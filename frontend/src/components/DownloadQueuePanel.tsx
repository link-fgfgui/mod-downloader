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

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { animateQueueExit, animateQueuePanel } from "@/lib/animations";
import { formatBytes, formatSpeed, progressPercent } from "@/lib/format";
import { useDownloadQueueStore } from "@/stores/downloadQueue";
import type { DownloadQueueItem, OptionalDependencyCandidate } from "@/types";
import { cn } from "@/lib/utils";

type QueueTab = "downloads" | "optional";

function statusIcon(status: string) {
  switch (status) {
    case "running":
      return <Download className="size-4" />;
    case "pending":
      return <Clock className="size-4" />;
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
  const visible = Boolean(queue.active || messageCount || items.length);
  const reminderOnly = !activeCount && messageCount > 0;
  const badgeCount = activeCount + messageCount;

  const [open, setOpen] = useState(false);
  const [tab, setTab] = useState<QueueTab>("downloads");
  const panelRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    if (!visible) setOpen(false);
  }, [visible]);

  useEffect(() => {
    if (reminders.length > 0 && items.length === 0) setTab("optional");
    else if (reminders.length === 0 && tab === "optional") setTab("downloads");
  }, [items.length, reminders.length, tab]);

  useEffect(() => {
    if (!open) return;
    return animateQueuePanel(panelRef.current);
  }, [open]);

  if (!visible) return null;

  const close = () => {
    void animateQueueExit(panelRef.current).then(() => setOpen(false));
  };

  const totalPercent = progressPercent(queue.bytesComplete, queue.totalBytes);

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

            {activeCount > 0 ? (
              <div className="grid gap-1.5">
                <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
                  <span>{t("download.queue.totalProgress")}</span>
                  <span>
                    {queue.totalBytes > 0
                      ? `${formatBytes(queue.bytesComplete)} / ${formatBytes(queue.totalBytes)} · ${totalPercent}%`
                      : t("download.queue.unknownSize")}
                  </span>
                </div>
                <Progress value={queue.totalBytes > 0 ? totalPercent : undefined} className={queue.totalBytes > 0 ? "h-1.5" : "h-1.5 animate-pulse"} />
                <div className="flex items-center justify-between text-xs text-muted-foreground">
                  <span>{t("download.queue.totalSpeed")}</span>
                  <strong className="text-foreground">{formatSpeed(queue.bytesPerSecond)}</strong>
                </div>
              </div>
            ) : null}
          </div>

          <div className="flex border-b">
            {(["downloads", "optional"] as QueueTab[]).map((value) => (
              <button
                key={value}
                type="button"
                className={cn(
                  "flex-1 border-b-2 px-3 py-2 text-xs font-medium transition-[color,border-color,opacity] duration-fast active:opacity-60",
                  tab === value ? "border-primary text-foreground" : "border-transparent text-muted-foreground hover:text-foreground",
                )}
                onClick={() => setTab(value)}
              >
                {t(`download.queue.tabs.${value}`)}
                {value === "optional" && reminders.length ? ` (${reminders.length})` : ""}
              </button>
            ))}
          </div>

          <div key={tab} className="max-h-[22rem] overflow-y-auto p-1.5 animate-in fade-in-0 slide-in-from-bottom-1 duration-150">
            {tab === "downloads" ? (
              items.length === 0 ? (
                <div className="p-6 text-center text-xs text-muted-foreground">{t("download.queue.emptyDownloads")}</div>
              ) : (
                items.map((item) => {
                  const percent = progressPercent(item.bytesComplete, item.totalBytes);
                  return (
                    <div key={item.id} className="grid grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-2 rounded-lg p-2 transition-colors duration-fast hover:bg-muted/50 animate-in fade-in-0 slide-in-from-bottom-1 duration-200">
                      <div
                        className={cn(
                          "flex size-8 items-center justify-center rounded-full",
                          item.status === "running" && "bg-primary/10 text-primary",
                          item.status === "pending" && "bg-sky-500/10 text-sky-600 dark:text-sky-400",
                          item.status === "failed" && "bg-destructive/10 text-destructive",
                          item.status === "canceled" && "bg-muted text-muted-foreground",
                        )}
                      >
                        {item.status === "running" ? <Loader2 className="size-4 animate-spin" /> : statusIcon(item.status)}
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
                        {isActiveItem(item) ? (
                          <div className="mt-1 grid gap-1">
                            <Progress value={item.totalBytes > 0 ? percent : undefined} className="h-1" />
                            <div className="flex justify-between text-[0.68rem] text-muted-foreground">
                              <span>{item.totalBytes > 0 ? `${formatBytes(item.bytesComplete)} / ${formatBytes(item.totalBytes)}` : t("download.queue.unknownSize")}</span>
                              {item.totalBytes > 0 ? <span>{percent}%</span> : null}
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
                                onClick={() => void cancel(item.id)}
                                onContextMenu={(event) => {
                                  if (!isRemovableItem(item)) return;
                                  event.preventDefault();
                                  void remove(item.id);
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
                                onClick={() => void retry(item.id)}
                                onContextMenu={(event) => {
                                  if (!isRemovableItem(item)) return;
                                  event.preventDefault();
                                  void remove(item.id);
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
                              <Button type="button" size="icon-xs" variant="ghost" aria-label={t("download.queue.removeItem")} onClick={() => void remove(item.id)}>
                                <Trash2 className="size-3.5" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>{t("download.queue.removeItem")}</TooltipContent>
                          </Tooltip>
                        ) : null}
                      </div>
                    </div>
                  );
                })
              )
            ) : reminders.length === 0 ? (
              <div className="p-6 text-center text-xs text-muted-foreground">{t("download.queue.emptyOptional")}</div>
            ) : (
              reminders.map((reminder) => (
                <div key={reminder.id} className="rounded-lg p-2 transition-colors duration-fast hover:bg-muted/50">
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
            {open ? (
              <ChevronDown key="open" className="size-5 animate-in fade-in-0 zoom-in-75 duration-150" />
            ) : reminderOnly ? (
              <Bell key="bell" className="size-5 animate-in fade-in-0 zoom-in-75 duration-150" />
            ) : (
              <Download key="download" className="size-5 animate-in fade-in-0 zoom-in-75 duration-150" />
            )}
            {badgeCount > 0 ? (
              <Badge key={badgeCount} className="absolute -right-1 -top-1 min-w-5 justify-center rounded-full px-1 text-[0.65rem] animate-in fade-in-0 zoom-in-75 duration-150">
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
