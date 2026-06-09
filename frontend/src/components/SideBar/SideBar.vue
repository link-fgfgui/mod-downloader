<template>
    <v-navigation-drawer permanent expand-on-hover :rail="!isPinned" @mouseenter="isHovered = true"
        @mouseleave="isHovered = false" @contextmenu.prevent>
        <div style="user-select: none;">
            <v-list>
                <v-list-item title="Mod Downloader">
                    <template #append><v-btn v-show="isHovered || isPinned" :icon="isPinned ? 'mdi-pin-off' : 'mdi-pin'"
                            variant="text" density="compact"
                            @click.stop="isPinned = !isPinned"></v-btn></template></v-list-item>
            </v-list>

            <v-divider></v-divider>
            <v-list density="compact" nav>
                <v-list-item prepend-icon="mdi-home" :title="$t('nav.home')" to="/"></v-list-item>

                <v-list-item prepend-icon="mdi-file-download" :title="$t('nav.download')" to="/download"></v-list-item>

                <v-list-item prepend-icon="mdi-list-status" :title="$t('nav.manage')" to="/manage"></v-list-item>
            </v-list>
            <v-divider></v-divider>
        </div>
        <template #append>
            <VersionChoose :isExpanded="isHovered || isPinned"/>
        </template>
    </v-navigation-drawer>
</template>

<script setup>
import { ref } from "vue";
import VersionChoose from "./VersionChoose.vue";
const isPinned = ref(true);
const isHovered = ref(false);

</script>
<style scoped></style>
