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
import type { models } from "../../wailsjs/go/models";

const props = withDefaults(defineProps<{
    versions?: models.ModVersion[];
    loading?: boolean;
    action?: "pin" | "replace";
    pinnedVersionId?: string;
    busyVersionId?: string;
    installedVersionId?: string;
    installedSha1?: string;
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
    disableInstalledAction: false,
    showPinAction: false,
    pinBusyVersionId: "",
    emptyTextKey: "download.noMatchingVersions",
});

const emit = defineEmits<{
    select: [version: models.ModVersion];
    pin: [version: models.ModVersion];
}>();

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
