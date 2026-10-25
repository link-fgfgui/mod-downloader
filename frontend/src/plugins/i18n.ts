import { createI18n } from "vue-i18n";

import en from "./locales/en";
import zh from "./locales/zh";

const messages = { zh, en };

export const languageSystem = "system";
export const languageChinese = "zh";
export const languageEnglish = "en";

export type LanguagePreference = typeof languageSystem | typeof languageChinese | typeof languageEnglish;
export type AppLocale = typeof languageChinese | typeof languageEnglish;

export function normalizeLanguagePreference(value: unknown): LanguagePreference {
  const normalized = String(value ?? "").trim().toLowerCase();
  if (normalized === languageChinese || normalized === "zh-cn" || normalized === "zh-hans") {
    return languageChinese;
  }
  if (normalized === languageEnglish) return languageEnglish;
  return languageSystem;
}

export function resolveLanguagePreference(
  preference: unknown,
  systemLocales?: readonly string[],
): AppLocale {
  const normalized = normalizeLanguagePreference(preference);
  if (normalized !== languageSystem) return normalized;
  const locales = systemLocales ?? (typeof navigator === "undefined"
    ? []
    : [...(navigator.languages || []), navigator.language]);
  const primaryLocale = locales.find((locale) => String(locale).trim()) ?? "";
  return String(primaryLocale).trim().toLowerCase().startsWith("zh")
    ? languageChinese
    : languageEnglish;
}

const i18n = createI18n({
  legacy: false,
  locale: resolveLanguagePreference(languageSystem),
  fallbackLocale: "en",
  messages,
});

export function applyLanguagePreference(preference: unknown): AppLocale {
  const locale = resolveLanguagePreference(preference);
  i18n.global.locale.value = locale;
  if (typeof document !== "undefined") {
    document.documentElement.lang = locale === languageChinese ? "zh-CN" : "en";
  }
  return locale;
}

export function currentLocale(): AppLocale {
  return i18n.global.locale.value as AppLocale;
}

export default i18n;
