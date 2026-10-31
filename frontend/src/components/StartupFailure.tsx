import { AlertTriangle, Power, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { api } from "@/services/app";
import type { StartupStatus } from "@/types";

interface StartupFailureProps {
  status: StartupStatus;
}

export function StartupFailure({ status }: StartupFailureProps) {
  const { t } = useTranslation();

  const stageKey = (() => {
    switch ((status.stage || "").trim().toLowerCase()) {
      case "config":
        return "startup.stageConfig";
      case "storage":
        return "startup.stageStorage";
      case "releases":
        return "startup.stageReleases";
      default:
        return "startup.stageUnknown";
    }
  })();

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <Card className="w-full max-w-xl">
        <CardHeader>
          <div className="mb-2 flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
            <AlertTriangle className="size-6" />
          </div>
          <CardTitle className="text-lg">{t("startup.title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <p className="text-muted-foreground">{t("startup.description")}</p>
          <div className="rounded-md border bg-muted/30 p-3">
            <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {t("startup.stageLabel")}
            </div>
            <div className="mt-1 font-medium">{t(stageKey)}</div>
            {status.error ? (
              <div className="mt-2 break-words font-mono text-xs text-destructive">{status.error}</div>
            ) : null}
          </div>
          {status.extensionServerError ? (
            <div className="rounded-md border border-amber-500/40 bg-amber-500/10 p-3 text-amber-700 dark:text-amber-300">
              {t("startup.extensionWarning", { message: status.extensionServerError })}
            </div>
          ) : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => window.location.reload()}>
              <RefreshCw className="size-4" />
              {t("startup.retry")}
            </Button>
            <Button type="button" variant="destructive" onClick={() => void api.Quit()}>
              <Power className="size-4" />
              {t("startup.quit")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
