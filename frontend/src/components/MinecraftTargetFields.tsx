import { useId } from "react";
import { useTranslation } from "react-i18next";

import { Combobox } from "@/components/ui/combobox";
import { cn } from "@/lib/utils";

interface MinecraftTargetFieldsProps {
  versions: string[];
  modLoaders: string[];
  minecraftVersion: string;
  modLoader: string;
  onMinecraftVersionChange: (version: string) => void;
  onModLoaderChange: (loader: string) => void;
  minecraftVersionLabel?: string;
  modLoaderLabel?: string;
  stacked?: boolean;
  disabled?: boolean;
  className?: string;
}

export function MinecraftTargetFields({
  versions,
  modLoaders,
  minecraftVersion,
  modLoader,
  onMinecraftVersionChange,
  onModLoaderChange,
  minecraftVersionLabel,
  modLoaderLabel,
  stacked = false,
  disabled = false,
  className,
}: MinecraftTargetFieldsProps) {
  const { t } = useTranslation();
  const listId = useId();

  return (
    <div
      className={cn(
        "grid w-full min-w-0 gap-3",
        stacked ? "grid-cols-1" : "grid-cols-1 sm:grid-cols-2",
        className,
      )}
    >
      <div className="grid w-full min-w-0 grid-cols-1 gap-1.5">
        <label htmlFor={`${listId}-version`} className="text-xs font-medium text-muted-foreground">
          {minecraftVersionLabel || t("sidebar.minecraftVersion")}
        </label>
        <Combobox
          id={`${listId}-version`}
          value={minecraftVersion}
          onValueChange={onMinecraftVersionChange}
          options={versions}
          disabled={disabled}
          size="sm"
          searchable
          searchPlaceholder={t("common.search")}
          emptyText={t("common.noResults")}
          placeholder={t("sidebar.minecraftVersion")}
          aria-label={minecraftVersionLabel || t("sidebar.minecraftVersion")}
          className="w-full"
        />
      </div>
      <div className="grid w-full min-w-0 grid-cols-1 gap-1.5">
        <label className="text-xs font-medium text-muted-foreground">
          {modLoaderLabel || t("sidebar.modLoader")}
        </label>
        <Combobox
          value={modLoader}
          onValueChange={onModLoaderChange}
          options={modLoaders}
          disabled={disabled}
          size="sm"
          placeholder={t("sidebar.modLoader")}
          aria-label={modLoaderLabel || t("sidebar.modLoader")}
          className="w-full"
        />
      </div>
    </div>
  );
}
