// Numeric-aware version comparison for local mod version strings.
//
// Mod versions are free-form (e.g. "1.2.3", "1.2.3+build.4", "v2.0-beta"),
// but provider `Version` fields are not always clean: CurseForge uses the
// file's display name ("Sodium 0.5.8") and local metadata may embed a loader
// or Minecraft prefix ("sodium-fabric-0.5.8.jar"). We therefore start
// comparing at the first version-looking token so a leading "sodium-fabric-"
// cannot dominate the order. Segments of different kinds (text vs number) are
// treated as incomparable instead of falling back to lexicographic order.

const VERSION_TOKEN = /v?\d+(?:[._-]\d+)*/i;

const versionBody = (value: string): string => {
    const text = (value || "").trim().toLocaleLowerCase();
    const match = VERSION_TOKEN.exec(text);
    if (!match) return text;
    return text.slice(match.index).replace(/^v(?=\d)/, "");
};

const splitSegments = (value: string): Array<number | string> =>
    versionBody(value)
        .split(/[^0-9A-Za-z]+/)
        .filter(Boolean)
        .map((segment) => (/^\d+$/.test(segment) ? Number(segment) : segment));

/** Returns -1, 0, or 1 for a < b, a == b, a > b. */
export function compareVersions(a?: string, b?: string): number {
    const left = splitSegments(a || "");
    const right = splitSegments(b || "");
    const length = Math.max(left.length, right.length);
    for (let index = 0; index < length; index += 1) {
        const l = left[index];
        const r = right[index];
        if (l === undefined) return -1;
        if (r === undefined) return 1;
        if (typeof l !== typeof r) {
            // e.g. "sodium" vs "0.5.8": no reliable ordering exists. Returning
            // zero avoids reporting a bogus downgrade for non-version text.
            return 0;
        }
        if (typeof l === "number" && typeof r === "number") {
            if (l !== r) return l < r ? -1 : 1;
            continue;
        }
        const ls = String(l);
        const rs = String(r);
        if (ls !== rs) return ls < rs ? -1 : 1;
    }
    return 0;
}

/** True when `candidate` is strictly older than `installed`. */
export function isVersionDowngrade(candidate?: string, installed?: string): boolean {
    if (!candidate || !installed) return false;
    return compareVersions(candidate, installed) < 0;
}

/** First numeric segment, used as the major version for change detection. */
export function versionMajor(value?: string): number | null {
    for (const segment of splitSegments(value || "")) {
        if (typeof segment === "number") return segment;
    }
    return null;
}

/**
 * True when both versions expose a numeric major and they differ (e.g.
 * 0.5.8 -> 1.0.0). Replacing across a major boundary is treated as risky even
 * when the candidate is technically newer.
 */
export function hasMajorVersionChange(candidate?: string, installed?: string): boolean {
    const left = versionMajor(candidate);
    const right = versionMajor(installed);
    if (left === null || right === null) return false;
    return left !== right;
}
