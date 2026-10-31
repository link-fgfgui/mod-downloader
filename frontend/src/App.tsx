import { useEffect, useState } from "react";
import { HashRouter, Navigate, Route, Routes } from "react-router-dom";

import { AppShell } from "@/components/AppShell";
import { StartupFailure } from "@/components/StartupFailure";
import { api } from "@/services/app";
import type { StartupStatus } from "@/types";
import { HomePage } from "@/pages/Home";
import { DownloadPage } from "@/pages/Download";
import { ManagePage } from "@/pages/Manage";
import { FavoritesPage } from "@/pages/Favorites";
import { UnpinPage } from "@/pages/Unpin";
import { SettingsPage } from "@/pages/Settings";

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
    </HashRouter>
  );
}
