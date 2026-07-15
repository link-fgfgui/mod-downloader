<template>
    <v-app>
        <SideBar />
        <v-main>
            <v-container fluid class="position-relative">
                <router-view v-slot="{ Component, route }">
                    <transition v-bind="routeTransitionProps">
                        <keep-alive>
                            <component :is="Component" :key="route.path" />
                        </keep-alive>
                    </transition>
                </router-view>
            </v-container>
            <v-overlay
                v-model="minecraftStore.isLoading"
                contained
                persistent
                scrim="transparent"
                class="main-loading-overlay"
            />
        </v-main>
        <transition v-bind="fabTransitionProps">
            <div v-if="queueSurfaceVisible" class="download-fab">
                <v-menu
                    v-model="downloadQueueOpen"
                    :close-on-content-click="false"
                    location="top end"
                    offset="12"
                >
                    <template #activator="{ props }">
                        <v-badge
                            :model-value="queueBadgeCount > 0"
                            :content="queueBadgeCount"
                            color="error"
                            floating
                        >
                            <v-btn
                                v-bind="props"
                                class="md-btn-press md-hover-scale"
                                color="primary"
                                icon
                                size="large"
                                :aria-label="$t('download.queue.title')"
                                @contextmenu.prevent.stop="clearQueueMessages"
                            >
                                <v-icon :icon="downloadQueueOpen ? 'mdi-chevron-down' : queueFabIcon"></v-icon>
                            </v-btn>
                        </v-badge>
                    </template>
                    <v-sheet class="download-queue-panel" elevation="8">
                        <div class="download-queue-header">
                            <div class="download-queue-heading">
                                <div>
                                    <div class="download-queue-title">{{ $t("download.queue.title") }}</div>
                                    <div class="download-queue-summary">
                                        {{ $t("download.queue.summary", { running: visibleDownloadQueue.running, pending: visibleDownloadQueue.pending }) }}
                                    </div>
                                </div>
                                <v-btn
                                    icon="mdi-close"
                                    variant="text"
                                    size="small"
                                    :aria-label="$t('download.queue.close')"
                                    @click="downloadQueueOpen = false"
                                />
                            </div>
                            <div v-if="visibleActiveDownloadCount > 0" class="download-queue-aggregate">
                                <div class="download-progress-labels">
                                    <span>{{ $t("download.queue.totalProgress") }}</span>
                                    <span v-if="visibleDownloadQueue.totalBytes > 0">
                                        {{ formatBytes(visibleDownloadQueue.bytesComplete) }} / {{ formatBytes(visibleDownloadQueue.totalBytes) }} · {{ totalProgressPercent }}%
                                    </span>
                                    <span v-else>{{ $t("download.queue.unknownSize") }}</span>
                                </div>
                                <v-progress-linear
                                    color="primary"
                                    height="6"
                                    :indeterminate="visibleDownloadQueue.totalBytes <= 0"
                                    :model-value="totalProgressPercent"
                                />
                                <div class="download-total-speed">
                                    <v-icon icon="mdi-speedometer" size="15" />
                                    <span>{{ $t("download.queue.totalSpeed") }}</span>
                                    <strong>{{ formatSpeed(visibleDownloadQueue.bytesPerSecond) }}</strong>
                                </div>
                            </div>
                        </div>
                        <v-divider />
                        <v-tabs v-model="downloadQueueTab" density="compact" class="download-queue-tabs">
                            <v-tab value="downloads">
                                {{ $t("download.queue.tabs.downloads") }}
                            </v-tab>
                            <v-tab value="optional">
                                {{ $t("download.queue.tabs.optional") }}
                            </v-tab>
                        </v-tabs>
                        <v-divider />
                        <v-window v-model="downloadQueueTab">
                            <v-window-item value="downloads">
                                <div class="download-queue-items">
                                    <div v-if="!downloadQueueItems.length" class="download-queue-empty">
                                        {{ $t("download.queue.emptyDownloads") }}
                                    </div>
                                    <div v-for="item in downloadQueueItems" :key="item.id" class="download-queue-item">
                                        <div class="download-queue-status" :class="`download-queue-status--${item.status}`">
                                            <v-icon :icon="queueStatusIcon(item.status)" size="20" />
                                        </div>
                                        <div class="download-queue-item-main">
                                            <div class="download-queue-item-title">{{ item.title || item.fileName || item.versionId }}</div>
                                            <div class="download-queue-item-meta">
                                                <span>{{ $t(`download.queue.status.${item.status}`) }}</span>
                                                <span v-if="queueItemMeta(item)"> · {{ queueItemMeta(item) }}</span>
                                            </div>
                                            <div v-if="item.reason" class="download-queue-item-reason">
                                                {{ item.reason }}
                                            </div>
                                            <div v-if="isActiveQueueItem(item)" class="download-item-progress">
                                                <v-progress-linear
                                                    color="primary"
                                                    height="4"
                                                    :indeterminate="item.totalBytes <= 0"
                                                    :model-value="itemProgressPercent(item)"
                                                />
                                                <div class="download-item-progress-meta">
                                                    <span v-if="item.totalBytes > 0">
                                                        {{ formatBytes(item.bytesComplete) }} / {{ formatBytes(item.totalBytes) }}
                                                    </span>
                                                    <span v-else>{{ $t("download.queue.unknownSize") }}</span>
                                                    <span v-if="item.totalBytes > 0">{{ itemProgressPercent(item) }}%</span>
                                                </div>
                                            </div>
                                        </div>
                                        <div class="download-queue-actions">
                                            <v-tooltip v-if="item.cancelable" :text="$t('download.queue.cancel')" location="top">
                                                <template #activator="{ props: tip }">
                                                    <v-btn
                                                        v-bind="tip"
                                                        icon="mdi-close-circle-outline"
                                                        variant="text"
                                                        color="error"
                                                        size="small"
                                                        :aria-label="$t('download.queue.cancel')"
                                                        :aria-description="isRemovableQueueItem(item) ? $t('download.queue.removeItem') : undefined"
                                                        @click.stop="cancelQueueItem(item.id)"
                                                        @contextmenu="removeQueueItem($event, item)"
                                                    />
                                                </template>
                                            </v-tooltip>
                                            <v-tooltip v-if="item.retryable" :text="$t('download.queue.retry')" location="top">
                                                <template #activator="{ props: tip }">
                                                    <v-btn
                                                        v-bind="tip"
                                                        icon="mdi-reload"
                                                        variant="text"
                                                        color="primary"
                                                        size="small"
                                                        :aria-label="$t('download.queue.retry')"
                                                        :aria-description="isRemovableQueueItem(item) ? $t('download.queue.removeItem') : undefined"
                                                        @click.stop="retryQueueItem(item.id)"
                                                        @contextmenu="removeQueueItem($event, item)"
                                                    />
                                                </template>
                                            </v-tooltip>
                                        </div>
                                    </div>
                                </div>
                            </v-window-item>
                            <v-window-item value="optional">
                                <div class="download-queue-items">
                                    <div v-if="!downloadQueueReminders.length" class="download-queue-empty">
                                        {{ $t("download.queue.emptyOptional") }}
                                    </div>
                                    <div v-for="reminder in downloadQueueReminders" :key="reminder.id" class="optional-reminder">
                                        <div class="optional-reminder-header">
                                            <div class="download-queue-item-main">
                                                <div class="download-queue-item-title">{{ reminder.mainTitle || reminder.mainProjectKey }}</div>
                                                <div class="download-queue-item-meta">
                                                    {{ [reminder.minecraftVersion, reminder.modLoader].filter(Boolean).join(" · ") }}
                                                </div>
                                            </div>
                                            <div class="download-queue-actions">
                                                <v-tooltip :text="$t('download.queue.installOptional')" location="top">
                                                    <template #activator="{ props: tip }">
                                                        <v-btn
                                                            v-bind="tip"
                                                            icon="mdi-download-multiple"
                                                            variant="text"
                                                            color="primary"
                                                            size="small"
                                                            :aria-label="$t('download.queue.installOptional')"
                                                            @click.stop="installOptionalDependencies(reminder.id)"
                                                        />
                                                    </template>
                                                </v-tooltip>
                                                <v-tooltip :text="$t('download.queue.dismissOptional')" location="top">
                                                    <template #activator="{ props: tip }">
                                                        <v-btn
                                                            v-bind="tip"
                                                            icon="mdi-bell-remove-outline"
                                                            variant="text"
                                                            size="small"
                                                            :aria-label="$t('download.queue.dismissOptional')"
                                                            @click.stop="dismissOptionalReminder(reminder.id)"
                                                        />
                                                    </template>
                                                </v-tooltip>
                                            </div>
                                        </div>
                                        <div class="optional-candidates">
                                            <div
                                                v-for="candidate in reminder.dependencies"
                                                :key="candidate.projectKey"
                                                class="optional-candidate"
                                                :class="{ 'optional-candidate--disabled': candidate.disabled }"
                                            >
                                                <v-icon :icon="optionalStatusIcon(candidate.status)" size="18" />
                                                <div class="download-queue-item-main">
                                                    <div class="download-queue-item-title">{{ candidate.title || candidate.projectKey }}</div>
                                                    <div class="download-queue-item-meta">
                                                        {{ optionalCandidateMeta(candidate) }}
                                                    </div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>
                                </div>
                            </v-window-item>
                        </v-window>
                    </v-sheet>
                </v-menu>
            </div>
        </transition>
    </v-app>
