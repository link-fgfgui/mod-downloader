// Download button status values shared by the GUI.
//
// The canonical wire values live in core/modbridge (BtnStatus*) and are locked
// by core/modbridge/button_status_test.go. Keep this module as the only GUI
// place that spells them out so a rename cannot drift across components.

export const DownloadStatus = {
  New: "new",
  Installed: "installed",
  Update: "update",
  Conflict: "conflict",
  Incompatible: "incompatible",
  MetadataLoading: "metadataLoading",
  MetadataFailed: "metadataFailed",
  NoSelectedVersion: "noSelectedVersion",
  InvalidRequest: "invalidRequest",
} as const;

export type DownloadStatusValue = (typeof DownloadStatus)[keyof typeof DownloadStatus];

/** Statuses that require a confirmation dialog before queueing. */
export const ConfirmStatuses: ReadonlySet<string> = new Set([
  DownloadStatus.Update,
  DownloadStatus.Conflict,
  DownloadStatus.Incompatible,
]);

/** Statuses that trigger a metadata refresh instead of queueing. */
export const MetadataRetryStatuses: ReadonlySet<string> = new Set([
  DownloadStatus.MetadataLoading,
  DownloadStatus.MetadataFailed,
]);

/** Core skip reasons mapped to i18n keys. */
export const DownloadErrorKeys: Readonly<Record<string, string>> = {
  "invalid request": "invalidRequest",
  "no matching version": "noMatchingVersion",
  "missing download url": "missingDownloadUrl",
  "no selected version": "noSelectedVersion",
};

/** Search result page size (mirrors uistate.SearchPageSize in the TUI). */
export const SearchPageSize = 10;
