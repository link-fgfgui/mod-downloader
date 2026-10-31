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
  apply();

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