</template>

<script setup lang="ts">
import SideBar from "./components/SideBar/SideBar.vue";
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { GetPreferences } from "../wailsjs/go/main/App";
import { useDownloadQueueStore } from "./stores/downloadQueue";
import { useMinecraftStore } from "./stores/minecraft";
import { initTheme, applyVuetifyTheme, stopThemeListener } from "./composables/useTheme";
import {
    afterGsapRouteLeave,
    animationModeGsap,
    animationModeOff,
    applyAnimationSettings,
    beforeGsapFabEnter,
    beforeGsapRouteEnter,
    enterGsapFab,
    enterGsapRoute,
    leaveGsapFab,
    leaveGsapRoute,
    useActiveAnimationMode,
} from "./composables/useAnimationSettings";
import type { structs } from "../wailsjs/go/models";

const themeDark = "dark";

initTheme();

const router = useRouter();
const downloadQueueStore = useDownloadQueueStore();
const minecraftStore = useMinecraftStore();
const activeAnimationMode = useActiveAnimationMode();
const downloadQueueOpen = ref(false);
const downloadQueueTab = ref("downloads");
const visibleQueueSnapshot = ref<DownloadQueueSnapshot | null>(null);

type OptionalDependencyReminderSnapshot = {
    id: string;
    mainProjectKey: string;
    mainTitle: string;
    mainVersionId: string;
    minecraftVersion: string;
    modLoader: string;
    dependencies?: structs.OptionalDependencyCandidate[];
};

