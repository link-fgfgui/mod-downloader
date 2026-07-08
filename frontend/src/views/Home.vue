<template>
  <v-container class="home-page pa-6 md-page" fluid>
    <v-sheet class="home-panel pa-6 pa-sm-8" elevation="0">
      <section class="home-header">
        <div class="home-header-icon">
          <v-icon icon="mdi-cube-send" size="42"></v-icon>
        </div>
        <div class="home-header-copy">
          <div class="text-overline text-primary font-weight-bold">
            {{ $t("home.kicker") }}
          </div>
          <h1 class="text-h4 font-weight-bold">
            {{ $t("home.title") }}
          </h1>
          <p class="text-body-1 text-medium-emphasis">
            {{ $t("home.subtitle") }}
          </p>
        </div>
      </section>

      <v-divider class="my-6"></v-divider>

      <v-list class="home-workflows bg-transparent pa-0" lines="two">
        <v-list-item
          v-for="item in workflowItems"
          :key="item.to"
          class="home-workflow-item md-hover-lift"
          rounded="lg"
          :to="item.to"
        >
          <template #prepend>
            <v-avatar class="home-workflow-icon" rounded="lg" size="44">
              <v-icon :icon="item.icon" size="24"></v-icon>
            </v-avatar>
          </template>

          <v-list-item-title class="home-workflow-title font-weight-medium">
            {{ $t(item.titleKey) }}
          </v-list-item-title>
          <v-list-item-subtitle class="home-workflow-description">
            {{ $t(item.descriptionKey) }}
          </v-list-item-subtitle>

          <template #append>
            <v-icon class="home-workflow-arrow" icon="mdi-arrow-right" size="20"></v-icon>
          </template>
        </v-list-item>
      </v-list>
    </v-sheet>
  </v-container>
</template>

<script setup lang="ts">
const workflowItems = [
  {
    to: "/download",
    icon: "mdi-file-download-outline",
    titleKey: "home.workflows.download.title",
    descriptionKey: "home.workflows.download.description",
  },
  {
    to: "/manage",
    icon: "mdi-list-status",
    titleKey: "home.workflows.manage.title",
    descriptionKey: "home.workflows.manage.description",
  },
  {
    to: "/favorites",
    icon: "mdi-playlist-star",
    titleKey: "home.workflows.favorites.title",
    descriptionKey: "home.workflows.favorites.description",
  },
  {
    to: "/unpin",
    icon: "mdi-pin-off-outline",
    titleKey: "home.workflows.unpin.title",
    descriptionKey: "home.workflows.unpin.description",
  },
  {
    to: "/settings",
    icon: "mdi-cog-outline",
    titleKey: "home.workflows.settings.title",
    descriptionKey: "home.workflows.settings.description",
  },
];
</script>

<style scoped>
.home-page {
  align-items: center;
  display: flex;
  min-height: calc(100vh - 24px);
}

.home-panel {
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 8px;
  margin: 0 auto;
  max-width: 980px;
  width: 100%;
}

.home-header {
  align-items: center;
  display: flex;
  gap: 20px;
  text-align: left;
}

.home-header-icon {
  align-items: center;
  background: rgb(var(--v-theme-primary-container));
  border-radius: 8px;
  color: rgb(var(--v-theme-on-primary-container));
  display: flex;
  flex: 0 0 72px;
  height: 72px;
  justify-content: center;
  width: 72px;
}

.home-header-copy {
  min-width: 0;
}

.home-header-copy p {
  margin: 8px 0 0;
  max-width: 680px;
}

.home-workflows {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.home-workflow-item {
  background: rgba(var(--v-theme-surface-variant), 0.34);
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  min-height: 88px;
  padding-inline: 16px;
  text-align: left;
}

.home-workflow-icon {
  background: rgb(var(--v-theme-surface));
  color: rgb(var(--v-theme-primary));
}

.home-workflow-title,
.home-workflow-description {
  white-space: normal;
}

.home-workflow-description {
  line-height: 1.35;
}

.home-workflow-arrow {
  color: rgb(var(--v-theme-primary));
}

@media (max-width: 600px) {
  .home-page {
    align-items: stretch;
    padding: 16px !important;
  }

  .home-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .home-header-icon {
    flex-basis: 60px;
    height: 60px;
    width: 60px;
  }

  .home-workflows {
    grid-template-columns: 1fr;
  }
}
</style>
