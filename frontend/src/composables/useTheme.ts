import { useTheme as useVuetifyTheme } from "vuetify";

const themeDark = "dark";
const themeLight = "light";
const themeSystem = "system";

let cachedTheme: ReturnType<typeof useVuetifyTheme> | null = null;
let stopListeningSystemTheme: (() => void) | null = null;

export function initTheme() {
    cachedTheme = useVuetifyTheme();
}

export function applyVuetifyTheme(theme: string) {
    stopListeningSystemTheme?.();
    stopListeningSystemTheme = null;

    if (!cachedTheme) return;

    const normalized = (theme || "").trim().toLowerCase();
    if (normalized === themeLight) {
        cachedTheme.change("light");
        return;
    }
    if (normalized !== themeSystem) {
        cachedTheme.change("dark");
        return;
    }

    const query = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
        cachedTheme!.change(query.matches ? "dark" : "light");
    };
    apply();
    query.addEventListener("change", apply);
    stopListeningSystemTheme = () => query.removeEventListener("change", apply);
}

export function stopThemeListener() {
    stopListeningSystemTheme?.();
    stopListeningSystemTheme = null;
}