type DownloadQueueSnapshot = {
    active: boolean;
    pending: number;
    running: number;
    messageCount: number;
    bytesComplete: number;
    totalBytes: number;
    bytesPerSecond: number;
    items?: structs.DownloadQueueItem[];
    optionalReminders?: OptionalDependencyReminderSnapshot[];
};

const cloneDownloadQueue = (queue: DownloadQueueSnapshot): DownloadQueueSnapshot => ({
    active: Boolean(queue.active),
    pending: queue.pending || 0,
    running: queue.running || 0,
    messageCount: queue.messageCount || 0,
    bytesComplete: queue.bytesComplete || 0,
    totalBytes: queue.totalBytes || 0,
    bytesPerSecond: queue.bytesPerSecond || 0,
    items: (queue.items || []).map((item) => ({ ...item })),
    optionalReminders: (queue.optionalReminders || []).map((reminder) => ({
        ...reminder,
        dependencies: (reminder.dependencies || []).map((candidate) => ({ ...candidate })),
    })),
});

const visibleDownloadQueue = computed(() => (
    downloadQueueStore.hasAnyQueueSurface
        ? downloadQueueStore.queue
        : (visibleQueueSnapshot.value || downloadQueueStore.queue)
));
const visibleActiveDownloadCount = computed(() => (visibleDownloadQueue.value.pending || 0) + (visibleDownloadQueue.value.running || 0));
const queueBadgeCount = computed(() => visibleActiveDownloadCount.value + (visibleDownloadQueue.value.messageCount || 0));
const totalProgressPercent = computed(() => progressPercent(
    visibleDownloadQueue.value.bytesComplete,
    visibleDownloadQueue.value.totalBytes,
));
const queueSurfaceVisible = computed(() => downloadQueueStore.hasAnyQueueSurface);
const reminderOnly = computed(() => downloadQueueStore.reminderOnly);
const queueFabIcon = computed(() => (reminderOnly.value ? "mdi-bell-outline" : "mdi-download"));
const downloadQueueItems = computed(() => visibleDownloadQueue.value.items || []);
const downloadQueueReminders = computed(() => visibleDownloadQueue.value.optionalReminders || []);

