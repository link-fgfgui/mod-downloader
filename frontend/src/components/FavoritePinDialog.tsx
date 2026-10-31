import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ModVersionList } from "@/components/ModVersionList";
import { api } from "@/services/app";
import { useFavoritesStore } from "@/stores/favorites";
import type { FavoriteModEntry, ModProject, ModVersion } from "@/types";

interface FavoritePinDialogProps {
  open: boolean;
  item: FavoriteModEntry | null;
  onOpenChange: (open: boolean) => void;
}

const displayName = (item: FavoriteModEntry) => item.title || item.slug || item.modId;

export function FavoritePinDialog({ open, item, onOpenChange }: FavoritePinDialogProps) {
  const { t } = useTranslation();
  const minecraftVersion = useFavoritesStore((state) => state.displayMinecraftVersion);
  const modLoader = useFavoritesStore((state) => state.displayModLoader);
  const [versions, setVersions] = useState<ModVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [pinnedVersionId, setPinnedVersionId] = useState("");
  const [busyVersionId, setBusyVersionId] = useState("");

  useEffect(() => {
    if (!open || !item?.platform || !item?.modId) return;
    let cancelled = false;
    setVersions([]);
    setPinnedVersionId("");
    setLoading(true);
    const project: ModProject = {
      id: `${item.platform}:${item.modId}`,
      platform: item.platform,
      projectId: item.modId,
      slug: item.slug || "",
      title: displayName(item),
      icon: "",
      iconUrl: item.iconUrl || "",
      description: item.description || "",
      downloads: 0,
      categories: item.categories || [],
      updatedAt: 0,
      cachedAt: 0,
    };
    void Promise.all([
      api.ListMatchingProjectVersions(project, minecraftVersion, modLoader),
      api.GetPinnedModVersion(item.platform, item.modId, minecraftVersion, modLoader),
    ])
      .then(([matched, pin]) => {
        if (cancelled) return;
        setVersions(matched ?? []);
        setPinnedVersionId(pin?.versionId || "");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [item, minecraftVersion, modLoader, open]);

  const pinVersion = async (version: ModVersion) => {
    if (!item || busyVersionId) return;
    setBusyVersionId(version.id);
    try {
      const pin = await api.PinModVersion({
        platform: item.platform,
        modId: item.modId,
        versionId: version.id,
        minecraftVersion,
        modLoader,
      });
      setPinnedVersionId(pin?.versionId || "");
    } finally {
      setBusyVersionId("");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl gap-0 p-0">
        <DialogHeader className="border-b px-4 py-3">
          <DialogTitle>{item ? displayName(item) : ""}</DialogTitle>
        </DialogHeader>
        <div className="min-h-48">
          <ModVersionList
            versions={versions}
            loading={loading}
            action="pin"
            pinnedVersionId={pinnedVersionId}
            busyVersionId={busyVersionId}
            emptyTextKey="download.noMatchingVersions"
            onSelect={(version) => void pinVersion(version)}
          />
        </div>
        <div className="sr-only">{t("favorites.actions.pin")}</div>
      </DialogContent>
    </Dialog>
  );
}
