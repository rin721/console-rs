import { z } from "zod";

const safeThemeValueSchema = z
  .string()
  .trim()
  .min(1)
  .refine((value) => !/[{}<>]/.test(value), "Theme values cannot contain CSS blocks or markup")
  .refine((value) => !/url\s*\(/i.test(value), "Theme values cannot reference URLs")
  .refine((value) => !/javascript\s*:/i.test(value), "Theme values cannot contain scripts");

const localAssetPathSchema = z
  .string()
  .trim()
  .min(1)
  .refine((value) => !/^(https?:)?\/\//i.test(value), "Theme assets must be local")
  .refine((value) => !value.includes(".."), "Theme assets cannot traverse directories")
  .refine((value) => !/[<>]/.test(value), "Theme asset paths cannot contain markup");

const themeModeValuesSchema = z.object({
  dark: z.record(z.string().regex(/^--aoi-[a-z0-9-]+$/), safeThemeValueSchema),
  light: z.record(z.string().regex(/^--aoi-[a-z0-9-]+$/), safeThemeValueSchema),
});

export const requiredThemeTemplateKeys = [
  "adminLayout",
  "adminSidebarNav",
  "authTemplate",
  "dashboardTemplate",
  "detailPageTemplate",
  "errorTemplate",
  "listPageTemplate",
  "loadingTemplate",
  "publicLayout",
  "settingsTemplate",
  "setupLayout",
] as const;

const chartPaletteSchema = z
  .object({
    border: safeThemeValueSchema,
    danger: safeThemeValueSchema,
    primary: safeThemeValueSchema,
    secondary: safeThemeValueSchema,
    success: safeThemeValueSchema,
    surface: safeThemeValueSchema,
    textPrimary: safeThemeValueSchema,
    textSecondary: safeThemeValueSchema,
    track: safeThemeValueSchema,
    warning: safeThemeValueSchema,
  })
  .strict();

const themeSettingsDefaultsSchema = z
  .object({
    accentColor: z.string().regex(/^#[0-9a-fA-F]{6}$/),
    mode: z.enum(["light", "dark"]),
    motionIntensity: z.enum(["none", "reduced", "standard"]),
    primaryColor: z.string().regex(/^#[0-9a-fA-F]{6}$/),
    radiusScale: z.number().min(0).max(24),
    shadowLevel: z.enum(["none", "soft", "standard"]),
    spacingScale: z.number().min(0.85).max(1.25),
    surfaceColor: z.string().regex(/^#[0-9a-fA-F]{6}$/),
    textColor: z.string().regex(/^#[0-9a-fA-F]{6}$/),
    typographyScale: z.number().min(0.92).max(1.12),
  })
  .strict();

const themeTemplateSchema = z
  .object({
    description: z.string().min(1),
    label: z.string().min(1),
  })
  .strict();

export const aoiThemePackageSchema = z
  .object({
    assets: z
      .object({
        fonts: z.array(localAssetPathSchema),
        icons: z.array(localAssetPathSchema),
        images: z.array(localAssetPathSchema),
      })
      .strict(),
    manifest: z
      .object({
        author: z.string().min(1),
        description: z.string().min(1),
        id: z.string().regex(/^[a-z0-9]+(?:[/-][a-z0-9]+)*$/),
        name: z.string().min(1),
        source: z.enum(["builtin", "custom", "example"]),
        version: z.string().regex(/^\d+\.\d+\.\d+$/),
      })
      .strict(),
    recipes: z.record(
      z.string().regex(/^[a-z][a-z0-9-]*$/),
      z
        .object({
          description: z.string().min(1),
          label: z.string().min(1),
          tokens: z.array(z.string().regex(/^--aoi-[a-z0-9-]+$/)),
        })
        .strict(),
    ),
    templates: z.object(
      Object.fromEntries(requiredThemeTemplateKeys.map((key) => [key, themeTemplateSchema])),
    ) as z.ZodObject<
      Record<(typeof requiredThemeTemplateKeys)[number], typeof themeTemplateSchema>
    >,
    tokens: z
      .object({
        chartPalette: z
          .object({
            dark: chartPaletteSchema,
            light: chartPaletteSchema,
          })
          .strict(),
        coverage: z.array(z.string().min(1)),
        tailwind: z.record(z.string().regex(/^--color-aoi-[a-z0-9-]+$/), safeThemeValueSchema),
        themeSettingsDefaults: themeSettingsDefaultsSchema,
        variables: themeModeValuesSchema,
      })
      .strict(),
  })
  .strict();

export type AoiThemePackage = z.infer<typeof aoiThemePackageSchema>;
export type AoiThemeTemplateKey = (typeof requiredThemeTemplateKeys)[number];
export type ThemeChartPalette = AoiThemePackage["tokens"]["chartPalette"]["light"];