const gsapAnimationsActive = computed(() => activeAnimationMode.value === animationModeGsap);
const animationsOff = computed(() => activeAnimationMode.value === animationModeOff);

// Route transition — one <transition>, props swapped per mode. `appear` makes the
// first paint (initial load / reload on any route) animate too, so gsap-owned
// entrances are never skipped and content can't get stuck hidden.
const routeTransitionProps = computed(() => {
    if (animationsOff.value) {
        return { css: false, mode: "out-in" as const };
    }
    if (gsapAnimationsActive.value) {
        return {
            css: false,
            appear: true,
            mode: "out-in" as const,
            onBeforeEnter: beforeGsapRouteEnter,
            onEnter: enterGsapRoute,
            onLeave: leaveGsapRoute,
            onAfterLeave: afterGsapRouteLeave,
        };
    }
    return { name: "slide-fade", appear: true, mode: "out-in" as const };
});

// FAB transition — each mode owns one layer, no cross-mode leakage:
//   off     → no-op (only the snapshot cleanup hook)
//   vuetify → CSS `md-fab-*` classes
//   gsap    → GSAP hooks
const fabTransitionProps = computed(() => {
    if (animationsOff.value) {
        return { css: false, onAfterLeave: clearDownloadQueueSnapshot };
    }
    if (gsapAnimationsActive.value) {
        return {
            css: false,
            onBeforeEnter: beforeGsapFabEnter,
            onEnter: enterGsapFab,
            onLeave: leaveGsapFab,
            onAfterLeave: clearDownloadQueueSnapshot,
        };
    }
    return { name: "md-fab", onAfterLeave: clearDownloadQueueSnapshot };
});

const queueStatusIcon = (status: string) => {
    switch (status) {
        case "running":
            return "mdi-progress-download";
        case "pending":
            return "mdi-clock-outline";
        case "failed":
            return "mdi-alert-circle-outline";
        case "canceled":
            return "mdi-cancel";
        default:
            return "mdi-download";
    }
};

const queueItemMeta = (item: structs.DownloadQueueItem) =>
    [item.platform, item.minecraftVersion, item.modLoader, item.fileName].filter(Boolean).join(" · ");

const progressPercent = (complete: number, total: number) => {
    if (!Number.isFinite(complete) || !Number.isFinite(total) || total <= 0) return 0;
    return Math.round(Math.min(100, Math.max(0, (complete / total) * 100)));
};

const itemProgressPercent = (item: structs.DownloadQueueItem) =>
    progressPercent(item.bytesComplete, item.totalBytes);

const isActiveQueueItem = (item: structs.DownloadQueueItem) =>
    item.status === "pending" || item.status === "running";

const formatBytes = (bytes: number) => {
    const value = Number.isFinite(bytes) ? Math.max(0, bytes) : 0;
    const units = ["B", "KiB", "MiB", "GiB", "TiB"];
    let scaled = value;
    let unitIndex = 0;
    while (scaled >= 1024 && unitIndex < units.length - 1) {
        scaled /= 1024;
        unitIndex += 1;
    }
    const digits = unitIndex === 0 || scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2;
    return `${Number(scaled.toFixed(digits))} ${units[unitIndex]}`;
};

