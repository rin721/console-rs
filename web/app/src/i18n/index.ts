import en from "./locales/en.json";
import zhCN from "./locales/zh-CN.json";
import type { Locale } from "../lib/api/types";

export const supportedLocales: Locale[] = ["zh-CN", "en"];
export const defaultLocale: Locale = "zh-CN";

const resources = {
  "zh-CN": zhCN,
  en
} as const;

type Params = Record<string, number | string | null | undefined>;

export function detectInitialLocale(): Locale {
  const stored = localStorage.getItem("console-locale");
  if (isLocale(stored)) return stored;
  const browser = navigator.language.toLowerCase();
  return browser.startsWith("en") ? "en" : defaultLocale;
}

export function isLocale(value: unknown): value is Locale {
  return typeof value === "string" && supportedLocales.includes(value as Locale);
}

export function translate(locale: Locale, key: string, params: Params = {}) {
  const value = lookup(resources[locale], key) ?? lookup(resources[defaultLocale], key) ?? key;
  return Object.entries(params).reduce(
    (current, [name, param]) => current.replaceAll(`{{${name}}}`, String(param ?? "")),
    value,
  );
}

function lookup(source: unknown, key: string): string | undefined {
  return key.split(".").reduce<unknown>((current, part) => {
    if (current && typeof current === "object" && part in current) {
      return (current as Record<string, unknown>)[part];
    }
    return undefined;
  }, source) as string | undefined;
}
