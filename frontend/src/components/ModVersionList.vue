<template>
    <div v-if="loading" class="pa-8 text-center">
        <v-progress-circular color="primary" indeterminate></v-progress-circular>
    </div>
    <v-list v-else-if="versions.length" class="mod-version-list py-0" density="comfortable" lines="two">
        <v-list-item
            v-for="version in versions"
            :key="versionId(version)"
            :class="{ 'mod-version-list__installed': isInstalled(version) }"
            :title="versionFileName(version)"
            :subtitle="versionSubtitle(version)"
        >
            <template #prepend>
                <v-tooltip :text="$t(releaseTypeMeta(version).label)" location="top">
                    <template #activator="{ props: releaseTip }">
                        <v-icon
                            v-bind="releaseTip"
                            class="me-3"
                            :color="releaseTypeMeta(version).color"
                            :icon="releaseTypeMeta(version).icon"
                            :aria-label="$t(releaseTypeMeta(version).label)"
                        ></v-icon>
                    </template>
                </v-tooltip>
            </template>
            <template #append>
                <v-tooltip v-if="isRisky(version)" :text="riskTooltip(version)" location="top" max-width="320">
                    <template #activator="{ props: riskTip }">
                        <v-chip v-bind="riskTip" class="me-2" color="warning" size="x-small" variant="tonal"
                            prepend-icon="mdi-link-variant-off">
                            {{ $t("versions.dependencyBreakWarning") }}
                        </v-chip>
                    </template>
                </v-tooltip>
                <v-chip v-if="isInstalled(version)" class="me-2" color="success" size="x-small" variant="tonal">
                    {{ $t("versions.installed") }}
                </v-chip>
                <v-tooltip :text="$t(actionLabel(version))" location="top">
                    <template #activator="{ props: tip }">
                        <v-btn
                            v-bind="tip"
                            :color="actionColor(version)"
                            :icon="actionIcon(version)"
                            variant="tonal"
                            size="small"
                            :loading="busyVersionId === versionId(version)"
                            :disabled="isActionDisabled(version)"
                            @click="emit('select', version)"
                        ></v-btn>
                    </template>
                </v-tooltip>
                <v-tooltip v-if="showPinAction && action !== 'pin'" :text="$t(isPinned(version) ? 'versions.unpin' : 'versions.pin')" location="top">
                    <template #activator="{ props: pinTip }">
                        <v-btn v-bind="pinTip" class="ms-2" :color="isPinned(version) ? 'primary' : 'surface-variant'"
                            :icon="isPinned(version) ? 'mdi-pin' : 'mdi-pin-outline'" variant="tonal" size="small"
                            :loading="pinBusyVersionId === versionId(version)" :disabled="Boolean(pinBusyVersionId && pinBusyVersionId !== versionId(version))"
                            @click="emit('pin', version)"></v-btn>
                    </template>
                </v-tooltip>
            </template>
        </v-list-item>
    </v-list>
    <div v-else class="pa-8 text-center text-medium-emphasis">
        {{ $t(emptyTextKey) }}
    </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import type { models } from "../../wailsjs/go/models";
import { hasMajorVersionChange, isVersionDowngrade } from "../utils/versionCompare";

type DependencyBreakDetail = {
    name: string;
    modId: string;
    version?: string;
};

const props = withDefaults(defineProps<{
    versions?: models.ModVersion[];
    loading?: boolean;
    action?: "pin" | "replace";
    pinnedVersionId?: string;
    busyVersionId?: string;
    installedVersionId?: string;
    installedSha1?: string;
    installedVersion?: string;
    dependencyBreakRisk?: boolean;
    dependencyBreakMods?: string[];
    dependencyBreakDetails?: DependencyBreakDetail[];
    disableInstalledAction?: boolean;
    showPinAction?: boolean;
    pinBusyVersionId?: string;
    emptyTextKey?: string;
}>(), {
    versions: () => [],
    loading: false,
    action: "replace",
    pinnedVersionId: "",
    busyVersionId: "",
    installedVersionId: "",
    installedSha1: "",
    installedVersion: "",
    dependencyBreakRisk: false,
    dependencyBreakMods: () => [],
    dependencyBreakDetails: () => [],
    disableInstalledAction: false,
    showPinAction: false,
    pinBusyVersionId: "",
    emptyTextKey: "download.noMatchingVersions",
});

