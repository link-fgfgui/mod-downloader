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
            <div v-if="downloadQueueStore.queue.active" class="download-fab">
                <v-menu
                    v-model="downloadQueueOpen"
                    :close-on-content-click="false"
                    location="top end"
                    offset="12"
                >
                    <template #activator="{ props }">
                        <v-badge
                            :model-value="activeDownloadCount > 1"
                            :content="activeDownloadCount"
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
                            >
                                <v-icon :icon="downloadQueueOpen ? 'mdi-chevron-down' : 'mdi-download'"></v-icon>
                            </v-btn>
                        </v-badge>
                    </template>
                    <v-sheet class="download-queue-panel" elevation="8">
                        <div class="download-queue-header">
                            <div>
                                <div class="download-queue-title">{{ $t("download.queue.title") }}</div>
                                <div class="download-queue-summary">
                                    {{ $t("download.queue.summary", { running: downloadQueueStore.queue.running, pending: downloadQueueStore.queue.pending }) }}
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
                        <v-divider />
                        <div class="download-queue-items">
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
                                                @click.stop="cancelQueueItem(item.id)"
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
                                                @click.stop="retryQueueItem(item.id)"
                                            />
                                        </template>
                                    </v-tooltip>
                                </div>
                            </div>
                        </div>
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
const activeDownloadCount = computed(() => downloadQueueStore.activeCount);
const downloadQueueItems = computed(() => downloadQueueStore.items);

const gsapAnimationsActive = computed(() => activeAnimationMode.value === animationModeGsap);

const routeTransitionProps = computed(() => (
    gsapAnimationsActive.value
        ? {
            css: false,
            mode: "out-in" as const,
            onBeforeEnter: beforeGsapRouteEnter,
            onEnter: enterGsapRoute,
            onLeave: leaveGsapRoute,
            onAfterLeave: afterGsapRouteLeave,
        }
        : { name: "slide-fade", mode: "out-in" as const }
));
const fabTransitionProps = computed(() => (
    gsapAnimationsActive.value
        ? {
            css: false,
            onBeforeEnter: beforeGsapFabEnter,
            onEnter: enterGsapFab,
            onLeave: leaveGsapFab,
        }
        : { name: "md-fab" }
));

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

const cancelQueueItem = (id: string) => {
    void downloadQueueStore.cancel(id);
};

const retryQueueItem = (id: string) => {
    void downloadQueueStore.retry(id);
};

watch(
    () => downloadQueueStore.hasVisibleItems,
    (hasVisibleItems) => {
        if (!hasVisibleItems) {
            downloadQueueOpen.value = false;
        }
    }
);

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
    align-items: center;
    display: flex;
    gap: 16px;
    justify-content: space-between;
    padding: 14px 14px 10px;
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

.download-queue-items {
    max-height: 340px;
    overflow-y: auto;
    padding: 6px;
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

.main-loading-overlay {
    cursor: not-allowed;
}
</style>
