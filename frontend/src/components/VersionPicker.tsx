import { FolderOpen, Loader2, RefreshCw, Repeat2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Combobox } from "@/components/ui/combobox";
import { MinecraftTargetFields } from "@/components/MinecraftTargetFields";
import { cn } from "@/lib/utils";
import {
  modLoaderOptions,
  selectConnectorDestination,
  useMinecraftStore,
} from "@/stores/minecraft";

interface VersionPickerProps {
  expanded: boolean;
}

export function VersionPicker({ expanded }: VersionPickerProps) {
  const { t } = useTranslation();
  const versions = useMinecraftStore((state) => state.versions);
  const releases = useMinecraftStore((state) => state.releases);
  const selected = useMinecraftStore((state) => state.selected);
  const minecraftVersion = useMinecraftStore((state) => state.minecraftVersion);
  const modLoader = useMinecraftStore((state) => state.modLoader);
  const dir = useMinecraftStore((state) => state.dir);
  const loading = useMinecraftStore((state) => state.loading);
  const refreshing = useMinecraftStore((state) => state.refreshing);
  const connectorAvailable = useMinecraftStore((state) => Boolean(state.selected?.connectorAvailable));
  const connectorDestination = useMinecraftStore(selectConnectorDestination);
  const selectVersion = useMinecraftStore((state) => state.selectVersion);
  const refreshSelected = useMinecraftStore((state) => state.refreshSelected);
  const chooseDir = useMinecraftStore((state) => state.chooseDir);
  const toggleConnector = useMinecraftStore((state) => state.toggleConnector);
  const setMinecraftVersion = useMinecraftStore((state) => state.setMinecraftVersion);
  const setModLoader = useMinecraftStore((state) => state.setModLoader);

  const versionOptions = versions
    .map((version) => ({ value: version.name || version.id, label: version.name || version.id }))
    .filter((item) => item.value);
  const selectedValue = selected?.name || selected?.id || "";
  const busy = loading || refreshing;

  return (
    <div className={cn("relative w-full min-w-0", expanded ? "min-h-0" : "h-16")}>
      <div
        className={cn(
          "grid w-full min-w-0 overflow-hidden transition-[grid-template-rows,opacity] duration-md",
          expanded
            ? "grid-rows-[1fr] opacity-100"
            : "pointer-events-none grid-rows-[0fr] opacity-0",
        )}
        aria-hidden={!expanded}
      >
        <div className="min-h-0 overflow-hidden">
          <div className="grid w-full min-w-0 grid-cols-1 gap-2 border-t p-3">
            <div className="flex w-full min-w-0 items-center gap-1.5">
              <Combobox
                value={selectedValue}
                onValueChange={(value) => void selectVersion(value)}
                options={versionOptions}
                disabled={busy || versionOptions.length === 0}
                size="sm"
                searchable
                searchPlaceholder={t("common.search")}
                emptyText={t("common.noResults")}
                placeholder={t("sidebar.version")}
                aria-label={t("sidebar.version")}
                className="min-w-0 flex-1"
              />
              <Button
                type="button"
                size="icon-sm"
                variant="ghost"
                className="shrink-0"
                title={t("sidebar.refresh")}
                aria-label={t("sidebar.refresh")}
                disabled={busy}
                onClick={() => void refreshSelected()}
              >
                {refreshing ? <Loader2 className="size-4 animate-spin" /> : <RefreshCw className="size-4" />}
              </Button>
            </div>

            <MinecraftTargetFields
              versions={releases}
              modLoaders={modLoaderOptions}
              minecraftVersion={minecraftVersion}
              modLoader={modLoader}
              onMinecraftVersionChange={setMinecraftVersion}
              onModLoaderChange={setModLoader}
              stacked
              disabled={busy}
            />

            {connectorAvailable ? (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                className="w-full min-w-0"
                disabled={busy}
                onClick={() => void toggleConnector()}
              >
                <Repeat2 className="size-4 shrink-0" />
                <span className="truncate">{t("sidebar.connector.switchTo", { loader: connectorDestination })}</span>
              </Button>
            ) : null}

            <Button
              type="button"
              variant="outline"
              size="sm"
              className="w-full min-w-0 justify-start"
              disabled={busy}
              title={dir || t("sidebar.minecraftDir")}
              onClick={() => void chooseDir()}
            >
              <FolderOpen className="size-4 shrink-0" />
              <span className="min-w-0 flex-1 truncate text-left">{dir || t("sidebar.minecraftDir")}</span>
            </Button>
          </div>
        </div>
      </div>

      <div
        className={cn(
          "absolute inset-x-0 bottom-0 flex justify-center pb-4 transition-opacity duration-md",
          expanded ? "pointer-events-none opacity-0" : "opacity-100",
        )}
        aria-hidden={expanded}
      >
        <div className="flex size-10 items-center justify-center rounded-lg border bg-muted/30 text-muted-foreground">
          <Button
            type="button"
            size="icon"
            variant="ghost"
            className="size-10"
            title={dir || t("sidebar.minecraftDir")}
            aria-label={t("sidebar.minecraftDir")}
            disabled={busy}
            onClick={() => void chooseDir()}
          >
            {busy ? <Loader2 className="size-4 animate-spin" /> : <FolderOpen className="size-4" />}
          </Button>
        </div>
      </div>
    </div>
  );
}
