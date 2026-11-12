import { Component, type ErrorInfo, type ReactNode } from "react";
import { AlertTriangle, RefreshCw } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface AppErrorBoundaryProps {
  children: ReactNode;
  /** Override the reload action in tests or custom hosts. */
  onReload?: () => void;
}

interface AppErrorBoundaryState {
  error: Error | null;
}

function ErrorFallback({ error, onReload }: { error: Error; onReload: () => void }) {
  const { t } = useTranslation();

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-6">
      <Card className="w-full max-w-xl">
        <CardHeader>
          <div className="mb-2 flex size-12 items-center justify-center rounded-full bg-destructive/10 text-destructive">
            <AlertTriangle className="size-6" />
          </div>
          <CardTitle className="text-lg">{t("errorBoundary.title")}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4 text-sm">
          <p className="text-muted-foreground">{t("errorBoundary.description")}</p>
          <div className="rounded-md border bg-muted/30 p-3">
            <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
              {t("errorBoundary.details")}
            </div>
            <div className="mt-1 break-words font-mono text-xs text-destructive">{error.message}</div>
          </div>
          <div className="flex justify-end">
            <Button type="button" onClick={onReload}>
              <RefreshCw className="size-4" />
              {t("errorBoundary.reload")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}

/**
 * Last-resort render boundary. React unmounts the whole tree when a render or
 * lifecycle error is uncaught, which leaves a blank WebView with no way back;
 * this boundary keeps a recovery screen on screen instead. Event-handler errors
 * are not caught by error boundaries and must be handled at their source.
 */
export class AppErrorBoundary extends Component<AppErrorBoundaryProps, AppErrorBoundaryState> {
  state: AppErrorBoundaryState = { error: null };

  static getDerivedStateFromError(error: unknown): AppErrorBoundaryState {
    return { error: error instanceof Error ? error : new Error(String(error)) };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    // There is no remote logger in the frontend; keep the component stack in
    // the WebView console so the failure can still be diagnosed.
    console.error("Uncaught render error", error, errorInfo.componentStack);
  }

  render() {
    if (this.state.error) {
      const reload = this.props.onReload ?? (() => window.location.reload());
      return <ErrorFallback error={this.state.error} onReload={reload} />;
    }
    return this.props.children;
  }
}
