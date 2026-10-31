import {
  ArrowLeftRight,
  BadgeCheck,
  CircleAlert,
  FlaskConical,
  Loader2,
  Pin,
  PinOff,
  TestTubeDiagonal,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { hasMajorVersionChange, isVersionDowngrade } from "@/utils/versionCompare";
import type { ModVersion } from "@/types";

export interface DependencyBreakDetail {
  name: string;
  modId: string;
  version?: string;
}

interface ModVersionListProps {
  versions?: ModVersion[];
  loading?: boolean;
  action?: "pin" | "replace";
  pinnedVersionId?: string;
  busyVersionId?: string;
  installedVersionId?: string;
  installedSha1?: string;
  installedVersion?: string;
  dependencyBreakRisk?: boolean;
  dependencyBreakMods?: string[];
  dependencyBreakDetails?: DependencyBreakDetail[];
  disableInstalledAction?: boolean;
  showPinAction?: boolean;
  pinBusyVersionId?: string;
  emptyTextKey?: string;
  onSelect: (version: ModVersion) => void;
  onPin?: (version: ModVersion) => void;
}

const normalized = (value?: string) => (value || "").trim().toLowerCase();

export function ModVersionList({
  versions = [],
  loading = false,
  action = "replace",
  pinnedVersionId = "",
  busyVersionId = "",
  installedVersionId = "",
  installedSha1 = "",
  installedVersion = "",
  dependencyBreakRisk = false,
  dependencyBreakMods = [],
  dependencyBreakDetails = [],
  disableInstalledAction = false,
  showPinAction = false,
  pinBusyVersionId = "",
  emptyTextKey = "download.noMatchingVersions",
  onSelect,
  onPin,
}: ModVersionListProps) {
  const { t } = useTranslation();

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (!versions.length) {
    return <div className="p-8 text-center text-sm text-muted-foreground">{t(emptyTextKey)}</div>;
  }

  const versionId = (version: ModVersion) => version.id || version.versionId;
  const isInstalled = (version: ModVersion) =>
    Boolean(
      (installedVersionId && versionId(version) === installedVersionId) ||
        (installedSha1 && normalized(version.sha1) === normalized(installedSha1)),
    );
  const isPinned = (version: ModVersion) =>
    Boolean(pinnedVersionId && pinnedVersionId === versionId(version));
  const isRisky = (version: ModVersion) =>
    dependencyBreakRisk &&
    dependencyBreakMods.length > 0 &&
    (isVersionDowngrade(version.version, installedVersion) ||
      hasMajorVersionChange(version.version, installedVersion));

  const releaseMeta = (version: ModVersion) => {
    switch (normalized(version.releaseType)) {
      case "alpha":
        return {
          icon: <TestTubeDiagonal className="size-4 text-amber-500" />,
          label: t("versions.releaseType.alpha"),
          color: "text-amber-500",
        };
      case "beta":
        return {
          icon: <FlaskConical className="size-4 text-sky-500" />,
          label: t("versions.releaseType.beta"),
          color: "text-sky-500",
        };
      default:
        return {
          icon: <BadgeCheck className="size-4 text-emerald-500" />,
          label: t("versions.releaseType.stable"),
          color: "text-emerald-500",
        };
    }
  };

  const riskTooltip = (version: ModVersion) => {
    const lines = [version.fileName || version.name || version.version || versionId(version)];
    if (dependencyBreakDetails.length > 0) {
      for (const detail of dependencyBreakDetails) {
        lines.push(
          t("versions.dependencyBreakDetail", {
            mod: detail.name,
            dep: detail.modId,
            version: (detail.version || "").trim() || t("versions.dependencyVersionUnknown"),
          }),
        );
      }
    } else {
      lines.push(...dependencyBreakMods);
    }
    return lines.join("\n");
  };

  return (
    <div className="max-h-[calc(100vh-10rem)] overflow-y-auto px-4 pb-4">
      {versions.map((version) => {
        const id = versionId(version);
        const installed = isInstalled(version);
        const pinned = isPinned(version);
        const fileName = version.fileName || version.name || version.version || id;
        const subtitle = [version.version, version.name]
          .filter(
            (value, index, all) =>
              Boolean(value) && value !== fileName && all.indexOf(value) === index,
          )
          .join(" · ");
        const actionDisabled =
          (busyVersionId && busyVersionId !== id) || (disableInstalledAction && installed);
        const actionIcon =
          action === "pin" ? (
            pinned ? (
              <Pin className="size-4" />
            ) : (
              <PinOff className="size-4" />
            )
          ) : (
            <ArrowLeftRight className="size-4" />
          );
        const actionLabel =
          action === "pin"
            ? pinned
              ? t("versions.unpin")
              : t("versions.pin")
            : t("versions.replace");

        return (
          <div
            key={id}
            className={`flex items-center gap-3 border-b py-3 last:border-b-0 ${installed ? "bg-emerald-500/10" : ""}`}
          >
            <Tooltip>
              <TooltipTrigger asChild>
                <span aria-label={releaseMeta(version).label} className={releaseMeta(version).color}>
                  {releaseMeta(version).icon}
                </span>
              </TooltipTrigger>
              <TooltipContent>{releaseMeta(version).label}</TooltipContent>
            </Tooltip>
            <div className="min-w-0 flex-1">
              <div className="truncate text-sm font-medium">{fileName}</div>
              {subtitle ? (
                <div className="truncate text-xs text-muted-foreground">{subtitle}</div>
              ) : null}
            </div>
            {isRisky(version) ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Badge variant="outline" className="gap-1 border-amber-500/40 text-amber-600 dark:text-amber-400">
                    <CircleAlert className="size-3" />
                    <span className="hidden sm:inline">{t("versions.dependencyBreakWarning")}</span>
                  </Badge>
                </TooltipTrigger>
                <TooltipContent className="max-w-xs whitespace-pre-line">{riskTooltip(version)}</TooltipContent>
              </Tooltip>
            ) : null}
            {installed ? (
              <Badge variant="secondary" className="text-emerald-700 dark:text-emerald-300">
                {t("versions.installed")}
              </Badge>
            ) : null}
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  size="icon"
                  variant={action === "replace" ? "outline" : pinned ? "default" : "secondary"}
                  disabled={Boolean(actionDisabled)}
                  onClick={() => onSelect(version)}
                  aria-label={actionLabel}
                >
                  {busyVersionId === id ? <Loader2 className="size-4 animate-spin" /> : actionIcon}
                </Button>
              </TooltipTrigger>
              <TooltipContent>{actionLabel}</TooltipContent>
            </Tooltip>
            {showPinAction && action !== "pin" ? (
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    type="button"
                    size="icon"
                    variant={pinned ? "default" : "secondary"}
                    disabled={Boolean(pinBusyVersionId && pinBusyVersionId !== id)}
                    onClick={() => onPin?.(version)}
                    aria-label={pinned ? t("versions.unpin") : t("versions.pin")}
                  >
                    {pinBusyVersionId === id ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : pinned ? (
                      <Pin className="size-4" />
                    ) : (
                      <PinOff className="size-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent>{pinned ? t("versions.unpin") : t("versions.pin")}</TooltipContent>
              </Tooltip>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}
