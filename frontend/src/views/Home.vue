<template>
  <v-container class="home-page pa-4 pa-sm-6 md-page" fluid>
    <header class="dashboard-header">
      <div>
        <div class="text-overline text-primary font-weight-bold">{{ $t("home.kicker") }}</div>
        <h1 class="text-h5 font-weight-bold">{{ $t("home.title") }}</h1>
        <p class="dashboard-subtitle text-body-2 text-medium-emphasis">{{ $t("home.subtitle") }}</p>
      </div>
      <div class="dashboard-header-actions">
        <v-chip :color="loadError ? 'error' : 'success'" size="small" variant="tonal">
          <v-icon :icon="loadError ? 'mdi-alert-circle-outline' : 'mdi-database-check-outline'" start></v-icon>
          {{ loadError ? $t("home.status.unavailable") : $t("home.status.tracking") }}
        </v-chip>
        <v-btn
          icon="mdi-refresh"
          size="small"
          variant="text"
          :loading="isLoading"
          :aria-label="$t('home.refresh')"
          @click="loadDashboard"
        ></v-btn>
      </div>
    </header>

    <div class="dashboard-progress">
      <v-progress-linear v-if="isLoading" color="primary" indeterminate height="3"></v-progress-linear>
    </div>

    <section aria-labelledby="usage-heading">
      <div class="section-heading">
        <div>
          <h2 id="usage-heading" class="text-subtitle-1 font-weight-bold">{{ $t("home.usage.title") }}</h2>
          <div class="text-caption text-medium-emphasis">{{ $t("home.usage.sinceTracking") }}</div>
        </div>
        <div class="usage-total">
          <span class="usage-total-value">{{ formatNumber(totalOperations) }}</span>
          <span class="text-caption text-medium-emphasis">{{ $t("home.usage.total") }}</span>
        </div>
      </div>

      <div class="metric-grid">
        <article v-for="metric in metrics" :key="metric.key" class="metric-item">
          <v-avatar :color="metric.color" rounded="lg" size="40" variant="tonal">
            <v-icon :icon="metric.icon" size="21"></v-icon>
          </v-avatar>
          <div class="metric-copy">
            <div class="metric-value">{{ formatNumber(metric.value) }}</div>
            <div class="metric-label text-caption text-medium-emphasis">{{ $t(metric.labelKey) }}</div>
          </div>
        </article>
      </div>
    </section>

    <v-divider class="my-5"></v-divider>

    <section class="status-grid" aria-labelledby="status-heading">
      <h2 id="status-heading" class="status-grid-title text-subtitle-1 font-weight-bold">{{ $t("home.status.title") }}</h2>

      <article class="status-panel">
        <div class="status-panel-heading">
          <v-icon icon="mdi-cube-outline" color="primary"></v-icon>
          <span>{{ $t("home.status.instance") }}</span>
        </div>
        <dl class="status-list">
          <div>
            <dt>{{ $t("home.status.selectedInstance") }}</dt>
            <dd>{{ selectedInstanceName || $t("home.status.none") }}</dd>
          </div>
          <div>
            <dt>{{ $t("home.status.target") }}</dt>
            <dd>{{ selectedTarget || $t("home.status.none") }}</dd>
          </div>
          <div>
            <dt>{{ $t("home.status.minecraftDir") }}</dt>
            <dd :title="minecraftDir">{{ minecraftDir || $t("home.status.notConfigured") }}</dd>
          </div>
        </dl>
      </article>

      <article class="status-panel">
        <div class="status-panel-heading">
          <v-icon icon="mdi-progress-download" color="secondary"></v-icon>
          <span>{{ $t("home.status.queue") }}</span>
        </div>
        <dl class="status-list">
          <div>
            <dt>{{ $t("home.status.queueState") }}</dt>
            <dd>{{ queueStateLabel }}</dd>
          </div>
          <div>
            <dt>{{ $t("home.status.running") }}</dt>
            <dd>{{ formatNumber(queue?.running || 0) }}</dd>
          </div>
          <div>
            <dt>{{ $t("home.status.pending") }}</dt>
            <dd>{{ formatNumber(queue?.pending || 0) }}</dd>
          </div>
        </dl>
      </article>

      <article class="status-panel">
        <div class="status-panel-heading">
          <v-icon icon="mdi-database-outline" color="info"></v-icon>
          <span>{{ $t("home.status.storage") }}</span>
        </div>
        <dl class="status-list">
          <div>
            <dt>{{ $t("home.status.source") }}</dt>
            <dd>{{ settings?.mcimEnabled ? $t("home.status.mcim") : $t("home.status.official") }}</dd>
          </div>
          <div>
            <dt>{{ $t("home.status.historyStore") }}</dt>
            <dd>SQLite · mod-favs.sqlite</dd>
          </div>
          <div>
            <dt>{{ $t("home.status.updated") }}</dt>
            <dd>{{ refreshedAt || $t("home.status.notAvailable") }}</dd>
          </div>
        </dl>
      </article>
    </section>
  </v-container>
