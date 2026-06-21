import en from "./locales/en.json";
import zhCN from "./locales/zh-CN.json";

export const fallbackLanguage = "zh-CN";

export const resources = {
  "zh-CN": zhCN,
  en,
} as const;

export type AppLocale = keyof typeof resources;
export type AppResource = (typeof resources)[AppLocale];
