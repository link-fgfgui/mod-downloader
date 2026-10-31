import { useCallback, useMemo, useRef, useState } from "react";

import { api } from "@/services/app";
import type { ModGroup } from "@/lib/modGroups";
import type { LocalModDependencyImpact, ModInfo, VersionInfo } from "@/types";

export type ManageSelectionActionIcon =
  | "copy"
  | "identifier"
  | "enable"
  | "disable"
  | "invert"
  | "delete"
  | "deselect";

export interface ManageSelectionAction {
  key: string;
  label: string;
  icon: ManageSelectionActionIcon;
  color?: "primary" | "warning" | "destructive";
  loading?: boolean;
  run: () => void;
}

type RestoredDependencies = Record<string, boolean>;

interface UseLocalModSelectionOptions {
  minecraftMods: ModInfo[];
  t: (key: string, options?: Record<string, unknown>) => string;
  applySelectedVersion: (version: VersionInfo) => void;
  refreshSelectedMods: () => Promise<unknown>;
  onError: (error: unknown) => void;
  onClearError: () => void;
  onNotify: (key: string, color?: "success" | "warning" | "error", params?: Record<string, unknown>) => void;
  onReplacementQueued: () => void;
}

const selectedGroupPaths = (groups: ModGroup[]) => {
  const seen = new Set<string>();
  const paths: string[] = [];
  for (const group of groups) {
    const path = (group.primary?.path || "").trim();
    if (!path || seen.has(path)) continue;
    seen.add(path);
    paths.push(path);
  }
  return paths;
};

const primaryBatchAction = (groups: ModGroup[]) =>
  groups.some((group) => group.primary?.enabled) ? "disable" : "enable";

