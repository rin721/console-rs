import type { AppLocale } from "./resources";

export const supportedLocales = ["zh-CN", "en"] as const satisfies readonly AppLocale[];
export const localeStorageKey = "aoi-locale";

export function isSupportedLocale(value: unknown): value is AppLocale {
  return typeof value === "string" && supportedLocales.includes(value as AppLocale);
}

export function toBackendLocale(locale: string): "zh-CN" | "en-US" {
  return locale === "en" ? "en-US" : "zh-CN";
}

export function normalizeLocale(value: string | undefined | null): AppLocale {
  if (!value) {
    return "zh-CN";
  }

  const normalized = value.toLowerCase();
  if (normalized === "en" || normalized.startsWith("en-")) {
    return "en";
  }
  if (normalized === "zh-cn" || normalized.startsWith("zh")) {
    return "zh-CN";
  }
  return "zh-CN";
}

export function loadStoredLocale(): AppLocale | null {
  if (typeof window === "undefined") {
    return null;
  }

  const stored = window.localStorage.getItem(localeStorageKey);
  return isSupportedLocale(stored) ? stored : null;
}

export function persistLocale(locale: AppLocale) {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(localeStorageKey, locale);
  }
}

export function detectPreferredLocale(): AppLocale {
  if (typeof navigator === "undefined") {
    return "zh-CN";
  }

  for (const language of navigator.languages || [navigator.language]) {
    const locale = normalizeLocale(language);
    if (isSupportedLocale(locale)) {
      return locale;
    }
  }

  return "zh-CN";
}
