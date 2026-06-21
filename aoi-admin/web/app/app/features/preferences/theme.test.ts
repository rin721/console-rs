import { describe, expect, it, vi } from "vitest";

import {
  applyResolvedThemeMode,
  loadStoredThemeMode,
  persistThemeMode,
  resolveThemeMode,
  themeModeStorageKey,
} from "./theme";

describe("theme preference helpers", () => {
  it("falls back to system when local storage contains an unknown value", () => {
    const storage = {
      getItem: vi.fn(() => "sepia"),
    };

    expect(loadStoredThemeMode(storage)).toBe("system");
  });

  it("resolves system mode from the current media preference", () => {
    expect(resolveThemeMode("system", true)).toBe("dark");
    expect(resolveThemeMode("system", false)).toBe("light");
    expect(resolveThemeMode("dark", false)).toBe("dark");
  });

  it("removes persisted mode when system mode is selected", () => {
    const storage = {
      removeItem: vi.fn(),
      setItem: vi.fn(),
    };

    persistThemeMode("system", storage);

    expect(storage.removeItem).toHaveBeenCalledWith(themeModeStorageKey);
    expect(storage.setItem).not.toHaveBeenCalled();
  });

  it("applies the resolved mode to the document element", () => {
    const element = document.createElement("html");

    applyResolvedThemeMode(element, "dark");

    expect(element.dataset.theme).toBe("dark");
    expect(element.style.colorScheme).toBe("dark");
  });
});
