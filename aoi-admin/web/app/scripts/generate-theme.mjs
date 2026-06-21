import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";
import { z } from "zod";

const rootDir = dirname(dirname(fileURLToPath(import.meta.url)));
const appDir = join(rootDir, "app");
const themeConfigPath = join(appDir, "theme", "theme.config.json");
const tokensOutputPath = join(appDir, "components", "aoi", "tokens", "tokens.css");
const metadataOutputPath = join(appDir, "theme", "generated", "theme-metadata.ts");
const templatesOutputPath = join(appDir, "theme", "generated", "templates.tsx");
const checkOnly = process.argv.includes("--check");

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

const requiredTemplateKeys = [
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
];

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

const themePackageSchema = z
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
      Object.fromEntries(
        requiredTemplateKeys.map((key) => [
          key,
          z
            .object({
              description: z.string().min(1),
              label: z.string().min(1),
            })
            .strict(),
        ]),
      ),
    ),
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
        themeSettingsDefaults: z
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
          .strict(),
        variables: z
          .object({
            dark: z.record(z.string().regex(/^--aoi-[a-z0-9-]+$/), safeThemeValueSchema),
            light: z.record(z.string().regex(/^--aoi-[a-z0-9-]+$/), safeThemeValueSchema),
          })
          .strict(),
      })
      .strict(),
  })
  .strict();

const themeConfigSchema = z
  .object({
    activeThemeId: z.string().regex(/^[a-z0-9]+(?:[/-][a-z0-9]+)*$/),
  })
  .strict();

const config = themeConfigSchema.parse(readJson(themeConfigPath));
const themePath = join(appDir, "theme", "packages", config.activeThemeId, "theme.json");
const themePackage = themePackageSchema.parse(readJson(themePath));

if (themePackage.manifest.id !== config.activeThemeId) {
  fail(
    `Theme manifest id ${themePackage.manifest.id} does not match activeThemeId ${config.activeThemeId}.`,
  );
}

const missingRecipeTokens = Object.entries(themePackage.recipes).flatMap(([recipeKey, recipe]) =>
  recipe.tokens
    .filter((token) => !themePackage.tokens.variables.light[token])
    .map((token) => ({
      recipeKey,
      token,
    })),
);
if (missingRecipeTokens.length > 0) {
  fail(
    `Theme recipes reference unknown light-mode tokens: ${missingRecipeTokens
      .map((item) => `${item.recipeKey}:${item.token}`)
      .join(", ")}`,
  );
}

const cssOutput = renderTokensCss(themePackage, config.activeThemeId);
const metadataOutput = renderMetadata(themePackage, config.activeThemeId);
const templatesOutput = renderTemplates(config.activeThemeId);

writeOrCheck(tokensOutputPath, cssOutput);
writeOrCheck(metadataOutputPath, metadataOutput);
writeOrCheck(templatesOutputPath, templatesOutput);

if (!checkOnly) {
  console.log(`Generated Aoi theme '${config.activeThemeId}'.`);
}

function readJson(path) {
  return JSON.parse(readFileSync(path, "utf8"));
}

function renderTokensCss(theme, activeThemeId) {
  const lines = [
    "/*",
    ` * Generated by scripts/generate-theme.mjs from app/theme/packages/${activeThemeId}/theme.json.`,
    " * Edit the source theme package, then run pnpm theme:generate.",
    " */",
    "",
    "@theme {",
    ...renderVars(theme.tokens.tailwind, "  "),
    "}",
    "",
    ":root {",
    "  color-scheme: light;",
    ...renderVars(theme.tokens.variables.light, "  "),
    "}",
    "",
    ':root[data-theme="dark"] {',
    "  color-scheme: dark;",
    ...renderVars(theme.tokens.variables.dark, "  "),
    "}",
    "",
  ];
  return lines.join("\n");
}

function renderVars(values, prefix) {
  return Object.entries(values)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([key, value]) => `${prefix}${key}: ${value};`);
}

function renderMetadata(theme, activeThemeId) {
  return [
    "/*",
    " * Generated by scripts/generate-theme.mjs.",
    " * Do not edit by hand.",
    " */",
    'import type { AoiThemePackage } from "../schema";',
    "",
    `export const activeThemeId = ${JSON.stringify(activeThemeId)};`,
    `export const activeThemePackage = ${JSON.stringify(theme, null, 2)} satisfies AoiThemePackage;`,
    "export const themeCssVariableNames = Object.keys(activeThemePackage.tokens.variables.light);",
    "export const themeTemplateKeys = Object.keys(activeThemePackage.templates);",
    "",
  ].join("\n");
}

function renderTemplates(activeThemeId) {
  const importPath = `../packages/${activeThemeId}/templates`;
  return [
    "/*",
    " * Generated by scripts/generate-theme.mjs.",
    " * Do not edit by hand.",
    " */",
    "export {",
    "  AdminSidebarNav,",
    "  AdminThemeLayout,",
    "  AuthThemeTemplate,",
    "  DashboardThemeTemplate,",
    "  DetailPageThemeTemplate,",
    "  ErrorThemeTemplate,",
    "  ListPageThemeTemplate,",
    "  LoadingThemeTemplate,",
    "  PublicThemeLayout,",
    "  SettingsThemeTemplate,",
    "  SetupThemeLayout,",
    `} from ${JSON.stringify(importPath)};`,
    "",
  ].join("\n");
}

function writeOrCheck(path, content) {
  if (!checkOnly) {
    mkdirSync(dirname(path), { recursive: true });
    writeFileSync(path, content, "utf8");
    return;
  }
  const current = readFileSync(path, "utf8");
  if (current !== content) {
    fail(`${relative(rootDir, path)} is out of date. Run pnpm theme:generate.`);
  }
}

function fail(message) {
  console.error(message);
  process.exit(1);
}
