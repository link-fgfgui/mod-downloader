import { useEffect, useRef, useState } from "react";
import { X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Outlet, useLocation, useNavigate } from "react-router-dom";

import { DownloadQueuePanel } from "@/components/DownloadQueuePanel";
import { Sidebar } from "@/components/Sidebar";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { animateRouteEnter } from "@/lib/animations";
import { resolveThemePreference } from "@/lib/theme";
import { useDownloadQueueStore } from "@/stores/downloadQueue";
import { useMinecraftStore } from "@/stores/minecraft";
import { useSettingsStore } from "@/stores/settings";

function isEditableTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return false;
  if (target.isContentEditable) return true;
  return Boolean(target.closest("input, textarea, select, [contenteditable='true']"));
}

export function AppShell({ extensionServerError }: { extensionServerError?: string }) {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const stageRef = useRef<HTMLDivElement | null>(null);
  const [extensionWarningDismissed, setExtensionWarningDismissed] = useState(false);
  const showExtensionWarning = Boolean(extensionServerError) && !extensionWarningDismissed;
  const themePreference = useSettingsStore((state) => state.settings?.theme ?? "dark");

  useEffect(() => {
    void useSettingsStore.getState().load().catch(() => undefined);
    void useMinecraftStore.getState().start().catch(() => undefined);
    void useDownloadQueueStore.getState().start().catch(() => undefined);
    return () => {
      useDownloadQueueStore.getState().stop();
    };
  }, []);

  useEffect(() => animateRouteEnter(stageRef.current), [location.pathname]);

  useEffect(() => {
    const onGlobalEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      if (event.defaultPrevented || isEditableTarget(event.target)) return;
      // Popovers own Escape while they are open. Checking here as well keeps
      // the global shortcut from navigating back if focus has drifted out of
      // the popup and its own handler runs after this one.
      const hasOverlay = document.querySelector(
        '[role="dialog"][data-state="open"], [role="alertdialog"][data-state="open"], [data-slot="combobox-panel"][data-state="open"], [data-radix-popper-content-wrapper]',
      );
      if (hasOverlay) return;
      if (location.pathname === "/") return;
      navigate(-1);
    };
    document.addEventListener("keydown", onGlobalEscape);
    return () => document.removeEventListener("keydown", onGlobalEscape);
  }, [location.pathname, navigate]);

  return (
    <TooltipProvider delayDuration={250}>
      <div className="flex h-screen overflow-hidden bg-background text-foreground">
        <Sidebar />
        <main className="flex min-w-0 flex-1 flex-col overflow-hidden">
          {showExtensionWarning ? (
            <div className="flex shrink-0 items-start gap-2 border-b border-amber-500/30 bg-amber-500/10 px-4 py-2 text-xs text-amber-700 dark:text-amber-300">
              <span className="min-w-0 flex-1">{t("startup.extensionWarning", { message: extensionServerError })}</span>
              <button
                type="button"
                className="shrink-0 rounded p-0.5 transition-[background-color,scale] duration-fast hover:bg-amber-500/20 active:scale-90"
                aria-label={t("download.queue.close")}
                onClick={() => setExtensionWarningDismissed(true)}
              >
                <X className="size-3.5" />
              </button>
            </div>
          ) : null}
          <div ref={stageRef} className="min-h-0 flex-1">
            <Outlet />
          </div>
        </main>
        <DownloadQueuePanel />
        <Toaster theme={resolveThemePreference(themePreference)} position="top-right" richColors closeButton />
      </div>
    </TooltipProvider>
  );
}