const formatSpeed = (bytesPerSecond: number) => `${formatBytes(bytesPerSecond)}/s`;

const optionalStatusIcon = (status: string) => {
    switch (status) {
        case "installed":
            return "mdi-check";
        case "update":
            return "mdi-arrow-up-bold-circle";
        case "conflict":
        case "incompatible":
            return "mdi-alert-octagon-outline";
        default:
            return "mdi-download";
    }
};

const optionalCandidateMeta = (candidate: structs.OptionalDependencyCandidate) =>
    [candidate.platform, candidate.versionId, candidate.status, candidate.reason].filter(Boolean).join(" · ");

const cancelQueueItem = (id: string) => {
    void downloadQueueStore.cancel(id);
};

const retryQueueItem = (id: string) => {
    void downloadQueueStore.retry(id);
};

const isRemovableQueueItem = (item: structs.DownloadQueueItem) =>
    item.status === "pending" || item.status === "failed" || item.status === "canceled";

const removeQueueItem = (event: MouseEvent, item: structs.DownloadQueueItem) => {
    if (!isRemovableQueueItem(item)) return;
    event.preventDefault();
    event.stopPropagation();
    void downloadQueueStore.remove(item.id);
};

const dismissOptionalReminder = (id: string) => {
    void downloadQueueStore.dismissOptionalReminder(id);
};

const installOptionalDependencies = (id: string) => {
    void downloadQueueStore.installOptionalDependencies(id);
};

const clearQueueMessages = () => {
    if (reminderOnly.value) {
        void downloadQueueStore.clearOptionalReminders();
    }
};

watch(
    () => downloadQueueStore.hasVisibleItems,
    (hasVisibleItems) => {
        if (!hasVisibleItems) {
            downloadQueueOpen.value = false;
        }
    }
);

watch(
    () => downloadQueueReminders.value.length,
    (count) => {
        if (count > 0 && !downloadQueueItems.value.length) {
            downloadQueueTab.value = "optional";
        } else if (count === 0 && downloadQueueTab.value === "optional") {
            downloadQueueTab.value = "downloads";
        }
    }
);

watch(
    () => downloadQueueStore.queue,
    (queue) => {
        if (downloadQueueStore.hasAnyQueueSurface) {
            visibleQueueSnapshot.value = cloneDownloadQueue(queue);
        }
    },
    { deep: true, immediate: true },
);

function clearDownloadQueueSnapshot() {
    if (!downloadQueueStore.hasAnyQueueSurface) {
        visibleQueueSnapshot.value = null;
    }
}

const isEditableTarget = (target: EventTarget | null) => {
    if (!(target instanceof HTMLElement)) return false;
    if (target.isContentEditable) return true;
    return Boolean(target.closest("input, textarea, select, [contenteditable='true']"));
};

const onGlobalEscape = (event: KeyboardEvent) => {
    if (event.key !== "Escape") return;
    if (event.defaultPrevented || isEditableTarget(event.target)) return;

    const hasModal = document.querySelector(
        ".v-overlay--active:not(.v-overlay--contained):not(.v-snackbar):not(.v-tooltip)"
    );
    if (hasModal) return;

    if (router.currentRoute.value.path === "/") return;

    router.back();
};

onMounted(async () => {
    document.addEventListener("keydown", onGlobalEscape);
    const preferences = await GetPreferences();
    applyVuetifyTheme(preferences?.theme ?? themeDark);
    applyAnimationSettings(preferences);
    void downloadQueueStore.start();
    void minecraftStore.start();
});

onUnmounted(() => {
    document.removeEventListener("keydown", onGlobalEscape);
    downloadQueueStore.stop();
    minecraftStore.stop();
    stopThemeListener();
});
</script>

<style>
.download-fab {
    bottom: 24px;
    position: fixed;
    right: 24px;
    z-index: 1000;
}

.download-queue-panel {
    border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
    border-radius: 8px;
    max-height: min(420px, calc(100vh - 120px));
    overflow: hidden;
    width: min(420px, calc(100vw - 32px));
}

.download-queue-header {
    display: grid;
    gap: 12px;
    padding: 14px 14px 10px;
}

