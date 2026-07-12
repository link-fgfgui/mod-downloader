<template>
    <div class="minecraft-target-fields" :class="{ 'minecraft-target-fields-stacked': stacked }">
        <v-combobox
            v-model="minecraftVersion"
            :items="versions"
            :label="minecraftVersionLabel || t('sidebar.minecraftVersion')"
            density="compact"
            hide-details
            variant="outlined"
            clearable
        ></v-combobox>
        <v-select
            v-model="modLoader"
            :items="modLoaders"
            :label="modLoaderLabel || t('sidebar.modLoader')"
            density="compact"
            hide-details
            variant="outlined"
        ></v-select>
    </div>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

const { t } = useI18n();

withDefaults(defineProps<{
    versions: string[];
    modLoaders: string[];
    minecraftVersionLabel?: string;
    modLoaderLabel?: string;
    stacked?: boolean;
}>(), {
    minecraftVersionLabel: "",
    modLoaderLabel: "",
    stacked: false,
});

const minecraftVersion = defineModel<string>("minecraftVersion", { required: true });
const modLoader = defineModel<string>("modLoader", { required: true });
</script>

<style scoped>
.minecraft-target-fields {
    display: grid;
    gap: 12px;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
}

.minecraft-target-fields-stacked {
    grid-template-columns: minmax(0, 1fr);
}

@media (max-width: 600px) {
    .minecraft-target-fields {
        grid-template-columns: minmax(0, 1fr);
    }
}
</style>
