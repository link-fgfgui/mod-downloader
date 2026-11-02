import type { RefObject } from "react";
import {
  AlertTriangle,
  ChevronDown,
  ChevronRight,
  FolderOpen,
  ListTree,
  Package,
  Search,
  ToggleLeft,
  ToggleRight,
  Trash2,
} from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import type { FlattenedTreeRow, TreeRelationKeys } from "@/lib/modDependencyGraph";
import type { LocalModGroup } from "@/types";

export interface ManageModTreeViewProps {
  scrollRef: RefObject<HTMLDivElement>;
  missingCount: number;
  rows: FlattenedTreeRow[];
  relations: TreeRelationKeys;
  selectedKey: string;
  hasActiveFilter: boolean;
  groupForPath: (path: string) => LocalModGroup | null;
  batchBusy: boolean;
  canReplaceVersion: (group: LocalModGroup) => boolean;
  onSelectNode: (key: string) => void;
  onToggleNode: (key: string) => void;
  onOpenLocation: (group: LocalModGroup | null) => void;
  onToggleGroup: (group: LocalModGroup) => void;
  onOpenDeleteDialog: (groups: LocalModGroup[]) => void;
  onSearchVersion: (group: LocalModGroup) => void;
}

export function ManageModTreeView({
  scrollRef,
  missingCount,
  rows,
  relations,
  selectedKey,
  hasActiveFilter,
  groupForPath,
  batchBusy,
  canReplaceVersion,
  onSelectNode,
  onToggleNode,
  onOpenLocation,
  onToggleGroup,
  onOpenDeleteDialog,
  onSearchVersion,
}: ManageModTreeViewProps) {
  const { t } = useTranslation();

  const treeStatus = (node: {
    isMissing: boolean;
    isJij: boolean;
    enabled: boolean;
    isOptional: boolean;
  }) => {
    if (node.isMissing)
      return { label: t("manage.tree.status.missing"), color: "text-destructive", icon: AlertTriangle };
    if (node.isJij)
      return { label: t("manage.tree.status.jij"), color: "text-violet-600 dark:text-violet-400", icon: Package };
    if (node.isOptional)
      return { label: t("manage.tree.status.optional"), color: "text-amber-600 dark:text-amber-400", icon: AlertTriangle };
    if (!node.enabled)
      return { label: t("manage.tree.status.disabled"), color: "text-muted-foreground", icon: ToggleLeft };
    return { label: t("manage.tree.status.enabled"), color: "text-emerald-600 dark:text-emerald-400", icon: ToggleRight };
  };

  return (
    <div
      ref={scrollRef}
      data-tree-scroll
      className="relative mt-4 min-h-0 flex-1 overflow-y-auto overflow-x-hidden rounded-xl border bg-card/40 animate-in fade-in-0 slide-in-from-right-2 duration-md"
    >
      {missingCount ? (
        <div className="border-b border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-700 dark:text-amber-300">
          {t("manage.tree.missingDep", { n: missingCount })}
        </div>
      ) : null}
      {rows.length === 0 ? (
        <div className="flex min-h-64 flex-col items-center justify-center gap-2 text-muted-foreground">
          <ListTree className="size-10" />
          <span className="text-sm">{hasActiveFilter ? t("manage.search.empty") : t("manage.tree.empty")}</span>
        </div>
      ) : (
        rows.map((row) => {
          const node = row.node;
          const status = treeStatus(node);
          const StatusIcon = status.icon;
          const group = groupForPath(node.path ?? "");
          return (
            <div
              key={row.key}
              data-tree-key={row.key}
              data-tree-depth={row.depth}
              className={cn(
                "md-row-interactive flex min-h-14 items-center gap-2 border-b px-3 py-2 last:border-b-0",
                selectedKey === node.key
                  ? "bg-primary/10 hover:bg-primary/15"
                  : relations.dependencies.has(node.key)
                    ? "bg-sky-500/10 hover:bg-sky-500/15"
                    : relations.dependents.has(node.key)
                      ? "bg-amber-500/10 hover:bg-amber-500/15"
                      : "hover:bg-accent/50",
              )}
              style={{ paddingLeft: 12 + row.depth * 20 }}
            >
              {row.hasChildren ? (
                <Button
                  type="button"
                  size="icon-xs"
                  variant="ghost"
                  aria-expanded={row.expanded}
                  onClick={() => onToggleNode(node.key)}
                >
                  <ChevronRight className={cn("size-3.5 transition-transform duration-md", row.expanded && "rotate-90")} />
                </Button>
              ) : (
                <span className="inline-block w-6" />
              )}
              <button
                type="button"
                className="flex min-w-0 flex-1 items-center gap-2 text-left transition-opacity duration-fast active:opacity-70"
                onClick={() => {
                  onSelectNode(node.key);
                  if (row.hasChildren) onToggleNode(node.key);
                }}
              >
                {node.iconUrl ? (
                  <img src={node.iconUrl} alt="" className="size-6 rounded object-cover" />
                ) : (
                  <StatusIcon className={cn("size-5 shrink-0", status.color)} />
                )}
                <span className="min-w-0 flex-1">
                  <span className="flex items-center gap-1" data-tree-label>
                    <span className="truncate text-sm font-medium">{node.name}</span>
                    {node.version ? <span className="truncate text-xs text-muted-foreground">· {node.version}</span> : null}
                  </span>
                  <span className="block truncate text-xs text-muted-foreground" data-tree-meta>
                    {node.isMissing
                      ? t("manage.tree.missingNode")
                      : node.isJij
                        ? t("manage.tree.jijNode")
                        : [
                            node.children.length ? t("manage.tree.dependsOn", { n: node.children.length }) : "",
                            node.parents.length ? t("manage.tree.requiredBy", { n: node.parents.length }) : "",
                            node.isOptional ? t("manage.tree.optionalNode") : "",
                          ]
                            .filter(Boolean)
                            .join(" · ")}
                  </span>
                </span>
              </button>
              {node.isRoot ? <Badge variant="outline">{t("manage.tree.root")}</Badge> : null}
              <Badge variant="secondary" className={status.color} data-tree-status>
                {status.label}
              </Badge>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button type="button" size="icon-sm" variant="ghost" aria-label={t("manage.tree.actions.menu")}>
                    <ChevronDown className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem disabled={!group} onClick={() => void onOpenLocation(group)}>
                    <FolderOpen className="size-4" />
                    {t("manage.tree.actions.openLocation")}
                  </DropdownMenuItem>
                  <DropdownMenuItem disabled={!group || batchBusy} onClick={() => group && void onToggleGroup(group)}>
                    {group?.primary?.enabled ? <ToggleLeft className="size-4" /> : <ToggleRight className="size-4" />}
                    {t(group?.primary?.enabled ? "manage.tree.actions.disable" : "manage.tree.actions.enable")}
                  </DropdownMenuItem>
                  <DropdownMenuItem disabled={!group || batchBusy} onClick={() => group && onOpenDeleteDialog([group])}>
                    <Trash2 className="size-4" />
                    {t("manage.tree.actions.uninstall")}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem disabled={!group || !canReplaceVersion(group)} onClick={() => group && onSearchVersion(group)}>
                    <Search className="size-4" />
                    {t("manage.tree.actions.searchNewVersion")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          );
        })
      )}
    </div>
  );
}
