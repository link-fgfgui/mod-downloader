// Passive unused-dependency scan. The scan never deletes anything; it only
// flags candidate groups so the normal list selection/delete flow can be used.

import { computed, ref, watch, type Ref } from "vue";
import { useI18n } from "vue-i18n";

import { ScanUnusedDependencies } from "../../wailsjs/go/main/App";
import { useSettingsStore } from "../stores/settings";

interface UnusedDependenciesOptions {
    mods: Ref<any[]>;
    hasSelectedInstance: Ref<boolean>;
    sortMode: Ref<string>;
    onError: (error: unknown) => void;
}

export function useUnusedDependencies(options: UnusedDependenciesOptions) {
    const { t } = useI18n();
    const settingsStore = useSettingsStore();
    const isScanningUnusedDependencies = ref(false);
    const unusedCandidates = ref<any[]>([]);
    let pending = false;

    const jijDetectionEnabled = computed(() => settingsStore.view?.enableJijDetection !== false);

    const unusedCandidateByPath = computed(() => {
        const map = new Map<string, any>();
        for (const candidate of unusedCandidates.value) {
            const path = (candidate?.path || "").trim();
            if (path) map.set(path, candidate);
        }
        return map;
    });

    const unusedCandidateByModId = computed(() => {
        const map = new Map<string, any>();
        for (const candidate of unusedCandidates.value) {
            for (const id of candidate?.modIds || []) {
                const key = (id || "").trim().toLowerCase();
                if (key) map.set(key, candidate);
            }
        }
        return map;
    });

    const unusedCandidateFor = (group: any) => {
        for (const mod of group.strong || []) {
            const path = (mod.path || "").trim();
            if (path && unusedCandidateByPath.value.has(path)) return unusedCandidateByPath.value.get(path);
        }
        for (const mod of group.strong || []) {
            const id = (mod.id || "").trim().toLowerCase();
            if (id && unusedCandidateByModId.value.has(id)) return unusedCandidateByModId.value.get(id);
        }
        return null;
    };

    const unusedEvidenceText = (candidate: any) => {
        const evidence = candidate?.evidence || [];
        if (evidence.includes("online-library") && evidence.includes("required-by-excluded")) {
            return t("manage.cleanup.evidence.onlineAndDeleted");
        }
        if (evidence.includes("required-by-excluded")) {
            return t("manage.cleanup.evidence.deleted");
        }
        return t("manage.cleanup.evidence.online");
    };

    const unusedTooltip = (group: any) => {
        const candidate = unusedCandidateFor(group);
        if (!candidate) return "";
        const ids = (candidate.modIds || []).join(", ");
        return [t("manage.unused.hint"), ids, unusedEvidenceText(candidate)].filter(Boolean).join("\n");
    };

    const loadUnusedDependencies = async () => {
        if (!options.hasSelectedInstance.value || !jijDetectionEnabled.value) {
            unusedCandidates.value = [];
            return;
        }
        if (isScanningUnusedDependencies.value) {
            pending = true;
            return;
        }
        isScanningUnusedDependencies.value = true;
        try {
            const result = await ScanUnusedDependencies({ excludedPaths: [] });
            unusedCandidates.value = result?.candidates || [];
        } catch (error) {
            unusedCandidates.value = [];
            options.onError(error);
        } finally {
            isScanningUnusedDependencies.value = false;
            if (pending) {
                pending = false;
                void loadUnusedDependencies();
            }
        }
    };

    watch(
        () => [
            options.sortMode.value,
            options.mods.value,
            settingsStore.view?.enableJijDetection,
            settingsStore.view?.autoScanUnusedDependencies,
        ],
        () => {
            const shouldScan = jijDetectionEnabled.value
                && (options.sortMode.value === "unused" || settingsStore.view?.autoScanUnusedDependencies === true);
            if (shouldScan) {
                void loadUnusedDependencies();
            } else {
                unusedCandidates.value = [];
            }
        },
        { immediate: true },
    );

    watch(jijDetectionEnabled, (enabled) => {
        if (!enabled && options.sortMode.value === "unused") {
            options.sortMode.value = "default";
        }
    });

    return {
        isScanningUnusedDependencies,
        unusedCandidates,
        unusedCandidateFor,
        unusedTooltip,
        jijDetectionEnabled,
    };
}
