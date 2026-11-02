import { Eye, Link2, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { favoriteModKey } from "@/stores/favorites";
import { cn } from "@/lib/utils";
import { displayName } from "@/components/favorites/shared";
import type { FavoriteModEntry } from "@/types";

export interface FavoriteModTableProps {
  items: FavoriteModEntry[];
  selectedKeys: Set<string>;
  exitingKeys: Set<string>;
  listRemoving: boolean;
  selectedListId: string;
  selectedCount: number;
  onToggleItem: (item: FavoriteModEntry, event: { ctrlKey: boolean; metaKey: boolean }) => void;
  onPinItem: (item: FavoriteModEntry) => void;
  onRemoveItem: (item: FavoriteModEntry) => void;
}

export function FavoriteModTable({
  items,
  selectedKeys,
  exitingKeys,
  listRemoving,
  selectedListId,
  selectedCount,
  onToggleItem,
  onPinItem,
  onRemoveItem,
}: FavoriteModTableProps) {
  const { t } = useTranslation();

  return (
    <div
      key={selectedListId}
      className={cn(
        "mt-3 min-h-0 flex-1 overflow-y-auto overflow-x-hidden pr-1 animate-in fade-in-0 slide-in-from-bottom-1 duration-md transition-[opacity,transform]",
        listRemoving && "pointer-events-none translate-y-1 opacity-0",
      )}
    >
      <div className={cn("grid gap-2", selectedCount > 0 && "pb-24")}>
        {items.map((item) => {
          const key = favoriteModKey(item);
          const selected = selectedKeys.has(key);
          return (
            <div
              key={item.id || key}
              className={cn(
                "md-row-interactive flex w-full min-w-0 cursor-pointer items-center gap-3 rounded-xl border bg-card p-3 shadow-sm",
                selected
                  ? "border-primary/50 bg-primary/10 shadow-sm hover:bg-primary/15"
                  : "hover:border-foreground/20 hover:bg-accent/60 hover:shadow-md",
                item.referenced && "border-l-4 border-l-cyan-500",
                exitingKeys.has(key) && "md-row-exiting",
              )}
              onClick={(event) => {
                if (!item.referenced) onToggleItem(item, event);
              }}
            >
              <div className={cn("flex size-5 shrink-0 items-center justify-center rounded border transition-colors duration-md", selected ? "border-primary bg-primary text-primary-foreground" : "border-input")}>
                {item.referenced ? (
                  <Link2 className="size-3 text-cyan-600 dark:text-cyan-400" />
                ) : selected ? (
                  <Eye className="size-3.5 animate-in fade-in-0 zoom-in-50 duration-md-fast" />
                ) : null}
              </div>
              <button
                type="button"
                className="size-12 shrink-0 overflow-hidden rounded-lg border bg-muted/40 transition-[scale] duration-fast active:scale-95"
                onClick={(event) => {
                  event.stopPropagation();
                  onPinItem(item);
                }}
                aria-label={t("favorites.actions.pin")}
              >
                {item.iconUrl ? (
                  <img src={item.iconUrl} alt="" className="size-full object-cover" />
                ) : (
                  <span className="flex size-full items-center justify-center text-lg font-bold text-muted-foreground">
                    {displayName(item).slice(0, 1).toUpperCase()}
                  </span>
                )}
              </button>
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  <span className="truncate text-sm font-semibold">{displayName(item)}</span>
                  {item.referenced ? <Badge variant="secondary">{item.sourceListName}</Badge> : null}
                </div>
                <div className="truncate text-xs text-muted-foreground">
                  {item.platform} / {item.modId}
                </div>
              </div>
              {!item.referenced ? (
                <Button
                  type="button"
                  size="icon-sm"
                  variant="secondary"
                  className="text-destructive"
                  aria-label={t("favorites.actions.removeSelected")}
                  onClick={(event) => {
                    event.stopPropagation();
                    onRemoveItem(item);
                  }}
                >
                  <Trash2 className="size-4" />
                </Button>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}