.download-queue-heading {
    align-items: center;
    display: flex;
    gap: 16px;
    justify-content: space-between;
}

.download-queue-title {
    font-size: 0.95rem;
    font-weight: 700;
    line-height: 1.3;
}

.download-queue-summary {
    color: rgba(var(--v-theme-on-surface), 0.68);
    font-size: 0.78rem;
    line-height: 1.4;
    margin-top: 2px;
}

.download-queue-aggregate {
    display: grid;
    gap: 6px;
}

.download-progress-labels,
.download-item-progress-meta,
.download-total-speed {
    align-items: center;
    color: rgba(var(--v-theme-on-surface), 0.68);
    display: flex;
    font-size: 0.72rem;
    gap: 6px;
    justify-content: space-between;
    line-height: 1.3;
}

.download-total-speed {
    justify-content: flex-start;
}

.download-total-speed strong {
    color: rgb(var(--v-theme-on-surface));
    font-weight: 650;
    margin-left: auto;
}

.download-queue-tabs {
    min-height: 40px;
}

.download-queue-items {
    max-height: 340px;
    overflow-y: auto;
    padding: 6px;
}

.download-queue-empty {
    color: rgba(var(--v-theme-on-surface), 0.62);
    font-size: 0.82rem;
    padding: 18px 12px;
    text-align: center;
}

.download-queue-item {
    align-items: center;
    border-radius: 6px;
    display: grid;
    gap: 10px;
    grid-template-columns: 32px minmax(0, 1fr) auto;
    min-height: 64px;
    padding: 8px;
}

.download-queue-item + .download-queue-item {
    border-top: 1px solid rgba(var(--v-border-color), 0.22);
}

.download-queue-status {
    align-items: center;
    border-radius: 50%;
    display: flex;
    height: 32px;
    justify-content: center;
    width: 32px;
}

.download-queue-status--running {
    background: rgba(var(--v-theme-primary), 0.12);
    color: rgb(var(--v-theme-primary));
}

.download-queue-status--pending {
    background: rgba(var(--v-theme-info), 0.12);
    color: rgb(var(--v-theme-info));
}

.download-queue-status--failed {
    background: rgba(var(--v-theme-error), 0.12);
    color: rgb(var(--v-theme-error));
}

.download-queue-status--canceled {
    background: rgba(var(--v-theme-on-surface), 0.08);
    color: rgba(var(--v-theme-on-surface), 0.72);
}

.download-queue-item-main {
    min-width: 0;
}

.download-item-progress {
    display: grid;
    gap: 4px;
    margin-top: 7px;
    min-height: 23px;
}

.download-queue-item-title,
.download-queue-item-meta,
.download-queue-item-reason {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.download-queue-item-title {
    font-size: 0.9rem;
    font-weight: 650;
    line-height: 1.35;
}

.download-queue-item-meta {
    color: rgba(var(--v-theme-on-surface), 0.64);
    font-size: 0.76rem;
    line-height: 1.35;
    margin-top: 2px;
}

.download-queue-item-reason {
    color: rgb(var(--v-theme-error));
    font-size: 0.74rem;
    line-height: 1.35;
    margin-top: 2px;
}

.download-queue-actions {
    align-items: center;
    display: flex;
    gap: 2px;
}

.optional-reminder {
    border-radius: 6px;
    padding: 8px;
}

.optional-reminder + .optional-reminder {
    border-top: 1px solid rgba(var(--v-border-color), 0.22);
}

.optional-reminder-header {
    align-items: center;
    display: grid;
    gap: 10px;
    grid-template-columns: minmax(0, 1fr) auto;
}

.optional-candidates {
    display: grid;
    gap: 4px;
    margin-top: 8px;
}

.optional-candidate {
    align-items: center;
    border-radius: 6px;
    display: grid;
    gap: 8px;
    grid-template-columns: 22px minmax(0, 1fr);
    min-height: 42px;
    padding: 6px 8px;
}

.optional-candidate--disabled {
    opacity: 0.58;
}

.main-loading-overlay {
    cursor: not-allowed;
}
</style>
