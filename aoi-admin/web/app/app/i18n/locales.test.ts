import { describe, expect, it } from "vitest";

import { normalizeLocale, toBackendLocale } from "./locales";
import { resources } from "./resources";

function flattenKeys(value: unknown, prefix = ""): string[] {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return [prefix];
  }

  return Object.entries(value).flatMap(([key, child]) =>
    flattenKeys(child, prefix ? `${prefix}.${key}` : key),
  );
}

describe("i18n locales", () => {
  it("keeps zh-CN and en resource keys aligned", () => {
    expect(flattenKeys(resources.en).sort()).toEqual(flattenKeys(resources["zh-CN"]).sort());
  });

  it("maps frontend locales to backend X-Locale values", () => {
    expect(toBackendLocale("zh-CN")).toBe("zh-CN");
    expect(toBackendLocale("en")).toBe("en-US");
  });

  it("normalizes browser language values", () => {
    expect(normalizeLocale("en-US")).toBe("en");
    expect(normalizeLocale("zh-Hans-CN")).toBe("zh-CN");
    expect(normalizeLocale("fr-FR")).toBe("zh-CN");
  });
});
