import { Events } from "@wailsio/runtime";
import * as EventNames from "../../bindings/events";
import type { DownloadFailedEvent, DownloadQueueState, SearchModsUpdate, UsageStats, VersionInfo } from "@/types";

export const appEventNames = EventNames;

export interface AppEventPayloads {
  [EventNames.MinecraftDirChanged]: string;
  [EventNames.SelectedVersionChanged]: VersionInfo;
  [EventNames.SearchModsUpdated]: SearchModsUpdate;
  [EventNames.ExtensionModsAccepted]: SearchModsUpdate;
  [EventNames.DownloadStatesUpdated]: undefined;
  [EventNames.DownloadQueueUpdated]: DownloadQueueState;
  [EventNames.DownloadFailed]: DownloadFailedEvent;
  [EventNames.DownloadCompleted]: undefined;
  [EventNames.UsageStatsUpdated]: UsageStats;
}

export function onAppEvent<K extends keyof AppEventPayloads>(
  eventName: K,
  handler: (payload: AppEventPayloads[K]) => void,
): () => void {
  return Events.On(eventName, ((event: { data: AppEventPayloads[K] }) => handler(event.data)) as never) as () => void;
}
