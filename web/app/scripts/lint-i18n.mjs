import { globSync, readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { join } from "node:path";

const root = new URL("..", import.meta.url);
const rootPath = fileURLToPath(root);
const locales = ["zh-CN", "en"];

function readJson(path) {
  return JSON.parse(readFileSync(new URL(path, root), "utf8"));
}

function flatten(value, prefix = "") {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return [prefix];
  }
  return Object.entries(value).flatMap(([key, child]) =>
    flatten(child, prefix ? `${prefix}.${key}` : key),
  );
}

const keySets = new Map(
  locales.map((locale) => [
    locale,
    new Set(flatten(readJson(`src/i18n/locales/${locale}.json`)).filter(Boolean)),
  ]),
);

const baseKeys = keySets.get("zh-CN");
let failed = false;
for (const locale of locales.filter((locale) => locale !== "zh-CN")) {
  const keys = keySets.get(locale);
  const missing = [...baseKeys].filter((key) => !keys.has(key));
  const extra = [...keys].filter((key) => !baseKeys.has(key));
  if (missing.length || extra.length) {
    failed = true;
    console.error(`Locale ${locale} does not match zh-CN.`);
    if (missing.length) console.error(`Missing keys:\n${missing.map((key) => `  - ${key}`).join("\n")}`);
    if (extra.length) console.error(`Extra keys:\n${extra.map((key) => `  - ${key}`).join("\n")}`);
  }
}

const sourceFiles = globSync("src/**/*.{ts,tsx}", { cwd: rootPath })
  .map((file) => file.replace(/\\/g, "/"))
  .filter((file) => !file.startsWith("src/i18n/locales/"));
const cjkHits = sourceFiles.filter((file) => /[\u4e00-\u9fff]/.test(readFileSync(join(rootPath, file), "utf8")));

if (cjkHits.length) {
  failed = true;
  console.error("Hardcoded CJK copy found outside locale resources:");
  console.error(cjkHits.map((file) => `  - ${file}`).join("\n"));
}

if (failed) {
  process.exitCode = 1;
} else {
  console.log("i18n resources are aligned.");
}
