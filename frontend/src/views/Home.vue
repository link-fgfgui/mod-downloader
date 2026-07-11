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
            <div class="metric-value">{{ formatCompactNumber(metric.value) }}</div>
            <div class="metric-label text-caption text-medium-emphasis">{{ $t(metric.labelKey) }}</div>
          </div>
        </article>
      </div>
    </section>

  </v-container>
</template>

<script setup lang="ts">
import { computed, onActivated, onDeactivated, ref } from "vue";
import { useI18n } from "vue-i18n";

import { GetUsageStats } from "../../wailsjs/go/main/App";
import type { storage } from "../../wailsjs/go/models";
import { EventsOn } from "../../wailsjs/runtime/runtime";

const usageStatsUpdatedEvent = "usage-stats-updated";

const { locale } = useI18n();
const stats = ref<storage.UsageStats | null>(null);
const isLoading = ref(false);
const loadError = ref("");
let loadSequence = 0;
let stopUsageListener: (() => void) | null = null;
let refreshQueued = false;
let active = false;

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

const formatNumber = (value: number) => new Intl.NumberFormat(locale.value).format(value);
const formatCompactNumber = (value: number) => {
  if (Math.abs(value) < 1_000) return formatNumber(value);
  const divisor = Math.abs(value) >= 500_000 ? 1_000_000 : 1_000;
  const suffix = divisor === 1_000_000 ? "M" : "K";
  return `${new Intl.NumberFormat(locale.value, { maximumSignificantDigits: 3 }).format(value / divisor)}${suffix}`;
};

const refreshStats = async () => {
  stats.value = await GetUsageStats();
};

const loadDashboard = async () => {
  const sequence = ++loadSequence;
  isLoading.value = true;
  loadError.value = "";
  try {
    const nextStats = await GetUsageStats();
    if (sequence !== loadSequence) return;
    stats.value = nextStats;
  } catch (error) {
    if (sequence === loadSequence) {
      loadError.value = error instanceof Error ? error.message : String(error);
    }
  } finally {
    if (sequence === loadSequence) isLoading.value = false;
  }
};

const startListeners = () => {
  if (!stopUsageListener) {
    stopUsageListener = EventsOn(usageStatsUpdatedEvent, (nextStats: storage.UsageStats) => {
      stats.value = nextStats;
      if (refreshQueued) return;
      refreshQueued = true;
      queueMicrotask(() => {
        refreshQueued = false;
        if (active) void refreshStats();
      });
    });
  }
};

onActivated(() => {
  active = true;
  startListeners();
  void loadDashboard();
});

onDeactivated(() => {
  active = false;
  stopUsageListener?.();
  stopUsageListener = null;
  refreshQueued = false;
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
.dashboard-header-actions {
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

@media (max-width: 900px) {
  .metric-grid {
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

  .metric-grid {
    grid-template-columns: 1fr;
  }

}
</style>
