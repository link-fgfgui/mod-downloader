<template>
    <v-app>
        <SideBar />
        <v-main>
            <v-container fluid class="position-relative">
                <router-view v-slot="{ Component, route }">
                    <transition name="fade" mode="out-in">
                        <keep-alive>
                            <component :is="Component" :key="route.path" />
                        </keep-alive>
                    </transition>
                </router-view>
            </v-container>
        </v-main>
        <div v-if="downloadQueue.active" class="download-fab">
            <v-badge :model-value="downloadQueue.pending + downloadQueue.running > 1"
                :content="downloadQueue.pending + downloadQueue.running" color="error" floating>
                <v-btn color="primary" icon size="large" aria-label="Download status">
                    <v-icon icon="mdi-download"></v-icon>
                </v-btn>
            </v-badge>
        </div>
    </v-app>
</template>

<script setup lang="ts">
import SideBar from "./components/SideBar/SideBar.vue";
import { onMounted, onUnmounted, ref } from "vue";
import { useTheme } from "vuetify";
import { GetDownloadQueueState, GetPreferences } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

const downloadQueueUpdatedEvent = "download-queue-updated";
const themeDark = "dark";
const themeLight = "light";
const themeSystem = "system";

const vuetifyTheme = useTheme();
const downloadQueue = ref({
    active: false,
    pending: 0,
    running: 0,
});

let stopListeningDownloadQueueUpdated: (() => void) | null = null;
let systemThemeQuery: MediaQueryList | null = null;
let stopListeningSystemTheme: (() => void) | null = null;

const applyVuetifyTheme = (theme: string) => {
    stopListeningSystemTheme?.();
    stopListeningSystemTheme = null;
    systemThemeQuery = null;

    const normalizedTheme = theme?.trim().toLowerCase();
    if (normalizedTheme === themeLight) {
        vuetifyTheme.global.name.value = "light";
        return;
    }
    if (normalizedTheme !== themeSystem) {
        vuetifyTheme.global.name.value = "dark";
        return;
    }

    systemThemeQuery = window.matchMedia("(prefers-color-scheme: dark)");
    const applySystemTheme = () => {
        vuetifyTheme.global.name.value = systemThemeQuery?.matches ? "dark" : "light";
    };
    applySystemTheme();
    systemThemeQuery.addEventListener("change", applySystemTheme);
    stopListeningSystemTheme = () => {
        systemThemeQuery?.removeEventListener("change", applySystemTheme);
    };
};

onMounted(async () => {
    const preferences = await GetPreferences();
    applyVuetifyTheme(preferences?.theme ?? themeDark);
    downloadQueue.value = await GetDownloadQueueState();
    stopListeningDownloadQueueUpdated = EventsOn(downloadQueueUpdatedEvent, (state) => {
        downloadQueue.value = state || { active: false, pending: 0, running: 0 };
    });
});

onUnmounted(() => {
    stopListeningDownloadQueueUpdated?.();
    stopListeningSystemTheme?.();
});
</script>

<style>
.fade-enter-active,
.fade-leave-active {
    transition: opacity 0.1s ease;
}

.fade-enter-from,
.fade-leave-to {
    opacity: 0;
}

.download-fab {
    bottom: 24px;
    position: fixed;
    right: 24px;
    z-index: 1000;
}
</style>