</template>

<script setup lang="ts">
import { computed, onActivated, onDeactivated, ref } from "vue";
import { useI18n } from "vue-i18n";

import {
  GetDownloadQueueState,
  GetMinecraftDir,
  GetSelectedVersion,
  GetSettings,
  GetUsageStats,
} from "../../wailsjs/go/main/App";
import type { main, storage, structs } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const downloadQueueUpdatedEvent = "download-queue-updated";
const minecraftDirChangedEvent = "minecraft-dir-changed";
const selectedVersionChangedEvent = "selected-version-changed";

const { locale, t } = useI18n();
const stats = ref<storage.UsageStats | null>(null);
const selectedInstance = ref<structs.VersionInfo | null>(null);
const queue = ref<structs.DownloadQueueState | null>(null);
const settings = ref<main.SettingsView | null>(null);
const minecraftDir = ref("");
const refreshedAt = ref("");
const isLoading = ref(false);
const loadError = ref("");
let loadSequence = 0;
let stopQueueListener: (() => void) | null = null;
let stopInstanceListener: (() => void) | null = null;
let stopDirectoryListener: (() => void) | null = null;

const totalOperations = computed(() => {
  const value = stats.value;
  if (!value) return 0;
  return value.downloadsCompleted
    + value.modsEnabled
    + value.modsDisabled
    + value.modsDeleted
    + value.favoritesAdded
    + value.packwizExports;
});

const metrics = computed(() => [
  { key: "downloads", labelKey: "home.usage.downloads", icon: "mdi-download", color: "primary", value: stats.value?.downloadsCompleted || 0 },
  { key: "enabled", labelKey: "home.usage.enabled", icon: "mdi-toggle-switch", color: "success", value: stats.value?.modsEnabled || 0 },
  { key: "disabled", labelKey: "home.usage.disabled", icon: "mdi-toggle-switch-off-outline", color: "warning", value: stats.value?.modsDisabled || 0 },
  { key: "deleted", labelKey: "home.usage.deleted", icon: "mdi-delete-outline", color: "error", value: stats.value?.modsDeleted || 0 },
  { key: "favorites", labelKey: "home.usage.favorites", icon: "mdi-playlist-star", color: "secondary", value: stats.value?.favoritesAdded || 0 },
  { key: "exports", labelKey: "home.usage.exports", icon: "mdi-package-variant-closed", color: "info", value: stats.value?.packwizExports || 0 },
]);

const selectedInstanceName = computed(() => selectedInstance.value?.name || selectedInstance.value?.id || "");
const selectedTarget = computed(() => [
  selectedInstance.value?.minecraftVersion,
  selectedInstance.value?.modLoader,
].filter(Boolean).join(" · "));
const queueStateLabel = computed(() => queue.value?.active ? t("home.status.active") : t("home.status.idle"));

const formatNumber = (value: number) => new Intl.NumberFormat(locale.value).format(value);

const refreshStats = async () => {
  stats.value = await GetUsageStats();
};

const loadDashboard = async () => {
  const sequence = ++loadSequence;
  isLoading.value = true;
  loadError.value = "";
  try {
    const [nextStats, nextInstance, nextQueue, nextSettings, nextMinecraftDir] = await Promise.all([
      GetUsageStats(),
      GetSelectedVersion(),
      GetDownloadQueueState(),
      GetSettings(),
      GetMinecraftDir(),
    ]);
    if (sequence !== loadSequence) return;
    stats.value = nextStats;
    selectedInstance.value = nextInstance;
    queue.value = nextQueue;
    settings.value = nextSettings;
    minecraftDir.value = nextMinecraftDir || "";
    refreshedAt.value = new Intl.DateTimeFormat(locale.value, { hour: "2-digit", minute: "2-digit", second: "2-digit" }).format(new Date());
  } catch (error) {
    if (sequence === loadSequence) {
      loadError.value = error instanceof Error ? error.message : String(error);
    }
  } finally {
    if (sequence === loadSequence) isLoading.value = false;
  }
};

