import { Suspense, lazy, useEffect, useState } from "react";
import { HashRouter, Navigate, Route, Routes } from "react-router-dom";

import { AppShell } from "@/components/AppShell";
import { StartupFailure } from "@/components/StartupFailure";
import { api } from "@/services/app";
import type { StartupStatus } from "@/types";
import { HomePage } from "@/pages/Home";

// The landing route stays static so the first paint needs no extra round
// trip; the remaining pages load on demand to keep the entry chunk small.
// Named exports are reshaped for React.lazy.
const DownloadPage = lazy(() => import("@/pages/Download").then((m) => ({ default: m.DownloadPage })));
const ManagePage = lazy(() => import("@/pages/Manage").then((m) => ({ default: m.ManagePage })));
const FavoritesPage = lazy(() => import("@/pages/Favorites").then((m) => ({ default: m.FavoritesPage })));
const UnpinPage = lazy(() => import("@/pages/Unpin").then((m) => ({ default: m.UnpinPage })));
const SettingsPage = lazy(() => import("@/pages/Settings").then((m) => ({ default: m.SettingsPage })));

export default function App() {
  const [status, setStatus] = useState<StartupStatus | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .GetStartupStatus()
      .then((result) => {
        if (!cancelled) setStatus(result ?? { ready: false, stage: "unknown", error: "" });
      })
      .catch((error) => {
        if (!cancelled) {
          setStatus({
            ready: false,
            stage: "unknown",
            error: error instanceof Error ? error.message : String(error),
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  if (!status) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background text-sm text-muted-foreground">
        Loading...
      </div>
    );
  }

  if (!status.ready) {
    return <StartupFailure status={status} />;
  }

  return (
    <HashRouter>
      <Suspense fallback={null}>
        <Routes>
          <Route element={<AppShell extensionServerError={status.extensionServerError} />}>
            <Route path="/" element={<HomePage />} />
            <Route path="/download" element={<DownloadPage />} />
            <Route path="/manage" element={<ManagePage />} />
            <Route path="/favorites" element={<FavoritesPage />} />
            <Route path="/unpin" element={<UnpinPage />} />
            <Route path="/settings" element={<SettingsPage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </Suspense>
    </HashRouter>
  );
}
