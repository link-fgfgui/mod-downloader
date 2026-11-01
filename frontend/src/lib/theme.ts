const themeDark = "dark";
const themeLight = "light";
const themeSystem = "system";

export type ThemePreference = "dark" | "light" | "system";

let stopListeningSystemTheme: (() => void) | null = null;

export function normalizeThemePreference(value: unknown): ThemePreference {
  const normalized = String(value ?? "").trim().toLowerCase();
  if (normalized === themeLight) return "light";
  if (normalized === themeSystem) return "system";
  return "dark";
}

export function resolveThemePreference(preference: unknown): "dark" | "light" {
  const normalized = normalizeThemePreference(preference);
  if (normalized !== themeSystem) return normalized;
  if (typeof window === "undefined" || !window.matchMedia) return "dark";
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

type ViewTransitionDocument = Document & {
  startViewTransition?: (callback: () => void) => { finished: Promise<void> };
};

/**
 * WebKitGTK crashes its web process when `startViewTransition()` is the first
 * call that enables accelerated compositing: FrameRenderer::graphicsLayerFactory()
 * hits WTFCrashWithInfo() because no layer tree host exists. That state is
 * reachable whenever hardware acceleration is disabled (WebviewGpuPolicyNever)
 * or DMA-BUF rendering is off, which Wails does automatically on NVIDIA.
 *
 * The theme cross-fade is cosmetic, so keep the fallback on WebKitGTK and apply
 * the theme directly instead of taking down the whole web view.
 *
 * https://bugs.webkit.org/show_bug.cgi?id=321683
 */
export function supportsViewTransitions(): boolean {
  if (typeof document === "undefined") return false;
  if (typeof (document as ViewTransitionDocument).startViewTransition !== "function") return false;
  if (typeof navigator === "undefined") return true;
  const ua = navigator.userAgent;
  const isWebKitGTK = /Linux/.test(ua) && /AppleWebKit/.test(ua) && !/Chrome|Chromium|Edg|OPR/.test(ua);
  return !isWebKitGTK;
}

let themeViewTransitionActive = false;

export function applyThemePreference(preference: unknown) {
  stopListeningSystemTheme?.();
  stopListeningSystemTheme = null;
  if (typeof document === "undefined") return;

  const normalized = normalizeThemePreference(preference);
  const apply = () => {
    const resolved = resolveThemePreference(normalized);
    document.documentElement.classList.toggle("dark", resolved === "dark");
    document.documentElement.style.colorScheme = resolved;
  };

  // Cross-fade the whole page only when the resolved theme actually changes.
  // Loading settings re-applies the same preference on every visit, and
  // starting a no-op transition there would interrupt an in-flight one.
  const viewTransitionDocument = document as ViewTransitionDocument;
  const reduceMotion =
    typeof window !== "undefined" &&
    window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
  const nextResolved = resolveThemePreference(normalized);
  const currentlyDark = document.documentElement.classList.contains("dark");
  const themeChanged = (nextResolved === "dark") !== currentlyDark;
  if (themeChanged && !themeViewTransitionActive && supportsViewTransitions() && !reduceMotion) {
    themeViewTransitionActive = true;
    const transition = viewTransitionDocument.startViewTransition(apply);
    void transition.finished.finally(() => {
      themeViewTransitionActive = false;
    });
  } else {
    apply();
  }

  if (normalized !== themeSystem || typeof window === "undefined" || !window.matchMedia) return;
  const query = window.matchMedia("(prefers-color-scheme: dark)");
  query.addEventListener("change", apply);
  stopListeningSystemTheme = () => query.removeEventListener("change", apply);
}

export function stopThemeListener() {
  stopListeningSystemTheme?.();
  stopListeningSystemTheme = null;
}

export { themeDark, themeLight, themeSystem };