const startListeners = () => {
  if (!stopQueueListener) {
    stopQueueListener = EventsOn(downloadQueueUpdatedEvent, (nextQueue: structs.DownloadQueueState) => {
      const completed = Boolean(queue.value?.active) && !nextQueue?.active;
      queue.value = nextQueue;
      if (completed) void refreshStats();
    });
  }
  if (!stopInstanceListener) {
    stopInstanceListener = EventsOn(selectedVersionChangedEvent, (version: structs.VersionInfo) => {
      selectedInstance.value = version;
    });
  }
  if (!stopDirectoryListener) {
    stopDirectoryListener = EventsOn(minecraftDirChangedEvent, () => {
      void loadDashboard();
    });
  }
};

onActivated(() => {
  startListeners();
  void loadDashboard();
});

onDeactivated(() => {
  stopQueueListener?.();
  stopInstanceListener?.();
  stopDirectoryListener?.();
  stopQueueListener = null;
  stopInstanceListener = null;
  stopDirectoryListener = null;
});
</script>

<style scoped>
.home-page {
  margin: 0 auto;
  max-width: 1180px;
  min-height: calc(100vh - 24px);
}

.dashboard-header,
.section-heading,
.dashboard-header-actions,
.status-panel-heading {
  align-items: center;
  display: flex;
}

.dashboard-header,
.section-heading {
  justify-content: space-between;
}

.dashboard-header {
  gap: 24px;
}

.dashboard-subtitle {
  margin: 6px 0 0;
}

.dashboard-header-actions {
  flex: 0 0 auto;
  gap: 8px;
}

.dashboard-progress {
  height: 3px;
  margin: 16px 0 18px;
}

.section-heading {
  gap: 16px;
  margin-bottom: 12px;
}

.usage-total {
  align-items: baseline;
  display: flex;
  gap: 8px;
}

.usage-total-value {
  font-size: 1.75rem;
  font-weight: 700;
  line-height: 1;
}

.metric-grid {
  display: grid;
  gap: 10px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.metric-item {
  align-items: center;
  background: rgba(var(--v-theme-surface-variant), 0.28);
  border: 1px solid rgba(var(--v-border-color), var(--v-border-opacity));
  border-radius: 6px;
  display: flex;
  gap: 12px;
  min-height: 76px;
  min-width: 0;
  padding: 12px 14px;
}

.metric-copy {
  min-width: 0;
}

.metric-value {
  font-size: 1.35rem;
  font-weight: 700;
  line-height: 1.15;
}

.metric-label {
  line-height: 1.25;
  margin-top: 3px;
}

.status-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.status-grid-title {
  grid-column: 1 / -1;
}

.status-panel {
  border-left: 3px solid rgb(var(--v-theme-primary));
  min-width: 0;
  padding: 4px 4px 4px 14px;
}

.status-panel-heading {
  font-size: 0.875rem;
  font-weight: 600;
  gap: 8px;
  margin-bottom: 10px;
}

.status-list {
  margin: 0;
}

.status-list > div {
  display: grid;
  gap: 8px;
  grid-template-columns: minmax(88px, auto) minmax(0, 1fr);
  padding: 4px 0;
}

.status-list dt {
  color: rgba(var(--v-theme-on-surface), var(--v-medium-emphasis-opacity));
  font-size: 0.75rem;
}

.status-list dd {
  font-size: 0.75rem;
  font-weight: 500;
  margin: 0;
  overflow: hidden;
  text-align: right;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 900px) {
  .metric-grid,
  .status-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 600px) {
  .dashboard-header,
  .section-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .dashboard-header-actions {
    justify-content: space-between;
    width: 100%;
  }

  .metric-grid,
  .status-grid {
    grid-template-columns: 1fr;
  }

  .status-grid-title {
    grid-column: auto;
  }
}
</style>
