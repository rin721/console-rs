import { readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, extname, join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const rootDir = dirname(dirname(fileURLToPath(import.meta.url)));
const appDir = join(rootDir, "app");
const tokensPath = join(appDir, "components", "aoi", "tokens", "tokens.css");
const tokenSource = readFileSync(tokensPath, "utf8");
const definedVariables = new Set(
  [...tokenSource.matchAll(/(--aoi-[a-z0-9-]+)\s*:/g)].map((match) => match[1]),
);
const allowedUndeclared = new Set([
  "--aoi-media-category-depth",
  "--theme-preview-motion",
  "--theme-preview-primary",
  "--theme-preview-radius",
  "--theme-preview-shadow",
  "--theme-preview-spacing",
  "--theme-preview-surface",
  "--theme-preview-text",
  "--theme-preview-type",
]);
const sourceFiles = collectFiles(appDir).filter(shouldScan);
const undefinedVariables = [];
const hardcodedColors = [];
const hardcodedShadows = [];
const hardcodedZIndexes = [];

for (const file of sourceFiles) {
  const source = readFileSync(file, "utf8");
  for (const match of source.matchAll(/var\((--aoi-[a-z0-9-]+)/g)) {
    const variable = match[1];
    if (!definedVariables.has(variable) && !allowedUndeclared.has(variable)) {
      undefinedVariables.push(
        `${relative(rootDir, file)}:${lineForOffset(source, match.index)} ${variable}`,
      );
    }
  }
  if (isRuntimeSource(file)) {
    for (const match of source.matchAll(/#[0-9a-fA-F]{3,8}|rgba?\(/g)) {
      hardcodedColors.push(
        `${relative(rootDir, file)}:${lineForOffset(source, match.index)} ${match[0]}`,
      );
    }
    for (const match of source.matchAll(/box-shadow\s*:\s*([^;]+)/g)) {
      const value = match[1].trim();
      if (!value.startsWith("var(") && value !== "none") {
        hardcodedShadows.push(
          `${relative(rootDir, file)}:${lineForOffset(source, match.index)} box-shadow: ${value}`,
        );
      }
    }
    for (const match of source.matchAll(/z-index\s*:\s*([^;]+)/g)) {
      const value = match[1].trim();
      if (
        !value.startsWith("var(") &&
        !["auto", "inherit", "initial", "revert", "unset"].includes(value)
      ) {
        hardcodedZIndexes.push(
          `${relative(rootDir, file)}:${lineForOffset(source, match.index)} z-index: ${value}`,
        );
      }
    }
  }
}

if (
  undefinedVariables.length > 0 ||
  hardcodedColors.length > 0 ||
  hardcodedShadows.length > 0 ||
  hardcodedZIndexes.length > 0
) {
  if (undefinedVariables.length > 0) {
    console.error("Undefined Aoi theme variables:");
    for (const issue of undefinedVariables) {
      console.error(`  ${issue}`);
    }
  }
  if (hardcodedColors.length > 0) {
    console.error("Hardcoded runtime color values outside theme packages:");
    for (const issue of hardcodedColors) {
      console.error(`  ${issue}`);
    }
  }
  if (hardcodedShadows.length > 0) {
    console.error("Hardcoded runtime box-shadow values outside theme packages:");
    for (const issue of hardcodedShadows) {
      console.error(`  ${issue}`);
    }
  }
  if (hardcodedZIndexes.length > 0) {
    console.error("Hardcoded runtime z-index values outside theme packages:");
    for (const issue of hardcodedZIndexes) {
      console.error(`  ${issue}`);
    }
  }
  process.exit(1);
}

console.log("Aoi theme surface check passed.");

function collectFiles(dir) {
  return readdirSync(dir).flatMap((entry) => {
    const path = join(dir, entry);
    const stats = statSync(path);
    return stats.isDirectory() ? collectFiles(path) : [path];
  });
}

function shouldScan(file) {
  const normalized = file.replaceAll("\\", "/");
  if (normalized.includes("/theme/packages/") || normalized.includes("/theme/generated/")) {
    return false;
  }
  if (normalized.endsWith("/components/aoi/tokens/tokens.css")) {
    return false;
  }
  if (normalized.endsWith(".test.ts") || normalized.endsWith(".test.tsx")) {
    return false;
  }
  return [".css", ".ts", ".tsx"].includes(extname(file));
}

function isRuntimeSource(file) {
  return [".css", ".ts", ".tsx"].includes(extname(file));
}

function lineForOffset(source, offset = 0) {
  return source.slice(0, offset).split("\n").length;
}
