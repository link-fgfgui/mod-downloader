import { useEffect, useMemo } from "react";
import {
  AlertCircle,
  CheckCircle2,
  Database,
  Download,
  HardDrive,
  PackageCheck,
  ListPlus,
  RefreshCw,
  Trash2,
  ToggleLeft,
  ToggleRight,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { formatCompactNumber, formatNumber } from "@/lib/format";
import i18n from "@/i18n";
import { useDownloadQueueStore } from "@/stores/downloadQueue";
import {
  selectHasSelectedInstance,
  selectSelectedInstanceLabel,
  useMinecraftStore,
} from "@/stores/minecraft";
import { useSettingsStore } from "@/stores/settings";
import { useStatsStore } from "@/stores/stats";

export function HomePage() {
  const { t } = useTranslation();
  const stats = useStatsStore((state) => state.stats);
  const loading = useStatsStore((state) => state.loading);
  const error = useStatsStore((state) => state.error);
  const updatedAt = useStatsStore((state) => state.updatedAt);
  const refresh = useStatsStore((state) => state.refresh);
  const queue = useDownloadQueueStore((state) => state.queue);
  const minecraftDir = useMinecraftStore((state) => state.dir);
  const minecraftVersion = useMinecraftStore((state) => state.minecraftVersion);
  const modLoader = useMinecraftStore((state) => state.modLoader);
  const hasSelectedInstance = useMinecraftStore(selectHasSelectedInstance);
  const selectedLabel = useMinecraftStore(selectSelectedInstanceLabel);
  const settings = useSettingsStore((state) => state.settings);
  const locale = i18n.language === "zh" ? "zh" : "en";

  useEffect(() => {
    void useStatsStore.getState().start();
    return () => useStatsStore.getState().stop();
  }, []);

  const totalOperations = useMemo(
    () =>
      stats.downloadsCompleted +
      stats.modsEnabled +
      stats.modsDisabled +
      stats.modsDeleted +
      stats.favoritesAdded +
      stats.packwizExports,
    [stats],
  );

  const metrics = [
    { key: "downloads", labelKey: "home.usage.downloads", value: stats.downloadsCompleted, icon: Download, color: "text-sky-600 dark:text-sky-400", bg: "bg-sky-500/10" },
    { key: "enabled", labelKey: "home.usage.enabled", value: stats.modsEnabled, icon: ToggleRight, color: "text-emerald-600 dark:text-emerald-400", bg: "bg-emerald-500/10" },
    { key: "disabled", labelKey: "home.usage.disabled", value: stats.modsDisabled, icon: ToggleLeft, color: "text-amber-600 dark:text-amber-400", bg: "bg-amber-500/10" },
    { key: "deleted", labelKey: "home.usage.deleted", value: stats.modsDeleted, icon: Trash2, color: "text-destructive", bg: "bg-destructive/10" },
    { key: "favorites", labelKey: "home.usage.favorites", value: stats.favoritesAdded, icon: ListPlus, color: "text-violet-600 dark:text-violet-400", bg: "bg-violet-500/10" },
    { key: "exports", labelKey: "home.usage.exports", value: stats.packwizExports, icon: PackageCheck, color: "text-cyan-600 dark:text-cyan-400", bg: "bg-cyan-500/10" },
  ];

  const queueActive = queue.running + queue.pending > 0;
  const updatedLabel = updatedAt ? new Intl.DateTimeFormat(locale, { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(updatedAt) : t("home.status.notAvailable");

  return (
    <div className="mx-auto flex h-full w-full max-w-6xl flex-col gap-6 overflow-y-auto p-6">
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="text-xs font-bold uppercase tracking-wider text-primary">{t("home.kicker")}</div>
          <h1 className="mt-1 text-2xl font-bold tracking-tight">{t("home.title")}</h1>
          <p className="mt-1 max-w-2xl text-sm text-muted-foreground">{t("home.subtitle")}</p>
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={error ? "destructive" : "secondary"} className="gap-1">
            {error ? <AlertCircle className="size-3" /> : <Database className="size-3" />}
            {error ? t("home.status.unavailable") : t("home.status.tracking")}
          </Badge>
          <Button type="button" size="sm" variant="outline" disabled={loading} onClick={() => void refresh()}>
            <RefreshCw className={loading ? "size-4 animate-spin" : "size-4"} />
            {t("home.refresh")}
          </Button>
        </div>
      </header>

      {loading ? <Progress value={undefined} className="h-0.5 animate-pulse" /> : null}

      {error ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
          {t("home.status.error", { reason: error })}
        </div>
      ) : null}

      <section className="grid gap-3">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h2 className="text-base font-semibold">{t("home.usage.title")}</h2>
            <p className="text-xs text-muted-foreground">{t("home.usage.sinceTracking")}</p>
          </div>
          <div className="flex items-baseline gap-2">
            <span key={totalOperations} className="text-3xl font-bold leading-none animate-in fade-in-0 duration-200">{formatNumber(totalOperations, locale)}</span>
            <span className="text-xs text-muted-foreground">{t("home.usage.total")}</span>
          </div>
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
          {metrics.map((metric, index) => {
            const Icon = metric.icon;
            return (
              <Card
                key={metric.key}
                className="gap-0 py-0 animate-in fade-in-0 slide-in-from-bottom-1 duration-300"
                style={{ animationDelay: `${index * 35}ms`, animationFillMode: "both" }}
              >
                <CardContent className="flex min-h-20 items-center gap-3 p-4">
                  <div className={`flex size-10 shrink-0 items-center justify-center rounded-lg ${metric.bg} ${metric.color}`}>
                    <Icon className="size-5" />
                  </div>
                  <div className="min-w-0">
                    <div key={metric.value} className="text-xl font-bold leading-tight animate-in fade-in-0 duration-200">{formatCompactNumber(metric.value, locale)}</div>
                    <div className="mt-0.5 text-xs text-muted-foreground">{t(metric.labelKey)}</div>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </section>

      <section className="grid gap-3">
        <h2 className="text-base font-semibold">{t("home.status.title")}</h2>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          <Card className="gap-3 py-4">
            <CardHeader className="px-4">
              <CardTitle className="flex items-center gap-2 text-sm">
                <CheckCircle2 className="size-4 text-emerald-500" />
                {t("home.status.instance")}
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-2 px-4 text-sm">
              <StatusRow label={t("home.status.selectedInstance")} value={hasSelectedInstance ? selectedLabel : t("home.status.none")} />
              <StatusRow label={t("home.status.target")} value={minecraftVersion && modLoader ? `${minecraftVersion} / ${modLoader}` : t("home.status.notConfigured")} />
              <StatusRow label={t("home.status.minecraftDir")} value={minecraftDir || t("home.status.notConfigured")} mono />
            </CardContent>
          </Card>

          <Card className="gap-3 py-4">
            <CardHeader className="px-4">
              <CardTitle className="flex items-center gap-2 text-sm">
                <Download className="size-4 text-sky-500" />
                {t("home.status.queue")}
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-2 px-4 text-sm">
              <StatusRow label={t("home.status.queueState")} value={queueActive ? t("home.status.active") : t("home.status.idle")} />
              <StatusRow label={t("home.status.running")} value={formatNumber(queue.running ?? 0, locale)} />
              <StatusRow label={t("home.status.pending")} value={formatNumber(queue.pending ?? 0, locale)} />
            </CardContent>
          </Card>

          <Card className="gap-3 py-4">
            <CardHeader className="px-4">
              <CardTitle className="flex items-center gap-2 text-sm">
                <HardDrive className="size-4 text-violet-500" />
                {t("home.status.storage")}
              </CardTitle>
            </CardHeader>
            <CardContent className="grid gap-2 px-4 text-sm">
              <StatusRow label={t("home.status.source")} value={settings?.mcimEnabled ? t("home.status.mcim") : t("home.status.official")} />
              <StatusRow label={t("home.status.historyStore")} value={error ? t("home.status.unavailable") : t("home.status.tracking")} />
              <StatusRow label={t("home.status.updated")} value={updatedLabel} />
            </CardContent>
          </Card>
        </div>
      </section>
    </div>
  );
}

function StatusRow({ label, value, mono = false }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="grid grid-cols-[minmax(0,9rem)_minmax(0,1fr)] gap-2">
      <span className="text-muted-foreground">{label}</span>
      <span className={`min-w-0 break-words text-right ${mono ? "font-mono text-xs" : ""}`}>{value}</span>
    </div>
  );
}
