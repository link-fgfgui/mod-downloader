import { useTheme } from "vuetify";

const themeDark = "dark";
const themeLight = "light";
const themeSystem = "system";

let stopListeningSystemTheme: (() => void) | null = null;

export function applyVuetifyTheme(theme: string) {
    stopListeningSystemTheme?.();
    stopListeningSystemTheme = null;

    const vuetifyTheme = useTheme();
    const normalized = (theme || "").trim().toLowerCase();
    if (normalized === themeLight) {
        vuetifyTheme.global.name.value = "light";
        return;
    }
    if (normalized !== themeSystem) {
        vuetifyTheme.global.name.value = "dark";
        return;
    }

    const query = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => {
        vuetifyTheme.global.name.value = query.matches ? "dark" : "light";
    };
    apply();
    query.addEventListener("change", apply);
    stopListeningSystemTheme = () => query.removeEventListener("change", apply);
}

export function stopThemeListener() {
    stopListeningSystemTheme?.();
    stopListeningSystemTheme = null;
}
