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
        <DownloadQueuePanel />
    </v-app>
</template>

<script setup lang="ts">
import SideBar from "./components/SideBar/SideBar.vue";
import DownloadQueuePanel from "./components/DownloadQueuePanel.vue";
import { computed, onMounted, onUnmounted } from "vue";
import { useRouter } from "vue-router";
import { GetPreferences } from "../wailsjs/go/main/App";
import { useMinecraftStore } from "./stores/minecraft";
import { initTheme, applyVuetifyTheme, stopThemeListener } from "./composables/useTheme";
import {
    afterGsapRouteLeave,
    animationModeGsap,
    animationModeOff,
    applyAnimationSettings,
    beforeGsapRouteEnter,
    enterGsapRoute,
    leaveGsapRoute,
    useActiveAnimationMode,
} from "./composables/useAnimationSettings";

const themeDark = "dark";

initTheme();

const router = useRouter();
const minecraftStore = useMinecraftStore();
const activeAnimationMode = useActiveAnimationMode();

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
