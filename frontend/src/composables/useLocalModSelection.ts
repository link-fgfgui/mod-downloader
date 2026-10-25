// Local mod selection/batch operations shared by the Manage list and tree.
// All destructive flows keep their confirmation dialogs; this composable only
// owns the operation state and the action definitions.

import { computed, ref } from "vue";

import { AnalyzeLocalModDisableImpact, ApplyLocalModBatchOperation, RestoreCachedDependency } from "../../wailsjs/go/main/App";

interface LocalModSelectionOptions {
    minecraftStore: any;
    t: (key: string, params?: Record<string, unknown>) => string;
    onError: (error: unknown) => void;
    onClearError: () => void;
    onNotify: (key: string, color?: string, params?: Record<string, unknown>) => void;
    onReplacementQueued: () => void;
}

export function useLocalModSelection(options: LocalModSelectionOptions) {
    const batchOperation = ref("");
    const pendingTogglePath = ref("");
    const deleteDialog = ref(false);
    const disableImpactDialog = ref(false);
    const disableImpacts = ref<any[]>([]);
    const restoringDependencyId = ref("");
    const restoredDependencies = ref<Record<string, boolean>>({});
    const pendingDeleteGroups = ref<any[]>([]);
    const pendingDeleteCount = ref(0);
    let pendingDeleteClearSelection: (() => void) | null = null;
    let pendingDisableOperation: any = null;

    const isBatchBusy = computed(() => batchOperation.value !== "");

    const selectedGroupPaths = (groups: any[]) => {
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

    const primaryBatchAction = (groups: any[]) =>
        groups.some((group) => group.primary?.enabled) ? "disable" : "enable";

    const copyModNames = async (groups: any[]) => {
        const names = groups.flatMap((group) => (group.strong || []).map((mod: any) => mod.name || mod.id)).filter(Boolean).join("\n");
        try { await navigator.clipboard.writeText(names); } catch {}
    };

    const copyModIds = async (groups: any[]) => {
        const ids = groups.flatMap((group) => (group.strong || []).map((mod: any) => mod.id)).filter(Boolean).join("\n");
        try { await navigator.clipboard.writeText(ids); } catch {}
    };

    const openDeleteDialog = (groups: any[], clearSelection?: (() => void) | null) => {
        pendingDeleteGroups.value = [...groups];
        pendingDeleteCount.value = selectedGroupPaths(groups).length;
        pendingDeleteClearSelection = clearSelection || null;
        deleteDialog.value = true;
    };

    const executeBatchOperation = async (groups: any[], action: string, clearSelection?: (() => void) | null) => {
        const paths = selectedGroupPaths(groups);
        if (!paths.length || batchOperation.value) return false;
        batchOperation.value = action;
        options.onClearError();
        try {
            const version = await ApplyLocalModBatchOperation({ paths, action });
            options.minecraftStore.applySelectedVersion(version);
            clearSelection?.();
            return true;
        } catch (error) {
            options.onError(error);
            await options.minecraftStore.refreshSelectedMods();
            return false;
        } finally {
            batchOperation.value = "";
        }
    };

    const applyBatchOperation = async (groups: any[], action: string, clearSelection?: (() => void) | null) => {
        const paths = selectedGroupPaths(groups);
        if (!paths.length || batchOperation.value) return false;
        if (action !== "disable" && action !== "invert") {
            return await executeBatchOperation(groups, action, clearSelection);
        }

        batchOperation.value = "analyze";
        options.onClearError();
        try {
            const result = await AnalyzeLocalModDisableImpact({ paths, action });
            const impacts = result?.impacts || [];
            if (!impacts.length) {
                batchOperation.value = "";
                return await executeBatchOperation(groups, action, clearSelection);
            }
            disableImpacts.value = impacts;
            restoredDependencies.value = {};
            pendingDisableOperation = { groups: [...groups], action, clearSelection };
            disableImpactDialog.value = true;
            return false;
        } catch (error) {
            options.onError(error);
            return false;
        } finally {
            if (batchOperation.value === "analyze") batchOperation.value = "";
        }
    };

    const manageSelectionActions = (groups: any[], clearSelection: () => void) => {
        const primaryAction = primaryBatchAction(groups);
        return [
            { label: options.t("manage.copyNames"), icon: "mdi-content-copy", run: () => copyModNames(groups) },
            { label: options.t("manage.copyIds"), icon: "mdi-identifier", run: () => copyModIds(groups) },
            {
                label: options.t(primaryAction === "disable" ? "manage.disableSelected" : "manage.enableSelected"),
                icon: primaryAction === "disable" ? "mdi-toggle-switch-off-outline" : "mdi-toggle-switch-outline",
                color: "primary",
                loading: batchOperation.value === primaryAction,
                run: () => applyBatchOperation(groups, primaryAction, clearSelection),
            },
            { label: options.t("manage.invertSelected"), icon: "mdi-swap-horizontal", color: "warning", loading: batchOperation.value === "invert", run: () => applyBatchOperation(groups, "invert", clearSelection) },
            { label: options.t("manage.deleteSelected"), icon: "mdi-delete", color: "error", run: () => openDeleteDialog(groups, clearSelection) },
            { label: options.t("manage.cleanup.deselectAll"), icon: "mdi-selection-off", color: "error", run: clearSelection },
        ];
    };

    const restoreDependency = async (impact: any, candidate: any) => {
        if (!candidate || restoringDependencyId.value) return;
        restoringDependencyId.value = impact.modId;
        try {
            const result = await RestoreCachedDependency({
                modId: impact.modId,
                platform: candidate.platform,
                projectId: candidate.projectId,
                versionId: candidate.versionId,
            });
            if (result?.queued) {
                restoredDependencies.value = { ...restoredDependencies.value, [impact.modId]: true };
                options.onReplacementQueued();
                options.onNotify("manage.disableImpact.queued", "success", { id: impact.modId });
            } else {
                options.onError(result?.reason || options.t("manage.disableImpact.restoreFailed"));
            }
        } catch (error) {
            options.onError(error);
        } finally {
            restoringDependencyId.value = "";
        }
    };

    const confirmDisableImpact = async () => {
        const pending = pendingDisableOperation;
        if (!pending || batchOperation.value) return;
        const ok = await executeBatchOperation(pending.groups, pending.action, pending.clearSelection);
        if (ok) disableImpactDialog.value = false;
    };

    const clearClosedDisableImpactDialog = () => {
        if (disableImpactDialog.value) return;
        disableImpacts.value = [];
        restoredDependencies.value = {};
        restoringDependencyId.value = "";
        pendingDisableOperation = null;
    };

    const toggleGroup = async (group: any) => {
        const path = (group.primary?.path || "").trim();
        if (!path || isBatchBusy.value) return;
        pendingTogglePath.value = path;
        try {
            await applyBatchOperation([group], group.primary.enabled ? "disable" : "enable");
        } finally {
            pendingTogglePath.value = "";
        }
    };

    const confirmDelete = async () => {
        const groups = pendingDeleteGroups.value;
        const clearSelection = pendingDeleteClearSelection;
        const ok = await applyBatchOperation(groups, "delete", clearSelection);
        if (!ok) return;
        deleteDialog.value = false;
        options.onNotify("manage.cleanup.deleteComplete", "success");
    };

    const clearClosedDeleteDialog = () => {
        if (deleteDialog.value) return;
        pendingDeleteGroups.value = [];
        pendingDeleteCount.value = 0;
        pendingDeleteClearSelection = null;
    };

    return {
        batchOperation,
        isBatchBusy,
        pendingTogglePath,
        deleteDialog,
        pendingDeleteCount,
        disableImpactDialog,
        disableImpacts,
        restoringDependencyId,
        restoredDependencies,
        manageSelectionActions,
        restoreDependency,
        confirmDisableImpact,
        clearClosedDisableImpactDialog,
        toggleGroup,
        openDeleteDialog,
        confirmDelete,
        clearClosedDeleteDialog,
        applyBatchOperation,
    };
}