export function useLocalModSelection(options: UseLocalModSelectionOptions) {
  const [batchOperation, setBatchOperation] = useState("");
  const [pendingTogglePath, setPendingTogglePath] = useState("");
  const [deleteDialog, setDeleteDialog] = useState(false);
  const [disableImpactDialog, setDisableImpactDialog] = useState(false);
  const [disableImpacts, setDisableImpacts] = useState<LocalModDependencyImpact[]>([]);
  const [restoringDependencyId, setRestoringDependencyId] = useState("");
  const [restoredDependencies, setRestoredDependencies] = useState<RestoredDependencies>({});
  const [pendingDeleteCount, setPendingDeleteCount] = useState(0);
  const pendingDeleteGroups = useRef<ModGroup[]>([]);
  const pendingDeleteClearSelection = useRef<(() => void) | null>(null);
  const pendingDisableOperation = useRef<{
    groups: ModGroup[];
    action: string;
    clearSelection?: (() => void) | null;
  } | null>(null);

  const isBatchBusy = batchOperation !== "";

  const copyText = useCallback(async (text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      return false;
    }
  }, []);

  const copyModNames = useCallback(
    (groups: ModGroup[]) => {
      const names = groups
        .flatMap((group) => (group.strong || []).map((mod) => mod.name || mod.id))
        .filter(Boolean)
        .join("\n");
      void copyText(names).then((ok) => options.onNotify(ok ? "manage.copiedNames" : "manage.copyFailed", ok ? "success" : "error"));
    },
    [copyText, options],
  );

  const copyModIds = useCallback(
    (groups: ModGroup[]) => {
      const ids = groups
        .flatMap((group) => (group.strong || []).map((mod) => mod.id))
        .filter(Boolean)
        .join("\n");
      void copyText(ids).then((ok) => options.onNotify(ok ? "manage.copiedIds" : "manage.copyFailed", ok ? "success" : "error"));
    },
    [copyText, options],
  );

  const openDeleteDialog = useCallback((groups: ModGroup[], clearSelection?: (() => void) | null) => {
    pendingDeleteGroups.current = [...groups];
    setPendingDeleteCount(selectedGroupPaths(groups).length);
    pendingDeleteClearSelection.current = clearSelection ?? null;
    setDeleteDialog(true);
  }, []);

  const executeBatchOperation = useCallback(
    async (groups: ModGroup[], action: string, clearSelection?: (() => void) | null) => {
      const paths = selectedGroupPaths(groups);
      if (!paths.length || batchOperation) return false;
      setBatchOperation(action);
      options.onClearError();
      try {
        const version = await api.ApplyLocalModBatchOperation({ paths, action });
        options.applySelectedVersion(version);
        clearSelection?.();
        return true;
      } catch (error) {
        options.onError(error);
        await options.refreshSelectedMods();
        return false;
      } finally {
        setBatchOperation("");
      }
    },
    [batchOperation, options],
  );

  const applyBatchOperation = useCallback(
    async (groups: ModGroup[], action: string, clearSelection?: (() => void) | null) => {
      const paths = selectedGroupPaths(groups);
      if (!paths.length || batchOperation) return false;
      if (action !== "disable" && action !== "invert") {
        return executeBatchOperation(groups, action, clearSelection);
      }

      setBatchOperation("analyze");
      options.onClearError();
      try {
        const result = await api.AnalyzeLocalModDisableImpact({ paths, action });
        const impacts = result?.impacts ?? [];
        if (!impacts.length) {
          setBatchOperation("");
          return executeBatchOperation(groups, action, clearSelection);
        }
        setDisableImpacts(impacts);
        setRestoredDependencies({});
        pendingDisableOperation.current = { groups: [...groups], action, clearSelection };
        setDisableImpactDialog(true);
        return false;
      } catch (error) {
        options.onError(error);
        return false;
      } finally {
        setBatchOperation((current) => (current === "analyze" ? "" : current));
      }
    },
    [batchOperation, executeBatchOperation, options],
  );

  const manageSelectionActions = useCallback(
    (groups: ModGroup[], clearSelection: () => void): ManageSelectionAction[] => {
      const primaryAction = primaryBatchAction(groups);
      return [
        {
          key: "names",
          label: options.t("manage.copyNames"),
          icon: "copy",
          run: () => copyModNames(groups),
        },
        {
          key: "ids",
          label: options.t("manage.copyIds"),
          icon: "identifier",
          run: () => copyModIds(groups),
        },
        {
          key: "primary",
          label: options.t(primaryAction === "disable" ? "manage.disableSelected" : "manage.enableSelected"),
          icon: primaryAction === "disable" ? "disable" : "enable",
          color: "primary",
          loading: batchOperation === primaryAction,
          run: () => void applyBatchOperation(groups, primaryAction, clearSelection),
        },
        {
          key: "invert",
          label: options.t("manage.invertSelected"),
          icon: "invert",
          color: "warning",
          loading: batchOperation === "invert",
          run: () => void applyBatchOperation(groups, "invert", clearSelection),
        },
        {
          key: "delete",
          label: options.t("manage.deleteSelected"),
          icon: "delete",
          color: "destructive",
          run: () => openDeleteDialog(groups, clearSelection),
        },
        {
          key: "deselect",
          label: options.t("manage.cleanup.deselectAll"),
          icon: "deselect",
          color: "destructive",
          run: clearSelection,
        },
      ];
    },
    [applyBatchOperation, batchOperation, copyModIds, copyModNames, openDeleteDialog, options],
  );

  const restoreDependency = useCallback(
    async (impact: LocalModDependencyImpact, candidate: NonNullable<LocalModDependencyImpact["candidates"]>[number]) => {
      if (!candidate || restoringDependencyId) return;
      setRestoringDependencyId(impact.modId);
      try {
        const result = await api.RestoreCachedDependency({
          modId: impact.modId,
          platform: candidate.platform,
          projectId: candidate.projectId,
          versionId: candidate.versionId,
        });
        if (result?.queued) {
          setRestoredDependencies((current) => ({ ...current, [impact.modId]: true }));
          options.onReplacementQueued();
          options.onNotify("manage.disableImpact.queued", "success", { id: impact.modId });
        } else {
          options.onError(result?.reason || options.t("manage.disableImpact.restoreFailed"));
        }
      } catch (error) {
        options.onError(error);
      } finally {
        setRestoringDependencyId("");
      }
    },
    [options, restoringDependencyId],
  );

  const confirmDisableImpact = useCallback(async () => {
    const pending = pendingDisableOperation.current;
    if (!pending || batchOperation) return;
    const ok = await executeBatchOperation(pending.groups, pending.action, pending.clearSelection);
    if (ok) setDisableImpactDialog(false);
  }, [batchOperation, executeBatchOperation]);

  const closeDisableImpactDialog = useCallback(() => {
    setDisableImpactDialog(false);
    setDisableImpacts([]);
    setRestoredDependencies({});
    setRestoringDependencyId("");
    pendingDisableOperation.current = null;
  }, []);

  const toggleGroup = useCallback(
    async (group: ModGroup) => {
      const path = (group.primary?.path || "").trim();
      if (!path || isBatchBusy) return;
      setPendingTogglePath(path);
      try {
        await applyBatchOperation([group], group.primary.enabled ? "disable" : "enable");
      } finally {
        setPendingTogglePath("");
      }
    },
    [applyBatchOperation, isBatchBusy],
  );

  const confirmDelete = useCallback(async () => {
    const groups = pendingDeleteGroups.current;
    const clearSelection = pendingDeleteClearSelection.current;
    const ok = await applyBatchOperation(groups, "delete", clearSelection);
    if (!ok) return;
    setDeleteDialog(false);
    options.onNotify("manage.cleanup.deleteComplete", "success");
  }, [applyBatchOperation, options]);

  const closeDeleteDialog = useCallback(() => {
    setDeleteDialog(false);
    pendingDeleteGroups.current = [];
    setPendingDeleteCount(0);
    pendingDeleteClearSelection.current = null;
  }, []);

  const clearRestored = useCallback(() => setRestoredDependencies({}), []);

  return useMemo(
    () => ({
      batchOperation,
      isBatchBusy,
      pendingTogglePath,
      deleteDialog,
      setDeleteDialog,
      pendingDeleteCount,
      disableImpactDialog,
      setDisableImpactDialog,
      disableImpacts,
      restoringDependencyId,
      restoredDependencies,
      manageSelectionActions,
      restoreDependency,
      confirmDisableImpact,
      closeDisableImpactDialog,
      toggleGroup,
      openDeleteDialog,
      confirmDelete,
      closeDeleteDialog,
      applyBatchOperation,
      clearRestored,
    }),
    [
      applyBatchOperation,
      batchOperation,
      clearRestored,
      closeDeleteDialog,
      closeDisableImpactDialog,
      confirmDelete,
      confirmDisableImpact,
      deleteDialog,
      disableImpactDialog,
      disableImpacts,
      isBatchBusy,
      manageSelectionActions,
      openDeleteDialog,
      pendingDeleteCount,
      pendingTogglePath,
      restoreDependency,
      restoredDependencies,
      restoringDependencyId,
      toggleGroup,
    ],
  );
}
