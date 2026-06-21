export const setupLanguageConfirmedStorageKey = "aoi-setup-language-confirmed";

export function hasConfirmedSetupLanguage() {
  if (typeof window === "undefined") {
    return false;
  }
  return window.localStorage.getItem(setupLanguageConfirmedStorageKey) === "true";
}

export function confirmSetupLanguage() {
  if (typeof window !== "undefined") {
    window.localStorage.setItem(setupLanguageConfirmedStorageKey, "true");
  }
}
