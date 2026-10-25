<template>
    <v-dialog v-model="versionDialog" max-width="680" @after-leave="clearClosedVersionDialog">
        <v-card>
            <v-toolbar density="compact" color="surface">
                <v-toolbar-title>{{ versionDialogTitle }}</v-toolbar-title>
                <v-btn icon="mdi-close" variant="text" @click="versionDialog = false"></v-btn>
            </v-toolbar>
            <v-divider></v-divider>
            <ModVersionList
                :versions="matchingVersions"
                :loading="isLoadingVersions"
                action="replace"
                :installed-version-id="selectedVersionGroup?.primary?.onlineVersionId || ''"
                :installed-sha1="selectedVersionGroup?.primary?.sha1 || ''"
                :installed-version="selectedVersionGroup?.primary?.version || ''"
                :dependency-break-risk="selectedGroupDependentMods.length > 0"
                :dependency-break-mods="selectedGroupDependentMods"
                :dependency-break-details="selectedGroupDependencyDetails"
                :busy-version-id="replacingVersionId"
                :pinned-version-id="pinnedVersionId"
                :pin-busy-version-id="pinningVersionId"
                show-pin-action
                disable-installed-action
                @select="replaceWithVersion"
                @pin="pinManageVersion"
            ></ModVersionList>
        </v-card>
    </v-dialog>

    <v-dialog v-model="replacementPreflightDialog" max-width="560" @after-leave="clearClosedReplacementPreflight">
        <v-card>
            <v-card-title>{{ $t("download.preflight.title") }}</v-card-title>
            <v-card-text>
                <div class="mb-3">
                    {{ $t("download.preflight.summary", { missing: replacementPreflight.missing, conflicts: replacementPreflight.conflicts }) }}
                </div>
                <v-list density="compact" class="replacement-preflight-list">
                    <v-list-item
                        v-for="(entry, index) in replacementPreflight.entries"
                        :key="`${entry.projectId}-${index}`"
                        :title="entry.title || entry.projectId"
                        :subtitle="entry.localName
                            ? $t('download.preflight.localVersion', { name: entry.localName, version: entry.localVersion || $t('download.preflight.unknownVersion') })
                            : entry.projectId"
                    >
                        <template #prepend>
                            <v-icon :color="preflightStatusMeta(entry).color" :icon="preflightStatusMeta(entry).icon"></v-icon>
                        </template>
                        <template #append>
                            <v-chip :color="preflightStatusMeta(entry).color" size="x-small" variant="tonal">
                                {{ $t(preflightStatusMeta(entry).label) }}
                            </v-chip>
                        </template>
                    </v-list-item>
                </v-list>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <v-btn variant="text" @click="replacementPreflightDialog = false">
                    {{ $t("download.preflight.cancel") }}
                </v-btn>
                <v-btn color="primary" variant="flat" @click="confirmReplacementPreflight">
                    {{ $t("download.preflight.confirm") }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>

    <v-dialog v-model="downgradeConfirmDialog" max-width="460" @after-leave="clearClosedDowngradeConfirm">
        <v-card>
            <v-card-title>{{ $t("versions.confirmDowngradeTitle") }}</v-card-title>
            <v-card-text>
                <v-alert type="warning" variant="tonal" density="compact" class="mb-3">
                    {{ $t("versions.confirmDowngradeMessage", { mods: selectedGroupDependentMods.join(", ") }) }}
                </v-alert>
                <div class="text-body-2">
                    {{ $t("versions.dependencyBreakWarning") }} · {{ pendingDowngradeVersionLabel }}
                </div>
            </v-card-text>
            <v-card-actions>
                <v-spacer></v-spacer>
                <v-btn variant="text" @click="downgradeConfirmDialog = false">
                    {{ $t("manage.confirmDelete.cancel") }}
                </v-btn>
                <v-btn color="warning" variant="tonal" :loading="Boolean(replacingVersionId)" @click="confirmDowngradeReplace">
                    {{ $t("versions.confirmDowngradeConfirm") }}
                </v-btn>
            </v-card-actions>
        </v-card>
    </v-dialog>
</template>

<script setup>
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";

import ModVersionList from "./ModVersionList.vue";
import { findReverseDependents, providedIdsOf } from "../composables/useModDependencyGraph";
import { hasMajorVersionChange, isVersionDowngrade } from "../utils/versionCompare";
import { useMinecraftStore } from "../stores/minecraft";
import {
    CheckDependencyFulfillment,
    GetPinnedModVersion,
    ListMatchingProjectVersions,
    PinModVersion,
    QueueModDownload,
} from "../../wailsjs/go/main/App";

const emit = defineEmits(["queued", "error", "notify"]);

const { t } = useI18n();
const minecraftStore = useMinecraftStore();

const versionDialog = ref(false);
const selectedVersionGroup = ref(null);
const matchingVersions = ref([]);
const isLoadingVersions = ref(false);
const replacingVersionId = ref("");
const pinnedVersionId = ref("");
const pinningVersionId = ref("");
const downgradeConfirmDialog = ref(false);
const pendingDowngradeVersion = ref(null);
const replacementPreflightDialog = ref(false);
const replacementPreflight = ref({ entries: [], missing: 0, conflicts: 0 });
const pendingPreflightVersion = ref(null);

const displayModName = (group) => {
    const mod = group?.primary || {};
    return mod.onlineName || mod.name || mod.id;
};

const modCategories = (group) => {
    const categories = group?.primary?.categories || [];
    const seen = new Set();
    return categories
        .map((category) => (category || "").trim())
        .filter((category) => {
            const key = category.toLowerCase();
            if (!key || seen.has(key)) return false;
            seen.add(key);
            return true;
        });
};

const canReplaceVersion = (group) => Boolean(
    group?.primary?.onlinePlatform && group?.primary?.onlineProjectId && minecraftStore.hasSelectedInstance,
);

const onlineProjectFromGroup = (group) => {
    const mod = group.primary;
    const platform = mod.onlinePlatform || "";
    const projectId = mod.onlineProjectId || "";
    return {
        id: `${platform.toLowerCase()}:${projectId}`,
        platform,
        projectId,
        slug: mod.onlineSlug || "",
        title: displayModName(group),
        icon: "mdi-package-variant",
        iconUrl: mod.iconUrl || "",
        description: mod.description || "",
        downloads: 0,
        categories: modCategories(group),
        updatedAt: 0,
        cachedAt: 0,
    };
};

const versionDialogTitle = computed(() =>
    selectedVersionGroup.value ? t("manage.version.title", { name: displayModName(selectedVersionGroup.value) }) : "",
);

const selectedGroupDependents = computed(() => {
    const group = selectedVersionGroup.value;
    if (!group) return [];
    const providedIDs = [];
    for (const mod of group.strong) providedIDs.push(...providedIdsOf(mod));
    const groupPaths = new Set(group.strong.map((mod) => mod.path));
    return findReverseDependents(minecraftStore.mods, providedIDs, {
        excludePaths: groupPaths,
        requireEnabled: true,
    });
});

const selectedGroupDependentMods = computed(() =>
    selectedGroupDependents.value.map((match) => match.name).filter(Boolean),
);

const selectedGroupDependencyDetails = computed(() => selectedGroupDependents.value.flatMap((match) =>
    match.requirements.map((requirement) => ({
        name: match.name,
        modId: requirement.modId,
        version: requirement.version,
    })),
));

const pendingDowngradeVersionLabel = computed(() => {
    const version = pendingDowngradeVersion.value;
    return version ? (version.fileName || version.name || version.version || version.id || "") : "";
});

const isVersionChangeRisk = (version) => {
    const group = selectedVersionGroup.value;
    if (!group || !selectedGroupDependentMods.value.length) return false;
    const candidate = version?.version;
    const installed = group.primary?.version;
    return isVersionDowngrade(candidate, installed) || hasMajorVersionChange(candidate, installed);
};

const preflightStatusMeta = (entry) => {
    switch (entry.status) {
        case "missing":
            return { icon: "mdi-download", color: "info", label: "download.preflight.status.missing" };
        case "conflict":
            return { icon: "mdi-alert-octagon-outline", color: "error", label: "download.preflight.status.conflict" };
        default:
            return { icon: "mdi-check-circle-outline", color: "success", label: "download.preflight.status.fulfilled" };
    }
};

const checkReplacementPreflight = async (version) => {
    const group = selectedVersionGroup.value;
    if (!group) return null;
    try {
        return await CheckDependencyFulfillment({
            projectId: group.primary.onlineProjectId,
            result: onlineProjectFromGroup(group),
            minecraftVersion: minecraftStore.selectedMinecraftVersion,
            modLoader: minecraftStore.selectedModLoader,
            versionId: version.id,
        });
    } catch {
        // A failed pre-flight must never block the replacement; the download
        // queue still performs its own dependency resolution.
        return null;
    }
};

const queueVersionReplacement = async (version) => {
    const group = selectedVersionGroup.value;
    if (!group || replacingVersionId.value) return;
    replacingVersionId.value = version.id;
    try {
        const project = onlineProjectFromGroup(group);
        const result = await QueueModDownload({
            projectId: group.primary.onlineProjectId,
            result: project,
            minecraftVersion: minecraftStore.selectedMinecraftVersion,
            modLoader: minecraftStore.selectedModLoader,
            versionId: version.id,
        });
        if (result?.skipped) {
            emit("notify", "download.errors.generic", "error");
            return;
        }
        emit("queued");
        versionDialog.value = false;
        emit("notify", "manage.version.queued", "success");
    } catch (error) {
        emit("error", error);
    } finally {
        replacingVersionId.value = "";
    }
};

const proceedWithVersionReplacement = async (version) => {
    if (isVersionChangeRisk(version)) {
        pendingDowngradeVersion.value = version;
        downgradeConfirmDialog.value = true;
        return;
    }
    await queueVersionReplacement(version);
};

const replaceWithVersion = async (version) => {
    const group = selectedVersionGroup.value;
    if (!group || replacingVersionId.value) return;
    replacingVersionId.value = version.id;
    let analysis = null;
    try {
        analysis = await checkReplacementPreflight(version);
    } finally {
        replacingVersionId.value = "";
    }
    if (analysis && (analysis.missing > 0 || analysis.conflicts > 0)) {
        pendingPreflightVersion.value = version;
        replacementPreflight.value = {
            entries: analysis.entries || [],
            missing: analysis.missing,
            conflicts: analysis.conflicts,
        };
        replacementPreflightDialog.value = true;
        return;
    }
    await proceedWithVersionReplacement(version);
};

const confirmReplacementPreflight = async () => {
    const version = pendingPreflightVersion.value;
    replacementPreflightDialog.value = false;
    if (version) {
        await proceedWithVersionReplacement(version);
    }
};

const clearClosedReplacementPreflight = () => {
    if (replacementPreflightDialog.value) return;
    pendingPreflightVersion.value = null;
    replacementPreflight.value = { entries: [], missing: 0, conflicts: 0 };
};

const pinManageVersion = async (version) => {
    const group = selectedVersionGroup.value;
    if (!group || pinningVersionId.value) return;
    const project = onlineProjectFromGroup(group);
    pinningVersionId.value = version.id;
    try {
        const pin = await PinModVersion({
            platform: project.platform,
            modId: project.projectId,
            versionId: version.id,
            minecraftVersion: minecraftStore.selectedMinecraftVersion,
            modLoader: minecraftStore.selectedModLoader,
        });
        pinnedVersionId.value = pin?.versionId || "";
    } finally {
        pinningVersionId.value = "";
    }
};

const confirmDowngradeReplace = async () => {
    const version = pendingDowngradeVersion.value;
    downgradeConfirmDialog.value = false;
    if (version) {
        await queueVersionReplacement(version);
    }
};

const clearClosedDowngradeConfirm = () => {
    if (downgradeConfirmDialog.value) return;
    pendingDowngradeVersion.value = null;
};

const clearClosedVersionDialog = () => {
    if (versionDialog.value) return;
    selectedVersionGroup.value = null;
    matchingVersions.value = [];
    pinnedVersionId.value = "";
};

const open = async (group) => {
    if (!canReplaceVersion(group)) return;
    selectedVersionGroup.value = group;
    matchingVersions.value = [];
    replacingVersionId.value = "";
    pinnedVersionId.value = "";
    pinningVersionId.value = "";
    versionDialog.value = true;
    isLoadingVersions.value = true;
    try {
        const project = onlineProjectFromGroup(group);
        const [versions, pin] = await Promise.all([
            ListMatchingProjectVersions(project, minecraftStore.selectedMinecraftVersion, minecraftStore.selectedModLoader),
            GetPinnedModVersion(project.platform, project.projectId, minecraftStore.selectedMinecraftVersion, minecraftStore.selectedModLoader),
        ]);
        matchingVersions.value = versions || [];
        pinnedVersionId.value = pin?.versionId || "";
    } catch (error) {
        emit("error", error);
    } finally {
        isLoadingVersions.value = false;
    }
};

defineExpose({ open });
</script>

<style scoped>
.replacement-preflight-list {
    max-height: min(320px, 52vh);
    overflow-y: auto;
}
</style>
