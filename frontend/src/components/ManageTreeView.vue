<template>
    <div ref="treeRoot" class="manage-tree">
        <v-alert v-if="dependencyGraph.missing.length" type="warning" variant="tonal" density="compact" class="mb-2">
            {{ $t("manage.tree.missingDep", { n: dependencyGraph.missing.length }) }}
        </v-alert>
        <div v-if="!treeRows.length" class="empty-state">
            <v-icon icon="mdi-file-tree" size="48"></v-icon>
            <div class="text-body-1 mt-3">
                {{ hasActiveFilter ? $t("manage.search.empty") : $t("manage.tree.empty") }}
            </div>
        </div>
        <v-virtual-scroll v-else class="manage-tree-list" :items="treeRows" :item-height="TREE_ROW_HEIGHT" item-key="key">
            <template #default="{ item: row }">
                <v-list-item
                    class="manage-tree-row"
                    density="compact"
                    lines="two"
                    :data-tree-key="encodeTreeRowKey(row.key)"
                    :class="treeRowClass(row.node)"
                    :style="{ paddingInlineStart: `${12 + row.depth * 20}px` }"
                    :aria-selected="selectedTreeKey === row.node.key"
                    @click="selectTreeRow(row)"
                    @contextmenu.prevent.stop="openTreeMenu($event, row.node)"
                >
                    <template #prepend>
                        <v-btn v-if="row.hasChildren"
                            class="manage-tree-toggle"
                            :class="{ 'manage-tree-toggle--expanded': row.expanded }"
                            icon="mdi-chevron-right"
                            variant="text" size="x-small" density="comfortable"
                            :aria-expanded="row.expanded"
                            @click.stop="toggleTreeNode(row)"></v-btn>
                        <span v-else class="manage-tree-spacer"></span>
                        <v-avatar v-if="row.node.iconUrl" class="ms-1 manage-tree-icon" rounded="sm" size="24">
                            <v-img :src="row.node.iconUrl" :alt="row.node.name"></v-img>
                        </v-avatar>
                        <v-icon v-else class="ms-1" :color="treeStatusColor(row.node)" :icon="treeNodeIcon(row.node)" size="20"></v-icon>
                    </template>
                    <v-list-item-title class="font-weight-medium">
                        {{ row.node.name }}
                        <span v-if="row.node.version" class="text-caption text-medium-emphasis ms-1">· {{ row.node.version }}</span>
                    </v-list-item-title>
                    <v-list-item-subtitle class="text-caption text-medium-emphasis">
                        {{ treeNodeSubtitle(row.node) }}
                    </v-list-item-subtitle>
                    <template #append>
                        <v-chip v-if="row.node.isRoot" class="me-2" color="primary" size="x-small" variant="text">
                            {{ $t("manage.tree.root") }}
                        </v-chip>
                        <v-chip :color="treeStatusColor(row.node)" size="x-small" variant="tonal">
                            {{ $t(treeStatusLabelKey(row.node)) }}
                        </v-chip>
                        <v-btn class="ms-1" icon="mdi-dots-vertical" variant="text" size="x-small" density="comfortable"
                            :aria-label="$t('manage.tree.actions.menu')"
                            @click.stop="openTreeMenu($event, row.node)"></v-btn>
                    </template>
                </v-list-item>
            </template>
        </v-virtual-scroll>

        <div ref="ghostLayer" class="manage-tree-ghost-layer" aria-hidden="true"></div>

        <v-menu v-model="treeMenuOpen" :target="treeMenuTarget" location="bottom start" min-width="220"
            @update:model-value="onTreeMenuOpenChange">
            <v-list density="compact">
                <v-list-item
                    v-if="treeMenuGroup"
                    prepend-icon="mdi-folder-open-outline"
                    :title="$t('manage.tree.actions.openLocation')"
                    @click="openTreeModLocation"
                ></v-list-item>
                <v-list-item
                    v-if="treeMenuGroup"
                    :prepend-icon="treeMenuGroup.primary.enabled ? 'mdi-toggle-switch-off-outline' : 'mdi-toggle-switch-outline'"
                    :title="$t(treeMenuGroup.primary.enabled ? 'manage.tree.actions.disable' : 'manage.tree.actions.enable')"
                    :disabled="busy"
                    @click="toggleTreeMod"
                ></v-list-item>
                <v-list-item
                    v-if="treeMenuGroup"
                    prepend-icon="mdi-delete-outline"
                    :title="$t('manage.tree.actions.uninstall')"
                    :disabled="busy"
                    @click="removeTreeMod"
                ></v-list-item>
                <v-list-item
                    v-if="treeMenuGroup"
                    prepend-icon="mdi-magnify"
                    :title="$t('manage.tree.actions.searchNewVersion')"
                    :disabled="!canReplaceVersion(treeMenuGroup)"
                    @click="searchTreeModNewVersion"
                ></v-list-item>
            </v-list>
        </v-menu>
    </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { matchesSearchExpression } from "../utils/searchExpression";
