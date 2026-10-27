<template>
    <v-app>
        <SideBar />
        <v-main>
            <div class="route-stage">
                <router-view v-slot="{ Component, route }">
                    <transition v-bind="routeTransitionProps" mode="out-in">
                        <keep-alive>
                            <component :is="Component" :key="route.path" />
                        </keep-alive>
                    </transition>
                </router-view>
            </div>
            <v-overlay
                v-model="minecraftStore.isLoading"
                contained
                persistent
                scrim="transparent"
                class="main-loading-overlay"
            />
        </v-main>
        <DownloadQueuePanel />
    </v-app>
</template>

<script setup lang="ts">
import SideBar from "./components/SideBar/SideBar.vue";
import DownloadQueuePanel from "./components/DownloadQueuePanel.vue";
import { onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { GetPreferences } from "../wailsjs/go/main/App";
import { useMinecraftStore } from "./stores/minecraft";
import { initTheme, applyVuetifyTheme, stopThemeListener } from "./composables/useTheme";
import {
    afterGsapRouteLeave,
    applyAnimationSettings,
    beforeGsapRouteEnter,
    enterGsapRoute,
    leaveGsapRoute,
} from "./composables/useAnimationSettings";

const themeDark = "dark";

initTheme();

const router = useRouter();
const minecraftStore = useMinecraftStore();

// Route transition — GSAP owns the page entrance. The leaving page is absolutely
// positioned by `leaveGsapRoute`, so the incoming page can enter at the same time
// without an `out-in` blank gap. `appear` covers the first paint and reloads.
const routeTransitionProps = {
    css: false,
    appear: true,
    onBeforeEnter: beforeGsapRouteEnter,
    onEnter: enterGsapRoute,
    onLeave: leaveGsapRoute,
    onAfterLeave: afterGsapRouteLeave,
};

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
    void minecraftStore.start();
});

onUnmounted(() => {
    document.removeEventListener("keydown", onGlobalEscape);
    minecraftStore.stop();
    stopThemeListener();
});
</script>

<style>
.main-loading-overlay {
    cursor: not-allowed;
}
</style>
