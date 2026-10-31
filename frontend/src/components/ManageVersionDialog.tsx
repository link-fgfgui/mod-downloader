import { useEffect, useMemo, useState } from "react";
import { AlertTriangle, CheckCircle2, Download, Loader2, ServerCrash } from "lucide-react";
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
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { ModVersionList, type DependencyBreakDetail } from "@/components/ModVersionList";
import { api } from "@/services/app";
import {
  findReverseDependents,
  providedIdsOf,
  type ReverseDependencyMatch,
} from "@/lib/modDependencyGraph";
import { displayModName, modCategories, type ModGroup } from "@/lib/modGroups";
import { hasMajorVersionChange, isVersionDowngrade } from "@/utils/versionCompare";
import { useMinecraftStore } from "@/stores/minecraft";
import type { DependencyPreflightResult, ModInfo, ModProject, ModVersion } from "@/types";

type DependencyPreflightEntry = NonNullable<DependencyPreflightResult["entries"]>[number];

const EMPTY_MODS: ModInfo[] = [];

interface ManageVersionDialogProps {
  open: boolean;
  group: ModGroup | null;
  onOpenChange: (open: boolean) => void;
  onQueued: () => void;
  onError: (error: unknown) => void;
  onNotify: (key: string, color?: "success" | "error") => void;
}

const onlineProjectFromGroup = (group: ModGroup): ModProject => {
  const mod = group.primary;
  const platform = mod.onlinePlatform || "";
  const projectId = mod.onlineProjectId || "";
  return {
    id: `${platform.toLowerCase()}:${projectId}`,
    platform,
    projectId,
    slug: mod.onlineSlug || "",
    title: displayModName(group),
    icon: "",
    iconUrl: mod.iconUrl || "",
    description: mod.description || "",
    downloads: 0,
    categories: modCategories(group),
    updatedAt: 0,
    cachedAt: 0,
  };
};

