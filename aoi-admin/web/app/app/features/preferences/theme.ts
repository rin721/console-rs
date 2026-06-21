export const themeModeStorageKey = "aoi-theme-mode";
export const themeModes = ["light", "dark", "system"] as const;

export type ResolvedThemeMode = "dark" | "light";
export type ThemeMode = (typeof themeModes)[number];

export function isThemeMode(value: unknown): value is ThemeMode {
  return typeof value === "string" && themeModes.includes(value as ThemeMode);
}

export function normalizeThemeMode(value: unknown): ThemeMode {
  return isThemeMode(value) ? value : "system";
}

export function resolveThemeMode(mode: ThemeMode, prefersDark: boolean): ResolvedThemeMode {
  if (mode === "system") {
    return prefersDark ? "dark" : "light";
  }
  return mode;
}

export function loadStoredThemeMode(storage: Pick<Storage, "getItem"> | undefined): ThemeMode {
  if (!storage) {
    return "system";
  }
  return normalizeThemeMode(storage.getItem(themeModeStorageKey));
}

export function persistThemeMode(
  mode: ThemeMode,
  storage: Pick<Storage, "removeItem" | "setItem"> | undefined,
) {
  if (!storage) {
    return;
  }
  if (mode === "system") {
    storage.removeItem(themeModeStorageKey);
    return;
  }
  storage.setItem(themeModeStorageKey, mode);
}

export function applyResolvedThemeMode(
  documentElement: HTMLElement,
  resolvedMode: ResolvedThemeMode,
) {
  documentElement.dataset.theme = resolvedMode;
  documentElement.style.colorScheme = resolvedMode;
}
