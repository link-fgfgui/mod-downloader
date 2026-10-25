import assert from "node:assert/strict";
import { test } from "node:test";

import { compareVersions, hasMajorVersionChange, isVersionDowngrade, versionMajor } from "../src/utils/versionCompare.ts";

test("compares numeric segments numerically", () => {
    assert.equal(compareVersions("1.10.0", "1.9.0"), 1);
    assert.equal(compareVersions("1.2.3", "1.2.3"), 0);
    assert.equal(compareVersions("1.2.0", "1.2.1"), -1);
});

test("handles separators and trailing segments", () => {
    assert.equal(compareVersions("1.2.3+build.4", "1.2.3"), 1);
    assert.equal(compareVersions("2.0.0", "2.0"), 1);
    assert.equal(compareVersions("1.20.1", "1.20.1"), 0);
});

test("ignores loader/file-name prefixes instead of comparing them lexically", () => {
    // Regression: "sodium-fabric-0.4.0.jar" used to sort above "0.5.8" because
    // the leading text segment was compared with the installed number.
    assert.equal(compareVersions("Sodium 0.5.8", "0.5.8"), 0);
    assert.equal(compareVersions("sodium-fabric-0.4.0.jar", "0.5.8"), -1);
    assert.equal(compareVersions("sodium-fabric-0.6.0.jar", "0.5.8"), 1);
    assert.equal(compareVersions("v0.4.0", "0.5.8"), -1);
});

test("treats text-versus-number segments as incomparable", () => {
    assert.equal(compareVersions("latest", "0.5.8"), 0);
    assert.equal(isVersionDowngrade("latest", "0.5.8"), false);
});

test("isVersionDowngrade only flags strictly older versions", () => {
    assert.equal(isVersionDowngrade("1.0.0", "2.0.0"), true);
    assert.equal(isVersionDowngrade("2.0.0", "2.0.0"), false);
    assert.equal(isVersionDowngrade("3.0.0", "2.0.0"), false);
    assert.equal(isVersionDowngrade("", "2.0.0"), false);
    assert.equal(isVersionDowngrade("0.4.0", "0.5.8"), true);
    assert.equal(isVersionDowngrade("0.6.0", "0.5.8"), false);
});

test("detects major version boundaries", () => {
    assert.equal(versionMajor("v2.0.0-beta"), 2);
    assert.equal(versionMajor("Sodium 0.5.8"), 0);
    assert.equal(versionMajor("no digits"), null);
    assert.equal(hasMajorVersionChange("1.0.0", "0.5.8"), true);
    assert.equal(hasMajorVersionChange("0.6.0", "0.5.8"), false);
    assert.equal(hasMajorVersionChange("latest", "0.5.8"), false);
});
