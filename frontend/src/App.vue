<template>
    <v-app>
        <SideBar />
        <v-main>
            <v-container fluid class="position-relative">
                <router-view v-slot="{ Component, route }">
                    <transition name="slide-fade" mode="out-in">
                        <keep-alive>
                            <component :is="Component" :key="route.path" />
                        </keep-alive>
                    </transition>
                </router-view>
            </v-container>
        </v-main>
        <transition name="md-fab">
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
import { onMounted, onUnmounted } from "vue";
import { useTheme } from "vuetify";
import { GetPreferences } from "../wailsjs/go/main/App";
import { useDownloadQueueStore } from "./stores/downloadQueue";
import { useMinecraftStore } from "./stores/minecraft";

const themeDark = "dark";
const themeLight = "light";
const themeSystem = "system";

const vuetifyTheme = useTheme();
const downloadQueueStore = useDownloadQueueStore();
const minecraftStore = useMinecraftStore();

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
    void downloadQueueStore.start();
    void minecraftStore.start();
});

onUnmounted(() => {
    downloadQueueStore.stop();
    minecraftStore.stop();
    stopListeningSystemTheme?.();
});
</script>

<style>
.download-fab {
    bottom: 24px;
    position: fixed;
    right: 24px;
    z-index: 1000;
}
</style>
