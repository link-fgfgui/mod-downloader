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
import { GetDownloadQueueState } from "../wailsjs/go/main/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

// const drawer = ref(true);

import { useI18n } from "vue-i18n";
const { locale } = useI18n();
function toggleLanguage() {
    locale.value = locale.value === "zh" ? "en" : "zh";
}

const downloadQueueUpdatedEvent = "download-queue-updated";
const downloadQueue = ref({
    active: false,
    pending: 0,
    running: 0,
});

let stopListeningDownloadQueueUpdated: (() => void) | null = null;

onMounted(async () => {
    downloadQueue.value = await GetDownloadQueueState();
    stopListeningDownloadQueueUpdated = EventsOn(downloadQueueUpdatedEvent, (state) => {
        downloadQueue.value = state || { active: false, pending: 0, running: 0 };
    });
});

onUnmounted(() => {
    stopListeningDownloadQueueUpdated?.();
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
