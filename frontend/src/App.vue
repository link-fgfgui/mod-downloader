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
                <v-badge :model-value="downloadQueueStore.queue.pending + downloadQueueStore.queue.running > 1"
                    :content="downloadQueueStore.queue.pending + downloadQueueStore.queue.running" color="error" floating>
                    <v-btn class="md-btn-press md-hover-scale" color="primary" icon size="large" aria-label="Download status">
                        <v-icon icon="mdi-download"></v-icon>
                    </v-btn>
                </v-badge>
            </div>
        </transition>
    </v-app>
</template>

<script setup lang="ts">
import SideBar from "./components/SideBar/SideBar.vue";
import { computed, onMounted, onUnmounted } from "vue";
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

const themeDark = "dark";

initTheme();

const router = useRouter();
const downloadQueueStore = useDownloadQueueStore();
const minecraftStore = useMinecraftStore();
const activeAnimationMode = useActiveAnimationMode();

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

.main-loading-overlay {
    cursor: not-allowed;
}
</style>