import { OpenLocalModLocation } from "../../wailsjs/go/main/App";
import {
    buildModDependencyGraph,
    collectMatchingTreeKeys,
    collectRelatedTreeKeys,
    flattenModDependencyTree,
} from "../composables/useModDependencyGraph";

const props = defineProps({
    mods: { type: Array, default: () => [] },
    groups: { type: Array, default: () => [] },
    appliedSearch: { type: String, default: "" },
    enabledFilter: { type: String, default: "all" },
    busy: { type: Boolean, default: false },
    hasSelectedInstance: { type: Boolean, default: false },
});

const emit = defineEmits(["toggle", "uninstall", "search-version", "error"]);

const TREE_ROW_HEIGHT = 56;
const TREE_MOTION_FALLBACK_MS = 220;
const TREE_ENTER_OFFSET_PX = 6;
const TREE_EASE = "cubic-bezier(0.4, 0, 0.2, 1)";

const { t } = useI18n();
const expandedTreeKeys = ref(new Set());
const selectedTreeKey = ref("");
const treeMenuOpen = ref(false);
const treeMenuTarget = ref([0, 0]);
const treeMenuNode = ref(null);
const treeRoot = ref(null);
const ghostLayer = ref(null);
const activeRowAnimations = new Set();
const activeGhosts = new Set();

const encodeTreeRowKey = (key) => encodeURIComponent(key);

const treeMotionAllowed = () => !window.matchMedia("(prefers-reduced-motion: reduce)").matches;

const cssDurationMs = (variable, fallback) => {
    const parsed = Number.parseFloat(getComputedStyle(document.documentElement).getPropertyValue(variable));
    return Number.isFinite(parsed) ? parsed : fallback;
};

const trackRowAnimation = (animation) => {
    activeRowAnimations.add(animation);
    animation.finished.catch(() => {}).finally(() => activeRowAnimations.delete(animation));
};

const clearTreeMotion = () => {
    for (const animation of activeRowAnimations) animation.cancel();
    activeRowAnimations.clear();
    for (const ghost of activeGhosts) ghost.remove();
    activeGhosts.clear();
};

const treeRowElements = () => [...(treeRoot.value?.querySelectorAll(".manage-tree-row[data-tree-key]") || [])];

const treeSurfaceColor = () => {
    for (let element = treeRoot.value; element; element = element.parentElement) {
        const color = getComputedStyle(element).backgroundColor;
        if (color && color !== "transparent" && color !== "rgba(0, 0, 0, 0)") return color;
    }
    return "rgb(var(--v-theme-background))";
};

const captureTreeRowTops = () => {
    const tops = new Map();
    for (const element of treeRowElements()) {
        tops.set(element.dataset.treeKey, element.getBoundingClientRect().top);
    }
    return tops;
};

// Collapsing rows vanish from the virtual list immediately, so clone the
// visible subtree into an opaque block and collapse its height. The rows below
// have already jumped up behind it, so the block hides the jump and then
// reveals the settled list without cross-fading two different texts.
const addCollapseGhosts = (row) => {
    if (!treeMotionAllowed() || !ghostLayer.value) return;
    const descendantPrefix = `${encodeTreeRowKey(row.key)}%00`;
    const rows = treeRowElements()
        .filter((element) => element.dataset.treeKey?.startsWith(descendantPrefix))
        .map((element) => ({ element, rect: element.getBoundingClientRect() }));
    if (!rows.length) return;

    const layerRect = ghostLayer.value.getBoundingClientRect();
    const top = Math.min(...rows.map((row) => row.rect.top));
    const bottom = Math.max(...rows.map((row) => row.rect.bottom));
    const surfaceColor = treeSurfaceColor();
    const block = document.createElement("div");
    block.className = "manage-tree-ghost-block";
    block.inert = true;
    block.style.position = "absolute";
    block.style.left = `${rows[0].rect.left - layerRect.left}px`;
    block.style.top = `${top - layerRect.top}px`;
    block.style.width = `${rows[0].rect.width}px`;
    block.style.height = `${bottom - top}px`;
    block.style.overflow = "hidden";
    block.style.pointerEvents = "none";
    block.style.backgroundColor = surfaceColor;
    ghostLayer.value.appendChild(block);
    activeGhosts.add(block);

    for (const { element, rect } of rows) {
        const ghost = element.cloneNode(true);
        ghost.removeAttribute("data-tree-key");
        ghost.removeAttribute("tabindex");
        ghost.classList.add("manage-tree-ghost");
        ghost.style.position = "absolute";
        ghost.style.left = "0";
        ghost.style.top = `${rect.top - top}px`;
        ghost.style.width = `${rect.width}px`;
        ghost.style.height = `${rect.height}px`;
        ghost.style.pointerEvents = "none";
        block.appendChild(ghost);
    }

    const duration = cssDurationMs("--md-transition-normal", TREE_MOTION_FALLBACK_MS);
    const animation = block.animate(
        [{ height: `${bottom - top}px` }, { height: "0px" }],
        { duration, easing: TREE_EASE },
    );
    animation.finished.catch(() => {}).finally(() => {
        block.remove();
        activeGhosts.delete(block);
    });
};