const emit = defineEmits<{
    select: [version: models.ModVersion];
    pin: [version: models.ModVersion];
}>();

const { t } = useI18n();

const normalized = (value?: string) => (value || "").trim().toLowerCase();
const versionId = (version: models.ModVersion) => version.id || version.versionId;

const versionFileName = (version: models.ModVersion) =>
    version.fileName || version.name || version.version || versionId(version);

const versionSubtitle = (version: models.ModVersion) => {
    const values = [version.version, version.name].filter((value, index, all) =>
        Boolean(value) && value !== versionFileName(version) && all.indexOf(value) === index,
    );
    return values.join(" · ");
};

const releaseTypeMeta = (version: models.ModVersion) => {
    switch (normalized(version.releaseType)) {
        case "alpha":
            return { icon: "mdi-alpha-a-circle-outline", color: "warning", label: "versions.releaseType.alpha" };
        case "beta":
            return { icon: "mdi-beta", color: "info", label: "versions.releaseType.beta" };
        default:
            return { icon: "mdi-check-circle-outline", color: "success", label: "versions.releaseType.stable" };
    }
};

const isInstalled = (version: models.ModVersion) => {
    return Boolean(
        (props.installedVersionId && versionId(version) === props.installedVersionId) ||
        (props.installedSha1 && normalized(version.sha1) === normalized(props.installedSha1)),
    );
};

const isPinned = (version: models.ModVersion) =>
    Boolean(props.pinnedVersionId && props.pinnedVersionId === versionId(version));

const actionIcon = (version: models.ModVersion) => {
    if (props.action === "pin") return isPinned(version) ? "mdi-pin" : "mdi-pin-outline";
    return "mdi-swap-horizontal";
};

const actionColor = (version: models.ModVersion) => {
    if (props.action === "pin" && isPinned(version)) return "primary";
    return props.action === "replace" ? "warning" : "surface-variant";
};

const actionLabel = (version: models.ModVersion) => {
    if (props.action === "pin") return isPinned(version) ? "versions.unpin" : "versions.pin";
    return "versions.replace";
};

const isActionDisabled = (version: models.ModVersion) =>
    Boolean(
        (props.busyVersionId && props.busyVersionId !== versionId(version)) ||
        (props.disableInstalledAction && isInstalled(version)),
    );

// A risky change breaks the dependency chain when another installed mod still
// requires this mod at the (higher) installed version. Compare the declared
// version field, never the file name: "sodium-fabric-0.5.8.jar" sorts above
// "0.5.11" segment-wise and would hide every real downgrade. Crossing a major
// version boundary is risky even when the candidate is newer.
const isRisky = (version: models.ModVersion) =>
    props.dependencyBreakRisk &&
    props.dependencyBreakMods.length > 0 &&
    (
        isVersionDowngrade(version.version, props.installedVersion) ||
        hasMajorVersionChange(version.version, props.installedVersion)
    );

const riskTooltip = (version: models.ModVersion) => {
    const lines = [versionFileName(version)];
    if (props.dependencyBreakDetails.length > 0) {
        for (const detail of props.dependencyBreakDetails) {
            lines.push(t("versions.dependencyBreakDetail", {
                mod: detail.name,
                dep: detail.modId,
                version: (detail.version || "").trim() || t("versions.dependencyVersionUnknown"),
            }));
        }
    } else {
        lines.push(...props.dependencyBreakMods);
    }
    return lines.join("\n");
};
</script>

<style scoped>
.mod-version-list {
    max-height: calc(100vh - 160px);
    overflow-y: auto;
}

.mod-version-list__installed {
    background: rgba(var(--v-theme-success), 0.1);
}
</style>
