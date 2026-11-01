import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { supportsViewTransitions } from "@/lib/theme";

const webKitGTKUserAgent =
  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15";
const chromiumUserAgent =
  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36";
const safariUserAgent =
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Safari/605.1.15";

function stubStartViewTransition() {
  Object.defineProperty(document, "startViewTransition", {
    configurable: true,
    value: () => ({ finished: Promise.resolve() }),
  });
}

describe("supportsViewTransitions", () => {
  beforeEach(() => {
    stubStartViewTransition();
  });

  afterEach(() => {
    vi.restoreAllMocks();
    Reflect.deleteProperty(document, "startViewTransition");
  });

  it("returns false when the browser has no View Transition API", () => {
    Object.defineProperty(document, "startViewTransition", { configurable: true, value: undefined });
    vi.spyOn(navigator, "userAgent", "get").mockReturnValue(chromiumUserAgent);

    expect(supportsViewTransitions()).toBe(false);
  });

  it("returns false on WebKitGTK, where the API crashes the web process", () => {
    vi.spyOn(navigator, "userAgent", "get").mockReturnValue(webKitGTKUserAgent);

    expect(supportsViewTransitions()).toBe(false);
  });

  it("returns true for Chromium on Linux", () => {
    vi.spyOn(navigator, "userAgent", "get").mockReturnValue(chromiumUserAgent);

    expect(supportsViewTransitions()).toBe(true);
  });

  it("returns true for Safari on macOS", () => {
    vi.spyOn(navigator, "userAgent", "get").mockReturnValue(safariUserAgent);

    expect(supportsViewTransitions()).toBe(true);
  });
});
