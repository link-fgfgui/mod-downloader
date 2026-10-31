import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import en from "./locales/en";
import zh from "./locales/zh";

export const languageSystem = "system";
export const languageChinese = "zh";
export const languageEnglish = "en";
export type LanguagePreference = "system" | "zh" | "en";
export type AppLocale = "zh" | "en";

export function normalizeLanguagePreference(value: unknown): LanguagePreference {
  const normalized = String(value ?? "").trim().toLowerCase();
  if (["zh", "zh-cn", "zh-hans"].includes(normalized)) return "zh";
  if (normalized === "en") return "en";
  return "system";
}

export function resolveLanguagePreference(preference: unknown, systemLocales?: readonly string[]): AppLocale {
  const normalized = normalizeLanguagePreference(preference);
  if (normalized !== "system") return normalized;
  const locales = systemLocales ?? (typeof navigator === "undefined" ? [] : [...(navigator.languages || []), navigator.language]);
  const primary = locales.find((locale) => String(locale).trim()) ?? "";
  return primary.toLowerCase().startsWith("zh") ? "zh" : "en";
}

void i18n.use(initReactI18next).init({
  resources: { zh: { translation: zh }, en: { translation: en } },
  lng: resolveLanguagePreference("system"),
  fallbackLng: "en",
  interpolation: { escapeValue: false, prefix: "{", suffix: "}" },
});

export async function applyLanguagePreference(preference: unknown): Promise<AppLocale> {
  const locale = resolveLanguagePreference(preference);
  await i18n.changeLanguage(locale);
  if (typeof document !== "undefined") document.documentElement.lang = locale === "zh" ? "zh-CN" : "en";
  return locale;
}

export function currentLocale(): AppLocale {
  return i18n.language === "zh" ? "zh" : "en";
}

export default i18n;
