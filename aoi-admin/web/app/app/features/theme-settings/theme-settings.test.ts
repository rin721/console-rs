import { describe, expect, test } from "vitest";

import { activeThemePackage } from "~/theme/generated/theme-metadata";
import * as generatedTemplates from "~/theme/generated/templates";
import builtinThemePackage from "~/theme/packages/builtin/aoi/theme.json";
import customThemePackage from "~/theme/packages/example/custom/theme.json";
import { aoiThemePackageSchema, requiredThemeTemplateKeys } from "~/theme/schema";

import {
  calculateContrastRatio,
  contrastStatus,
  defaultThemeDraft,
  parseThemeImport,
  serializeThemeDraft,
} from "./theme-settings";

describe("theme settings helpers", () => {
  test("validates builtin and example source theme packages", () => {
    expect(aoiThemePackageSchema.safeParse(builtinThemePackage).success).toBe(true);
    expect(aoiThemePackageSchema.safeParse(customThemePackage).success).toBe(true);
  });

  test("uses active theme package defaults for local token drafts", () => {
    expect(defaultThemeDraft).toEqual(activeThemePackage.tokens.themeSettingsDefaults);
  });

  test("rejects remote theme asset URLs", () => {
    const result = aoiThemePackageSchema.safeParse({
      ...builtinThemePackage,
      assets: {
        ...builtinThemePackage.assets,
        images: ["https://assets.example.test/theme.png"],
      },
    });

    expect(result.success).toBe(false);
  });

  test("exports all required theme templates from the generated active package", () => {
    const exportNames = {
      adminLayout: "AdminThemeLayout",
      adminSidebarNav: "AdminSidebarNav",
      authTemplate: "AuthThemeTemplate",
      dashboardTemplate: "DashboardThemeTemplate",
      detailPageTemplate: "DetailPageThemeTemplate",
      errorTemplate: "ErrorThemeTemplate",
      listPageTemplate: "ListPageThemeTemplate",
      loadingTemplate: "LoadingThemeTemplate",
      publicLayout: "PublicThemeLayout",
      settingsTemplate: "SettingsThemeTemplate",
      setupLayout: "SetupThemeLayout",
    } satisfies Record<(typeof requiredThemeTemplateKeys)[number], keyof typeof generatedTemplates>;

    for (const key of requiredThemeTemplateKeys) {
      expect(typeof generatedTemplates[exportNames[key]]).toBe("function");
    }
  });

  test("calculates WCAG contrast status for readable text", () => {
    const ratio = calculateContrastRatio("#202322", "#ffffff");

    expect(ratio).toBeGreaterThanOrEqual(4.5);
    expect(contrastStatus(ratio)).toBe("pass");
  });

  test("flags low-contrast pairs as failing normal text requirements", () => {
    const ratio = calculateContrastRatio("#aaaaaa", "#ffffff");

    expect(ratio).toBeLessThan(4.5);
    expect(contrastStatus(ratio)).toBe("fail");
  });

  test("imports a strict theme draft and normalizes hex casing", () => {
    const result = parseThemeImport(
      JSON.stringify({
        ...defaultThemeDraft,
        primaryColor: "#2F6F6A",
      }),
    );

    expect(result).toMatchObject({
      draft: {
        primaryColor: "#2f6f6a",
      },
      ok: true,
    });
  });

  test("rejects arbitrary CSS-like color payloads", () => {
    const result = parseThemeImport(
      JSON.stringify({
        ...defaultThemeDraft,
        surfaceColor: "background:url(javascript:alert(1))",
      }),
    );

    expect(result).toEqual({ message: "invalid-schema", ok: false });
  });

  test("rejects unknown draft fields instead of silently accepting new token scope", () => {
    const result = parseThemeImport(
      JSON.stringify({
        ...defaultThemeDraft,
        unsafeToken: "#000000",
      }),
    );

    expect(result).toEqual({ message: "invalid-schema", ok: false });
  });

  test("serializes stable import and export payloads", () => {
    expect(serializeThemeDraft(defaultThemeDraft)).toContain('"primaryColor": "#d95f7f"');
  });
});