// New rows fade in behind the existing rows while those rows are FLIPed to
// their pushed-down positions. The existing rows get an opaque background and
// a higher stacking order for the duration, so their text never overlaps the
// fading rows underneath.
const animateTreeRows = (beforeTops, expanding) => {
    if (!expanding || !treeMotionAllowed()) return;
    const duration = cssDurationMs("--md-transition-normal", TREE_MOTION_FALLBACK_MS);
    const surfaceColor = treeSurfaceColor();
    for (const element of treeRowElements()) {
        const beforeTop = beforeTops.get(element.dataset.treeKey);
        if (beforeTop === undefined) {
            // Stay invisible while the rows above slide over this slot; the
            // fade only starts once most of the overlap is gone.
            trackRowAnimation(element.animate(
                [
                    { opacity: 0, transform: `translateY(-${TREE_ENTER_OFFSET_PX}px)`, offset: 0 },
                    { opacity: 0, transform: "translateY(0)", offset: 0.4 },
                    { opacity: 1, transform: "translateY(0)", offset: 1 },
                ],
                { duration, easing: TREE_EASE },
            ));
            continue;
        }
        const shift = beforeTop - element.getBoundingClientRect().top;
        if (Math.abs(shift) < 1) continue;
        element.style.backgroundColor = surfaceColor;
        element.style.position = "relative";
        element.style.zIndex = "1";
        const animation = element.animate(
            [{ transform: `translateY(${shift}px)` }, { transform: "translateY(0)" }],
            { duration, easing: TREE_EASE },
        );
        trackRowAnimation(animation);
        animation.finished.catch(() => {}).finally(() => {
            element.style.backgroundColor = "";
            element.style.position = "";
            element.style.zIndex = "";
        });
    }
};

const hasActiveFilter = computed(() => props.appliedSearch !== "" || props.enabledFilter !== "all");
const dependencyGraph = computed(() => buildModDependencyGraph(props.mods));
const treeNodeMatchesFilters = (node) => {
    if (node.isMissing) return false;
    const matchesEnabledState = props.enabledFilter === "all"
        || node.enabled === (props.enabledFilter === "enabled");
    return matchesEnabledState && matchesSearchExpression(node.searchText, props.appliedSearch);
};
const treeVisibleKeys = computed(() => {
    if (!hasActiveFilter.value) return null;
    return collectMatchingTreeKeys(dependencyGraph.value.roots, treeNodeMatchesFilters);
});
const treeRows = computed(() => flattenModDependencyTree(
    dependencyGraph.value.roots,
    expandedTreeKeys.value,
    { visibleKeys: treeVisibleKeys.value, forceExpand: props.appliedSearch !== "" },
));
const treeRelations = computed(() => collectRelatedTreeKeys(dependencyGraph.value.roots, selectedTreeKey.value));
const treeMenuGroup = computed(() => {
    const path = treeMenuNode.value?.path || "";
    if (!path) return null;
    return props.groups.find((group) => group.strong.some((mod) => (mod.path || "") === path)) || null;
});

watch(treeRows, (rows) => {
    if (selectedTreeKey.value && !rows.some((row) => row.node.key === selectedTreeKey.value)) {
        selectedTreeKey.value = "";
    }
});

const toggleTreeNode = (row) => {
    const expanding = !expandedTreeKeys.value.has(row.node.key);
    clearTreeMotion();
    const beforeTops = expanding ? captureTreeRowTops() : new Map();
    const next = new Set(expandedTreeKeys.value);
    if (expanding) {
        next.add(row.node.key);
    } else {
        next.delete(row.node.key);
        addCollapseGhosts(row);
    }
    expandedTreeKeys.value = next;
    void nextTick(() => animateTreeRows(beforeTops, expanding));
};

onBeforeUnmount(clearTreeMotion);

const selectTreeRow = (row) => {
    selectedTreeKey.value = row.node.key;
    if (row.hasChildren) toggleTreeNode(row);
};

const treeRowClass = (node) => ({
    "manage-tree-row--selected": selectedTreeKey.value === node.key,
    "manage-tree-row--dependency": treeRelations.value.dependencies.has(node.key),
    "manage-tree-row--dependent": treeRelations.value.dependents.has(node.key),
});

