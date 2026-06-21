import { z } from "zod";

import { activeThemePackage } from "~/theme/generated/theme-metadata";

const hexColorSchema = z
  .string()
  .trim()
  .regex(/^#[0-9a-fA-F]{6}$/);

export const themeModeSchema = z.enum(["light", "dark"]);
export const motionIntensitySchema = z.enum(["none", "reduced", "standard"]);
export const shadowLevelSchema = z.enum(["none", "soft", "standard"]);

export const themeDraftSchema = z
  .object({
    accentColor: hexColorSchema,
    mode: themeModeSchema,
    motionIntensity: motionIntensitySchema,
    primaryColor: hexColorSchema,
    radiusScale: z.number().min(0).max(24),
    shadowLevel: shadowLevelSchema,
    spacingScale: z.number().min(0.85).max(1.25),
    surfaceColor: hexColorSchema,
    textColor: hexColorSchema,
    typographyScale: z.number().min(0.92).max(1.12),
  })
  .strict();

export type ThemeDraft = z.infer<typeof themeDraftSchema>;
export type ThemeMode = z.infer<typeof themeModeSchema>;
export type MotionIntensity = z.infer<typeof motionIntensitySchema>;
export type ShadowLevel = z.infer<typeof shadowLevelSchema>;

export type ContrastStatus = "fail" | "pass";

export type ThemeImportResult =
  | {
      draft: ThemeDraft;
      ok: true;
    }
  | {
      message: string;
      ok: false;
    };

export const defaultThemeDraft = activeThemePackage.tokens
  .themeSettingsDefaults satisfies ThemeDraft;

export function normalizeThemeDraft(draft: ThemeDraft): ThemeDraft {
  return {
    ...draft,
    accentColor: normalizeHexColor(draft.accentColor),
    primaryColor: normalizeHexColor(draft.primaryColor),
    radiusScale: roundTo(draft.radiusScale, 1),
    spacingScale: roundTo(draft.spacingScale, 2),
    surfaceColor: normalizeHexColor(draft.surfaceColor),
    textColor: normalizeHexColor(draft.textColor),
    typographyScale: roundTo(draft.typographyScale, 2),
  };
}

export function normalizeHexColor(value: string) {
  const trimmed = value.trim();
  if (!hexColorSchema.safeParse(trimmed).success) {
    return trimmed;
  }
  return trimmed.toLowerCase();
}

export function parseThemeImport(rawValue: string): ThemeImportResult {
  try {
    const parsed = JSON.parse(rawValue) as unknown;
    const result = themeDraftSchema.safeParse(parsed);
    if (!result.success) {
      return { message: "invalid-schema", ok: false };
    }
    return { draft: normalizeThemeDraft(result.data), ok: true };
  } catch {
    return { message: "invalid-json", ok: false };
  }
}

export function serializeThemeDraft(draft: ThemeDraft) {
  return `${JSON.stringify(normalizeThemeDraft(draft), null, 2)}\n`;
}

export function calculateContrastRatio(foreground: string, background: string) {
  const foregroundLuminance = relativeLuminance(hexToRgb(foreground));
  const backgroundLuminance = relativeLuminance(hexToRgb(background));
  const lighter = Math.max(foregroundLuminance, backgroundLuminance);
  const darker = Math.min(foregroundLuminance, backgroundLuminance);
  return roundTo((lighter + 0.05) / (darker + 0.05), 2);
}

export function contrastStatus(ratio: number): ContrastStatus {
  return ratio >= 4.5 ? "pass" : "fail";
}

function hexToRgb(value: string) {
  const normalized = normalizeHexColor(value);
  const result = hexColorSchema.safeParse(normalized);
  if (!result.success) {
    return { blue: 0, green: 0, red: 0 };
  }

  return {
    blue: Number.parseInt(normalized.slice(5, 7), 16) / 255,
    green: Number.parseInt(normalized.slice(3, 5), 16) / 255,
    red: Number.parseInt(normalized.slice(1, 3), 16) / 255,
  };
}

function relativeLuminance(color: { blue: number; green: number; red: number }) {
  const red = linearChannel(color.red);
  const green = linearChannel(color.green);
  const blue = linearChannel(color.blue);

  return 0.2126 * red + 0.7152 * green + 0.0722 * blue;
}

function linearChannel(channel: number) {
  return channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4;
}

function roundTo(value: number, digits: number) {
  const scale = 10 ** digits;
  return Math.round(value * scale) / scale;
}
