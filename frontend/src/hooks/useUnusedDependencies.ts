// Passive unused-dependency scan. The scan never deletes anything; it only
// flags candidate groups so the normal list selection/delete flow can be used.

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { api } from "@/services/app";
import { useSettingsStore } from "@/stores/settings";
import type { ModInfo, UnusedDependencyCandidate } from "@/types";
import type { ModGroup } from "@/lib/modGroups";

interface UnusedDependenciesOptions {
  mods: ModInfo[];
  hasSelectedInstance: boolean;
  sortMode: string;
  setSortMode: (mode: string) => void;
  onError: (error: unknown) => void;
  t: (key: string, options?: Record<string, unknown>) => string;
}

export function useUnusedDependencies(options: UnusedDependenciesOptions) {
  const [isScanning, setIsScanning] = useState(false);
  const [candidates, setCandidates] = useState<UnusedDependencyCandidate[]>([]);
  const pending = useRef(false);
  const scanning = useRef(false);
  const enabledJij = useSettingsStore((state) => state.settings?.enableJijDetection !== false);
  const autoScan = useSettingsStore((state) => state.settings?.autoScanUnusedDependencies === true);

  const onErrorRef = useRef(options.onError);
  const tRef = useRef(options.t);
  const setSortModeRef = useRef(options.setSortMode);
  onErrorRef.current = options.onError;
  tRef.current = options.t;
  setSortModeRef.current = options.setSortMode;

  const byPath = useMemo(() => {
    const map = new Map<string, UnusedDependencyCandidate>();
    for (const candidate of candidates) {
      const path = (candidate?.path || "").trim();
      if (path) map.set(path, candidate);
    }
    return map;
  }, [candidates]);

  const byModId = useMemo(() => {
    const map = new Map<string, UnusedDependencyCandidate>();
    for (const candidate of candidates) {
      for (const id of candidate?.modIds || []) {
        const key = (id || "").trim().toLowerCase();
        if (key) map.set(key, candidate);
      }
    }
    return map;
  }, [candidates]);

  const candidateFor = useCallback(
    (group: ModGroup): UnusedDependencyCandidate | null => {
      for (const mod of group.strong || []) {
        const path = (mod.path || "").trim();
        if (path && byPath.has(path)) return byPath.get(path)!;
      }
      for (const mod of group.strong || []) {
        const id = (mod.id || "").trim().toLowerCase();
        if (id && byModId.has(id)) return byModId.get(id)!;
      }
      return null;
    },
    [byModId, byPath],
  );

  const evidenceText = useCallback((candidate: UnusedDependencyCandidate) => {
    const evidence = candidate?.evidence || [];
    if (evidence.includes("online-library") && evidence.includes("required-by-excluded")) {
      return tRef.current("manage.cleanup.evidence.onlineAndDeleted");
    }
    if (evidence.includes("required-by-excluded")) {
      return tRef.current("manage.cleanup.evidence.deleted");
    }
    return tRef.current("manage.cleanup.evidence.online");
  }, []);

  const tooltipFor = useCallback(
    (group: ModGroup) => {
      const candidate = candidateFor(group);
      if (!candidate) return "";
      const ids = (candidate.modIds || []).join(", ");
      return [tRef.current("manage.unused.hint"), ids, evidenceText(candidate)].filter(Boolean).join("\n");
    },
    [candidateFor, evidenceText],
  );

  const load = useCallback(async () => {
    if (!options.hasSelectedInstance || !enabledJij) {
      setCandidates([]);
      return;
    }
    if (scanning.current) {
      pending.current = true;
      return;
    }
    scanning.current = true;
    setIsScanning(true);
    try {
      const result = await api.ScanUnusedDependencies({ excludedPaths: [] });
      setCandidates(result?.candidates ?? []);
    } catch (error) {
      setCandidates([]);
      onErrorRef.current(error);
    } finally {
      scanning.current = false;
      setIsScanning(false);
      if (pending.current) {
        pending.current = false;
        void load();
      }
    }
  }, [enabledJij, options.hasSelectedInstance]);

  useEffect(() => {
    const shouldScan = enabledJij && (options.sortMode === "unused" || autoScan);
    if (shouldScan) void load();
    else setCandidates([]);
    // The mods array reference changes after a scan of the active instance.
  }, [autoScan, enabledJij, load, options.mods, options.sortMode]);

  useEffect(() => {
    if (!enabledJij && options.sortMode === "unused") setSortModeRef.current("default");
  }, [enabledJij, options.sortMode]);

  return {
    isScanning,
    candidates,
    candidateFor,
    tooltipFor,
    jijDetectionEnabled: enabledJij,
    reload: load,
  };
}
