import { describe, expect, it } from "vitest";

import {
  ConfirmStatuses,
  DownloadErrorKeys,
  DownloadStatus,
  MetadataRetryStatuses,
} from "@/lib/downloadStatus";

describe("downloadStatus", () => {
  it("keeps the strict missing-URL state terminal", () => {
    expect(DownloadStatus.MissingDownloadUrl).toBe("missingDownloadUrl");
    expect(MetadataRetryStatuses.has(DownloadStatus.MissingDownloadUrl)).toBe(false);
    expect(ConfirmStatuses.has(DownloadStatus.MissingDownloadUrl)).toBe(false);
  });

  it("maps the missing-URL skip reason to the friendly message", () => {
    expect(DownloadErrorKeys["missing download url"]).toBe("missingDownloadUrl");
  });
});