export function ManageVersionDialog({
  open,
  group,
  onOpenChange,
  onQueued,
  onError,
  onNotify,
}: ManageVersionDialogProps) {
  const { t } = useTranslation();
  const minecraftVersion = useMinecraftStore((state) => state.minecraftVersion);
  const modLoader = useMinecraftStore((state) => state.modLoader);
  const mods = useMinecraftStore((state) => state.selected?.mods ?? EMPTY_MODS);
  const hasSelectedInstance = useMinecraftStore((state) => Boolean(state.selected?.id || state.selected?.name));
  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [replacingVersionId, setReplacingVersionId] = useState("");
  const [pinnedVersionId, setPinnedVersionId] = useState("");
  const [pinningVersionId, setPinningVersionId] = useState("");
  const [preflightResult, setPreflightResult] = useState<DependencyPreflightResult | null>(null);
  const [preflightOpen, setPreflightOpen] = useState(false);
  const [pendingVersion, setPendingVersion] = useState<ModVersion | null>(null);
  const [downgradeOpen, setDowngradeOpen] = useState(false);

  const canReplace = Boolean(
    group?.primary?.onlinePlatform && group?.primary?.onlineProjectId && hasSelectedInstance,
  );

  useEffect(() => {
    if (!open || !group || !canReplace) return;
    let cancelled = false;
    setVersions([]);
    setPinnedVersionId("");
    setReplacingVersionId("");
    setLoading(true);
    const project = onlineProjectFromGroup(group);
    void Promise.all([
      api.ListMatchingProjectVersions(project, minecraftVersion, modLoader),
      api.GetPinnedModVersion(project.platform, project.projectId, minecraftVersion, modLoader),
    ])
      .then(([matched, pin]) => {
        if (cancelled) return;
        setVersions(matched ?? []);
        setPinnedVersionId(pin?.versionId || "");
      })
      .catch(onError)
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
    // onError is intentionally not part of the trigger: it is a notification sink.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [canReplace, group, minecraftVersion, modLoader, open]);

  const dependents: ReverseDependencyMatch[] = useMemo(() => {
    if (!group) return [];
    const providedIds: string[] = [];
    for (const mod of group.strong) providedIds.push(...providedIdsOf(mod));
    const groupPaths = new Set(group.strong.map((mod) => mod.path));
    return findReverseDependents(mods, providedIds, { excludePaths: groupPaths, requireEnabled: true });
  }, [group, mods]);

  const dependentMods = dependents.map((match) => match.name).filter(Boolean);
  const dependencyDetails: DependencyBreakDetail[] = dependents.flatMap((match) =>
    match.requirements.map((requirement) => ({
      name: match.name,
      modId: requirement.modId,
      version: requirement.version,
    })),
  );

  const versionChangeIsRisky = (version: ModVersion) => {
    if (!group || dependentMods.length === 0) return false;
    const candidate = version?.version;
    const installed = group.primary?.version;
    return isVersionDowngrade(candidate, installed) || hasMajorVersionChange(candidate, installed);
  };

  const checkPreflight = async (version: ModVersion) => {
    if (!group) return null;
    try {
      return await api.CheckDependencyFulfillment({
        projectId: group.primary.onlineProjectId || "",
        result: onlineProjectFromGroup(group),
        minecraftVersion,
        modLoader,
        versionId: version.id,
      });
    } catch {
      return null;
    }
  };

  const queueReplacement = async (version: ModVersion) => {
    if (!group || replacingVersionId) return;
    setReplacingVersionId(version.id);
    try {
      const project = onlineProjectFromGroup(group);
      const result = await api.QueueModDownload({
        projectId: group.primary.onlineProjectId || "",
        result: project,
        minecraftVersion,
        modLoader,
        versionId: version.id,
      });
      if (result?.skipped) {
        onNotify("download.errors.generic", "error");
        return;
      }
      onQueued();
      onOpenChange(false);
      onNotify("manage.version.queued", "success");
    } catch (error) {
      onError(error);
    } finally {
      setReplacingVersionId("");
    }
  };

  const proceed = async (version: ModVersion) => {
    if (versionChangeIsRisky(version)) {
      setPendingVersion(version);
      setDowngradeOpen(true);
      return;
    }
    await queueReplacement(version);
  };

  const selectVersion = async (version: ModVersion) => {
    if (!group || replacingVersionId) return;
    setReplacingVersionId(version.id);
    let analysis: DependencyPreflightResult | null = null;
    try {
      analysis = await checkPreflight(version);
    } finally {
      setReplacingVersionId("");
    }
    if (analysis && (analysis.missing > 0 || analysis.conflicts > 0)) {
      setPendingVersion(version);
      setPreflightResult(analysis);
      setPreflightOpen(true);
      return;
    }
    await proceed(version);
  };

  const pinVersion = async (version: ModVersion) => {
    if (!group || pinningVersionId) return;
    setPinningVersionId(version.id);
    try {
      const project = onlineProjectFromGroup(group);
      const pin = await api.PinModVersion({
        platform: project.platform,
        modId: project.projectId,
        versionId: version.id,
        minecraftVersion,
        modLoader,
      });
      setPinnedVersionId(pin?.versionId || "");
    } catch (error) {
      onError(error);
    } finally {
      setPinningVersionId("");
    }
  };

  const statusMeta = (entry: DependencyPreflightEntry) => {
    switch (entry.status) {
      case "missing":
        return {
          icon: <Download className="size-4" />,
          color: "text-sky-600 dark:text-sky-400",
          label: t("download.preflight.status.missing"),
        };
      case "conflict":
        return {
          icon: <ServerCrash className="size-4" />,
          color: "text-destructive",
          label: t("download.preflight.status.conflict"),
        };
      default:
        return {
          icon: <CheckCircle2 className="size-4" />,
          color: "text-emerald-600 dark:text-emerald-400",
          label: t("download.preflight.status.fulfilled"),
        };
    }
  };

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-2xl gap-0 p-0" showCloseButton>
          <DialogHeader className="border-b px-4 py-3 pr-12">
            <DialogTitle>
              {group ? t("manage.version.title", { name: displayModName(group) }) : ""}
            </DialogTitle>
          </DialogHeader>
          <div className="min-h-48">
            <ModVersionList
              versions={versions}
              loading={loading}
              action="replace"
              installedVersionId={group?.primary?.onlineVersionId || ""}
              installedSha1={group?.primary?.sha1 || ""}
              installedVersion={group?.primary?.version || ""}
              dependencyBreakRisk={dependentMods.length > 0}
              dependencyBreakMods={dependentMods}
              dependencyBreakDetails={dependencyDetails}
              busyVersionId={replacingVersionId}
              pinnedVersionId={pinnedVersionId}
              pinBusyVersionId={pinningVersionId}
              showPinAction
              disableInstalledAction
              onSelect={(version) => void selectVersion(version)}
              onPin={(version) => void pinVersion(version)}
            />
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={preflightOpen} onOpenChange={setPreflightOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>{t("download.preflight.title")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-3 text-sm">
            <p>
              {t("download.preflight.summary", {
                missing: preflightResult?.missing || 0,
                conflicts: preflightResult?.conflicts || 0,
              })}
            </p>
            <div className="max-h-72 overflow-y-auto rounded-md border">
              {(preflightResult?.entries ?? []).map((entry, index) => (
                <div key={`${entry.projectId}-${index}`} className="flex items-center gap-2 border-b px-2 py-2 last:border-b-0">
                  <span className={statusMeta(entry).color}>{statusMeta(entry).icon}</span>
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">{entry.title || entry.projectId}</div>
                    <div className="truncate text-xs text-muted-foreground">
                      {entry.localName
                        ? t("download.preflight.localVersion", {
                            name: entry.localName,
                            version: entry.localVersion || t("download.preflight.unknownVersion"),
                          })
                        : entry.projectId}
                    </div>
                  </div>
                  <Badge variant="outline">{statusMeta(entry).label}</Badge>
                </div>
              ))}
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="ghost" onClick={() => setPreflightOpen(false)}>
              {t("download.preflight.cancel")}
            </Button>
            <Button
              type="button"
              onClick={() => {
                const version = pendingVersion;
                setPreflightOpen(false);
                setPreflightResult(null);
                setPendingVersion(null);
                if (version) void proceed(version);
              }}
            >
              {t("download.preflight.confirm")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog open={downgradeOpen} onOpenChange={setDowngradeOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("versions.confirmDowngradeTitle")}</AlertDialogTitle>
            <AlertDialogDescription>
              <span className="flex items-start gap-2 rounded-md border border-amber-500/40 bg-amber-500/10 p-2 text-amber-700 dark:text-amber-300">
                <AlertTriangle className="mt-0.5 size-4 shrink-0" />
                {t("versions.confirmDowngradeMessage", { mods: dependentMods.join(", ") })}
              </span>
              <span className="mt-2 block text-xs">
                {t("versions.dependencyBreakWarning")} ·{" "}
                {pendingVersion?.fileName || pendingVersion?.name || pendingVersion?.version || ""}
              </span>
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setPendingVersion(null)}>
              {t("manage.confirmDelete.cancel")}
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                const version = pendingVersion;
                setDowngradeOpen(false);
                setPendingVersion(null);
                if (version) void queueReplacement(version);
              }}
            >
              {replacingVersionId ? <Loader2 className="size-4 animate-spin" /> : null}
              {t("versions.confirmDowngradeConfirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}
