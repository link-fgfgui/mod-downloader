// Every Wails event name used by the frontend. Keep this as the only source of
// truth so renaming an event cannot silently leave a listener behind.

export const minecraftDirChangedEvent = "minecraft-dir-changed";
export const selectedVersionChangedEvent = "selected-version-changed";
export const searchModsUpdatedEvent = "search-mods-updated";
export const downloadStatesUpdatedEvent = "download-states-updated";
export const downloadQueueUpdatedEvent = "download-queue-updated";
export const downloadFailedEvent = "download-failed";
export const downloadCompletedEvent = "download-completed";
export const usageStatsUpdatedEvent = "usage-stats-updated";
export const extensionModsAcceptedEvent = "extension-mods-accepted";