const treeStatusMeta = (node) => {
    if (node.isMissing) return { color: "error", icon: "mdi-alert-circle-outline", label: "manage.tree.status.missing" };
    if (node.isJij) return { color: "deep-purple", icon: "mdi-package-variant-closed", label: "manage.tree.status.jij" };
    if (!node.enabled) return { color: "grey", icon: "mdi-toggle-switch-off-outline", label: "manage.tree.status.disabled" };
    if (node.isOptional) return { color: "warning", icon: "mdi-link-variant", label: "manage.tree.status.optional" };
    return { color: "success", icon: "mdi-check-circle-outline", label: "manage.tree.status.enabled" };
};
const treeStatusColor = (node) => treeStatusMeta(node).color;
const treeStatusLabelKey = (node) => treeStatusMeta(node).label;
const treeNodeIcon = (node) => {
    if (node.isMissing) return "mdi-alert-circle-outline";
    if (node.isJij) return "mdi-package-variant-closed";
    if (node.isOptional) return "mdi-link-variant";
    if (!node.enabled) return "mdi-toggle-switch-off-outline";
    return node.isRoot ? "mdi-cube-outline" : "mdi-package-variant";
};

const treeNodeSubtitle = (node) => {
    if (node.isMissing) return t("manage.tree.missingNode");
    if (node.isJij) return t("manage.tree.jijNode");
    const parts = [];
    if (node.children.length) parts.push(t("manage.tree.dependsOn", { n: node.children.length }));
    if (node.parents.length) parts.push(t("manage.tree.requiredBy", { n: node.parents.length }));
    if (node.isOptional) parts.push(t("manage.tree.optionalNode"));
    return parts.join(" · ");
};

const canReplaceVersion = (group) => Boolean(
    group?.primary?.onlinePlatform && group?.primary?.onlineProjectId && props.hasSelectedInstance,
);

const openTreeMenu = (event, node) => {
    selectedTreeKey.value = node.key;
    treeMenuNode.value = node;
    const rect = event?.currentTarget?.getBoundingClientRect?.();
    const x = event?.clientX ?? (rect ? rect.right : 0);
    const y = event?.clientY ?? (rect ? rect.bottom : 0);
    treeMenuOpen.value = false;
    treeMenuTarget.value = [x, y];
    void nextTick(() => {
        treeMenuOpen.value = true;
    });
};

const onTreeMenuOpenChange = (open) => {
    if (open) return;
    // Keep the node until the click handler has run.
    treeMenuOpen.value = false;
};

const openTreeModLocation = async () => {
    const group = treeMenuGroup.value;
    treeMenuOpen.value = false;
    if (!group?.primary?.path) return;
    try {
        await OpenLocalModLocation(group.primary.path);
    } catch (error) {
        emit("error", error);
    }
};

const toggleTreeMod = () => {
    const group = treeMenuGroup.value;
    treeMenuOpen.value = false;
    if (!group) return;
    emit("toggle", group, group.primary.enabled ? "disable" : "enable");
};

const removeTreeMod = () => {
    const group = treeMenuGroup.value;
    treeMenuOpen.value = false;
    if (!group) return;
    emit("uninstall", group);
};

const searchTreeModNewVersion = () => {
    const group = treeMenuGroup.value;
    treeMenuOpen.value = false;
    if (!group) return;
    emit("search-version", group);
};
</script>

<style scoped>
.manage-tree {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-height: 0;
    position: relative;
}

.manage-tree-list {
    flex: 1 1 auto;
    min-height: 0;
    overflow-y: auto;
    padding-inline-end: 16px;
    position: relative;
    z-index: 1;
}

.manage-tree-toggle :deep(.v-icon) {
    transition: transform var(--md-transition-fast) var(--md-ease-out);
}

.manage-tree-toggle--expanded :deep(.v-icon) {
    transform: rotate(90deg);
}

.manage-tree-ghost-layer {
    inset: 0;
    overflow: hidden;
    pointer-events: none;
    position: absolute;
    z-index: 2;
}

.manage-tree-row {
    cursor: default;
    transition: background-color var(--md-transition-fast) var(--md-ease-out);
}

.manage-tree-row--selected {
    background: rgba(var(--v-theme-primary), 0.16);
}

.manage-tree-row--dependency {
    background: rgba(var(--v-theme-info), 0.08);
}

.manage-tree-row--dependent {
    background: rgba(var(--v-theme-warning), 0.08);
}

.manage-tree-row--selected.manage-tree-row--dependency,
.manage-tree-row--selected.manage-tree-row--dependent {
    background: rgba(var(--v-theme-primary), 0.16);
}

.manage-tree-icon {
    flex: 0 0 auto;
}

.manage-tree-spacer {
    display: inline-block;
    width: 28px;
}

.empty-state {
    align-items: center;
    color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 320px;
}
</style>
