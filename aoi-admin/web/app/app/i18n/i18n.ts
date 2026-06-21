import i18next from "i18next";
import { initReactI18next } from "react-i18next";

import { fallbackLanguage, resources } from "./resources";

export const i18n = i18next.createInstance();

void i18n.use(initReactI18next).init({
  fallbackLng: fallbackLanguage,
  interpolation: {
    escapeValue: false,
  },
  lng: fallbackLanguage,
  resources: {
    "zh-CN": { translation: resources["zh-CN"] },
    en: { translation: resources.en },
  },
  returnNull: false,
});
