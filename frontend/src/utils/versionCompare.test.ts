import { describe, expect, it } from "vitest";

import {
  compareVersions,
  hasMajorVersionChange,
  isVersionDowngrade,
  versionMajor,
} from "@/utils/versionCompare";

describe("versionCompare", () => {
  it("compares numeric segments", () => {
    expect(compareVersions("1.2.3", "1.2.4")).toBe(-1);
    expect(compareVersions("1.10.0", "1.9.9")).toBe(1);
    expect(compareVersions("v2.0.0", "2.0.0")).toBe(0);
  });

  it("starts at the first version-looking token", () => {
    expect(isVersionDowngrade("sodium-fabric-0.5.8", "0.5.11")).toBe(true);
    expect(isVersionDowngrade("sodium-fabric-0.6.0", "0.5.11")).toBe(false);
  });

  it("does not invent an order for text vs numeric segments", () => {
    expect(compareVersions("snapshot", "1.0.0")).toBe(0);
  });

  it("detects major version changes", () => {
    expect(versionMajor("mod-1.20.1")).toBe(1);
    expect(hasMajorVersionChange("1.0.0", "0.9.0")).toBe(true);
    expect(hasMajorVersionChange("0.10.0", "0.9.0")).toBe(false);
    expect(hasMajorVersionChange("snapshot", "0.9.0")).toBe(false);
  });
});
