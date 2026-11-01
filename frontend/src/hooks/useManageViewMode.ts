import { useCallback, useState } from "react";

export type ManageViewMode = "list" | "tree";

export const MANAGE_VIEW_MODE_STORAGE_KEY = "mod-downloader.manage.view-mode";

const defaultViewMode: ManageViewMode = "list";

function isManageViewMode(value: unknown): value is ManageViewMode {
  return value === "list" || value === "tree";
}

export function readStoredManageViewMode(): ManageViewMode {
  if (typeof window === "undefined") return defaultViewMode;
  try {
    const stored = window.localStorage.getItem(MANAGE_VIEW_MODE_STORAGE_KEY);
    return isManageViewMode(stored) ? stored : defaultViewMode;
  } catch {
    return defaultViewMode;
  }
}

/**
 * Persists the Manage page's list/tree choice across route changes and reloads.
 * This is a GUI-only presentation preference, so it stays out of core settings.
 */
export function useManageViewMode(): [ManageViewMode, (mode: ManageViewMode) => void] {
  const [viewMode, setViewModeState] = useState<ManageViewMode>(readStoredManageViewMode);

  const setViewMode = useCallback((mode: ManageViewMode) => {
    setViewModeState(mode);
    if (typeof window === "undefined") return;
    try {
      window.localStorage.setItem(MANAGE_VIEW_MODE_STORAGE_KEY, mode);
    } catch {
      // Storage may be blocked (private mode); keep the in-memory preference.
    }
  }, []);

  return [viewMode, setViewMode];
}
