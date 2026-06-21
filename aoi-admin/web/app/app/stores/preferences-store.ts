import { create } from "zustand";

import {
  loadStoredThemeMode,
  persistThemeMode,
  resolveThemeMode,
  type ResolvedThemeMode,
  type ThemeMode,
} from "~/features/preferences/theme";

type PreferencesState = {
  hydrated: boolean;
  prefersDark: boolean;
  resolvedThemeMode: ResolvedThemeMode;
  setSystemPrefersDark: (prefersDark: boolean) => void;
  setThemeMode: (mode: ThemeMode) => void;
  themeMode: ThemeMode;
  hydrateThemeMode: () => void;
};

export const usePreferencesStore = create<PreferencesState>((set, get) => ({
  hydrated: false,
  prefersDark: false,
  resolvedThemeMode: "light",
  setSystemPrefersDark: (prefersDark) => {
    set({
      prefersDark,
      resolvedThemeMode: resolveThemeMode(get().themeMode, prefersDark),
    });
  },
  setThemeMode: (themeMode) => {
    persistThemeMode(themeMode, browserStorage());
    set({
      resolvedThemeMode: resolveThemeMode(themeMode, get().prefersDark),
      themeMode,
    });
  },
  themeMode: "system",
  hydrateThemeMode: () => {
    const themeMode = loadStoredThemeMode(browserStorage());
    set({
      hydrated: true,
      resolvedThemeMode: resolveThemeMode(themeMode, get().prefersDark),
      themeMode,
    });
  },
}));

function browserStorage() {
  return typeof window === "undefined" ? undefined : window.localStorage;
}
